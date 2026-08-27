// Package config 加载并校验 DreamTexture 的运行配置。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Mode 决定后端如何得到一个可用的 ComfyUI。
type Mode string

const (
	// ModeManaged：后端拉起并监管 ComfyUI 子进程。
	ModeManaged Mode = "managed"
	// ModeAttach：连接用户自己开着的实例（含 Comfy Desktop），只做健康检查不管生命周期。
	ModeAttach Mode = "attach"
)

type Comfy struct {
	Mode Mode `json:"mode"`

	// BaseURL 是 ComfyUI 的地址。managed 模式下后端按此地址的端口启动子进程。
	BaseURL string `json:"base_url"`

	// 以下仅 managed 模式使用。
	Python    string   `json:"python"`
	MainPy    string   `json:"main_py"`
	ExtraArgs []string `json:"extra_args"`

	// StartTimeout 是等待 ComfyUI 就绪的上限。
	StartTimeout Duration `json:"start_timeout"`

	// SettleQuiet 是"启动已安定"的判据：ComfyUI 能应答之后，还要等它的日志
	// 连续这么久没有新输出，才认为可以派活。
	//
	// 起因是实测发现的一个大坑：ComfyUI 应答 /system_stats 之后，自定义节点
	// （典型是 ComfyUI-Manager 抓取节点registry）还会在后台忙很久。此时 HTTP
	// 探针依旧是毫秒级，但 Python 侧的 GIL 被占，落在这个窗口里的任务会慢上
	// 一个数量级——实测同一张图 878 秒 vs 34.7 秒。
	//
	// 这里不去认哪个节点在忙，只看日志有没有安静下来，因此不与任何插件耦合；
	// 没装 Manager 的环境日志本来就立刻安静，几乎无额外等待。设为 0 可关闭。
	SettleQuiet Duration `json:"settle_quiet"`
	// SettleTimeout 是等待安定的上限，到点就照常放行。
	SettleTimeout Duration `json:"settle_timeout"`
	// HealthInterval 是就绪后的健康巡检间隔。
	HealthInterval Duration `json:"health_interval"`
	// AutoRestart 决定子进程异常退出后是否自动拉起。
	AutoRestart bool `json:"auto_restart"`
	// MaxQueueDepth 超过后 readiness 探针报未就绪；0 表示不限。
	MaxQueueDepth int `json:"max_queue_depth"`

	// ReserveVRAM 是留给别的程序的显存（GB），转成 ComfyUI 的 --reserve-vram。
	//
	// 这台机器上多半还开着虚幻编辑器。ComfyUI 只在装载模型那一刻看一眼空闲显存，
	// 之后并不复查；等它把显存吃到只剩几百兆，UE 一涨、或者解码环节要一大块连续
	// 显存，Windows 显卡驱动就会启用"回退到系统内存"——**不报错**，改用内存
	// 硬算，慢几十倍。表面症状是 GPU 占用 99%、显存满、进度条纹丝不动。
	//
	// 留一点余量让 ComfyUI 主动把模型换出到内存，比让驱动去悄悄降级好得多。
	// 0 表示不留。
	ReserveVRAM float64 `json:"reserve_vram_gb"`
}

// Imagen 是外部底图来源的配置。
type Imagen struct {
	// Proxy 优先于 HTTPS_PROXY 等环境变量。
	//
	// 留这个出口是因为环境变量并不总是读得到：后端若以服务或计划任务方式启动，
	// 拿到的是另一套环境。而 api.openai.com 在部分网络下只能经代理访问，
	// 配不上的症状是干等到超时，很难一眼看出原因。
	Proxy string `json:"proxy"`

	// OpenAIBaseURL 可指向兼容网关（自建中转、Azure 等）。留空用官方地址。
	//
	// 不内置任何第三方中转商地址：那等于替用户决定把令牌和图交给谁。
	// 要用就自己填，界面上也会明示"你的令牌与图像会经过这个地址"。
	OpenAIBaseURL string `json:"openai_base_url"`

	// Flatten 是底图亮度场压平强度（0~1）。
	//
	// 默认开满：云端模型普遍有暗角，不压平的话平铺会出可见网格。
	// 想保留模型原本的明暗氛围就调低。
	Flatten float64 `json:"flatten"`

	// RefineModel 是扩写提示词用的文本模型。留空用内置默认值。
	//
	// 与图像模型分开配：它们往往不是同一个模型，但走同一个网关、同一把令牌。
	RefineModel string `json:"refine_model"`
}

type Config struct {
	// Addr 是 DreamTexture 自身的监听地址。
	Addr string `json:"addr"`
	// OutputDir 存放材质套装，每个套装一个子目录。
	OutputDir string `json:"output_dir"`
	// DataDir 存放 SQLite 等运行时数据。
	DataDir string `json:"data_dir"`
	// WorkflowsDir 存放工作流模板与参数声明。
	WorkflowsDir string `json:"workflows_dir"`

	Comfy  Comfy  `json:"comfy"`
	Imagen Imagen `json:"imagen"`
}

// Duration 让 time.Duration 能以 "30s" 这样的字符串写在 JSON 里。
type Duration time.Duration

func (d Duration) D() time.Duration { return time.Duration(d) }

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("时长格式无效 %q: %w", s, err)
	}
	*d = Duration(v)
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func Default() Config {
	return Config{
		Addr:         "127.0.0.1:8777",
		OutputDir:    "output",
		DataDir:      "data",
		WorkflowsDir: "workflows",
		Comfy: Comfy{
			Mode:           ModeAttach,
			BaseURL:        "http://127.0.0.1:8188",
			StartTimeout:   Duration(3 * time.Minute),
			SettleQuiet:    Duration(12 * time.Second),
			SettleTimeout:  Duration(6 * time.Minute),
			HealthInterval: Duration(10 * time.Second),
			AutoRestart:    true,
			MaxQueueDepth:  8,
			ReserveVRAM:    1,
		},
		Imagen: Imagen{Flatten: 1},
	}
}

// Load 读取配置文件；文件不存在时返回默认配置，便于开箱即用。
func Load(path string) (Config, error) {
	cfg := Default()
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, cfg.finalize()
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("解析配置 %s: %w", path, err)
	}
	return cfg, cfg.finalize()
}

// Validate 只做校验，不改动任何字段。
//
// 与 finalize 分开是为了让界面上改设置时能先验一遍再落盘——校验函数一旦
// 顺手改字段，"试着验一下"就变成了"已经改了一半"。
func (c Config) Validate() error {
	if c.Imagen.Flatten < 0 || c.Imagen.Flatten > 1 {
		return fmt.Errorf("imagen.flatten 只能落在 0~1，收到 %v", c.Imagen.Flatten)
	}
	switch c.Comfy.Mode {
	case ModeManaged:
		if c.Comfy.Python == "" || c.Comfy.MainPy == "" {
			return fmt.Errorf("managed 模式需要同时配置 comfy.python 与 comfy.main_py")
		}
	case ModeAttach:
	default:
		return fmt.Errorf("comfy.mode 只能是 managed 或 attach，收到 %q", c.Comfy.Mode)
	}
	return nil
}

func (c *Config) finalize() error {
	// 相对路径按配置文件所在目录的上一级（也就是程序根目录）解析，
	// 这样整个 DreamTexture 目录搬到别的盘也照样能跑。
	for _, p := range []*string{&c.OutputDir, &c.DataDir, &c.WorkflowsDir,
		&c.Comfy.Python, &c.Comfy.MainPy} {
		if *p == "" {
			continue
		}
		abs, err := filepath.Abs(*p)
		if err != nil {
			return err
		}
		*p = abs
	}
	return c.Validate()
}
