package comfy

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"
)

func cat() []Option { return Catalog([]GPU{{Index: 0, Name: "RTX 4070 Ti SUPER", TotalM: 16376}}) }

// 认不出来的参数必须原样留着。目录跟不上上游 cli_args.py 是早晚的事，
// 而把用户手写的参数悄悄吃掉是不可接受的。
func TestParseKeepsUnknown(t *testing.T) {
	in := []string{
		"--lowvram",
		"--whitelist-custom-nodes", "ComfyUI-Chord",
		"--use-sage-attention",
		"--some-future-flag",
	}
	values, leftover := Parse(in, cat())

	if values["vram"] != "lowvram" {
		t.Fatalf("vram = %q，期望 lowvram", values["vram"])
	}
	if values["attn"] != "sage" {
		t.Fatalf("attn = %q，期望 sage", values["attn"])
	}
	want := []string{"--whitelist-custom-nodes", "ComfyUI-Chord", "--some-future-flag"}
	if !reflect.DeepEqual(leftover, want) {
		t.Fatalf("leftover = %v，期望 %v", leftover, want)
	}
}

// --k=v 和 --k v 是 argparse 都认的两种写法。不统一的话界面会显示成默认值，
// 而实际跑的是另一个——最难发现的那类不一致。
func TestParseEqualsForm(t *testing.T) {
	values, leftover := Parse([]string{"--preview-method=latent2rgb", "--cuda-device=0"}, cat())
	if values["preview"] != "latent2rgb" {
		t.Fatalf("preview = %q", values["preview"])
	}
	if values["device"] != "cuda0" {
		t.Fatalf("device = %q", values["device"])
	}
	if len(leftover) != 0 {
		t.Fatalf("不该有剩余，实际 %v", leftover)
	}
}

// --fast fp16_accumulation 必须先于光秃秃的 --fast 匹配上，
// 否则界面会把"仅 fp16 累加"显示成"全开"，而那是两种不同的行为。
func TestParseLongestChoiceWins(t *testing.T) {
	v, _ := Parse([]string{"--fast", "fp16_accumulation"}, cat())
	if v["fast"] != "fp16acc" {
		t.Fatalf("fast = %q，期望 fp16acc", v["fast"])
	}
	v, _ = Parse([]string{"--fast"}, cat())
	if v["fast"] != "all" {
		t.Fatalf("fast = %q，期望 all", v["fast"])
	}
}

// Invert 的项：界面上"开着"才是常态，对应的是**不加**参数。
func TestInvertedBool(t *testing.T) {
	v, _ := Parse(nil, cat())
	if v["smartmem"] != "true" {
		t.Fatalf("没写参数时智能显存优化该是开着的，实际 %q", v["smartmem"])
	}
	if got := Build(v, nil, cat()); contains(got, "--disable-smart-memory") {
		t.Fatalf("开着却加了 --disable-smart-memory: %v", got)
	}

	v["smartmem"] = "false"
	if got := Build(v, nil, cat()); !contains(got, "--disable-smart-memory") {
		t.Fatalf("关掉了却没加 --disable-smart-memory: %v", got)
	}
}

// Parse → Build 要能还原：界面上什么都不改就保存，参数不该发生变化。
func TestRoundTrip(t *testing.T) {
	in := []string{
		"--cuda-device", "0",
		"--lowvram",
		"--use-pytorch-cross-attention",
		"--preview-method", "none",
		"--disable-smart-memory",
		"--cpu-vae",
		"--fp32-vae",
		"--deterministic",
		"--whitelist-custom-nodes", "ComfyUI-Chord",
	}
	values, leftover := Parse(in, cat())
	out := Build(values, leftover, cat())

	sortedEq := func(a, b []string) bool {
		x, y := append([]string{}, a...), append([]string{}, b...)
		sort.Strings(x)
		sort.Strings(y)
		return reflect.DeepEqual(x, y)
	}
	if !sortedEq(in, out) {
		t.Fatalf("往返之后变了：\n  进 %v\n  出 %v", in, out)
	}

	// 再走一圈必须完全稳定，否则每次打开设置页都会显示"有改动"。
	values2, leftover2 := Parse(out, cat())
	out2 := Build(values2, leftover2, cat())
	if !reflect.DeepEqual(out, out2) {
		t.Fatalf("第二圈不稳定：\n  %v\n  %v", out, out2)
	}
}

// 后端自己管的参数要认出来，否则会出现"两个地方说了不同的话，后写的赢"。
func TestManagedIn(t *testing.T) {
	got := ManagedIn([]string{"--listen", "0.0.0.0", "--lowvram", "--reserve-vram=2"})
	want := []string{"--listen", "--reserve-vram"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	// 一个都没有时必须是空切片而不是 nil。nil 进 JSON 会变成 null，
	// 前端 `managed.length` 一炸，设置页上整块「性能」就消失了——
	// 除了控制台没有任何线索。这条踩过一次。
	empty := ManagedIn([]string{"--lowvram"})
	if empty == nil {
		t.Fatal("返回了 nil，序列化出去会变成 null")
	}
	b, err := json.Marshal(map[string]any{"managed": empty})
	if err != nil {
		t.Fatal(err)
	}
	if w := `{"managed":[]}`; string(b) != w {
		t.Fatalf("序列化成了 %s，期望 %s", b, w)
	}
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
