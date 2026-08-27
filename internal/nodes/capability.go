package nodes

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// API 是 ComfyUI-Manager 的接口世代。
//
// Manager 在 V4 做了破坏性改动：全部端点加了 /v2/ 前缀，五个逐操作端点合并成
// 一个 /v2/manager/queue/task（用 body 里的 kind 区分），节点列表端点在默认
// UI 模式下更是直接没有了。所以不能写死一套路径，必须探测。
type API int

const (
	APIUnknown API = iota
	APIv3          // 裸路径：/manager/queue/install 等
	APIv4          // /v2/ 前缀 + 统一 task 端点
)

func (a API) String() string {
	switch a {
	case APIv3:
		return "v3"
	case APIv4:
		return "v4"
	}
	return "unknown"
}

// Capability 是一次探测的结果。
type Capability struct {
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	API       string `json:"api,omitempty"`
	// HasNodeList 表示能不能从 Manager 拿到节点目录。
	// V4 的默认 UI 模式没有这个端点，得改从 Registry 取。
	HasNodeList bool   `json:"has_node_list"`
	Reason      string `json:"reason,omitempty"`
}

type caps struct {
	once sync.Once
	mu   sync.RWMutex
	val  Capability
	api  API
}

// Detect 探测 Manager 的版本与接口世代，结果会缓存。
//
// 按版本号字符串去猜能力是不可靠的（发行版可能带自己的版本号，本机的秋叶整合包
// 报的就是 V3.39.2），所以直接试端点通不通。
func (m *Manager) Detect(ctx context.Context, force bool) Capability {
	if force {
		m.caps.mu.Lock()
		m.caps.val, m.caps.api = Capability{}, APIUnknown
		m.caps.mu.Unlock()
	} else {
		m.caps.mu.RLock()
		v, a := m.caps.val, m.caps.api
		m.caps.mu.RUnlock()
		if a != APIUnknown {
			return v
		}
	}

	out := Capability{}
	api := APIUnknown

	// V4 先试：它同时保留了裸路径的兼容注册，先问 /v2 才能区分出来。
	if v, err := m.text(ctx, "/api/v2/manager/version"); err == nil && v != "" {
		api, out.Available, out.Version = APIv4, true, v
		// V4 默认 UI 没有 getlist，只有开了 legacy UI 才有。
		if legacy, err := m.text(ctx, "/api/v2/manager/is_legacy_manager_ui"); err == nil {
			out.HasNodeList = strings.Contains(strings.ToLower(legacy), "true")
		}
	} else if v, err := m.text(ctx, "/manager/version"); err == nil && v != "" {
		api, out.Available, out.Version, out.HasNodeList = APIv3, true, v, true
	} else {
		out.Reason = "这个 ComfyUI 没有启用 ComfyUI-Manager，节点管理不可用"
	}
	out.API = api.String()

	m.caps.mu.Lock()
	m.caps.val, m.caps.api = out, api
	m.caps.mu.Unlock()
	return out
}

func (m *Manager) apiGen(ctx context.Context) API {
	m.Detect(ctx, false)
	m.caps.mu.RLock()
	defer m.caps.mu.RUnlock()
	return m.caps.api
}

// path 按当前接口世代给出正确的端点路径。
func (m *Manager) path(ctx context.Context, v3, v4 string) string {
	if m.apiGen(ctx) == APIv4 {
		return v4
	}
	return v3
}

func (m *Manager) text(ctx context.Context, path string) (string, error) {
	var sb strings.Builder
	if err := m.raw(ctx, path, &sb); err != nil {
		return "", err
	}
	s := strings.TrimSpace(sb.String())
	if strings.HasPrefix(s, "<") {
		// 404 落到了前端的 index.html 上，不算真的有这个接口。
		return "", fmt.Errorf("端点 %s 不存在", path)
	}
	return s, nil
}
