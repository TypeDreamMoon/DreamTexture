package workflow

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Rendered 是一次注入完成、可直接提交给 ComfyUI 的工作流。
type Rendered struct {
	Graph Graph

	// Expect 是期望产出的节点 id → 通道名。
	//
	// 这份清单是判定任务成败的依据：ComfyUI 在部分输出分支校验失败时仍会把整体
	// 状态报成 success，只能靠比对实际 outputs 的键集合发现缺失。
	Expect map[string]string

	// Values 是套用默认值、做完类型转换与范围钳制之后的最终参数。
	Values map[string]any

	titleByID map[string]string
}

// TitleOf 把节点 id 翻译成模板里的标题，用于把 ComfyUI 的报错说人话。
func (r *Rendered) TitleOf(nodeID string) string { return r.titleByID[nodeID] }

// Resolve 只做参数解析：套默认值、转类型、钳范围，不碰节点图。
//
// 单独拆出来是给"底图来自云端"的工作流用的：后端要先按解析后的参数去调
// 外部接口拿图，把图上传给 ComfyUI，拿到文件名再回过头来渲染节点图。
// 让调用方为此渲染两遍整张图是浪费，也容易两次结果不一致。
func (t *Template) Resolve(values map[string]any) (map[string]any, error) {
	resolved := map[string]any{}
	for _, p := range t.Meta.AllParams() {
		raw, ok := values[p.Key]
		if !ok || raw == nil || raw == "" {
			raw = p.Default
		}
		v, err := coerce(p, raw)
		if err != nil {
			return nil, err
		}
		resolved[p.Key] = v
	}
	return resolved, nil
}

// Render 把用户传入的参数注入模板。
//
// outPrefix 会重写全部 SaveImage 的 filename_prefix：每个任务用互不相同的前缀，
// 既避免不同任务的产物互相覆盖，也让 ComfyUI 的节点缓存不会把旧任务的文件名
// 当成新任务的产物报回来。
func (t *Template) Render(values map[string]any, outPrefix string) (*Rendered, error) {
	// 直出的模板没有节点图，走到这里说明调用方选错了路径。
	// 不拦的话报出来的是 "unexpected end of JSON input"，看不出是怎么回事。
	if t.Meta.Direct() {
		return nil, fmt.Errorf("工作流 %s 是纯云端直出，没有节点图可渲染", t.Meta.ID)
	}
	g, err := t.graph()
	if err != nil {
		return nil, err
	}

	idByTitle := map[string]string{}
	titleByID := map[string]string{}
	for id, n := range g {
		if n.Meta.Title == "" {
			continue
		}
		idByTitle[n.Meta.Title] = id
		titleByID[id] = n.Meta.Title
	}

	resolved, err := t.Resolve(values)
	if err != nil {
		return nil, err
	}

	// 先处理 bypass：摘节点会改动连线，必须在写入参数之前完成。
	for _, p := range t.Meta.AllParams() {
		b := p.BypassWhenZero
		if b == nil {
			continue
		}
		f, ok := toFloat(resolved[p.Key])
		if !ok || f != 0 {
			continue
		}
		if err := bypass(g, idByTitle[b.Node], b.Passthrough); err != nil {
			return nil, fmt.Errorf("摘除节点 %s: %w", b.Node, err)
		}
		delete(idByTitle, b.Node)
	}

	// 条件接线：有值就改接，没值就把那条支路删掉。参考图（img2img）走这里。
	for _, p := range t.Meta.AllParams() {
		set := hasValue(resolved[p.Key])
		if set && p.RewireWhenSet != nil {
			if err := rewire(g, idByTitle, *p.RewireWhenSet); err != nil {
				return nil, fmt.Errorf("参数 %s: %w", p.Key, err)
			}
		}
		if !set {
			drop(g, idByTitle, p.DropWhenEmpty)
		}

		// 数值为零的分支。改接必须排在删除之前，否则改接要找的源节点
		// 可能已经被同一个参数的 DropWhenZero 删掉了。
		if len(p.RewireWhenZero) == 0 && len(p.DropWhenZero) == 0 {
			continue
		}
		if f, ok := toFloat(resolved[p.Key]); !ok || f != 0 {
			continue
		}
		for _, rw := range p.RewireWhenZero {
			if err := rewire(g, idByTitle, rw); err != nil {
				return nil, fmt.Errorf("参数 %s: %w", p.Key, err)
			}
		}
		drop(g, idByTitle, p.DropWhenZero)
	}

	for _, p := range t.Meta.AllParams() {
		v := resolved[p.Key]
		if v == nil {
			continue
		}
		if s, ok := v.(string); ok && (p.Prefix != "" || p.Suffix != "") {
			v = p.Prefix + s + p.Suffix
		}
		for _, target := range p.targets() {
			title, input, _ := splitTarget(target)
			id, ok := idByTitle[title]
			if !ok {
				continue // 该节点已被 bypass 摘掉，跳过即可
			}
			g[id].Inputs[input] = v
		}
	}

	expect := map[string]string{}
	for ch, o := range t.Meta.Outputs {
		id, ok := idByTitle[o.Node]
		if !ok {
			continue
		}
		g[id].Inputs["filename_prefix"] = outPrefix + "/" + ch
		expect[id] = ch
	}
	if len(expect) == 0 {
		return nil, fmt.Errorf("工作流 %s 没有任何有效的输出节点", t.Meta.ID)
	}
	if err := checkDangling(g, titleByID); err != nil {
		return nil, fmt.Errorf("工作流 %s: %w", t.Meta.ID, err)
	}

	return &Rendered{Graph: g, Expect: expect, Values: resolved, titleByID: titleByID}, nil
}

// checkDangling 找出指向已删除节点的连线。
//
// 条件删除很容易漏掉一条下游引用，而 ComfyUI 那边的报错是
// "Node ID '#9' not found"——既不说是谁引用的，也不说该节点原本是什么。
// 在提交前用模板里的标题把话说清楚，省得每次都要回去对着 id 数图。
func checkDangling(g Graph, titleByID map[string]string) error {
	name := func(id string) string {
		if t := titleByID[id]; t != "" {
			return t
		}
		return "#" + id
	}
	var bad []string
	for id, n := range g {
		for input, v := range n.Inputs {
			link, ok := v.([]any)
			if !ok || len(link) != 2 {
				continue
			}
			src, ok := link[0].(string)
			if !ok {
				continue
			}
			if _, alive := g[src]; !alive {
				bad = append(bad, fmt.Sprintf("%s.%s 指向已被删除的 %s", name(id), input, name(src)))
			}
		}
	}
	if len(bad) == 0 {
		return nil
	}
	sort.Strings(bad)
	return fmt.Errorf("条件删除后有连线悬空（模板声明有误）:\n  %s", strings.Join(bad, "\n  "))
}

// rewire 把 rw.Node 的某个输入改接到 rw.Source 的第 rw.Slot 路输出。
func rewire(g Graph, idByTitle map[string]string, rw Rewire) error {
	target, ok := idByTitle[rw.Node]
	src, ok2 := idByTitle[rw.Source]
	if !ok || !ok2 {
		return fmt.Errorf("改接目标 %s <- %s 已不在图中", rw.Node, rw.Source)
	}
	g[target].Inputs[rw.Input] = []any{src, float64(rw.Slot)}
	return nil
}

// drop 按标题删除节点。调用方须保证被删的节点已无下游引用。
func drop(g Graph, idByTitle map[string]string, titles []string) {
	for _, title := range titles {
		if id, ok := idByTitle[title]; ok {
			delete(g, id)
			delete(idByTitle, title)
		}
	}
}

// bypass 删除一个直通节点，并把所有指向它的连线改接到它对应的上游。
func bypass(g Graph, id string, passthrough []string) error {
	n, ok := g[id]
	if !ok {
		return fmt.Errorf("节点不存在")
	}
	// 第 i 路输出对应 passthrough[i] 这个输入所连的上游。
	upstream := make([]any, len(passthrough))
	for i, in := range passthrough {
		v, ok := n.Inputs[in]
		if !ok {
			return fmt.Errorf("输入 %q 不存在", in)
		}
		if _, isLink := v.([]any); !isLink {
			return fmt.Errorf("输入 %q 不是连线，无法直通", in)
		}
		upstream[i] = v
	}
	delete(g, id)
	for _, other := range g {
		for k, v := range other.Inputs {
			link, ok := v.([]any)
			if !ok || len(link) != 2 {
				continue
			}
			src, _ := link[0].(string)
			if src != id {
				continue
			}
			slot := 0
			if f, ok := toFloat(link[1]); ok {
				slot = int(f)
			}
			if slot < 0 || slot >= len(upstream) {
				return fmt.Errorf("下游引用了第 %d 路输出，但 passthrough 只声明了 %d 路", slot, len(upstream))
			}
			other.Inputs[k] = upstream[slot]
		}
	}
	return nil
}

// hasValue 判断参数是否"有值"。空字符串与 nil 都算没有。
func hasValue(v any) bool {
	if v == nil {
		return false
	}
	s, ok := v.(string)
	return !ok || strings.TrimSpace(s) != ""
}

// coerce 把外部传入的值转成参数声明要求的类型，并做范围钳制。
func coerce(p Param, raw any) (any, error) {
	if raw == nil {
		return nil, nil
	}
	switch p.Type {
	case "string", "image":
		return fmt.Sprint(raw), nil

	case "bool":
		switch v := raw.(type) {
		case bool:
			return v, nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return nil, fmt.Errorf("参数 %s 需要布尔值，收到 %q", p.Key, v)
			}
			return b, nil
		}
		f, ok := toFloat(raw)
		if !ok {
			return nil, fmt.Errorf("参数 %s 需要布尔值", p.Key)
		}
		return f != 0, nil

	case "int":
		f, ok := toFloat(raw)
		if !ok {
			return nil, fmt.Errorf("参数 %s 需要整数，收到 %v", p.Key, raw)
		}
		return int64(clamp(p, math.Trunc(f))), nil

	case "float":
		f, ok := toFloat(raw)
		if !ok {
			return nil, fmt.Errorf("参数 %s 需要数值，收到 %v", p.Key, raw)
		}
		return clamp(p, f), nil

	case "enum":
		for _, opt := range p.Options {
			if sameScalar(opt, raw) {
				return opt, nil
			}
		}
		return nil, fmt.Errorf("参数 %s 的取值 %v 不在允许范围 %v 内", p.Key, raw, p.Options)

	default:
		return raw, nil
	}
}

func clamp(p Param, f float64) float64 {
	if p.Min != nil && f < *p.Min {
		return *p.Min
	}
	if p.Max != nil && f > *p.Max {
		return *p.Max
	}
	return f
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(n), 64)
		return f, err == nil
	}
	return 0, false
}

func sameScalar(a, b any) bool {
	if fa, ok := toFloat(a); ok {
		if fb, ok := toFloat(b); ok {
			return fa == fb
		}
		return false
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}
