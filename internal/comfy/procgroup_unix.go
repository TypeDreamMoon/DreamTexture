//go:build !windows

package comfy

import (
	"os/exec"
	"syscall"
)

// procGroup 在类 Unix 上用进程组做同样的事：给子进程单独开一个进程组，
// 收尾时对整组发信号，避免留下孤儿进程占着端口和显存。
type procGroup struct {
	pgid int
}

func newProcGroup() (*procGroup, error) { return &procGroup{}, nil }

func (g *procGroup) prepare(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
}

func (g *procGroup) assign(cmd *exec.Cmd) error {
	if cmd.Process != nil {
		g.pgid = cmd.Process.Pid
	}
	return nil
}

func (g *procGroup) close() {
	if g.pgid > 0 {
		_ = syscall.Kill(-g.pgid, syscall.SIGKILL)
		g.pgid = 0
	}
}
