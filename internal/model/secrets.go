package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Provider 是一个需要令牌的外部服务。
type Provider struct {
	ID    string
	Label string
	// Host 是该令牌适用的域名后缀，用于下载时按目标地址挑令牌。
	// 留空表示这个令牌不用于下载（例如 OpenAI 只用来调生成接口）。
	Host string
	// Help 在界面上提示去哪里申请。
	Help string
	// Endpoint 为 true 表示这个来源还允许自定义接口地址（兼容网关、自建中转）。
	Endpoint bool
	// EndpointDefault 是留空时用的官方地址，界面上作为占位提示。
	EndpointDefault string
	// EndpointHelp 说明改这个地址意味着什么。
	EndpointHelp string
}

// Providers 是全部支持的令牌来源。新增来源只要往这里加一条。
var Providers = []Provider{
	{ID: "huggingface", Label: "HuggingFace", Host: "huggingface.co",
		Help: "用于下载受限模型（如 CHORD）。在 huggingface.co/settings/tokens 生成 read 权限令牌"},
	{ID: "civitai", Label: "Civitai", Host: "civitai.com",
		Help: "用于下载需要登录的模型。在 civitai.com/user/account 生成 API Key"},
	{ID: "openai", Label: "OpenAI",
		Help:            "用于调用 API 生成底图，按量计费。在 platform.openai.com/api-keys 生成",
		Endpoint:        true,
		EndpointDefault: "https://api.openai.com/v1",
		EndpointHelp: "改成兼容网关（自建中转、Azure、第三方代理）的地址。" +
			"注意：你的令牌和图像都会经过这个地址，只填你信得过的",
	},
}

func providerByID(id string) (Provider, bool) {
	for _, p := range Providers {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}

// Secrets 保存各服务的访问令牌，以及可自定义的接口地址。
//
// 令牌是用户自己在设置页填的，只保存在本机。这里的设计原则是**只写不读**：
// 对外只报告某个令牌"有没有设置"，永远不把值回传给任何接口。想换就重新填一次。
//
// 接口地址也放这个文件，不放普通配置：第三方网关的地址里经常直接嵌着密钥
// （形如 https://gw.example.com/sk-xxxx/v1），当成普通配置存就会跟着仓库
// 一起泄出去。放这儿它自动获得 0600 权限和 gitignore。
type Secrets struct {
	path string

	mu        sync.RWMutex
	tokens    map[string]string
	endpoints map[string]string
}

// 磁盘格式保持扁平的 <id>_token 键，与最初的两个字段兼容——
// 换成 map 是内部重构，不该让已经填过令牌的用户重填一遍。
func diskKey(id string) string {
	if id == "huggingface" {
		return "hf_token"
	}
	return id + "_token"
}

func endpointKey(id string) string { return id + "_base_url" }

func LoadSecrets(path string) (*Secrets, error) {
	s := &Secrets{path: path, tokens: map[string]string{}, endpoints: map[string]string{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, err
	}
	var raw map[string]string
	if err := json.Unmarshal(b, &raw); err != nil {
		return s, err
	}
	for _, p := range Providers {
		if v := strings.TrimSpace(raw[diskKey(p.ID)]); v != "" {
			s.tokens[p.ID] = v
		}
		if v := strings.TrimSpace(raw[endpointKey(p.ID)]); v != "" && p.Endpoint {
			s.endpoints[p.ID] = v
		}
	}
	return s, nil
}

// Status 报告各令牌是否已设置——只有布尔值，没有内容。
func (s *Secrets) Status() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := map[string]bool{}
	for _, p := range Providers {
		out[p.ID] = s.tokens[p.ID] != ""
	}
	return out
}

// Has 报告某个令牌是否已设置。
func (s *Secrets) Has(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens[id] != ""
}

// Token 取出令牌供后端自己调用外部接口。
//
// 这是唯一的读出口，只给同进程的调用方用；它的返回值绝不能出现在任何
// HTTP 响应里——对外一律只经过 Status。
func (s *Secrets) Token(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens[id]
}

// Endpoint 返回某来源的自定义接口地址；没设置则返回空串。
//
// 这个值可以回传给界面（与令牌不同），但只回传 origin——见 EndpointOrigin。
func (s *Secrets) Endpoint(id string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.endpoints[id]
}

// EndpointOrigin 只返回地址的协议与主机，去掉路径与查询串。
//
// 界面上要让用户确认"我现在指着官方还是自己的网关"，但完整地址不能回传：
// 不少第三方网关把密钥直接放在路径里。给出 origin 足以确认目标，又不泄密。
func (s *Secrets) EndpointOrigin(id string) string {
	raw := s.Endpoint(id)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "（地址无法解析）"
	}
	return u.Scheme + "://" + u.Host
}

// SetEndpoint 设置自定义接口地址。传入空字符串表示恢复官方地址。
func (s *Secrets) SetEndpoint(provider, raw string) error {
	p, ok := providerByID(provider)
	if !ok {
		return fmt.Errorf("未知的来源 %q", provider)
	}
	if !p.Endpoint {
		return fmt.Errorf("%s 不支持自定义接口地址", p.Label)
	}
	v := strings.TrimSpace(raw)
	if v != "" {
		u, err := url.Parse(v)
		if err != nil {
			return fmt.Errorf("接口地址无法解析: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("接口地址要以 http:// 或 https:// 开头，收到 %q", v)
		}
		if u.Host == "" {
			return fmt.Errorf("接口地址里没有主机名: %q", v)
		}
		// 结尾的斜杠会拼出 //images/generations 这种双斜杠路径，
		// 有的网关认、有的直接 404，统一在这里削掉。
		v = strings.TrimRight(v, "/")
	}
	s.mu.Lock()
	if v == "" {
		delete(s.endpoints, provider)
	} else {
		s.endpoints[provider] = v
	}
	s.mu.Unlock()
	return s.flush()
}

// Set 更新令牌。传入空字符串表示清除。
func (s *Secrets) Set(provider, token string) error {
	if _, ok := providerByID(provider); !ok {
		return fmt.Errorf("未知的令牌来源 %q", provider)
	}
	s.mu.Lock()
	if t := strings.TrimSpace(token); t == "" {
		delete(s.tokens, provider)
	} else {
		s.tokens[provider] = t
	}
	s.mu.Unlock()
	return s.flush()
}

func (s *Secrets) flush() error {
	s.mu.RLock()
	payload := map[string]string{}
	for id, v := range s.tokens {
		payload[diskKey(id)] = v
	}
	for id, v := range s.endpoints {
		payload[endpointKey(id)] = v
	}
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	// 手写而不用 MarshalIndent：map 的序列化顺序不定，每次改一个令牌
	// 都会把整个文件搅乱，用户想 diff 一眼都看不出改了什么。
	keys := make([]string, 0, len(payload))
	for k := range payload {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString("{\n")
	for i, k := range keys {
		kb, _ := json.Marshal(k)
		vb, _ := json.Marshal(payload[k])
		sb.WriteString("  ")
		sb.Write(kb)
		sb.WriteString(": ")
		sb.Write(vb)
		if i < len(keys)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("}\n")

	// 0600：这是凭据文件，不给同机其他用户读。
	return os.WriteFile(s.path, []byte(sb.String()), 0o600)
}

// Header 实现 Credentials：按下载源域名选用对应令牌。
func (s *Secrets) Header(sourceURL string) (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range Providers {
		if p.Host == "" {
			continue
		}
		if strings.Contains(sourceURL, p.Host) && s.tokens[p.ID] != "" {
			return "Authorization", "Bearer " + s.tokens[p.ID]
		}
	}
	return "", ""
}
