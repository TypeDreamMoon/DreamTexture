package workflow

import (
	"path/filepath"
	"runtime"
	"testing"
)

func repoWorkflows(t *testing.T) *Registry {
	t.Helper()
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("拿不到源文件路径")
	}
	dir := filepath.Join(filepath.Dir(self), "..", "..", "workflows")
	reg := NewRegistry(dir)
	if err := reg.Load(); err != nil {
		t.Fatalf("加载 %s: %v", dir, err)
	}
	return reg
}

// 全部模板都应当能按默认值渲染出一张合法的图。
// 这条测试是模板声明与模板本体对不上时的第一道拦截。
func TestAllTemplatesRenderWithDefaults(t *testing.T) {
	for _, tpl := range repoWorkflows(t).List() {
		// 直出的没有节点图，由 image_test.go 单独覆盖
		if tpl.Meta.Direct() {
			t.Logf("%-18s 纯云端直出，无节点图", tpl.Meta.ID)
			continue
		}
		r, err := tpl.Render(map[string]any{}, "probe")
		if err != nil {
			t.Errorf("%s 渲染失败: %v", tpl.Meta.ID, err)
			continue
		}
		t.Logf("%-18s %d 节点  %d 路产物  无缝=%v",
			tpl.Meta.ID, len(r.Graph), len(r.Expect), tpl.Meta.Tileable)
	}
}

// 云端底图模板的两种形态都必须成立：
//   - 无缝重整 > 0：走本地 SDXL 重画一道，采样链完整
//   - 无缝重整 = 0：直出，整条 SDXL 支路被删掉，且没有连线悬空
func TestAPISourceTemplatesBothModes(t *testing.T) {
	reg := repoWorkflows(t)
	found := 0
	for _, tpl := range reg.List() {
		// 只看「云端出底图 + 本地做 PBR 分解」那两条；
		// 纯图片管线没有无缝重整可言，也没有节点图
		if tpl.Meta.Source == nil || tpl.Meta.Kind != KindMaterial {
			continue
		}
		found++
		id := tpl.Meta.ID

		if tpl.Meta.TileableWhenPositive != "tile_fix" {
			t.Errorf("%s 应当声明 tileable_when_positive=tile_fix", id)
		}

		refine, err := tpl.Render(map[string]any{"tile_fix": 0.45, "source_image": "x.png"}, "probe")
		if err != nil {
			t.Fatalf("%s 重整模式渲染失败: %v", id, err)
		}
		if !hasTitle(refine, "dt.sampler") || !hasTitle(refine, "dt.checkpoint") {
			t.Errorf("%s 重整模式下采样链不应缺失", id)
		}

		direct, err := tpl.Render(map[string]any{"tile_fix": 0, "source_image": "x.png"}, "probe")
		if err != nil {
			t.Fatalf("%s 直出模式渲染失败: %v", id, err)
		}
		for _, gone := range []string{"dt.sampler", "dt.checkpoint", "dt.decode", "dt.source_encode"} {
			if hasTitle(direct, gone) {
				t.Errorf("%s 直出模式下 %s 应当被删掉", id, gone)
			}
		}
		if !hasTitle(direct, "dt.source_scale") {
			t.Errorf("%s 直出模式下底图缩放节点必须保留", id)
		}
		// 产物一路都不能少——直出只是换了入口，不是少做几张图。
		if len(direct.Expect) != len(refine.Expect) {
			t.Errorf("%s 两种模式产物数不一致: 直出 %d / 重整 %d",
				id, len(direct.Expect), len(refine.Expect))
		}
		// Render 内部已经查过悬空连线，这里再确认一次删得干净。
		if err := checkDangling(direct.Graph, titleMap(direct)); err != nil {
			t.Errorf("%s 直出模式有连线悬空: %v", id, err)
		}
		t.Logf("%-16s 重整 %2d 节点 / 直出 %2d 节点，产物各 %d 路",
			id, len(refine.Graph), len(direct.Graph), len(direct.Expect))
	}
	if found == 0 {
		t.Fatal("没有找到任何声明了 source 的工作流")
	}
}

// 提示词的前后缀必须原样带进送往云端的文本里——
// CHORD 依赖那段平光约束，丢了会把烘焙阴影当成真实起伏。
func TestAPISourcePromptKeepsAffixes(t *testing.T) {
	for _, tpl := range repoWorkflows(t).List() {
		src := tpl.Meta.Source
		// 只对材质管线成立：普通出图不该被平光约束限死画面
		if src == nil || tpl.Meta.Kind != KindMaterial {
			continue
		}
		key := src.Roles["prompt"]
		var p *Param
		for i := range tpl.Meta.Params {
			if tpl.Meta.Params[i].Key == key {
				p = &tpl.Meta.Params[i]
			}
		}
		if p == nil {
			t.Errorf("%s 的 source.roles.prompt 指向的参数不在基础参数里", tpl.Meta.ID)
			continue
		}
		if p.Prefix == "" && p.Suffix == "" {
			t.Errorf("%s 的提示词没有任何前后缀约束，云端底图会带上烘焙光照", tpl.Meta.ID)
		}
	}
}

func hasTitle(r *Rendered, title string) bool {
	for id := range r.Graph {
		if r.TitleOf(id) == title {
			return true
		}
	}
	return false
}

func titleMap(r *Rendered) map[string]string {
	out := map[string]string{}
	for id := range r.Graph {
		out[id] = r.TitleOf(id)
	}
	return out
}
