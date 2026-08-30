// Package nodes 代理 ComfyUI-Manager 的节点管理能力。
//
// 不自己实现 git clone + pip install：Manager 已经把安装、依赖、版本、快照、
// 风险等级这些做得很完整，重写一遍只会多一套要维护的坑。我们做的是把它的能力
// 搬进 DreamTexture 的界面，让用户不用在两个页面之间来回跳。
package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Pack 是一个自定义节点包。字段名对齐 Manager 的 /customnode/getlist。
type Pack struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Description string `json:"description"`
	Repository  string `json:"repository"`
	Reference   string `json:"reference"`
	// State 取 enabled / disabled / not-installed。
	State       string     `json:"state"`
	Version     string     `json:"version"`
	CnrLatest   string     `json:"cnr_latest"`
	Stars       int        `json:"stars"`
	LastUpdate  flexString `json:"last_update"`
	Trust       bool       `json:"trust"`
	Files       []string   `json:"files"`
	UpdateState flexString `json:"update-state"`
}

// flexString 容忍同一字段有时是字符串、有时是数字或布尔。
//
// Manager 的节点库是社区数据拼起来的，字段类型并不整齐——例如 last_update
// 既出现过 "2026-04-19 17:07:11" 也出现过纯数字时间戳。为一条脏数据让整个
// 列表解析失败不值得。
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "null" {
		*f = ""
		return nil
	}
	if len(s) > 1 && s[0] == '"' {
		var v string
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flexString(v)
		return nil
	}
	*f = flexString(strings.Trim(s, `"`))
	return nil
}

func (f flexString) MarshalJSON() ([]byte, error) { return json.Marshal(string(f)) }

func (p Pack) Installed() bool { return p.State == "enabled" || p.State == "disabled" }

type QueueStatus struct {
	Total      int  `json:"total_count"`
	Done       int  `json:"done_count"`
	InProgress int  `json:"in_progress_count"`
	Processing bool `json:"is_processing"`
}

// Manager 是 ComfyUI-Manager 的客户端。
type Manager struct {
	base string
	http *http.Client

	mu       sync.RWMutex
	cache    []Pack
	cachedAt time.Time
	caps     caps
	// clientID 供 V4 关联任务与 WebSocket 回调。
	clientID string
}

// raw 把响应体原样写进 w，供探测端点是否存在。
func (m *Manager) raw(ctx context.Context, path string, w io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	_, err = io.Copy(w, io.LimitReader(resp.Body, 4096))
	return err
}

func New(comfyBaseURL string) *Manager {
	return &Manager{
		base: strings.TrimRight(comfyBaseURL, "/"),
		// 节点列表有 4~5MB，装节点更是要 git clone + pip install，给足时间。
		http:     &http.Client{Timeout: 10 * time.Minute},
		clientID: "dreamtexture",
	}
}

// SetBase 在 ComfyUI 地址变化时更新（例如从 attach 切到 managed）。
func (m *Manager) SetBase(u string) {
	m.mu.Lock()
	m.base = strings.TrimRight(u, "/")
	m.cache = nil
	m.mu.Unlock()
}

// Version 返回 Manager 版本，同时用来判断它装没装。
func (m *Manager) Version(ctx context.Context) (string, error) {
	c := m.Detect(ctx, false)
	if !c.Available {
		return "", fmt.Errorf("%s", c.Reason)
	}
	return c.Version, nil
}

const cacheTTL = 10 * time.Minute

// List 返回全部节点包。列表 4~5MB 且很少变，缓存起来，界面搜索才跟得上手。
func (m *Manager) List(ctx context.Context, refresh bool) ([]Pack, error) {
	m.mu.RLock()
	fresh := time.Since(m.cachedAt) < cacheTTL && len(m.cache) > 0
	cached := m.cache
	m.mu.RUnlock()
	if fresh && !refresh {
		return cached, nil
	}

	mode := "cache"
	if refresh {
		mode = "remote"
	}
	// V4 的默认 UI 模式不再提供节点目录（前端直接查 Registry）。
	// 只有开了 legacy UI 才有这个端点。
	if c := m.Detect(ctx, false); c.Available && !c.HasNodeList {
		return nil, fmt.Errorf("当前 ComfyUI-Manager（%s，%s 接口）没有提供节点目录端点。"+
			"用 --enable-manager-legacy-ui 启动 ComfyUI 可以恢复它", c.Version, c.API)
	}
	var payload struct {
		Channel   string          `json:"channel"`
		NodePacks map[string]Pack `json:"node_packs"`
	}
	listPath := m.path(ctx, "/customnode/getlist?mode=", "/api/v2/customnode/getlist?mode=") + mode
	if err := m.getJSON(ctx, listPath, &payload); err != nil {
		if fresh {
			return cached, nil // 拉取失败就先用旧的，别让界面空白
		}
		return nil, err
	}

	list := make([]Pack, 0, len(payload.NodePacks))
	for id, p := range payload.NodePacks {
		if p.ID == "" {
			p.ID = id
		}
		list = append(list, p)
	}
	// 已安装的排前面，其次按星数——搜索结果的默认顺序要有意义。
	sort.Slice(list, func(i, j int) bool {
		if list[i].Installed() != list[j].Installed() {
			return list[i].Installed()
		}
		if list[i].Stars != list[j].Stars {
			return list[i].Stars > list[j].Stars
		}
		return list[i].ID < list[j].ID
	})

	m.mu.Lock()
	m.cache, m.cachedAt = list, time.Now()
	m.mu.Unlock()
	return list, nil
}

type Query struct {
	Text          string
	State         string // enabled | disabled | not-installed | installed
	Limit, Offset int
}

// Search 在缓存的列表上做过滤。7757 个包全塞给前端不现实，在后端筛完再给。
func (m *Manager) Search(ctx context.Context, q Query) ([]Pack, int, error) {
	all, err := m.List(ctx, false)
	if err != nil {
		return nil, 0, err
	}
	text := strings.ToLower(strings.TrimSpace(q.Text))
	terms := strings.Fields(text)

	var hits []Pack
	for _, p := range all {
		switch q.State {
		case "installed":
			if !p.Installed() {
				continue
			}
		case "not-installed", "enabled", "disabled":
			if p.State != q.State {
				continue
			}
		}
		if len(terms) > 0 {
			hay := strings.ToLower(p.ID + " " + p.Title + " " + p.Author + " " + p.Description)
			ok := true
			for _, t := range terms {
				if !strings.Contains(hay, t) {
					ok = false
					break
				}
			}
			if !ok {
				continue
			}
		}
		hits = append(hits, p)
	}

	total := len(hits)
	if q.Offset > 0 && q.Offset < total {
		hits = hits[q.Offset:]
	} else if q.Offset >= total {
		hits = nil
	}
	if q.Limit > 0 && len(hits) > q.Limit {
		hits = hits[:q.Limit]
	}
	return hits, total, nil
}

// Install 把一个节点包加入 Manager 的安装队列。
//
// version 传空则装注册表最新的稳定版；传 "nightly" 走 git 仓库最新提交。
func (m *Manager) Install(ctx context.Context, p Pack, version string) error {
	if version == "" {
		if p.CnrLatest != "" {
			version = p.CnrLatest
		} else {
			version = "nightly"
		}
	}
	// 默认走注册表的稳定版（CDN zip）。nightly 会打到 github.com，
	// 国内建连常常十几秒，只在用户明确要的时候才用。
	body := map[string]any{
		"id":                p.ID,
		"version":           firstNonEmpty(p.Version, version),
		"selected_version":  version,
		"repository":        p.Repository,
		"files":             p.Files,
		"channel":           "default",
		"mode":              "cache",
		"skip_post_install": false,
	}
	return m.task(ctx, "install", "/manager/queue/install", body)
}

func (m *Manager) Uninstall(ctx context.Context, p Pack) error {
	return m.task(ctx, "uninstall", "/manager/queue/uninstall", map[string]any{
		"id": p.ID, "version": p.Version, "files": p.Files,
		"node_name": p.ID, "is_unknown": false,
		"channel": "default", "mode": "cache",
	})
}

// SetEnabled 启用或停用一个已安装的节点包。
func (m *Manager) SetEnabled(ctx context.Context, p Pack, enabled bool) error {
	body := map[string]any{
		"id": p.ID, "version": p.Version, "files": p.Files,
		"node_name": p.ID, "cnr_id": p.ID,
		"channel": "default", "mode": "cache",
	}
	if !enabled {
		return m.task(ctx, "disable", "/manager/queue/disable", body)
	}
	// V3 没有独立的 enable：走 install 且跳过后置步骤，是 Manager 自己的约定。
	body["selected_version"] = p.Version
	body["repository"] = p.Repository
	body["skip_post_install"] = true
	return m.task(ctx, "enable", "/manager/queue/install", body)
}

func (m *Manager) Update(ctx context.Context, p Pack) error {
	return m.task(ctx, "update", "/manager/queue/update", map[string]any{
		"id": p.ID, "version": p.Version, "files": p.Files,
		"node_name": p.ID, "node_ver": p.Version,
		"channel": "default", "mode": "cache",
	})
}

// task 提交一个任务并立刻让 Manager 开始处理。
//
// 两代接口形状完全不同：V3 是逐操作端点、请求体就是参数；V4 是统一的
// /v2/manager/queue/task，参数塞进 params 并用 kind 区分。
//
// 无论哪一代，入队之后都**必须**显式调 queue/start —— Manager 的 worker
// 不会自己醒来，不调的话任务会一直排着不动。
func (m *Manager) task(ctx context.Context, kind, v3Path string, params map[string]any) error {
	if m.apiGen(ctx) == APIv4 {
		body := map[string]any{
			"ui_id":     "dt-" + kind + "-" + fmt.Sprint(time.Now().UnixNano()),
			"client_id": m.clientID,
			"kind":      kind,
			"params":    params,
		}
		if err := m.post(ctx, "/api/v2/manager/queue/task", body); err != nil {
			return err
		}
	} else if err := m.post(ctx, v3Path, params); err != nil {
		return err
	}

	m.invalidate()
	if err := m.start(ctx); err != nil {
		return fmt.Errorf("任务已入队但启动失败: %w", err)
	}
	return nil
}

// start 唤醒 Manager 的 worker。已经在跑时它会返回 201，那不是错误。
func (m *Manager) start(ctx context.Context) error {
	if m.apiGen(ctx) == APIv4 {
		return m.post(ctx, "/api/v2/manager/queue/start", map[string]any{})
	}
	return m.getJSON(ctx, "/manager/queue/start", nil)
}

func (m *Manager) QueueStatus(ctx context.Context) (*QueueStatus, error) {
	var s QueueStatus
	p := m.path(ctx, "/manager/queue/status", "/api/v2/manager/queue/status?client_id="+m.clientID)
	if err := m.getJSON(ctx, p, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Reboot 让 ComfyUI 重启。装完节点必须重启才会注册进来。
func (m *Manager) Reboot(ctx context.Context) error {
	// 重启途中连接一定会断，那不是错误。
	if m.apiGen(ctx) == APIv4 {
		_ = m.post(ctx, "/api/v2/manager/reboot", map[string]any{})
	} else {
		_ = m.getJSON(ctx, "/manager/reboot", nil)
	}
	m.invalidate()
	m.caps.mu.Lock()
	m.caps.api = APIUnknown // 重启后重新探测，版本可能变了
	m.caps.mu.Unlock()
	return nil
}

func (m *Manager) invalidate() {
	m.mu.Lock()
	m.cachedAt = time.Time{}
	m.mu.Unlock()
}

func (m *Manager) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, m.base+path, nil)
	if err != nil {
		return err
	}
	resp, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp, path); err != nil {
		return err
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (m *Manager) post(ctx context.Context, path string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.base+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkStatus(resp, path)
}

func checkStatus(resp *http.Response, path string) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	msg := strings.TrimSpace(string(b))
	if resp.StatusCode == http.StatusForbidden {
		// Manager 的安全级别默认可能拦住安装类操作，直接告诉用户去哪儿改。
		return fmt.Errorf("ComfyUI-Manager 以安全级别为由拒绝了这个操作。"+
			"请把它配置里的 security_level 调成 normal 或 weak（%s）",
			"ComfyUI/user/__manager/config.ini")
	}
	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("ComfyUI-Manager 没有 %s 这个接口，可能是版本过旧", path)
	}
	return fmt.Errorf("ComfyUI-Manager 返回 HTTP %d: %s", resp.StatusCode, msg)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return "unknown"
}
