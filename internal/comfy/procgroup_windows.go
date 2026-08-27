//go:build windows

package comfy

import (
	"fmt"
	"os/exec"
	"unsafe"

	"golang.org/x/sys/windows"
)

// procGroup 用 Windows Job Object 兜住整棵进程树。
//
// ComfyUI 会派生子进程；只对父进程调 Kill 会留下僵尸进程继续占着 8188 端口和显存，
// 下次启动就报端口占用或 CUDA 错误。把父进程放进设了 KILL_ON_JOB_CLOSE 的 Job 里，
// 关闭句柄即可连同所有后代一起终止。
type procGroup struct {
	job windows.Handle
}

func newProcGroup() (*procGroup, error) {
	h, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 Job Object: %w", err)
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{
		BasicLimitInformation: windows.JOBOBJECT_BASIC_LIMIT_INFORMATION{
			LimitFlags: windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE,
		},
	}
	if _, err := windows.SetInformationJobObject(
		h,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		windows.CloseHandle(h)
		return nil, fmt.Errorf("配置 Job Object: %w", err)
	}
	return &procGroup{job: h}, nil
}

// prepare 在进程启动前设置属性。Windows 下无需额外设置。
func (g *procGroup) prepare(_ *exec.Cmd) {}

// assign 在进程启动后把它纳入 Job。
func (g *procGroup) assign(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("进程尚未启动")
	}
	h, err := windows.OpenProcess(
		windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		return fmt.Errorf("打开进程 %d: %w", cmd.Process.Pid, err)
	}
	defer windows.CloseHandle(h)
	if err := windows.AssignProcessToJobObject(g.job, h); err != nil {
		return fmt.Errorf("将进程加入 Job Object: %w", err)
	}
	return nil
}

// close 终止 Job 内的所有进程。
func (g *procGroup) close() {
	if g.job != 0 {
		windows.CloseHandle(g.job)
		g.job = 0
	}
}
