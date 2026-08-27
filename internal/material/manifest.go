// Package material 负责把一次生成的产物落成标准材质套装。
//
// 契约见 docs/manifest-v1.md：消费端（Web 前端、UE 插件）只读 manifest，
// 不假设贴图是怎么来的。底层换工作流、换 PBR 估计模型都不影响消费端。
package material

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"time"
)

const (
	Schema        = "dreamtexture/material-manifest"
	SchemaVersion = 1
)

type Map struct {
	File       string `json:"file"`
	Colorspace string `json:"colorspace"`
	Y          string `json:"y,omitempty"`
	Packing    string `json:"packing,omitempty"`
	Role       string `json:"role,omitempty"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Bytes      int64  `json:"bytes"`
	SHA256     string `json:"sha256"`
}

type WorkflowRef struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}

// Reference 记录 img2img 用的参考图。
type Reference struct {
	// File 是 ComfyUI input 目录下的文件名。
	File string `json:"file"`
	// Origin 是来源：web 上传，或（Phase B）从 UE 里选中的 Texture。
	Origin string `json:"origin,omitempty"`
	// Denoise 是当时用的重绘幅度，越低越贴近原图。
	Denoise float64 `json:"denoise,omitempty"`
}

// Source 记录底图来自外部服务时的出处。
//
// 字段全部摊平成基本类型，不引用其它包的结构：manifest 是 Web 前端和 UE 插件
// 共同的契约，它的形状不该跟着某个内部包一起变。
//
// 这个字段存在本身就是一条信息——**它一出现就说明这份材质复现不了**。
// 云端图像接口不支持种子，同样的提示词每次结果都不同；manifest 里的 seed
// 只作用于本地那一半（无缝重整），单凭它拿不回同一张图。所以这里不再另加
// 一个"可否复现"的布尔量：老素材没有 source 字段，天然读成"本地、可复现"，
// 不需要迁移。
type Source struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	// Prompt 是真正送出去的完整文本（含模板固定的前后缀）。
	Prompt        string `json:"prompt"`
	Size          string `json:"size,omitempty"`
	Quality       string `json:"quality,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`

	InputTokens  int     `json:"input_tokens,omitempty"`
	OutputTokens int     `json:"output_tokens,omitempty"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
	ElapsedMS    int64   `json:"elapsed_ms,omitempty"`

	// Flattened 记录有没有压平亮度场，以及压平前后的边缘/中心亮度比。
	Flattened     bool    `json:"flattened"`
	FalloffBefore float64 `json:"falloff_before,omitempty"`
	FalloffAfter  float64 `json:"falloff_after,omitempty"`
}

type Generator struct {
	ComfyUI      string   `json:"comfyui,omitempty"`
	Checkpoint   string   `json:"checkpoint,omitempty"`
	PBREstimator string   `json:"pbr_estimator,omitempty"`
	NodePacks    []string `json:"node_packs,omitempty"`
}

type LicenseFlags struct {
	CommercialUse bool   `json:"commercial_use"`
	Reason        string `json:"reason,omitempty"`
}

type Manifest struct {
	Schema        string `json:"schema"`
	SchemaVersion int    `json:"schema_version"`

	ID    string `json:"id"`
	Name  string `json:"name"`
	Style string `json:"style"`

	Workflow WorkflowRef `json:"workflow"`

	Prompt     string `json:"prompt"`
	Negative   string `json:"negative,omitempty"`
	Seed       int64  `json:"seed"`
	Resolution int    `json:"resolution"`
	Tileable   bool   `json:"tileable"`

	Reference *Reference `json:"reference"`
	// Source 非空表示底图来自外部服务，见类型注释。
	Source *Source `json:"source,omitempty"`

	Maps    map[string]Map `json:"maps"`
	Preview string         `json:"preview,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	Generator    Generator     `json:"generator"`
	LicenseFlags *LicenseFlags `json:"license_flags,omitempty"`

	// Params 保留完整的注入参数，用于"参数回填再来一张"。
	Params map[string]any `json:"params,omitempty"`
}

const manifestFile = "manifest.json"

// Dir 是一个材质套装所在的目录。
type Dir struct{ path string }

func NewDir(root, id string) (*Dir, error) {
	p := filepath.Join(root, id)
	if err := os.MkdirAll(p, 0o755); err != nil {
		return nil, err
	}
	return &Dir{path: p}, nil
}

func (d *Dir) Path() string { return d.path }

// WriteMap 落盘一路贴图并返回它的 manifest 条目。
func (d *Dir) WriteMap(channel string, data []byte, colorspace, y, packing, role string) (Map, error) {
	name := channel + ".png"
	full := filepath.Join(d.path, name)
	if err := os.WriteFile(full, data, 0o644); err != nil {
		return Map{}, err
	}
	sum := sha256.Sum256(data)
	m := Map{
		File:       name,
		Colorspace: colorspace,
		Y:          y,
		Packing:    packing,
		Role:       role,
		Bytes:      int64(len(data)),
		SHA256:     hex.EncodeToString(sum[:])[:16],
	}
	if cfg, _, err := image.DecodeConfig(bytesReader(data)); err == nil {
		m.Width, m.Height = cfg.Width, cfg.Height
	}
	return m, nil
}

func (d *Dir) WriteFile(name string, data []byte) error {
	return os.WriteFile(filepath.Join(d.path, name), data, 0o644)
}

func (d *Dir) WriteManifest(m *Manifest) error {
	m.Schema, m.SchemaVersion = Schema, SchemaVersion
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(d.path, manifestFile), b, 0o644)
}

// Remove 清掉半成品目录。任务失败时调用，避免素材库里留下残缺套装。
func (d *Dir) Remove() { _ = os.RemoveAll(d.path) }

// ReadManifest 读取一个已存在的套装。
func ReadManifest(root, id string) (*Manifest, error) {
	b, err := os.ReadFile(filepath.Join(root, id, manifestFile))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("解析 %s 的 manifest: %w", id, err)
	}
	if m.Schema != Schema {
		return nil, fmt.Errorf("%s 不是 DreamTexture 材质套装（schema=%q）", id, m.Schema)
	}
	if m.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf("%s 的 manifest 版本 %d 高于本程序支持的 %d，请升级 DreamTexture",
			id, m.SchemaVersion, SchemaVersion)
	}
	return &m, nil
}
