package job

import (
	"strings"
	"testing"

	"github.com/mengye/dreamtexture/internal/workflow"
)

// 直出模板没有节点图。曾经 Submit 一律拿 Render 去校验参数，而 Render 见到
// 直出模板就报"没有节点图可渲染"——于是云端出图在提交那一步就死了，
// 一次 API 都没发出去。这个测试盯的就是那条路。
func TestValidateParamsDirect(t *testing.T) {
	tpl := &workflow.Template{Meta: workflow.Meta{
		ID:   "cloud-image-test",
		Kind: workflow.KindImage,
		Source: &workflow.Source{
			Kind: "api", Provider: "openai",
			ImageParam:   "prompt",
			DirectOutput: true,
		},
		Params: []workflow.Param{
			{Key: "prompt", Type: "string"},
			{Key: "api_size", Type: "enum", Options: []any{"1024x1024", "1536x1024"},
				Default: "1024x1024"},
		},
	}}
	if !tpl.Meta.Direct() {
		t.Fatal("这个模板应当是直出的，测试前提就不成立")
	}

	if err := validateParams(tpl, map[string]any{"prompt": "a lighthouse"}); err != nil {
		t.Fatalf("直出模板的参数校验不该失败: %v", err)
	}

	// 参数本身错了还是要拦住——绕开 Render 不等于不校验。
	err := validateParams(tpl, map[string]any{"prompt": "x", "api_size": "17x17"})
	if err == nil {
		t.Fatal("枚举外的取值应当被拦下")
	}
	if !strings.Contains(err.Error(), "api_size") {
		t.Fatalf("报错该指出是哪个参数，实际是: %v", err)
	}
}
