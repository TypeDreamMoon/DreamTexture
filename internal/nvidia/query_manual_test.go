//go:build windows

package nvidia

import (
	"os"
	"testing"
)

// 手动跑：DT_NVAPI_EXE=<某个 exe 的完整路径> go test ./internal/nvidia/ -run TestQueryReal -v
//
// 不设环境变量就跳过：这个测试读的是本机真实的驱动配置，没法在别的机器上
// 断言任何东西——写死一个路径进来只会让别人看着一头雾水。
func TestQueryReal(t *testing.T) {
	exe := os.Getenv("DT_NVAPI_EXE")
	if exe == "" {
		t.Skip("设 DT_NVAPI_EXE 指向一个 exe 才跑")
	}
	st := Query(exe)
	t.Logf("exe=%s\n  supported=%v value=%s profile=%q reason=%s",
		exe, st.Supported, st.Value, st.Profile, st.Reason)
}
