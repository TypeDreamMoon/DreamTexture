package workflow

import "fmt"

// SegmentKind 标出一份声明是不是"半条管线"。
//
// 材质管线天然分成两段：出图（产出一张底图）和分解（把一张底图拆成 PBR 通道）。
// 原先每种组合都是一份完整模板，于是四个预设其实是 2×2 的笛卡尔积——加一个
// 分解模型就要为每个出图源各补一份文件，而那些文件之间只有中间一根连线不同。
//
// 拆成段之后，组合在**加载期**拼成一份普通 Template，下游（Render、提交、
// manifest、环境自检）完全不知道有这回事。
type SegmentKind string

const (
	// SegmentNone 是完整模板：自己从头跑到尾，不参与组合。
	// 用户手搓的自定义工作流走这条，保持原样。
	SegmentNone SegmentKind = ""
	// SegmentSource 是出图段，产出一张底图。
	SegmentSource SegmentKind = "source"
	// SegmentDecompose 是分解段，吃一张底图，产出整套 PBR 通道。
	SegmentDecompose SegmentKind = "decompose"
)

// SegmentPort 是两段之间那根线的接口。
//
// 出图段用 Export 声明"底图从哪个节点的第几路输出出来"，分解段用 Import 声明
// "底图接到哪个节点的哪个输入上"。两边都按标题寻址，与图里的节点 id 无关——
// 拼接时 id 会重编号，标题不会。
type SegmentPort struct {
	Node  string `json:"node"`
	Slot  int    `json:"slot,omitempty"`  // 仅 Export
	Input string `json:"input,omitempty"` // 仅 Imports
}

// validateSegment 检查段声明本身是否自洽。
func validateSegment(m *Meta) error {
	switch m.Segment {
	case SegmentNone:
		if m.Export != nil || len(m.Imports) > 0 {
			return fmt.Errorf("不是段却声明了 export/imports；要参与组合请写明 segment")
		}
		return nil
	case SegmentSource:
		if m.Export == nil || m.Export.Node == "" {
			return fmt.Errorf("出图段必须声明 export.node（底图从哪个节点出来）")
		}
		if len(m.Imports) > 0 {
			return fmt.Errorf("出图段不该声明 imports")
		}
		if len(m.Outputs) > 0 {
			return fmt.Errorf("出图段不该声明 outputs：最终产物由分解段负责落盘")
		}
	case SegmentDecompose:
		if len(m.Imports) == 0 {
			return fmt.Errorf("分解段必须声明 imports（底图接到哪些节点的哪些输入上）")
		}
		for i, im := range m.Imports {
			if im.Node == "" || im.Input == "" {
				return fmt.Errorf("imports[%d] 缺少 node 或 input", i)
			}
		}
		if m.Export != nil {
			return fmt.Errorf("分解段不该声明 export")
		}
		if len(m.Outputs) == 0 {
			return fmt.Errorf("分解段必须声明 outputs")
		}
	default:
		return fmt.Errorf("segment 只能是 source 或 decompose，收到 %q", m.Segment)
	}
	return nil
}
