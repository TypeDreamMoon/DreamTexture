// Package nvidia 通过 NVAPI 读写 N 卡的应用配置。
//
// 目前只用来管一件事：**CUDA 系统内存回退策略**。
//
// 为什么非要走驱动接口——显存被别的程序占满时，Windows 显卡驱动默认会悄悄
// 改用系统内存走 PCIe 算，不抛 OOM，慢几十倍。表现是 GPU 占用 99%、显存打满、
// 进度条纹丝不动，等半小时也不结束（详见 docs/comfyui-notes.md）。关掉这个回退
// 之后，显存不够会**立刻报错**，至少知道发生了什么。
//
// 这不是命令行参数，只能改驱动里针对某个可执行文件的应用配置。
package nvidia

// Fallback 是"显存不够时允不允许退回系统内存"。
type Fallback int

const (
	// FallbackDefault：没有为这个程序设过，跟随驱动全局默认（也就是允许回退）。
	FallbackDefault Fallback = iota
	// FallbackAllow：显式允许。慢，但不会因为显存不够而失败。
	FallbackAllow
	// FallbackDeny：显式禁止。显存不够立刻报 OOM，而不是无声地慢几十倍。
	FallbackDeny
)

func (f Fallback) String() string {
	switch f {
	case FallbackAllow:
		return "allow"
	case FallbackDeny:
		return "deny"
	default:
		return "default"
	}
}

// ParseFallback 解析界面传来的取值。
func ParseFallback(s string) Fallback {
	switch s {
	case "allow":
		return FallbackAllow
	case "deny":
		return FallbackDeny
	default:
		return FallbackDefault
	}
}

// Status 是对外报告的状态。
type Status struct {
	// Supported 为 false 时 Reason 说明为什么用不了（没有 N 卡、不是 Windows…）。
	Supported bool   `json:"supported"`
	Reason    string `json:"reason,omitempty"`
	// Exe 是这条配置针对的可执行文件。
	Exe string `json:"exe,omitempty"`
	// Value 是当前取值。
	Value string `json:"value"`
	// Profile 是配置在驱动里的名字，便于用户去 NVIDIA 控制面板核对。
	Profile string `json:"profile,omitempty"`
}
