package workflow

import (
	"encoding/json"
	"strings"
	"testing"
)

// 两个最小段：出图段出一张图，分解段吃一张图存一张图。
const srcGraph = `{
  "1": {"class_type":"CheckpointLoaderSimple","inputs":{"ckpt_name":"x.safetensors"},"_meta":{"title":"dt.src.ckpt"}},
  "2": {"class_type":"CLIPTextEncode","inputs":{"text":"hi","clip":["1",1]},"_meta":{"title":"dt.src.positive"}},
  "3": {"class_type":"VAEDecode","inputs":{"samples":["2",0],"vae":["1",2]},"_meta":{"title":"dt.src.out"}}
}`

// 分解段有**两个**入口：一个喂给处理链，一个把底图原样存一份。
// 真实模板正是这样——CHORD 段要 dt.pbr_estimate + dt.out.source，
// 派生段要 dt.luminance + dt.out.basecolor。
const decGraph = `{
  "1": {"class_type":"ImageScale","inputs":{"image":null,"width":1024},"_meta":{"title":"dt.dec.in"}},
  "2": {"class_type":"SaveImage","inputs":{"images":["1",0],"filename_prefix":"dt"},"_meta":{"title":"dt.dec.save"}},
  "3": {"class_type":"SaveImage","inputs":{"images":null,"filename_prefix":"src"},"_meta":{"title":"dt.dec.save_src"}}
}`

func srcTemplate() *Template {
	return &Template{
		raw: []byte(srcGraph),
		Meta: Meta{
			ID: "src-a", Name: "出图A", Segment: SegmentSource, Domain: "realistic",
			Resolution: 1024, Tileable: true,
			Export:            &SegmentPort{Node: "dt.src.out", Slot: 0},
			Params:            []Param{{Key: "prompt", Type: "string", Target: json.RawMessage(`"dt.src.positive.text"`)}},
			NodePacks:         []string{"pack-a"},
			Licenses:          []LicenseNotice{{Component: "A", Commercial: true}},
			ModelRequirements: []ModelRequirement{{Dir: "checkpoints", File: "x.safetensors"}},
		},
	}
}

func decTemplate() *Template {
	return &Template{
		raw: []byte(decGraph),
		Meta: Meta{
			ID: "dec-b", Name: "分解B", Segment: SegmentDecompose, ExpectsDomain: []string{"realistic"},
			Tileable:          true,
			Imports:           []SegmentPort{{Node: "dt.dec.in", Input: "image"}, {Node: "dt.dec.save_src", Input: "images"}},
			Params:            []Param{{Key: "width", Type: "int", Target: json.RawMessage(`"dt.dec.in.width"`)}},
			Outputs:           map[string]Output{"basecolor": {Node: "dt.dec.save"}, "source": {Node: "dt.dec.save_src"}},
			NodePacks:         []string{"pack-b"},
			Licenses:          []LicenseNotice{{Component: "B", Commercial: false}},
			ModelRequirements: []ModelRequirement{{Dir: "checkpoints", File: "y.safetensors"}},
		},
	}
}

func TestComposeWiresSegments(t *testing.T) {
	got, err := Compose(srcTemplate(), decTemplate())
	if err != nil {
		t.Fatal(err)
	}
	if got.Meta.ID != "src-a.dec-b" {
		t.Errorf("id = %q", got.Meta.ID)
	}
	g, err := got.graph()
	if err != nil {
		t.Fatal(err)
	}
	if len(g) != 6 {
		t.Fatalf("节点数 = %d，两段各 3/3 个应当是 6", len(g))
	}

	// 出图段的 id 原样保留，分解段避开它们重编号。
	inID, ok := idByTitle(g, "dt.dec.in")
	if !ok {
		t.Fatal("找不到 dt.dec.in")
	}
	outID, _ := idByTitle(g, "dt.src.out")
	link, ok := g[inID].Inputs["image"].([]any)
	if !ok || len(link) != 2 {
		t.Fatalf("入口的 image 没接上，是 %#v", g[inID].Inputs["image"])
	}
	if link[0] != outID {
		t.Errorf("接到了 %v，应当接 dt.src.out (%s)", link[0], outID)
	}

	// 分解段内部那条连线必须跟着重编号，不能还指向旧 id。
	saveID, _ := idByTitle(g, "dt.dec.save")
	sl := g[saveID].Inputs["images"].([]any)
	if sl[0] != inID {
		t.Errorf("dt.dec.save.images 指向 %v，应当是重编号后的 dt.dec.in (%s)", sl[0], inID)
	}
	if sl[0] == "1" && inID != "1" {
		t.Error("连线还指着分解段的旧 id")
	}
}

func TestComposeWiresEveryImport(t *testing.T) {
	got, err := Compose(srcTemplate(), decTemplate())
	if err != nil {
		t.Fatal(err)
	}
	g, _ := got.graph()
	outID, _ := idByTitle(g, "dt.src.out")
	// 第二个入口漏接的话它会悬空，ComfyUI 会直接拒绝整张图——而错误信息
	// 只会说某个节点缺输入，看不出是拼接漏了。
	for _, title := range []string{"dt.dec.in", "dt.dec.save_src"} {
		id, ok := idByTitle(g, title)
		if !ok {
			t.Fatalf("找不到 %s", title)
		}
		var input string
		if title == "dt.dec.in" {
			input = "image"
		} else {
			input = "images"
		}
		link, ok := g[id].Inputs[input].([]any)
		if !ok || link[0] != outID {
			t.Errorf("%s.%s 没接到底图，是 %#v", title, input, g[id].Inputs[input])
		}
	}
}

func TestComposeMergesMeta(t *testing.T) {
	got, err := Compose(srcTemplate(), decTemplate())
	if err != nil {
		t.Fatal(err)
	}
	// 许可必须取并集：只留一边的话，"云端底图 + CHORD"会把 research-only 弄丢。
	if len(got.Meta.Licenses) != 2 {
		t.Errorf("许可提示 = %d 条，应当是两段的并集 2 条", len(got.Meta.Licenses))
	}
	if len(got.Meta.NodePacks) != 2 || len(got.Meta.ModelRequirements) != 2 {
		t.Errorf("节点包 %v / 模型 %d 没合全", got.Meta.NodePacks, len(got.Meta.ModelRequirements))
	}
	if got.Meta.SourceSegment != "src-a" || got.Meta.DecomposeSegment != "dec-b" {
		t.Error("没记下两段的来源，界面还原不出那两个下拉")
	}
	if len(got.Meta.Params) != 2 {
		t.Errorf("参数 = %d，两段各一个应当是 2", len(got.Meta.Params))
	}
	if got.Meta.Mismatch != "" {
		t.Errorf("画面类型是配的，不该报不匹配：%s", got.Meta.Mismatch)
	}
}

func TestComposeFlagsDomainMismatch(t *testing.T) {
	src := srcTemplate()
	src.Meta.Domain = "stylized" // 手绘图喂给只吃写实的分解模型
	got, err := Compose(src, decTemplate())
	if err != nil {
		t.Fatalf("画面类型不匹配只该提醒，不该拒绝：%v", err)
	}
	if got.Meta.Mismatch == "" {
		t.Error("手绘 × 只吃写实的分解模型，应当给出提醒")
	}
}

func TestComposeRejectsTitleCollision(t *testing.T) {
	dec := decTemplate()
	dec.raw = []byte(strings.Replace(decGraph, "dt.dec.in", "dt.src.out", 1))
	dec.Meta.Imports = []SegmentPort{{Node: "dt.src.out", Input: "image"}}
	_, err := Compose(srcTemplate(), dec)
	if err == nil {
		t.Fatal("标题撞了必须拒绝：不然参数会注入到另一段的同名节点上，图还合法，行为全错")
	}
	if !strings.Contains(err.Error(), "标题") {
		t.Errorf("报错没说清是标题撞了：%v", err)
	}
}

func TestComposeRejectsParamKeyCollision(t *testing.T) {
	dec := decTemplate()
	dec.Meta.Params[0].Key = "prompt" // 与出图段撞键
	dec.Meta.Params[0].Target = json.RawMessage(`"dt.dec.in.width"`)
	_, err := Compose(srcTemplate(), dec)
	if err == nil {
		t.Fatal("参数键撞了必须拒绝：前端只会渲染一个控件，另一个永远是默认值且看不出来")
	}
}

func TestComposeIsDeterministic(t *testing.T) {
	a, err := Compose(srcTemplate(), decTemplate())
	if err != nil {
		t.Fatal(err)
	}
	b, err := Compose(srcTemplate(), decTemplate())
	if err != nil {
		t.Fatal(err)
	}
	if string(a.raw) != string(b.raw) {
		t.Error("同样的输入拼出了不同的图；节点 id 每次变的话，日志和报错都对不上")
	}
}

func TestSplitComposedID(t *testing.T) {
	for _, c := range []struct {
		in   string
		s, d string
		ok   bool
	}{
		{"src-a.dec-b", "src-a", "dec-b", true},
		{"realistic-chord-v1", "", "", false},
		{".dec", "", "", false},
		{"src.", "", "", false},
	} {
		s, d, ok := SplitComposedID(c.in)
		if s != c.s || d != c.d || ok != c.ok {
			t.Errorf("SplitComposedID(%q) = %q,%q,%v；想要 %q,%q,%v", c.in, s, d, ok, c.s, c.d, c.ok)
		}
	}
}
