package config

import (
	"os"
	"path/filepath"
	"testing"
)

// 相对路径必须按**配置文件的位置**解析，不能按当前工作目录。
//
// 以前是按 cwd 的。从仓库根手动跑时两者恰好一样，所以一直没暴露；
// 装成程序、被外壳拉起来、或者从服务里启动时 cwd 各不相同，
// output/ 和 data/ 就会落到谁也想不到的地方——而且不报错，
// 只是"我的素材呢"。
func TestRelativePathsFollowConfigNotCwd(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "dreamtexture.json")
	body := `{"output_dir":"output","data_dir":"data","workflows_dir":"workflows",
	          "comfy":{"mode":"attach","base_url":"http://127.0.0.1:8188"}}`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	// 故意在别处跑，模拟"双击启动"那种 cwd 和程序目录对不上的情形。
	elsewhere := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(elsewhere); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, c := range []struct{ name, got string }{
		{"output_dir", cfg.OutputDir},
		{"data_dir", cfg.DataDir},
		{"workflows_dir", cfg.WorkflowsDir},
	} {
		want := filepath.Join(root, filepath.Base(c.got))
		if c.got != want {
			t.Errorf("%s = %s\n  期望 %s（配置文件所在目录的上一级）", c.name, c.got, want)
		}
		if filepath.Dir(c.got) == elsewhere {
			t.Errorf("%s 落到了当前工作目录，正是要避免的情况", c.name)
		}
	}
}

// 已经是绝对路径的不该被动。
func TestAbsolutePathsUntouched(t *testing.T) {
	root := t.TempDir()
	cfgDir := filepath.Join(root, "configs")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "c.json")
	abs := filepath.Join(t.TempDir(), "somewhere")
	body := `{"output_dir":` + quote(abs) + `,"comfy":{"mode":"attach"}}`
	if err := os.WriteFile(cfgPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.OutputDir != abs {
		t.Fatalf("绝对路径被改了：%s → %s", abs, cfg.OutputDir)
	}
}

func quote(s string) string {
	out := []rune{'"'}
	for _, r := range s {
		if r == '\\' || r == '"' {
			out = append(out, '\\')
		}
		out = append(out, r)
	}
	return string(append(out, '"'))
}
