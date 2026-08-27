package workflow

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// NodeSpec 是 ComfyUI /object_info 里单个节点类的定义，只取转换需要的部分。
type NodeSpec struct {
	Input struct {
		Required map[string]json.RawMessage `json:"required"`
		Optional map[string]json.RawMessage `json:"optional"`
	} `json:"input"`
	InputOrder struct {
		Required []string `json:"required"`
		Optional []string `json:"optional"`
	} `json:"input_order"`
	Output     []string `json:"output"`
	OutputName []string `json:"output_name"`
}

// orderedInputs 返回该节点全部输入的规范顺序。
//
// 顺序必须来自 input_order：Go 的 map 无序，而 widgets_values 是位置相关的数组，
// 顺序错了参数就会串到别的控件上。
func (s *NodeSpec) orderedInputs() []string {
	out := make([]string, 0, len(s.Input.Required)+len(s.Input.Optional))
	seen := map[string]bool{}
	add := func(names []string, pool map[string]json.RawMessage) {
		for _, n := range names {
			if _, ok := pool[n]; ok && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
		// input_order 没覆盖到的兜底，按名字排序保证确定性。
		var rest []string
		for n := range pool {
			if !seen[n] {
				rest = append(rest, n)
			}
		}
		sort.Strings(rest)
		for _, n := range rest {
			seen[n] = true
			out = append(out, n)
		}
	}
	add(s.InputOrder.Required, s.Input.Required)
	add(s.InputOrder.Optional, s.Input.Optional)
	return out
}

// inputMeta 解析某个输入的类型与附加属性。
func (s *NodeSpec) inputMeta(name string) (typeName string, isCombo bool, control bool) {
	raw, ok := s.Input.Required[name]
	if !ok {
		raw = s.Input.Optional[name]
	}
	var spec []json.RawMessage
	if json.Unmarshal(raw, &spec) != nil || len(spec) == 0 {
		return "", false, false
	}
	// 第一项要么是类型名字符串，要么是候选值数组（下拉框）。
	var t string
	if json.Unmarshal(spec[0], &t) == nil {
		typeName = t
	} else {
		isCombo = true
		typeName = "COMBO"
	}
	if len(spec) > 1 {
		var opts struct {
			ControlAfterGenerate bool `json:"control_after_generate"`
		}
		if json.Unmarshal(spec[1], &opts) == nil {
			control = opts.ControlAfterGenerate
		}
	}
	return
}

// ToEditorFormat 把 API-format 工作流转成 ComfyUI 编辑器能打开的 UI 格式。
//
// 这一步是必需的：API 格式只有节点与取值，没有槽位、连线和坐标；直接丢给编辑器
// 会得到一张空图（实测确认）。要让用户在 ComfyUI 里可视化地改我们的模板，
// 就得把这些信息按 /object_info 的声明重建出来。
func ToEditorFormat(g Graph, specs map[string]NodeSpec, name string) (map[string]any, error) {
	ids := make([]string, 0, len(g))
	for id := range g {
		ids = append(ids, id)
	}
	// 按数字大小排序，保证每次导出的结果一致（便于 diff）。
	sort.Slice(ids, func(i, j int) bool { return numID(ids[i]) < numID(ids[j]) })

	type outSlot struct{ links []int }
	nodeOutputs := map[string][]outSlot{}
	for _, id := range ids {
		spec := specs[g[id].ClassType]
		nodeOutputs[id] = make([]outSlot, len(spec.Output))
	}

	var links [][]any
	linkID := 0
	// 先建连线表：每条连线要同时登记到源节点的输出槽和目标节点的输入槽。
	type inLink struct {
		name   string
		linkID int
		typ    string
	}
	nodeInLinks := map[string][]inLink{}

	for _, id := range ids {
		node := g[id]
		spec, ok := specs[node.ClassType]
		if !ok {
			return nil, fmt.Errorf("ComfyUI 里没有注册节点类 %q，无法转换（可能是自定义节点包未安装）", node.ClassType)
		}
		for _, in := range spec.orderedInputs() {
			v, present := node.Inputs[in]
			if !present {
				continue
			}
			link, isLink := v.([]any)
			if !isLink || len(link) != 2 {
				continue
			}
			src, _ := link[0].(string)
			slot := 0
			if f, ok := toFloat(link[1]); ok {
				slot = int(f)
			}
			srcSpec, ok := specs[g[src].ClassType]
			if !ok || slot >= len(srcSpec.Output) {
				return nil, fmt.Errorf("节点 %s 的第 %d 路输出不存在", src, slot)
			}
			linkID++
			typ := srcSpec.Output[slot]
			links = append(links, []any{linkID, numID(src), slot, numID(id), len(nodeInLinks[id]), typ})
			nodeInLinks[id] = append(nodeInLinks[id], inLink{name: in, linkID: linkID, typ: typ})
			nodeOutputs[src][slot].links = append(nodeOutputs[src][slot].links, linkID)
		}
	}

	depth := layerDepths(g)
	perLayer := map[int]int{}
	nodes := make([]map[string]any, 0, len(ids))

	for _, id := range ids {
		node := g[id]
		spec := specs[node.ClassType]

		var widgets []any
		for _, in := range spec.orderedInputs() {
			v, present := node.Inputs[in]
			if !present {
				continue
			}
			if link, isLink := v.([]any); isLink && len(link) == 2 {
				continue // 连线不进 widgets_values
			}
			widgets = append(widgets, v)
			// 种子这类控件在编辑器里额外带一个"生成后如何变化"的下拉，
			// 它也占 widgets_values 的一个位置，漏了后面的值就全错位了。
			if _, _, control := spec.inputMeta(in); control {
				widgets = append(widgets, "randomize")
			}
		}

		inputs := make([]map[string]any, 0, len(nodeInLinks[id]))
		for _, l := range nodeInLinks[id] {
			inputs = append(inputs, map[string]any{"name": l.name, "type": l.typ, "link": l.linkID})
		}

		outputs := make([]map[string]any, 0, len(spec.Output))
		for i, t := range spec.Output {
			n := t
			if i < len(spec.OutputName) {
				n = spec.OutputName[i]
			}
			o := map[string]any{"name": n, "type": t}
			if ls := nodeOutputs[id][i].links; len(ls) > 0 {
				o["links"] = ls
			} else {
				o["links"] = nil
			}
			outputs = append(outputs, o)
		}

		d := depth[id]
		entry := map[string]any{
			"id":    numID(id),
			"type":  node.ClassType,
			"pos":   []int{80 + d*360, 80 + perLayer[d]*200},
			"size":  []int{300, 120},
			"flags": map[string]any{},
			"order": len(nodes),
			"mode":  0,
			"properties": map[string]any{
				"Node name for S&R": node.ClassType,
			},
		}
		perLayer[d]++
		if len(inputs) > 0 {
			entry["inputs"] = inputs
		}
		if len(outputs) > 0 {
			entry["outputs"] = outputs
		}
		if len(widgets) > 0 {
			entry["widgets_values"] = widgets
		}
		if node.Meta.Title != "" {
			entry["title"] = node.Meta.Title
		}
		nodes = append(nodes, entry)
	}

	last := 0
	for _, id := range ids {
		if n := numID(id); n > last {
			last = n
		}
	}

	return map[string]any{
		"id":           name,
		"revision":     0,
		"last_node_id": last,
		"last_link_id": linkID,
		"nodes":        nodes,
		"links":        links,
		"groups":       []any{},
		"config":       map[string]any{},
		"extra":        map[string]any{},
		"version":      0.4,
	}, nil
}

// layerDepths 给每个节点算一个"离源头多远"的层号，用于自动排布坐标。
//
// 没有坐标信息时节点会全挤在原点，图打开也看不懂；按依赖深度分列至少能让人
// 一眼看出数据从左往右流。
func layerDepths(g Graph) map[string]int {
	depth := map[string]int{}
	var visit func(string, map[string]bool) int
	visit = func(id string, path map[string]bool) int {
		if d, ok := depth[id]; ok {
			return d
		}
		if path[id] {
			return 0 // 理论上不该有环，真有也不能死循环
		}
		path[id] = true
		max := 0
		if n, ok := g[id]; ok {
			for _, v := range n.Inputs {
				link, isLink := v.([]any)
				if !isLink || len(link) != 2 {
					continue
				}
				src, _ := link[0].(string)
				if d := visit(src, path) + 1; d > max {
					max = d
				}
			}
		}
		delete(path, id)
		depth[id] = max
		return max
	}
	for id := range g {
		visit(id, map[string]bool{})
	}
	return depth
}

func numID(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// RawTemplate 返回模板的原始 API-format JSON。
func (t *Template) RawTemplate() []byte { return t.raw }

// GraphForEditor 解析模板并转成编辑器格式。
func (t *Template) GraphForEditor(specs map[string]NodeSpec, name string) (map[string]any, error) {
	g, err := t.graph()
	if err != nil {
		return nil, err
	}
	return ToEditorFormat(g, specs, name)
}
