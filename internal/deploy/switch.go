package deploy

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mengye/dreamtexture/internal/comfyver"
)

// switchPlan 是切换版本要走的三步。
//
// 复用部署那套 step 机制，界面上就能看到进度——切版本可能要装几分钟依赖，
// 只给一个转圈的话用户不知道它卡在哪儿，也不知道还要多久。
func switchPlan() []Step {
	return []Step{
		{Key: "checkout", Title: "切换 ComfyUI 版本", State: StatePending},
		{Key: "deps", Title: "重装 ComfyUI 依赖", State: StatePending},
		{Key: "verify", Title: "验证环境", State: StatePending},
	}
}

// SwitchOptions 是一次版本切换。
type SwitchOptions struct {
	// MainPy 定位 ComfyUI 仓库；Python 是那个虚拟环境的解释器。
	// 两个都从当前配置里取，不接受客户端指定。
	MainPy string
	Python string
	// Ref 是要切到的提交号或 tag。
	Ref string
	// Mirror / Proxy 与部署同义，装依赖时要用。
	Mirror bool
	Proxy  string
}

// Switch 把 ComfyUI 切到指定版本，并重装依赖。
//
// **重装依赖不是可选项。** ComfyUI 的 requirements.txt 在版本之间是会变的，
// 只 checkout 不装依赖，轻则某些节点报缺包，重则一起来就崩——而症状看着
// 完全不像"版本切歪了"。
func (d *Deployer) Switch(parent context.Context, opt SwitchOptions) error {
	repo, err := comfyver.Open(opt.MainPy)
	if err != nil {
		return err
	}
	if opt.Python == "" {
		return fmt.Errorf("还没有配置 Python 解释器")
	}

	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return errBusy
	}
	ctx, cancel := context.WithCancel(parent)
	d.running, d.cancel = true, cancel
	d.status = Status{Running: true, StartedAt: time.Now(), Steps: switchPlan()}
	d.mu.Unlock()

	go func() {
		err := d.runSwitch(ctx, repo, opt)
		d.mu.Lock()
		d.running = false
		d.status.Running = false
		d.status.FinishedAt = time.Now()
		if err != nil {
			d.status.Error = err.Error()
		}
		took := d.status.FinishedAt.Sub(d.status.StartedAt)
		d.mu.Unlock()

		if err != nil {
			d.log.Append("ERROR", "deploy", "切换版本失败: "+err.Error())
		} else {
			d.say("切换完成，用时 %s。重启 ComfyUI 后生效", took.Round(time.Second))
		}
	}()
	return nil
}

func (d *Deployer) runSwitch(ctx context.Context, repo *comfyver.Repo, opt SwitchOptions) error {
	// 装依赖那步要用 uv，和部署找的是同一个。
	uvPath, err := findUV()
	if err != nil {
		return err
	}

	var target comfyver.Version
	if err := d.step(ctx, "checkout", "切换 ComfyUI 版本", func() (bool, string, error) {
		ref, err := repo.Resolve(ctx, opt.Ref)
		if err != nil {
			return false, "", err
		}
		before := repo.Status(ctx)
		if before.Ref == ref {
			return true, "已经在这个版本上了", nil
		}
		// 切之前把当前版本记进日志。出了问题要退回来时，这一行就是唯一的线索。
		d.say("  当前 %s（%s），即将切到 %s", before.Short, before.Name, ref[:7])
		if err := repo.Checkout(ctx, ref); err != nil {
			return false, "", err
		}
		after := repo.Status(ctx)
		target = comfyver.Version{Ref: after.Ref, Short: after.Short, Name: after.Name}
		return false, after.Short + "（" + after.Name + "）", nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "deps", "重装 ComfyUI 依赖", func() (bool, string, error) {
		req := filepath.Join(repo.Dir(), "requirements.txt")
		if !fileExists(req) {
			return false, "", fmt.Errorf("没有找到 %s", req)
		}
		o := Options{Mirror: opt.Mirror, Proxy: opt.Proxy}
		if err := d.pipInstall(ctx, o, uvPath, opt.Python, "-r", req); err != nil {
			return false, "", err
		}
		// 和部署那步同一个理由：ComfyUI 的 requirements 里列着 torch，
		// 而 PyPI 上的是 CPU 版，装完很可能把 CUDA 版顶掉。静默故障，
		// 表现只是"生成慢了十几倍"，所以每次装完都要回头验一遍。
		if fixed, err := d.ensureCUDATorch(ctx, o, uvPath, opt.Python); err != nil {
			return false, "", err
		} else if fixed {
			return false, "依赖把 torch 换成了 CPU 版，已重新装回 CUDA 版", nil
		}
		return false, "", nil
	}); err != nil {
		return err
	}

	return d.step(ctx, "verify", "验证环境", func() (bool, string, error) {
		out, err := d.capture(ctx, opt.Python, "-c",
			"import torch;print(torch.__version__, torch.cuda.is_available())")
		if err != nil {
			return false, "", fmt.Errorf("虚拟环境跑不起来了：%w", err)
		}
		name := target.Name
		if name == "" {
			name = target.Short
		}
		return false, fmt.Sprintf("%s · torch %s", name, out), nil
	})
}
