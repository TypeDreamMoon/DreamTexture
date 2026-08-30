package workflow

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

// TestComposedMatchesLegacy 证明"拆段之后拼回来"与原先手写的完整模板逐节点逐连线一致。
//
// 这是整套分段改造唯一的安全网。拼错一根线不会有任何报错——图仍然合法，ComfyUI
// 照跑，只是出来的材质悄悄变了；而材质好不好看，人是分辨不出"是不是接错了"的。
// testdata/legacy 下那四份是改造前的原件，冻在那儿就是为了当基准。
//
// 比的是**渲染后**的图：模板里的差异（例如组合版多一个默认关闭的 LoRA 节点）
// 会被 bypass 消掉，真正该一致的是最终提交给 ComfyUI 的那张图。
func TestComposedMatchesLegacy(t *testing.T) {
	now := NewRegistry("../../workflows")
	if err := now.Load(); err != nil {
		t.Fatalf("加载 workflows: %v", err)
	}
	old := NewRegistry("testdata/legacy")
	if err := old.Load(); err != nil {
		t.Fatalf("加载 testdata/legacy: %v", err)
	}

	cases := []struct {
		composed string
		legacy   string
	}{
		{"src-local-realistic-v1.dec-chord-v1", "realistic-chord-v1"},
		{"src-local-stylized-v1.dec-derive-v1", "stylized-derive-v1"},
		{"src-api-realistic-v1.dec-chord-v1", "api-chord-v1"},
		{"src-api-stylized-v1.dec-derive-v1", "api-derive-v1"},
	}

	for _, c := range cases {
		t.Run(c.legacy, func(t *testing.T) {
			ct, ok := now.Get(c.composed)
			if !ok {
				t.Fatalf("没有拼出组合 %s", c.composed)
			}
			lt, ok := old.Get(c.legacy)
			if !ok {
				t.Fatalf("基准 %s 不见了", c.legacy)
			}

			// 两边喂同一份参数：先取基准的默认值，再套上覆盖项。这样任何差异
			// 都只可能来自接线或声明，不会是"默认值不同"这种噪声。
			vals, err := lt.Resolve(nil)
			if err != nil {
				t.Fatal(err)
			}

			cr, err := ct.Render(vals, "t")
			if err != nil {
				t.Fatalf("渲染组合版: %v", err)
			}
			lr, err := lt.Render(vals, "t")
			if err != nil {
				t.Fatalf("渲染基准: %v", err)
			}

			gotG := byTitle(t, cr.Graph)
			wantG := byTitle(t, lr.Graph)

			if d := diffTitles(gotG, wantG); d != "" {
				t.Errorf("节点集合不一致：%s", d)
			}
			for title, want := range wantG {
				got, ok := gotG[title]
				if !ok {
					continue // 上面已经报过
				}
				if !reflect.DeepEqual(got, want) {
					t.Errorf("节点 %s 不一致\n  组合版: %v\n  基准:   %v", title, got, want)
				}
			}
		})
	}
}

// byTitle 把图整理成"标题 → 归一化输入"，连线里的节点 id 换成对端标题。
// 直接比 id 是没意义的——拼接本来就会重编号。
func byTitle(t *testing.T, g Graph) map[string]map[string]any {
	t.Helper()
	title := map[string]string{}
	for id, n := range g {
		if n.Meta.Title != "" {
			title[id] = n.Meta.Title
		}
	}
	out := map[string]map[string]any{}
	for _, n := range g {
		if n.Meta.Title == "" {
			continue
		}
		norm := map[string]any{"__class": n.ClassType}
		for k, v := range n.Inputs {
			if link, ok := v.([]any); ok && len(link) == 2 {
				if id, ok := link[0].(string); ok {
					norm[k] = fmt.Sprintf("@%s:%v", title[id], link[1])
					continue
				}
			}
			norm[k] = normNum(v)
		}
		out[n.Meta.Title] = norm
	}
	return out
}

// normNum 把数值统一成 float64。同一个 0，从模板 JSON 读出来是 float64，
// 经参数注入写进去可能是 int——值一样却比不过 DeepEqual。
func normNum(v any) any {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	}
	return v
}

func diffTitles(got, want map[string]map[string]any) string {
	var missing, extra []string
	for k := range want {
		if _, ok := got[k]; !ok {
			missing = append(missing, k)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			extra = append(extra, k)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) == 0 && len(extra) == 0 {
		return ""
	}
	return fmt.Sprintf("组合版缺少 %v，多出 %v", missing, extra)
}
