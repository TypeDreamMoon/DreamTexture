// Package settings 让配置可以在界面上改，并落回配置文件。
//
// 与 internal/config 的分工：config 负责"启动时怎么读、怎么校验"，
// settings 负责"运行中怎么改、怎么写回、哪些改完立刻生效"。
package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/mengye/dreamtexture/internal/config"
)

// Store 持有当前配置，并负责写回磁盘。
type Store struct {
	path string

	mu  sync.RWMutex
	cfg config.Config
}

func New(path string, cfg config.Config) *Store {
	return &Store{path: path, cfg: cfg}
}

func (s *Store) Path() string { return s.path }

func (s *Store) Get() config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// Imagen 供 imagen 包在每次请求时现取代理设置，改完不用重启。
func (s *Store) Imagen() config.Imagen {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg.Imagen
}

// Proxy 实现 imagen 需要的动态代理查询。
func (s *Store) Proxy() string { return s.Imagen().Proxy }

// Flatten 是底图亮度场压平强度。
func (s *Store) Flatten() float64 { return s.Imagen().Flatten }

// Patch 是一次配置改动。指针字段为 nil 表示"这一项不动"——
// 用零值当"不改"会让人没法把某项真的设成空串或 0。
type Patch struct {
	Proxy         *string  `json:"proxy,omitempty"`
	OpenAIBaseURL *string  `json:"openai_base_url,omitempty"`
	Flatten       *float64 `json:"flatten,omitempty"`
	RefineModel   *string  `json:"refine_model,omitempty"`

	ComfyMode        *string   `json:"comfy_mode,omitempty"`
	ComfyBaseURL     *string   `json:"comfy_base_url,omitempty"`
	ComfyPython      *string   `json:"comfy_python,omitempty"`
	ComfyMainPy      *string   `json:"comfy_main_py,omitempty"`
	ComfyExtraArgs   *[]string `json:"comfy_extra_args,omitempty"`
	ComfyAutoStart   *bool     `json:"comfy_auto_restart,omitempty"`
	ComfyReserveVRAM *float64  `json:"comfy_reserve_vram,omitempty"`
}

// restartKeys 列出改完必须重启后端才生效的字段，用于如实告诉用户。
//
// 不去做"热重载 ComfyUI 监管器"：那要把子进程、巡检、任务队列都摘干净再重建，
// 出错面积远大于让用户点一下重启。
var restartKeys = map[string]string{
	"comfy_mode":         "ComfyUI 运行模式",
	"comfy_base_url":     "ComfyUI 地址",
	"comfy_python":       "Python 解释器",
	"comfy_main_py":      "ComfyUI 主程序",
	"comfy_extra_args":   "ComfyUI 启动参数",
	"comfy_auto_restart": "自动重启",
	"comfy_reserve_vram": "显存余量",
}

// Apply 校验并写入改动，返回需要重启才生效的项。
func (s *Store) Apply(p Patch) (needRestart []string, err error) {
	s.mu.Lock()
	next := s.cfg
	touched := map[string]bool{}

	if p.Proxy != nil {
		next.Imagen.Proxy = *p.Proxy
		touched["proxy"] = true
	}
	if p.OpenAIBaseURL != nil {
		next.Imagen.OpenAIBaseURL = *p.OpenAIBaseURL
		touched["openai_base_url"] = true
	}
	if p.Flatten != nil {
		next.Imagen.Flatten = *p.Flatten
		touched["flatten"] = true
	}
	if p.RefineModel != nil {
		next.Imagen.RefineModel = *p.RefineModel
		touched["refine_model"] = true
	}
	if p.ComfyMode != nil {
		next.Comfy.Mode = config.Mode(*p.ComfyMode)
		touched["comfy_mode"] = true
	}
	if p.ComfyBaseURL != nil {
		next.Comfy.BaseURL = *p.ComfyBaseURL
		touched["comfy_base_url"] = true
	}
	if p.ComfyPython != nil {
		next.Comfy.Python = *p.ComfyPython
		touched["comfy_python"] = true
	}
	if p.ComfyMainPy != nil {
		next.Comfy.MainPy = *p.ComfyMainPy
		touched["comfy_main_py"] = true
	}
	if p.ComfyExtraArgs != nil {
		next.Comfy.ExtraArgs = *p.ComfyExtraArgs
		touched["comfy_extra_args"] = true
	}
	if p.ComfyAutoStart != nil {
		next.Comfy.AutoRestart = *p.ComfyAutoStart
		touched["comfy_auto_restart"] = true
	}
	if p.ComfyReserveVRAM != nil {
		v := *p.ComfyReserveVRAM
		if v < 0 || v > 16 {
			s.mu.Unlock()
			return nil, fmt.Errorf("显存余量只能落在 0~16 GB，收到 %v", v)
		}
		next.Comfy.ReserveVRAM = v
		touched["comfy_reserve_vram"] = true
	}

	if err := next.Validate(); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	// 先落盘再改内存：写失败时内存里还是旧值，与磁盘一致，
	// 不会出现"界面显示改了、重启后又变回去"这种最难查的不一致。
	if err := s.write(next, touched); err != nil {
		s.mu.Unlock()
		return nil, err
	}
	s.cfg = next
	s.mu.Unlock()

	for k := range touched {
		if label, ok := restartKeys[k]; ok {
			needRestart = append(needRestart, label)
		}
	}
	return needRestart, nil
}

// write 把改动合并进配置文件。
//
// 读原始 JSON 再合并，而不是把整个结构体序列化覆盖：后者会把 output_dir
// 这类路径写成 finalize 之后的绝对路径，本来写相对路径的用户会莫名其妙地
// 失去可移植性；也会抹掉文件里我们还不认识的键。
func (s *Store) write(next config.Config, touched map[string]bool) error {
	raw := map[string]any{}
	if b, err := os.ReadFile(s.path); err == nil {
		if err := json.Unmarshal(b, &raw); err != nil {
			return fmt.Errorf("配置文件 %s 不是合法 JSON，先修好再改设置: %w", s.path, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	sub := func(key string) map[string]any {
		if m, ok := raw[key].(map[string]any); ok {
			return m
		}
		m := map[string]any{}
		raw[key] = m
		return m
	}
	imagen, comfy := sub("imagen"), sub("comfy")

	set := map[string]func(){
		"proxy":              func() { imagen["proxy"] = next.Imagen.Proxy },
		"openai_base_url":    func() { imagen["openai_base_url"] = next.Imagen.OpenAIBaseURL },
		"flatten":            func() { imagen["flatten"] = next.Imagen.Flatten },
		"refine_model":       func() { imagen["refine_model"] = next.Imagen.RefineModel },
		"comfy_mode":         func() { comfy["mode"] = string(next.Comfy.Mode) },
		"comfy_base_url":     func() { comfy["base_url"] = next.Comfy.BaseURL },
		"comfy_python":       func() { comfy["python"] = next.Comfy.Python },
		"comfy_main_py":      func() { comfy["main_py"] = next.Comfy.MainPy },
		"comfy_extra_args":   func() { comfy["extra_args"] = next.Comfy.ExtraArgs },
		"comfy_auto_restart": func() { comfy["auto_restart"] = next.Comfy.AutoRestart },
		"comfy_reserve_vram": func() { comfy["reserve_vram_gb"] = next.Comfy.ReserveVRAM },
	}
	for k := range touched {
		if fn, ok := set[k]; ok {
			fn()
		}
	}

	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	// 写临时文件再改名：直接覆写时若中途断电或磁盘满，配置文件会变成半截，
	// 下次连启动都启动不了。
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
