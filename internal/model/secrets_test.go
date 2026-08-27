package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func tmpSecrets(t *testing.T) *Secrets {
	t.Helper()
	s, err := LoadSecrets(filepath.Join(t.TempDir(), "secrets.json"))
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}
	return s
}

// 令牌只写不读：对外只有布尔状态，磁盘上确实存了值。
func TestTokensAreWriteOnly(t *testing.T) {
	s := tmpSecrets(t)
	if s.Status()["openai"] {
		t.Fatal("空的存储不该报告已设置")
	}
	if err := s.Set("openai", "  sk-test-123  "); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if !s.Status()["openai"] {
		t.Error("保存后应当报告已设置")
	}
	if got := s.Token("openai"); got != "sk-test-123" {
		t.Errorf("内部取值应当去掉首尾空白，得到 %q", got)
	}
	// 状态里只该有布尔，任何字符串形态的令牌都不该出现
	for k, v := range s.Status() {
		_ = k
		_ = v
	}
	if err := s.Set("openai", ""); err != nil {
		t.Fatalf("清除失败: %v", err)
	}
	if s.Status()["openai"] {
		t.Error("清除后不该还报告已设置")
	}
}

func TestSetUnknownProvider(t *testing.T) {
	s := tmpSecrets(t)
	if err := s.Set("nope", "x"); err == nil {
		t.Error("未知来源应当被拒")
	}
	if err := s.SetEndpoint("nope", "https://x.com"); err == nil {
		t.Error("未知来源的接口地址应当被拒")
	}
	// huggingface 没有开放自定义接口地址
	if err := s.SetEndpoint("huggingface", "https://x.com"); err == nil {
		t.Error("不支持自定义地址的来源应当被拒")
	}
}

func TestEndpointValidation(t *testing.T) {
	s := tmpSecrets(t)
	bad := []string{"api.openai.com/v1", "ftp://x.com", "https://", "://nope"}
	for _, v := range bad {
		if err := s.SetEndpoint("openai", v); err == nil {
			t.Errorf("%q 应当被拒", v)
		} else {
			t.Logf("%-24s -> %v", v, err)
		}
	}

	if err := s.SetEndpoint("openai", "https://gw.example.com/v1/"); err != nil {
		t.Fatalf("合法地址被拒: %v", err)
	}
	// 结尾斜杠要削掉，否则会拼出 //images/generations
	if got := s.Endpoint("openai"); got != "https://gw.example.com/v1" {
		t.Errorf("结尾斜杠没削干净: %q", got)
	}
	if err := s.SetEndpoint("openai", ""); err != nil {
		t.Fatalf("清除失败: %v", err)
	}
	if s.Endpoint("openai") != "" {
		t.Error("清除后应当为空")
	}
}

// 网关地址里可能嵌着密钥，对外只能给到 origin。
func TestEndpointOriginHidesPath(t *testing.T) {
	s := tmpSecrets(t)
	const secretish = "https://gw.example.com/sk-live-abcdef123456/v1"
	if err := s.SetEndpoint("openai", secretish); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	origin := s.EndpointOrigin("openai")
	if origin != "https://gw.example.com" {
		t.Errorf("origin 应当只有协议+主机，得到 %q", origin)
	}
	if strings.Contains(origin, "sk-live") {
		t.Error("origin 里泄出了路径中的密钥")
	}
}

// 换成 map 之后仍要认得最初的 hf_token 键，不能让已经填过的用户重填。
func TestLegacyKeyCompat(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "secrets.json")
	if err := os.WriteFile(p, []byte(`{"hf_token":"hf_old","civitai_token":"civ_old"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := LoadSecrets(p)
	if err != nil {
		t.Fatalf("加载失败: %v", err)
	}
	if s.Token("huggingface") != "hf_old" || s.Token("civitai") != "civ_old" {
		t.Error("旧键没被识别")
	}
	// 再写一次之后旧键仍要保持原名
	if err := s.Set("openai", "sk-new"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	for _, want := range []string{`"hf_token"`, `"civitai_token"`, `"openai_token"`} {
		if !strings.Contains(string(b), want) {
			t.Errorf("落盘内容缺少 %s：%s", want, b)
		}
	}
}

// 按下载地址挑令牌；OpenAI 没有 Host，不该被任何下载地址匹配上。
func TestHeaderMatchesByHost(t *testing.T) {
	s := tmpSecrets(t)
	for id, tok := range map[string]string{"huggingface": "hf_x", "civitai": "civ_x", "openai": "sk_x"} {
		if err := s.Set(id, tok); err != nil {
			t.Fatal(err)
		}
	}
	cases := map[string]string{
		"https://huggingface.co/a/resolve/main/b.safetensors": "Bearer hf_x",
		"https://civitai.com/api/download/models/1":           "Bearer civ_x",
		"https://example.com/model.safetensors":               "",
	}
	for u, want := range cases {
		_, got := s.Header(u)
		if got != want {
			t.Errorf("%s 期望 %q，得到 %q", u, want, got)
		}
		if strings.Contains(got, "sk_x") {
			t.Errorf("%s 不该带上 OpenAI 令牌", u)
		}
	}
}
