// Package deploy 一键部署一套独立的 ComfyUI 运行环境到 DreamTexture 目录下。
//
// 目标是让这个工具自带运行时，而不是依赖某个外部整合包——整合包换了位置、
// 被清理、或者被别的工具改坏，DreamTexture 就跟着废了。
//
// 每一步都是幂等的：已经装好的直接跳过。部署动辄十几分钟且要下几个 GB，
// 中途断网、关机、手滑取消都很正常，重跑一遍必须能接着走而不是从头来。
package deploy

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Sink 收一行输出，与 comfy.Sink 同形，直接复用同一个日志缓冲。
type Sink interface {
	Append(level, source, text string)
}

// Options 是一次部署的参数。
type Options struct {
	// Dir 是运行时目录，ComfyUI 与 Python 都装在它下面。
	Dir string `json:"dir"`
	// PyVersion 形如 "3.13"。
	PyVersion string `json:"py_version"`
	// Torch 决定装哪一档 PyTorch：cu130 / cu128 / cpu。
	Torch string `json:"torch"`
	// Mirror 为真时优先用国内镜像（PyPI 走清华，ComfyUI 源走 jihulab）。
	Mirror bool `json:"mirror"`
	// ModelBasePath 是已有的模型库目录，写进 extra_model_paths.yaml 复用，
	// 免得几十 GB 的模型再下一遍。
	ModelBasePath string `json:"model_base_path"`
	// SkipTorch 跳过 PyTorch 安装，用于已经装好时的重跑。
	SkipTorch bool `json:"skip_torch"`

	// Proxy 由后端按设置页里的配置填进来，不接受客户端指定。
	//
	// 必须显式往下传：git 和 uv 读的是 HTTPS_PROXY 环境变量，而后端进程
	// 可能是从一个没有这些变量的环境里起来的（服务、计划任务、双击 exe）。
	// 不传的话 git clone GitHub 会卡到超时，"一键"就成了空话。
	Proxy string `json:"-"`
}

// nodePacks 是 DreamTexture 的工作流真正依赖的节点包。
//
// 不把整合包里那十几个都搬过来：那些是用户自己按别的需求装的，
// 部署一个"能跑 DreamTexture"的环境不需要它们，装了反而拖慢启动、
// 增加依赖冲突的面积。想要什么另外在节点页里装。
var nodePacks = []struct {
	Name string
	Repo string
	Why  string
}{
	{"ComfyUI-Manager", "https://github.com/Comfy-Org/ComfyUI-Manager", "节点页要靠它装/卸节点"},
	{"ComfyUI-Chord", "https://github.com/ubisoft/ComfyUI-Chord.git", "写实管线的 PBR 分解"},
	{"ComfyUI-seamless-tiling", "https://github.com/spinagon/ComfyUI-seamless-tiling.git", "循环卷积无缝平铺"},
	{"ComfyUI-TextureAlchemy", "https://github.com/amtarr/ComfyUI-TextureAlchemy.git", "法线/AO/通道打包等派生节点"},
}

const (
	comfyRepoGitHub = "https://github.com/Comfy-Org/ComfyUI"
	comfyRepoMirror = "https://jihulab.com/hanamizuki/comfyui"
	pypiMirror      = "https://pypi.tuna.tsinghua.edu.cn/simple"
)

// StepState 是一步的状态。
type StepState string

const (
	StatePending StepState = "pending"
	StateRunning StepState = "running"
	StateDone    StepState = "done"
	StateSkipped StepState = "skipped"
	StateFailed  StepState = "failed"
)

type Step struct {
	Key    string    `json:"key"`
	Title  string    `json:"title"`
	State  StepState `json:"state"`
	Detail string    `json:"detail,omitempty"`
	MS     int64     `json:"ms,omitempty"`
}

// Status 是部署的整体状态，供界面轮询。
type Status struct {
	Running    bool      `json:"running"`
	Steps      []Step    `json:"steps"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	// Result 是装完之后应当写进配置的两个路径。
	Python string `json:"python,omitempty"`
	MainPy string `json:"main_py,omitempty"`
}

// Deployer 同时只允许跑一次部署。
type Deployer struct {
	log Sink

	mu      sync.RWMutex
	status  Status
	cancel  context.CancelFunc
	running bool
}

func New(log Sink) *Deployer { return &Deployer{log: log} }

func (d *Deployer) Status() Status {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s := d.status
	s.Steps = append([]Step{}, d.status.Steps...)
	return s
}

func (d *Deployer) Running() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// Cancel 中断正在跑的部署。已经下好的东西留着，下次重跑会跳过。
func (d *Deployer) Cancel() {
	d.mu.Lock()
	c := d.cancel
	d.mu.Unlock()
	if c != nil {
		c()
	}
}

func (d *Deployer) say(format string, args ...any) {
	d.log.Append("INFO", "deploy", fmt.Sprintf(format, args...))
}

func (d *Deployer) warn(format string, args ...any) {
	d.log.Append("WARN", "deploy", fmt.Sprintf(format, args...))
}

var errBusy = errors.New("已经有一个部署在跑了")

// Start 在后台开始部署，立刻返回。
func (d *Deployer) Start(parent context.Context, opt Options) error {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return errBusy
	}
	if err := opt.normalize(); err != nil {
		d.mu.Unlock()
		return err
	}
	ctx, cancel := context.WithCancel(parent)
	d.running, d.cancel = true, cancel
	d.status = Status{Running: true, StartedAt: time.Now(), Steps: plan()}
	d.mu.Unlock()

	go func() {
		err := d.run(ctx, opt)
		d.mu.Lock()
		d.running = false
		d.status.Running = false
		d.status.FinishedAt = time.Now()
		if err != nil {
			d.status.Error = err.Error()
		}
		// 在锁内取出来再用，别在锁外读 status。
		took := d.status.FinishedAt.Sub(d.status.StartedAt)
		d.mu.Unlock()

		if err != nil {
			d.log.Append("ERROR", "deploy", "部署失败: "+err.Error())
		} else {
			d.say("部署完成，用时 %s", took.Round(time.Second))
		}
	}()
	return nil
}

func (o *Options) normalize() error {
	if strings.TrimSpace(o.Dir) == "" {
		return fmt.Errorf("没有指定运行时目录")
	}
	abs, err := filepath.Abs(o.Dir)
	if err != nil {
		return err
	}
	o.Dir = abs
	if o.PyVersion == "" {
		o.PyVersion = "3.13"
	}
	switch o.Torch {
	case "", "cu130":
		o.Torch = "cu130"
	case "cu128", "cpu":
	default:
		return fmt.Errorf("不认识的 PyTorch 档位 %q", o.Torch)
	}
	if runtime.GOOS != "windows" {
		return fmt.Errorf("一键部署目前只做了 Windows；其它平台请手动装 ComfyUI 后在设置里填路径")
	}
	return nil
}

func plan() []Step {
	return []Step{
		{Key: "tools", Title: "检查 uv 与 git", State: StatePending},
		{Key: "python", Title: "准备 Python", State: StatePending},
		{Key: "venv", Title: "创建虚拟环境", State: StatePending},
		{Key: "torch", Title: "安装 PyTorch", State: StatePending},
		{Key: "comfyui", Title: "获取 ComfyUI", State: StatePending},
		{Key: "deps", Title: "安装 ComfyUI 依赖", State: StatePending},
		{Key: "nodes", Title: "安装所需节点包", State: StatePending},
		{Key: "dtnodes", Title: "接入 DreamTexture 自有节点", State: StatePending},
		{Key: "modelpaths", Title: "接上已有模型库", State: StatePending},
		{Key: "verify", Title: "验证环境", State: StatePending},
	}
}

func (d *Deployer) setStep(key string, state StepState, detail string, ms int64) {
	d.mu.Lock()
	for i := range d.status.Steps {
		if d.status.Steps[i].Key == key {
			d.status.Steps[i].State = state
			d.status.Steps[i].Detail = detail
			if ms > 0 {
				d.status.Steps[i].MS = ms
			}
			break
		}
	}
	d.mu.Unlock()
}

// step 跑一步并记录状态。fn 返回 (skipped, detail, err)。
func (d *Deployer) step(ctx context.Context, key, title string,
	fn func() (bool, string, error)) error {

	if err := ctx.Err(); err != nil {
		d.setStep(key, StateFailed, "已取消", 0)
		return err
	}
	d.setStep(key, StateRunning, "", 0)
	d.say("▸ %s", title)
	start := time.Now()

	skipped, detail, err := fn()
	ms := time.Since(start).Milliseconds()
	switch {
	case err != nil:
		d.setStep(key, StateFailed, err.Error(), ms)
		return fmt.Errorf("%s：%w", title, err)
	case skipped:
		d.setStep(key, StateSkipped, detail, ms)
		d.say("  跳过：%s", detail)
	default:
		d.setStep(key, StateDone, detail, ms)
		if detail != "" {
			d.say("  完成：%s（%s）", detail, time.Duration(ms)*time.Millisecond)
		} else {
			d.say("  完成（%s）", time.Duration(ms)*time.Millisecond)
		}
	}
	return nil
}
