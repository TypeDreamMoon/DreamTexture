// Package catalog 提供模型的浏览与检索，让用户不用离开界面去别的网站找模型。
//
// 两个来源互补：
//   - curated：ComfyUI-Manager 维护的精选清单，五百多条，人工核对过目标目录，
//     一键下载最稳，但覆盖窄。
//   - civitai：社区库，量大、有预览图和触发词，但需要判断文件类型该落到哪个目录。
package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Entry 是一条可下载的模型，两个来源统一成这个形状。
type Entry struct {
	Source      string `json:"source"` // curated | civitai
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Author      string `json:"author,omitempty"`
	// Kind 是来源自己的类型名（Checkpoint / lora / VAE ...）。
	Kind string `json:"kind"`
	// Dir 是建议落到 ComfyUI 的哪个模型目录。
	Dir      string `json:"dir"`
	Base     string `json:"base,omitempty"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
	// SizeBytes 为 0 表示来源没给大小。
	SizeBytes    int64    `json:"size_bytes,omitempty"`
	SHA256       string   `json:"sha256,omitempty"`
	Downloads    int      `json:"downloads,omitempty"`
	TrainedWords []string `json:"trained_words,omitempty"`
	Preview      string   `json:"preview,omitempty"`
	Page         string   `json:"page,omitempty"`
	// Auth 说明下载是否需要凭据：none | civitai-login | hf-gated。
	Auth string `json:"auth"`
	// Installed 由调用方按本地清单回填。
	Installed bool `json:"installed"`
}

type Query struct {
	Text   string
	Kind   string // 统一后的目录名，例如 checkpoints / loras
	Source string // curated | civitai，空则两个都查
	Limit  int
}

type Catalog struct {
	comfyBase string
	http      *http.Client

	mu       sync.RWMutex
	curated  []Entry
	fetched  time.Time
}

func New(comfyBaseURL string) *Catalog {
	return &Catalog{
		comfyBase: strings.TrimRight(comfyBaseURL, "/"),
		http:      &http.Client{Timeout: 90 * time.Second},
	}
}

// dirForKind 把来源的类型名归一到 ComfyUI 的模型目录。
//
// 这张表是启发式的：来源的分类和 ComfyUI 的目录并非一一对应，认不出来的
// 一律留空，交给用户在界面上选——猜错目录会让模型静默失效，比让用户点一下更糟。
var dirForKind = map[string]string{
	"checkpoint": "checkpoints", "checkpoints": "checkpoints",
	"lora": "loras", "loras": "loras", "locon": "loras", "dora": "loras",
	"vae": "vae",
	"controlnet": "controlnet", "t2i-adapter": "controlnet",
	"upscale": "upscale_models", "upscaler": "upscale_models",
	"embedding": "embeddings", "textualinversion": "embeddings",
	"hypernetwork": "hypernetworks",
	"clip": "text_encoders", "textencoder": "text_encoders",
	"clip_vision": "clip_vision", "clipvision": "clip_vision",
	"diffusion_model": "diffusion_models", "unet": "diffusion_models",
	"taesd": "vae_approx",
	"gligen": "gligen",
}

func normalizeDir(kind, savePath string) string {
	// 来源自己给了明确目录就优先用它（Manager 的精选清单会给 controlnet/SDXL 这种子路径）。
	if savePath != "" && savePath != "default" {
		return savePath
	}
	return dirForKind[strings.ToLower(strings.TrimSpace(kind))]
}

// ---------- Manager 精选清单 ----------

const curatedTTL = 30 * time.Minute

func (c *Catalog) loadCurated(ctx context.Context) ([]Entry, error) {
	c.mu.RLock()
	if time.Since(c.fetched) < curatedTTL && len(c.curated) > 0 {
		defer c.mu.RUnlock()
		return c.curated, nil
	}
	c.mu.RUnlock()

	var payload struct {
		Models []struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Base        string `json:"base"`
			SavePath    string `json:"save_path"`
			Description string `json:"description"`
			Reference   string `json:"reference"`
			Filename    string `json:"filename"`
			URL         string `json:"url"`
			Size        string `json:"size"`
			Installed   string `json:"installed"`
		} `json:"models"`
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.comfyBase+"/externalmodel/getlist?mode=cache", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ComfyUI-Manager 没有提供精选模型清单（HTTP %d）", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]Entry, 0, len(payload.Models))
	for _, m := range payload.Models {
		out = append(out, Entry{
			Source: "curated", ID: m.Filename, Name: m.Name,
			Description: m.Description, Kind: m.Type, Base: m.Base,
			Dir: normalizeDir(m.Type, m.SavePath), Filename: m.Filename,
			URL: m.URL, SizeBytes: parseSize(m.Size), Page: m.Reference,
			Auth:      authForURL(m.URL),
			Installed: strings.EqualFold(m.Installed, "true"),
		})
	}
	c.mu.Lock()
	c.curated, c.fetched = out, time.Now()
	c.mu.Unlock()
	return out, nil
}

// parseSize 解析 "4.71MB" / "6.9GB" 这类人写的大小。
func parseSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}
	mult := int64(1)
	switch {
	case strings.HasSuffix(s, "GB"):
		mult, s = 1<<30, strings.TrimSuffix(s, "GB")
	case strings.HasSuffix(s, "MB"):
		mult, s = 1<<20, strings.TrimSuffix(s, "MB")
	case strings.HasSuffix(s, "KB"):
		mult, s = 1<<10, strings.TrimSuffix(s, "KB")
	case strings.HasSuffix(s, "B"):
		s = strings.TrimSuffix(s, "B")
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return int64(f * float64(mult))
}

func authForURL(u string) string {
	switch {
	case strings.Contains(u, "civitai.com"):
		return "civitai-login"
	case strings.Contains(u, "huggingface.co"):
		return "none" // 公开仓库匿名可下；gated 的会在下载时报 401 并给出提示
	}
	return "none"
}

// ---------- Civitai ----------

type civitaiFile struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	SizeKB      float64 `json:"sizeKB"`
	Primary     bool    `json:"primary"`
	DownloadURL string  `json:"downloadUrl"`
	Hashes      struct {
		SHA256 string `json:"SHA256"`
	} `json:"hashes"`
	Metadata struct {
		Format string `json:"format"`
	} `json:"metadata"`
}

type civitaiResponse struct {
	Items []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Type    string `json:"type"`
		Creator struct {
			Username string `json:"username"`
		} `json:"creator"`
		Stats struct {
			DownloadCount int `json:"downloadCount"`
		} `json:"stats"`
		ModelVersions []struct {
			Name         string   `json:"name"`
			BaseModel    string   `json:"baseModel"`
			TrainedWords []string `json:"trainedWords"`
			Images       []struct {
				URL string `json:"url"`
			} `json:"images"`
			Files []civitaiFile `json:"files"`
		} `json:"modelVersions"`
	} `json:"items"`
}

// pickFile 从一个版本的多个文件里挑真正该下的那个。
//
// 一个版本常常同时挂着模型本体、训练数据、配置甚至 VAE。优先 primary，
// 其次挑 safetensors 格式的模型文件——.ckpt/.pt 是 pickle，有执行代码的风险。
func pickFile(files []civitaiFile) *civitaiFile {
	var best *civitaiFile
	for i := range files {
		f := &files[i]
		if f.Primary {
			return f
		}
		if best == nil && strings.EqualFold(f.Type, "Model") &&
			strings.EqualFold(f.Metadata.Format, "SafeTensor") {
			best = f
		}
	}
	return best
}

// civitaiTypes 把 ComfyUI 的目录名反查成 Civitai 的 type 参数。
var civitaiTypes = map[string][]string{
	"checkpoints":    {"Checkpoint"},
	"loras":          {"LORA", "LoCon", "DoRA"},
	"vae":            {"VAE"},
	"controlnet":     {"Controlnet"},
	"upscale_models": {"Upscaler"},
	"embeddings":     {"TextualInversion"},
	"hypernetworks":  {"Hypernetwork"},
}

func (c *Catalog) searchCivitai(ctx context.Context, q Query) ([]Entry, error) {
	v := url.Values{}
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 24 // Civitai 的 limit 上限是 100，超了直接报错
	}
	v.Set("limit", strconv.Itoa(limit))
	v.Set("nsfw", "false")
	if t := strings.TrimSpace(q.Text); t != "" {
		v.Set("query", t)
	} else {
		v.Set("sort", "Most Downloaded")
	}
	for _, t := range civitaiTypes[q.Kind] {
		v.Add("types", t)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://civitai.com/api/v1/models?"+v.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 Civitai 失败（国内网络可能需要代理）: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("Civitai 限流了，稍后再试")
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("Civitai 返回 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var payload civitaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make([]Entry, 0, len(payload.Items))
	for _, it := range payload.Items {
		if len(it.ModelVersions) == 0 {
			continue
		}
		ver := it.ModelVersions[0]
		file := pickFile(ver.Files)
		if file == nil {
			continue
		}
		preview := ""
		if len(ver.Images) > 0 {
			preview = ver.Images[0].URL
		}
		out = append(out, Entry{
			Source: "civitai", ID: strconv.Itoa(it.ID),
			Name:   it.Name + " · " + ver.Name,
			Author: it.Creator.Username,
			Kind:   it.Type, Base: ver.BaseModel,
			Dir:          dirForKind[strings.ToLower(it.Type)],
			Filename:     file.Name,
			URL:          file.DownloadURL,
			SizeBytes:    int64(file.SizeKB * 1024),
			SHA256:       file.Hashes.SHA256,
			Downloads:    it.Stats.DownloadCount,
			TrainedWords: ver.TrainedWords,
			Preview:      preview,
			Page:         "https://civitai.com/models/" + strconv.Itoa(it.ID),
			Auth:         "civitai-login",
		})
	}
	return out, nil
}

// Search 合并两个来源的结果。
func (c *Catalog) Search(ctx context.Context, q Query) ([]Entry, []string, error) {
	var (
		out   []Entry
		warns []string
	)

	if q.Source == "" || q.Source == "curated" {
		cur, err := c.loadCurated(ctx)
		if err != nil {
			warns = append(warns, "精选清单不可用: "+err.Error())
		} else {
			out = append(out, filterCurated(cur, q)...)
		}
	}
	if q.Source == "" || q.Source == "civitai" {
		civ, err := c.searchCivitai(ctx, q)
		if err != nil {
			warns = append(warns, "Civitai 搜索失败: "+err.Error())
		} else {
			out = append(out, civ...)
		}
	}

	// 精选的排前面：它的目标目录是人工核对过的，比启发式判断可靠。
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].Source == "curated") != (out[j].Source == "curated") {
			return out[i].Source == "curated"
		}
		return out[i].Downloads > out[j].Downloads
	})
	if q.Limit > 0 && len(out) > q.Limit*2 {
		out = out[:q.Limit*2]
	}
	return out, warns, nil
}

func filterCurated(all []Entry, q Query) []Entry {
	terms := strings.Fields(strings.ToLower(q.Text))
	var hits []Entry
	for _, e := range all {
		if q.Kind != "" && !strings.HasPrefix(e.Dir, q.Kind) {
			continue
		}
		if len(terms) > 0 {
			hay := strings.ToLower(e.Name + " " + e.Description + " " + e.Filename + " " + e.Base)
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
		hits = append(hits, e)
	}
	return hits
}
