//go:build !windows

package nvidia

import "fmt"

// 非 Windows 上没有这回事：系统内存回退是 Windows 显卡驱动（WDDM）的行为，
// Linux 上显存不够就是干脆利落地 OOM。

func Query(exe string) Status {
	return Status{Exe: exe, Value: FallbackDefault.String(),
		Reason: "系统内存回退是 Windows 显卡驱动的行为，当前系统上不存在这个设置"}
}

func Apply(exe string, want Fallback) error {
	return fmt.Errorf("当前系统上没有这个设置")
}
