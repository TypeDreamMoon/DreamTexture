package workflow

import "testing"

// manifest 里记的底模必须是这一次真的进了图的。
//
// 条件接线会整段删掉支路：云端底图调成直出时，SDXL 那一段根本不在提交的图里，
// 底模一次都没加载过。照抄模板声明的 model_requirements 就会写上一个从未参与的
// 底模——排障时能把人带得很远，所以专门守一道。
func TestDeclaredModelsMatchRenderedGraph(t *testing.T) {
	reg := repoWorkflows(t)

	inGraph := func(r *Rendered, file string) bool {
		for _, n := range r.Graph {
			for _, v := range n.Inputs {
				if s, ok := v.(string); ok && s == file {
					return true
				}
			}
		}
		return false
	}

	for _, tpl := range reg.List() {
		if tpl.Meta.Direct() {
			continue // 没有节点图，无从比对
		}
		// 默认参数下，声明的每个 checkpoint 都应当真的出现在图里；
		// 对不上说明声明和模板脱节了。
		def, err := tpl.Render(map[string]any{}, "probe")
		if err != nil {
			t.Fatalf("%s: %v", tpl.Meta.ID, err)
		}
		for _, req := range tpl.Meta.ModelRequirements {
			if req.Kind != "checkpoint" {
				continue
			}
			if tpl.Meta.Source != nil && !inGraph(def, req.File) {
				// 云端管线的底模可能因为直出而被删掉——但默认是 0.45，
				// 也就是走重整，此时它必须在。
				t.Errorf("%s 默认参数下声明的 %s 不在图里", tpl.Meta.ID, req.File)
			}
			if tpl.Meta.Source == nil && !inGraph(def, req.File) {
				t.Errorf("%s 声明了 %s 但图里没有引用它", tpl.Meta.ID, req.File)
			}
		}

		if tpl.Meta.Source == nil || tpl.Meta.Kind != KindMaterial {
			continue
		}
		// 直出模式：SDXL 底模必须消失，CHORD（如有）必须还在。
		direct, err := tpl.Render(map[string]any{"tile_fix": 0, "source_image": "x.png"}, "probe")
		if err != nil {
			t.Fatalf("%s 直出渲染失败: %v", tpl.Meta.ID, err)
		}
		for _, req := range tpl.Meta.ModelRequirements {
			if req.Kind != "checkpoint" {
				continue
			}
			isChord := req.File == "chord_v1.safetensors"
			got := inGraph(direct, req.File)
			if isChord && !got {
				t.Errorf("%s 直出模式下分解模型 %s 不该被删掉", tpl.Meta.ID, req.File)
			}
			if !isChord && got {
				t.Errorf("%s 直出模式下不该还引用底模 %s", tpl.Meta.ID, req.File)
			}
		}
	}
}
