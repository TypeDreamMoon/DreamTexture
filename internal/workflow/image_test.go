package workflow

import "testing"

// 图片管线的两种形态都要立得住：
//
//	本地——有节点图，出且只出一张图
//	云端直出——没有节点图，参数只喂给 API
func TestImageWorkflows(t *testing.T) {
	reg := repoWorkflows(t)
	var local, direct int

	for _, tpl := range reg.List() {
		if tpl.Meta.Kind != KindImage {
			continue
		}
		id := tpl.Meta.ID

		if tpl.Meta.Direct() {
			direct++
			// 直出没有节点图，Render 不该被调用；参数必须全由 source.roles 消费
			if len(tpl.Meta.Outputs) != 0 {
				t.Errorf("%s 是直出，不该声明 outputs", id)
			}
			roles := map[string]bool{}
			for _, k := range tpl.Meta.Source.Roles {
				roles[k] = true
			}
			for _, p := range tpl.Meta.AllParams() {
				if len(p.targets()) > 0 {
					t.Errorf("%s 的参数 %s 声明了 target，但没有节点图可注入", id, p.Key)
				}
				if !roles[p.Key] && !p.Hidden {
					t.Errorf("%s 的参数 %s 没有任何用处", id, p.Key)
				}
			}
			if tpl.Meta.Source.Roles["prompt"] == "" {
				t.Errorf("%s 没有把提示词接进 source.roles", id)
			}
			continue
		}

		local++
		r, err := tpl.Render(map[string]any{}, "probe")
		if err != nil {
			t.Fatalf("%s 渲染失败: %v", id, err)
		}
		// 图片管线只该有一路产物——多于一路的话 collectImage 取哪一张就成了随机的
		if len(r.Expect) != 1 {
			t.Errorf("%s 应当只有一路产物，实际 %d 路", id, len(r.Expect))
		}
		// 普通出图不该带材质管线那套平光前后缀
		for _, p := range tpl.Meta.AllParams() {
			if p.Key == "prompt" && (p.Prefix != "" || p.Suffix != "") {
				t.Errorf("%s 的提示词带了固定前后缀，普通出图不该限制画面", id)
			}
		}
		// 给了参考图应当改接到重绘支路
		withRef, err := tpl.Render(map[string]any{"reference": "x.png"}, "probe")
		if err != nil {
			t.Fatalf("%s 带参考图渲染失败: %v", id, err)
		}
		if !hasTitle(withRef, "dt.reference_encode") {
			t.Errorf("%s 给了参考图却没有编码节点", id)
		}
		if hasTitle(r, "dt.reference_encode") {
			t.Errorf("%s 没给参考图时不该留着编码节点", id)
		}
		t.Logf("%-16s 本地 %d 节点（带参考图 %d），产物 %d 路",
			id, len(r.Graph), len(withRef.Graph), len(r.Expect))
	}

	if local == 0 || direct == 0 {
		t.Fatalf("本地图片管线 %d 条、云端直出 %d 条，两种都该有", local, direct)
	}
}

// 材质管线不该被误标成 image，反之亦然。
func TestKindsAreExplicit(t *testing.T) {
	for _, tpl := range repoWorkflows(t).List() {
		switch tpl.Meta.Kind {
		case KindMaterial:
			if len(tpl.Meta.Outputs) < 2 {
				t.Errorf("%s 标成材质却只有 %d 路产物", tpl.Meta.ID, len(tpl.Meta.Outputs))
			}
		case KindImage:
		default:
			t.Errorf("%s 的 kind 非法: %q", tpl.Meta.ID, tpl.Meta.Kind)
		}
	}
}
