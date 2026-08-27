// Package workflow 负责工作流模板的加载与参数注入。
//
// 模板是 ComfyUI 导出的 API-format JSON；与之配套的 <id>.params.json 声明哪些参数
// 可以注入、注入到哪个节点的哪个输入上。定位一律用 _meta.title（统一 dt. 前缀）而不是
// 节点 id —— 在 ComfyUI 里重排节点图会改 id，但 title 是作者自己起的，稳定得多。
package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Node 是 API-format 里的一个节点。
type Node struct {
	ClassType string         `json:"class_type"`
	Inputs    map[string]any `json:"inputs"`
	Meta      struct {
		Title string `json:"title"`
	} `json:"_meta"`
}

// Graph 是节点 id 到节点的映射。
type Graph map[string]*Node

// Param 是一条可注入参数的声明。
type Param struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Type      string `json:"type"` // string | int | float | bool | enum
	Multiline bool   `json:"multiline,omitempty"`
	Options   []any  `json:"options,omitempty"`

	// Target 是 "标题.输入名"，也可以是它们的数组（一个参数同时驱动多个输入，
	// 例如分辨率要同时改 latent 的宽和高）。
	Target json.RawMessage `json:"target"`

	// Prefix/Suffix 仅对提示词类参数有效，用来固定那些不该让用户删掉的约束
	// （例如 CHORD 需要的平光描述）。
	Prefix string `json:"prefix,omitempty"`
	Suffix string `json:"suffix,omitempty"`

	Default any      `json:"default,omitempty"`
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Note    string   `json:"note,omitempty"`

	// Hidden 表示这个参数由后端填，界面上不显示。
	// API 底图的落盘文件名走这条路：它是参数（要注入进图里），但不是用户能填的东西。
	Hidden bool `json:"hidden,omitempty"`

	// Widget 是给前端的控件提示，用于那些选项只有运行时才知道的参数
	// （例如可用的云端模型列表要现问服务端）。留空则按 Type 渲染。
	Widget string `json:"widget,omitempty"`

	// BypassWhenZero：该参数为 0 时摘掉指定节点，把它的输入直通给下游。
	// 用于"风格强度为 0 就不加载 LoRA"这类场景。
	BypassWhenZero *Bypass `json:"bypass_when_zero,omitempty"`

	// RewireWhenSet：该参数有值时改接一条连线。
	//
	// 参考图（img2img）就靠它：模板里同时留着"空 latent"和"参考图编码"两条路，
	// 没传参考图走前者，传了就把采样器的 latent 输入改接到后者。这样一套模板
	// 同时覆盖文生图和图生图，不必维护两份几乎一样的工作流。
	RewireWhenSet *Rewire `json:"rewire_when_set,omitempty"`

	// DropWhenEmpty：该参数为空时删掉这些节点。
	//
	// 与 BypassWhenZero 不同，这里是直接删除而不做直通——被删的是没有下游引用
	// 的分支（例如没传参考图时的 LoadImage 链），留着只会让 ComfyUI 去校验
	// 一个填不出值的输入。
	DropWhenEmpty []string `json:"drop_when_empty,omitempty"`

	// RewireWhenZero / DropWhenZero：该参数为 0 时改接连线、删掉节点。
	//
	// 与上面两个"空值"版本的区别在于判据是数值零而不是空串——数值参数永远
	// "有值"，走不到 DropWhenEmpty。API 底图的"直出"就靠这对：无缝重整强度
	// 调到 0 时，把分解节点的输入从采样器改接到底图本身，并把整条 SDXL 支路
	// 删掉。删掉而不是留着，是因为留着 ComfyUI 仍会校验 CheckpointLoaderSimple，
	// 于是没下过底模的人光想用 API 也会被拦住。
	RewireWhenZero []Rewire `json:"rewire_when_zero,omitempty"`
	DropWhenZero   []string `json:"drop_when_zero,omitempty"`
}

// Rewire 描述把某个节点的某个输入改接到另一个节点的某路输出。
type Rewire struct {
	Node   string `json:"node"`
	Input  string `json:"input"`
	Source string `json:"source"`
	Slot   int    `json:"slot"`
}

// Bypass 描述摘掉一个直通节点时如何改接下游。
//
// Passthrough[i] 是该节点第 i 路输出所对应的输入名——必须显式写出来，
// 因为节点的 inputs 在 JSON 里是无序对象，无法从中推断输出顺序。
type Bypass struct {
	Node        string   `json:"node"`
	Passthrough []string `json:"passthrough"`
}

func (p Param) targets() []string {
	if len(p.Target) == 0 {
		return nil
	}
	var one string
	if json.Unmarshal(p.Target, &one) == nil {
		return []string{one}
	}
	var many []string
	if json.Unmarshal(p.Target, &many) == nil {
		return many
	}
	return nil
}

// Output 描述一路产物。
type Output struct {
	Node       string `json:"node"`
	Colorspace string `json:"colorspace"`
	Y          string `json:"y,omitempty"`
	YFromParam string `json:"y_from_param,omitempty"`
	Packing    string `json:"packing,omitempty"`
	Role       string `json:"role,omitempty"`
	Note       string `json:"note,omitempty"`
}

type ModelRequirement struct {
	Kind      string `json:"kind"`
	File      string `json:"file"`
	Dir       string `json:"dir"`
	SizeBytes int64  `json:"size_bytes"`

	// Source 是给人看的页面地址（模型主页）。
	Source string `json:"source"`
	// DownloadURL 是可直接 GET 到文件本体的地址。
	//
	// 与 Source 分开是必要的：模型主页的 URL 下下来是一张 HTML，存成 .safetensors
	// 只会让 ComfyUI 在加载时报一堆莫名其妙的错。没有直链就老实让用户手动下。
	DownloadURL string `json:"download_url,omitempty"`

	// Auth：none | hf-gated | civitai-login。决定要不要带令牌，以及失败时怎么提示。
	Auth string `json:"auth"`
	Note string `json:"note,omitempty"`
}

type LicenseNotice struct {
	Component          string   `json:"component"`
	License            string   `json:"license"`
	Commercial         bool     `json:"commercial"`
	ReplaceableSegment string   `json:"replaceable_segment"`
	Alternatives       []string `json:"alternatives"`
}

// Source 声明底图从哪来。缺省（nil）表示由 ComfyUI 自己采样出图。
//
// 云端模型只负责这一步：它对提示词的理解更好，但既不可复现、也不保证无缝，
// 更不该把用户的素材整条管线都往外送。所以 PBR 分解一律留在本地。
type Source struct {
	Kind     string `json:"kind"`     // api
	Provider string `json:"provider"` // openai

	// ImageParam 是接收底图文件名的参数键。
	//
	// 后端把云端出的图上传到 ComfyUI，再把 ComfyUI 回的文件名写进这个参数——
	// 于是"注入一张图"复用了既有的参数注入机制，不必为它单开一条路径。
	ImageParam string `json:"image_param"`

	// Roles 把 API 请求字段映射到参数键，形如 {"prompt": "prompt", "size": "resolution"}。
	// 认得的字段：model、prompt、size、quality、background、reference。
	Roles map[string]string `json:"roles"`
}

// Meta 是 <id>.params.json 的内容。
type Meta struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	Name        string `json:"name"`
	Style       string `json:"style"`
	Description string `json:"description"`
	TemplateRef string `json:"template"`
	Resolution  int    `json:"resolution"`
	Tileable    bool   `json:"tileable"`

	// TileableWhenPositive 指定一个参数：它大于 0 时产物才是无缝的。
	//
	// 云端底图那两条管线需要这个——底图本身不保证无缝，只有开了本地无缝重整
	// 才真的能平铺。manifest 里的 tileable 是给消费端（UE、三维预览）看的，
	// 写死成 false 会让本来能平铺的材质被当成不能平铺，写死成 true 更糟。
	TileableWhenPositive string `json:"tileable_when_positive,omitempty"`

	// Source 非空表示底图来自外部服务，由后端先取图再提交工作流。
	Source *Source `json:"source,omitempty"`

	LicenseNotice     *LicenseNotice     `json:"license_notice,omitempty"`
	ModelRequirements []ModelRequirement `json:"model_requirements"`
	NodePacks         []string           `json:"node_packs"`

	Params   []Param           `json:"params"`
	Advanced []Param           `json:"advanced"`
	Outputs  map[string]Output `json:"outputs"`
}

// AllParams 返回基础参数与高级参数的合集。
func (m *Meta) AllParams() []Param {
	out := make([]Param, 0, len(m.Params)+len(m.Advanced))
	out = append(out, m.Params...)
	out = append(out, m.Advanced...)
	return out
}

// Template 是一份加载好的模板。零值不可用，通过 Registry 获取。
type Template struct {
	Meta Meta
	raw  []byte // 原始 API-format JSON，每次渲染都从它反序列化，天然是深拷贝
}

// Registry 持有 workflows 目录里的全部模板。
type Registry struct {
	mu   sync.RWMutex
	dir  string
	list map[string]*Template
}

func NewRegistry(dir string) *Registry {
	return &Registry{dir: dir, list: map[string]*Template{}}
}

// Load 扫描目录并载入全部 <id>.params.json 及其模板。
func (r *Registry) Load() error {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return fmt.Errorf("读取工作流目录 %s: %w", r.dir, err)
	}
	loaded := map[string]*Template{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".params.json") {
			continue
		}
		t, err := loadTemplate(r.dir, e.Name())
		if err != nil {
			return err
		}
		if _, dup := loaded[t.Meta.ID]; dup {
			return fmt.Errorf("工作流 id 重复: %s", t.Meta.ID)
		}
		loaded[t.Meta.ID] = t
	}
	if len(loaded) == 0 {
		return fmt.Errorf("%s 下没有找到任何 *.params.json 工作流声明", r.dir)
	}
	r.mu.Lock()
	r.list = loaded
	r.mu.Unlock()
	return nil
}

func loadTemplate(dir, paramsFile string) (*Template, error) {
	pb, err := os.ReadFile(filepath.Join(dir, paramsFile))
	if err != nil {
		return nil, err
	}
	var meta Meta
	if err := json.Unmarshal(pb, &meta); err != nil {
		return nil, fmt.Errorf("解析 %s: %w", paramsFile, err)
	}
	if meta.ID == "" {
		return nil, fmt.Errorf("%s 缺少 id 字段", paramsFile)
	}
	ref := meta.TemplateRef
	if ref == "" {
		ref = strings.TrimSuffix(paramsFile, ".params.json") + ".json"
	}
	raw, err := os.ReadFile(filepath.Join(dir, ref))
	if err != nil {
		return nil, fmt.Errorf("工作流 %s 的模板文件: %w", meta.ID, err)
	}
	t := &Template{Meta: meta, raw: raw}
	// 提前校验一次：声明里引用的每个 title 都必须在图里存在，
	// 免得等到用户点了生成才发现模板和声明对不上。
	g, err := t.graph()
	if err != nil {
		return nil, err
	}
	if err := validate(&meta, g); err != nil {
		return nil, fmt.Errorf("工作流 %s: %w", meta.ID, err)
	}
	return t, nil
}

func validate(m *Meta, g Graph) error {
	titles := map[string]bool{}
	for _, n := range g {
		if n.Meta.Title != "" {
			titles[n.Meta.Title] = true
		}
	}
	// 服务于外部底图来源的参数可以没有 target：它们送去调 API，不注入节点图
	// （例如云端模型名、画质档）。
	forSource := map[string]bool{}
	if m.Source != nil {
		for _, key := range m.Source.Roles {
			forSource[key] = true
		}
	}

	for _, p := range m.AllParams() {
		ts := p.targets()
		if len(ts) == 0 && !forSource[p.Key] {
			return fmt.Errorf("参数 %s 既没有 target，也没有被 source.roles 引用", p.Key)
		}
		for _, target := range ts {
			title, _, ok := splitTarget(target)
			if !ok {
				return fmt.Errorf("参数 %s 的 target %q 不是 \"标题.输入名\" 形式", p.Key, target)
			}
			if !titles[title] {
				return fmt.Errorf("参数 %s 指向的节点标题 %q 在模板里不存在", p.Key, title)
			}
		}
		rewires := p.RewireWhenZero
		if rw := p.RewireWhenSet; rw != nil {
			rewires = append(rewires, *rw)
		}
		for _, rw := range rewires {
			for _, n := range []string{rw.Node, rw.Source} {
				if !titles[n] {
					return fmt.Errorf("参数 %s 的改接目标 %q 在模板里不存在", p.Key, n)
				}
			}
		}
		for _, n := range append(append([]string{}, p.DropWhenEmpty...), p.DropWhenZero...) {
			if !titles[n] {
				return fmt.Errorf("参数 %s 要删除的节点 %q 在模板里不存在", p.Key, n)
			}
		}
		if b := p.BypassWhenZero; b != nil {
			if !titles[b.Node] {
				return fmt.Errorf("参数 %s 的 bypass 目标 %q 在模板里不存在", p.Key, b.Node)
			}
			if len(b.Passthrough) == 0 {
				return fmt.Errorf("参数 %s 的 bypass 缺少 passthrough 输出映射", p.Key)
			}
		}
	}
	for ch, o := range m.Outputs {
		if !titles[o.Node] {
			return fmt.Errorf("产物 %s 指向的节点标题 %q 在模板里不存在", ch, o.Node)
		}
	}
	return validateSource(m)
}

func validateSource(m *Meta) error {
	s := m.Source
	if s == nil {
		return nil
	}
	if s.Kind != "api" {
		return fmt.Errorf("source.kind 只支持 api，收到 %q", s.Kind)
	}
	if s.Provider == "" {
		return fmt.Errorf("source 缺少 provider")
	}
	keys := map[string]bool{}
	for _, p := range m.AllParams() {
		keys[p.Key] = true
	}
	if s.ImageParam == "" {
		return fmt.Errorf("source 缺少 image_param")
	}
	if !keys[s.ImageParam] {
		return fmt.Errorf("source.image_param %q 不是本工作流声明过的参数", s.ImageParam)
	}
	known := map[string]bool{
		"model": true, "prompt": true, "size": true,
		"quality": true, "background": true, "reference": true,
	}
	for role, key := range s.Roles {
		if !known[role] {
			return fmt.Errorf("source.roles 里的 %q 不是认得的字段", role)
		}
		if !keys[key] {
			return fmt.Errorf("source.roles.%s 指向的参数 %q 未声明", role, key)
		}
	}
	if s.Roles["prompt"] == "" {
		return fmt.Errorf("source.roles 必须至少映射 prompt")
	}
	return nil
}

func splitTarget(t string) (title, input string, ok bool) {
	i := strings.LastIndex(t, ".")
	if i <= 0 || i == len(t)-1 {
		return "", "", false
	}
	return t[:i], t[i+1:], true
}

func (t *Template) graph() (Graph, error) {
	var g Graph
	if err := json.Unmarshal(t.raw, &g); err != nil {
		return nil, fmt.Errorf("解析工作流 %s 的模板: %w", t.Meta.ID, err)
	}
	return g, nil
}

// UsedClasses 返回这份模板用到的全部节点类，去重并排序。
//
// 环境自检据此判断"能不能跑"——比对节点包名是不够的：装了包也可能因为依赖
// 缺失而没注册成功，只有节点类真的在 /object_info 里才算数。
func (t *Template) UsedClasses() []string {
	g, err := t.graph()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(g))
	for _, n := range g {
		if n.ClassType == "" || seen[n.ClassType] {
			continue
		}
		seen[n.ClassType] = true
		out = append(out, n.ClassType)
	}
	sort.Strings(out)
	return out
}

// Dir 返回工作流目录，供导入功能写文件。
func (r *Registry) Dir() string { return r.dir }

func (r *Registry) Get(id string) (*Template, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.list[id]
	return t, ok
}

// List 按 id 排序返回全部模板。
func (r *Registry) List() []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Template, 0, len(r.list))
	for _, t := range r.list {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Meta.ID < out[j].Meta.ID })
	return out
}
