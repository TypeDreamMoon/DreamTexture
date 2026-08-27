// Package imagen 把"底图从哪来"抽象成一个可替换的来源。
//
// 本地 SDXL 之外还接了云端 API：云端模型对提示词的理解明显更好，出图更"听话"，
// 代价是按量计费、不可复现、且天然不无缝。所以它在 DreamTexture 里的定位是
// **只负责出底图**，PBR 分解仍然全程走本地 ComfyUI——分解才是这个工具的核心，
// 而且分解不该把用户的图往外发。
package imagen

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

// Model 是一个可用的生成模型。
type Model struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Provider string `json:"provider"`
	// Sizes 是该模型支持的尺寸；空表示未知，交给服务端裁决。
	Sizes []string `json:"sizes,omitempty"`
	// Qualities 是可选的质量档。
	Qualities []string `json:"qualities,omitempty"`
	// Edits 表示支持图生图（在参考图基础上改）。
	Edits bool `json:"edits"`
	// Note 给界面显示的一句话说明。
	Note string `json:"note,omitempty"`
	// Known 为 true 表示这是我们认得的模型，参数表可信；
	// false 表示是从服务端列表里发现的新模型，参数只能靠猜。
	Known bool `json:"known"`
}

// Request 是一次底图生成请求。
type Request struct {
	Model      string
	Prompt     string
	Size       string
	Quality    string
	Background string
	// Reference 有值时走图生图（编辑）接口。
	Reference     []byte
	ReferenceName string
	// Mask 可选，仅在有 Reference 时有意义；白色区域会被重绘。
	Mask []byte
}

// Usage 是服务端报告的用量，用于把真实花费记进 manifest。
type Usage struct {
	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

// Result 是一次生成的产物。
type Result struct {
	Image []byte
	// Revised 是服务端改写后的提示词（部分模型会重写），没有则为空。
	Revised string
	Usage   Usage
	Model   string
	Elapsed time.Duration
}

// Provider 是一个能出底图的外部服务。
type Provider interface {
	ID() string
	Label() string
	// Configured 报告令牌是否已设置。没设置时 Models/Generate 都会失败，
	// 但界面仍要能列出这个来源并提示去设置。
	Configured() bool
	// Models 返回可用模型。实现应当优先向服务端问，问不到再退回内置清单。
	Models(ctx context.Context) ([]Model, error)
	// Ping 快速探一次可达性与鉴权，供环境自检使用。
	//
	// 放进接口而不是当成可选能力：这条链路上能出错的地方太多（代理、身份
	// 验证、余额），而每一种的表现都是"提交任务后干等几分钟"。任何来源都
	// 必须能便宜地自证可用。
	Ping(ctx context.Context) (detail string, err error)
	Generate(ctx context.Context, req Request) (*Result, error)
}

// ErrNoToken 表示该来源还没配置令牌。
var ErrNoToken = errors.New("尚未配置访问令牌")

// Refusal 表示服务端以内容策略为由拒绝了这次请求。
//
// 单独成型是因为它和网络错误的处理方式完全不同：重试没有意义，
// 界面上要直接告诉用户改提示词。
type Refusal struct{ Reason string }

func (e *Refusal) Error() string { return "内容策略拒绝: " + e.Reason }

// Registry 持有全部已注册的来源。
type Registry struct {
	mu   sync.RWMutex
	list []Provider
}

func NewRegistry(ps ...Provider) *Registry {
	return &Registry{list: ps}
}

func (r *Registry) Get(id string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.list {
		if p.ID() == id {
			return p, true
		}
	}
	return nil, false
}

func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]Provider{}, r.list...)
}

// TokenSource 让 Provider 在每次调用时现取令牌与接口地址。
//
// 现取而不是构造时注入，是因为用户可能在运行中才把它们填上——
// 缓存一份的话得重启后端才生效，而"填完要重启"是很糟的体验。
type TokenSource interface {
	Token(id string) string
	// Endpoint 返回用户自定义的接口地址；空串表示用官方地址。
	Endpoint(id string) string
}

// CheckProxy 校验一个代理地址是否可用作配置值。空串合法（表示走环境变量）。
func CheckProxy(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return fmt.Errorf("代理地址 %q 无法解析: %w", s, err)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("代理地址 %q 需要形如 http://127.0.0.1:7890", s)
	}
	return nil
}

// Transport 造一个尊重代理设置的 HTTP 传输层。
//
// 显式写 Proxy 是必须的：这台机器上 api.openai.com 只能经由本地代理访问
// （实测直连不通，走 127.0.0.1:7890 才有 401 回来）。http.DefaultTransport
// 本来就读环境变量，但一旦自定义 Transport 而忘了带 Proxy 字段，代理就
// 悄悄失效了——症状是超时，很难一眼看出是这个原因。
//
// proxyOf 每次请求现取，所以在设置页改完代理立刻生效，不用重启后端。
// 返回空串表示回落到环境变量，给"后端以服务方式启动、读不到用户环境变量"
// 的场景留个出口。
func Transport(proxyOf func() string) *http.Transport {
	return &http.Transport{
		Proxy: func(r *http.Request) (*url.URL, error) {
			if proxyOf != nil {
				if s := strings.TrimSpace(proxyOf()); s != "" {
					u, err := url.Parse(s)
					if err != nil {
						// 配置里的地址坏了不该让请求直接失败——回落到环境变量，
						// 真正的提示留给设置页保存时的校验。
						return http.ProxyFromEnvironment(r)
					}
					return u, nil
				}
			}
			return http.ProxyFromEnvironment(r)
		},
		MaxIdleConns:          8,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   20 * time.Second,
		ExpectContinueTimeout: 5 * time.Second,
	}
}

// ParseSize 把 "1024x1024" 拆成宽高。
func ParseSize(s string) (w, h int, ok bool) {
	parts := strings.SplitN(strings.ToLower(strings.TrimSpace(s)), "x", 2)
	if len(parts) != 2 {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[0], "%d", &w); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &h); err != nil {
		return 0, 0, false
	}
	return w, h, w > 0 && h > 0
}

// sortModels 把认得的模型排在前面，同组内按 id 倒序——
// 新版本的模型 id 通常带更大的数字，倒序能让它冒到最上面。
func sortModels(ms []Model) {
	sort.SliceStable(ms, func(i, j int) bool {
		if ms[i].Known != ms[j].Known {
			return ms[i].Known
		}
		return ms[i].ID > ms[j].ID
	})
}
