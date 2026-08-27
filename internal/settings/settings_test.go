package settings

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/mengye/dreamtexture/internal/config"
)

// need_restart 直接进 JSON 给前端。Go 的 nil 切片序列化成 null，前端一句
// `r.need_restart.length` 就炸——而且是在保存**成功**之后炸，用户看到的是
// 一个红色报错，配置其实已经写进去了。这个测试盯住"永远是个数组"。
func TestApplyReturnsArrayNotNull(t *testing.T) {
	dir := t.TempDir()
	st := New(filepath.Join(dir, "dreamtexture.json"), config.Default())

	v := 0.5
	got, err := st.Apply(Patch{Flatten: &v})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got == nil {
		t.Fatal("needRestart 是 nil，序列化出去会变成 null")
	}
	if len(got) != 0 {
		t.Fatalf("flatten 不需要重启，却报了 %v", got)
	}

	b, err := json.Marshal(map[string]any{"need_restart": got})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"need_restart":[]}`; string(b) != want {
		t.Fatalf("序列化成了 %s，期望 %s", b, want)
	}

	// 要重启的项照常报出来。
	gb := 2.0
	got, err = st.Apply(Patch{ComfyReserveVRAM: &gb})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(got) != 1 || got[0] != "显存余量" {
		t.Fatalf("期望报出「显存余量」需重启，实际 %v", got)
	}
}
