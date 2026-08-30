package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// ImportPlaceholder 让出图段在不知道对面是谁的前提下引用分解段的入口。
//
// 云端底图那条管线需要它：无缝重整强度调到 0 时要把底图**直接**喂给分解链、
// 整条 SDXL 支路删掉。但"分解链的入口"叫什么，取决于配的是哪个分解段
// （CHORD 是 dt.pbr_estimate，传统派生是 dt.luminance）。写死任何一个名字都会
// 让出图段只能配那一种分解，正是分段要消除的耦合。
//
// 拼接时它展开成分解段的每一个 import 端口。
const ImportPlaceholder = "@import"

// ComposedID 拼出组合管线的 id。
//
// 分隔符用 '.'：这个 id 会进任务记录、也会出现在接口路径里，'+' 在查询串里
// 会被解成空格，'/' 会把路径切开，都不合适。
func ComposedID(sourceID, decomposeID string) string { return sourceID + "." + decomposeID }

// SplitComposedID 是 ComposedID 的逆操作，第二个返回值表示这是不是一个组合 id。
func SplitComposedID(id string) (string, string, bool) {
	i := strings.Index(id, ".")
	if i <= 0 || i == len(id)-1 {
		return "", "", false
	}
	return id[:i], id[i+1:], true
}

// Compose 把出图段和分解段拼成一份完整模板。
//
// 拼出来的是一份**普通 Template**，跟手写的完整模板没有任何区别——正因如此，
// 渲染、提交、manifest、环境自检那一整条下游一行都不用改。
//
// 能这么干是因为这套模板全程按 _meta.title 寻址：参数的 target、改接、bypass、
// 删节点、产物声明，引用的都是标题。节点 id 只是 JSON 的 map 键和连线里的第一个
// 元素，重编号一遍就完事了。
func Compose(src, dec *Template) (*Template, error) {
	if src.Meta.Segment != SegmentSource {
		return nil, fmt.Errorf("%s 不是出图段", src.Meta.ID)
	}
	if dec.Meta.Segment != SegmentDecompose {
		return nil, fmt.Errorf("%s 不是分解段", dec.Meta.ID)
	}

	sg, err := src.graph()
	if err != nil {
		return nil, err
	}
	dg, err := dec.graph()
	if err != nil {
		return nil, err
	}

	// 标题必须全局唯一：整套寻址都靠它，撞了就会静默指错节点——参数注入到
	// 另一段的同名节点上，图还是合法的，只是行为完全不对。
	if dup := commonTitles(sg, dg); len(dup) > 0 {
		return nil, fmt.Errorf("两段的节点标题撞了 %v；出图段用 dt.src.* 前缀，分解段用 dt.dec.*", dup)
	}

	merged, remap := mergeGraphs(sg, dg)

	// 接线：把分解段的入口输入指向出图段的出口。
	srcOutID, ok := idByTitle(sg, src.Meta.Export.Node)
	if !ok {
		return nil, fmt.Errorf("出图段 %s 的 export.node %q 在图里不存在", src.Meta.ID, src.Meta.Export.Node)
	}
	for _, im := range dec.Meta.Imports {
		decInOld, ok := idByTitle(dg, im.Node)
		if !ok {
			return nil, fmt.Errorf("分解段 %s 的 imports 里 %q 在图里不存在", dec.Meta.ID, im.Node)
		}
		inNode := merged[remap[decInOld]]
		if _, has := inNode.Inputs[im.Input]; !has {
			return nil, fmt.Errorf("分解段 %s 的入口节点 %q 上没有输入 %q", dec.Meta.ID, im.Node, im.Input)
		}
		inNode.Inputs[im.Input] = []any{srcOutID, src.Meta.Export.Slot}
	}

	meta, err := mergeMeta(&src.Meta, &dec.Meta)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(merged)
	if err != nil {
		return nil, err
	}
	t := &Template{Meta: *meta, raw: raw}
	if err := validate(meta, merged); err != nil {
		return nil, fmt.Errorf("组合 %s: %w", meta.ID, err)
	}
	return t, nil
}

func commonTitles(a, b Graph) []string {
	seen := map[string]bool{}
	for _, n := range a {
		if n.Meta.Title != "" {
			seen[n.Meta.Title] = true
		}
	}
	var dup []string
	for _, n := range b {
		if n.Meta.Title != "" && seen[n.Meta.Title] {
			dup = append(dup, n.Meta.Title)
		}
	}
	sort.Strings(dup)
	return dup
}

func idByTitle(g Graph, title string) (string, bool) {
	for id, n := range g {
		if n.Meta.Title == title {
			return id, true
		}
	}
	return "", false
}

// mergeGraphs 把 b 并进 a 的副本，b 的节点 id 全部重编号避开 a。
// 返回合并后的图，以及 b 的旧 id → 新 id 映射。
func mergeGraphs(a, b Graph) (Graph, map[string]string) {
	out := make(Graph, len(a)+len(b))
	maxID := 0
	for id, n := range a {
		out[id] = n
		if v, err := strconv.Atoi(id); err == nil && v > maxID {
			maxID = v
		}
	}

	// 按旧 id 排序后再分配，保证同样的输入拼出同样的图——不然每次加载
	// 节点 id 都在变，日志和报错对不上，diff 也没法看。
	olds := make([]string, 0, len(b))
	for id := range b {
		olds = append(olds, id)
	}
	sort.Slice(olds, func(i, j int) bool {
		x, ex := strconv.Atoi(olds[i])
		y, ey := strconv.Atoi(olds[j])
		if ex == nil && ey == nil {
			return x < y
		}
		return olds[i] < olds[j]
	})

	remap := make(map[string]string, len(b))
	for _, old := range olds {
		maxID++
		remap[old] = strconv.Itoa(maxID)
	}
	for _, old := range olds {
		n := b[old]
		cp := &Node{ClassType: n.ClassType, Inputs: make(map[string]any, len(n.Inputs))}
		cp.Meta.Title = n.Meta.Title
		for k, v := range n.Inputs {
			cp.Inputs[k] = relinkValue(v, remap)
		}
		out[remap[old]] = cp
	}
	return out, remap
}

// relinkValue 把连线里的节点 id 换成新 id。
//
// API-format 里一条连线长这样：["3", 0]——第一个元素是上游节点 id。只认这个形状，
// 别的值（字符串、数字、对象）原样返回。
func relinkValue(v any, remap map[string]string) any {
	link, ok := v.([]any)
	if !ok || len(link) != 2 {
		return v
	}
	id, ok := link[0].(string)
	if !ok {
		return v
	}
	nid, ok := remap[id]
	if !ok {
		return v
	}
	return []any{nid, link[1]}
}

func mergeMeta(src, dec *Meta) (*Meta, error) {
	m := &Meta{
		ID:          ComposedID(src.ID, dec.ID),
		Kind:        KindMaterial,
		Version:     1,
		Name:        src.Name + " · " + dec.Name,
		Style:       src.Domain,
		Description: strings.TrimSpace(src.Description + " " + dec.Description),
		Resolution:  src.Resolution,
		Tileable:    src.Tileable && dec.Tileable,

		TileableWhenPositive: src.TileableWhenPositive,
		Source:               src.Source,
		Outputs:              dec.Outputs,

		SourceSegment:    src.ID,
		DecomposeSegment: dec.ID,
	}
	if m.Resolution == 0 {
		m.Resolution = dec.Resolution
	}

	var err error
	if m.Params, err = mergeParams(src.Params, dec.Params, dec.Imports); err != nil {
		return nil, err
	}
	if m.Advanced, err = mergeParams(src.Advanced, dec.Advanced, dec.Imports); err != nil {
		return nil, err
	}

	m.Licenses = append(append([]LicenseNotice{}, src.Licenses...), dec.Licenses...)
	m.ModelRequirements = mergeRequirements(src.ModelRequirements, dec.ModelRequirements)
	m.NodePacks = mergeStrings(src.NodePacks, dec.NodePacks)

	// 画面类型对不上只做标记，不拦。手绘图喂给 CHORD 是分布外输入，法线会糊，
	// 但要不要试是用户的事——真拦掉就等于把一条合法的探索路堵死了。
	if src.Domain != "" && len(dec.ExpectsDomain) > 0 {
		fit := false
		for _, d := range dec.ExpectsDomain {
			if d == src.Domain {
				fit = true
				break
			}
		}
		if !fit {
			m.Mismatch = fmt.Sprintf("%s 是为 %s 画面训练/调参的，%s 的产物对它是分布外输入，结果可能明显变差",
				dec.Name, strings.Join(dec.ExpectsDomain, "/"), src.Name)
		}
	}
	return m, nil
}

func mergeRequirements(a, b []ModelRequirement) []ModelRequirement {
	out := append([]ModelRequirement{}, a...)
	seen := map[string]bool{}
	for _, r := range a {
		seen[r.Dir+"/"+r.File] = true
	}
	for _, r := range b {
		if k := r.Dir + "/" + r.File; !seen[k] {
			seen[k] = true
			out = append(out, r)
		}
	}
	return out
}

func mergeStrings(a, b []string) []string {
	out := append([]string{}, a...)
	seen := map[string]bool{}
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// mergeParams 合并两段的参数声明，并把 @import 展开成分解段的入口。
//
// 同名参数**不是**错误，而是合并：分辨率就是典型——出图段要用它定 latent 尺寸，
// 传统派生段要用它定常量图尺寸，两边说的是同一个值。取 target 的并集即可。
//
// 但只在类型一致时才合并。类型都不一样说明是两个不相干的东西撞了名字，那种情况
// 必须拦：前端只会渲染一个控件，另一段那个参数永远停在默认值——图照跑，结果不对，
// 而且完全看不出来。
func mergeParams(a, b []Param, imports []SegmentPort) ([]Param, error) {
	out := make([]Param, 0, len(a)+len(b))
	at := map[string]int{}
	for _, p := range append(append([]Param{}, a...), b...) {
		p = expandImports(p, imports)
		i, dup := at[p.Key]
		if !dup {
			at[p.Key] = len(out)
			out = append(out, p)
			continue
		}
		merged, err := mergeParam(out[i], p)
		if err != nil {
			return nil, err
		}
		out[i] = merged
	}
	return out, nil
}

func mergeParam(a, b Param) (Param, error) {
	if a.Type != b.Type {
		return a, fmt.Errorf("参数键 %q 在两段里类型不同（%s vs %s），这是撞名不是同一个参数", a.Key, a.Type, b.Type)
	}
	if a.Default != nil && b.Default != nil && !sameScalar(a.Default, b.Default) {
		return a, fmt.Errorf("参数键 %q 在两段里默认值不同（%v vs %v）", a.Key, a.Default, b.Default)
	}
	ts := append(a.targets(), b.targets()...)
	raw, err := json.Marshal(ts)
	if err != nil {
		return a, err
	}
	a.Target = raw
	if a.Default == nil {
		a.Default = b.Default
	}
	return a, nil
}

// expandImports 把改接声明里的 @import 换成分解段的每一个入口端口。
func expandImports(p Param, imports []SegmentPort) Param {
	expand := func(in []Rewire) []Rewire {
		var out []Rewire
		for _, rw := range in {
			if rw.Node != ImportPlaceholder {
				out = append(out, rw)
				continue
			}
			for _, im := range imports {
				out = append(out, Rewire{Node: im.Node, Input: im.Input, Source: rw.Source, Slot: rw.Slot})
			}
		}
		return out
	}
	if len(p.RewireWhenZero) > 0 {
		p.RewireWhenZero = expand(p.RewireWhenZero)
	}
	if rw := p.RewireWhenSet; rw != nil && rw.Node == ImportPlaceholder && len(imports) > 0 {
		// 单条改接展开成多条时只能保留第一个入口——RewireWhenSet 的字段是单数。
		// 分解段有多个入口时应当改用 rewire_when_zero 那种数组形式。
		first := imports[0]
		p.RewireWhenSet = &Rewire{Node: first.Node, Input: first.Input, Source: rw.Source, Slot: rw.Slot}
	}
	return p
}
