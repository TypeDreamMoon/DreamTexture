// Package picture 负责单张图片的落盘。
//
// 与 material 分开：材质套装是一个目录、一份 manifest、多路通道，给 UE 读；
// 图片就是一张图加一小段元信息。给一张图开一个目录太重，所以这里是
// <root>/images/<id>.png 与同名 .json 并排放。
//
// 元信息仍然落在文件里而不是只进数据库——沿用整个项目的一贯做法：
// **目录是事实，数据库只是索引**。把 output 整个拷到别的机器上，
// 该有的东西都还在。
package picture

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	Schema        = "dreamtexture/picture"
	SchemaVersion = 1
	subdir        = "images"
)

// Meta 是与图片并排的那份元信息。
type Meta struct {
	Schema        string `json:"schema"`
	SchemaVersion int    `json:"schema_version"`

	ID         string `json:"id"`
	Name       string `json:"name"`
	File       string `json:"file"`
	WorkflowID string `json:"workflow_id"`

	Prompt   string `json:"prompt"`
	Negative string `json:"negative,omitempty"`
	// Seed 为 0 表示这次出图没有可复现的种子（云端接口不支持）。
	Seed   int64 `json:"seed"`
	Width  int   `json:"width"`
	Height int   `json:"height"`

	// Source 非空表示图是云端模型出的，同 material 的约定：
	// 这个字段一出现就意味着复现不了。
	Source *Source `json:"source,omitempty"`

	// Reference 记录这次用的参考图（走图生图时）。
	Reference string `json:"reference,omitempty"`

	Params    map[string]any `json:"params,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

type Source struct {
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	Size          string  `json:"size,omitempty"`
	Quality       string  `json:"quality,omitempty"`
	RevisedPrompt string  `json:"revised_prompt,omitempty"`
	InputTokens   int     `json:"input_tokens,omitempty"`
	OutputTokens  int     `json:"output_tokens,omitempty"`
	CostUSD       float64 `json:"cost_usd,omitempty"`
	ElapsedMS     int64   `json:"elapsed_ms,omitempty"`
}

// Dir 返回图片目录，必要时创建。
func Dir(root string) (string, error) {
	d := filepath.Join(root, subdir)
	return d, os.MkdirAll(d, 0o755)
}

// Write 落盘一张图片与它的元信息。data 必须是 PNG 或 JPEG。
func Write(root string, m *Meta, data []byte) error {
	d, err := Dir(root)
	if err != nil {
		return err
	}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		m.Width, m.Height = cfg.Width, cfg.Height
	}
	m.Schema, m.SchemaVersion = Schema, SchemaVersion
	m.File = m.ID + ".png"

	if err := os.WriteFile(filepath.Join(d, m.File), data, 0o644); err != nil {
		return err
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(d, m.ID+".json"), b, 0o644); err != nil {
		// 图写进去了、元信息没写成，等于留下一张来历不明的图。
		// 一起删掉比留个半成品好。
		_ = os.Remove(filepath.Join(d, m.File))
		return err
	}
	return nil
}

// Read 读回一张图片的元信息。
func Read(root, id string) (*Meta, error) {
	if err := safeID(id); err != nil {
		return nil, err
	}
	d := filepath.Join(root, subdir)
	b, err := os.ReadFile(filepath.Join(d, id+".json"))
	if err != nil {
		return nil, err
	}
	var m Meta
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, fmt.Errorf("解析 %s 的元信息: %w", id, err)
	}
	if m.Schema != Schema {
		return nil, fmt.Errorf("%s 不是 DreamTexture 图片（schema=%q）", id, m.Schema)
	}
	return &m, nil
}

// Path 返回图片本体的绝对路径。
func Path(root, id string) (string, error) {
	if err := safeID(id); err != nil {
		return "", err
	}
	return filepath.Join(root, subdir, id+".png"), nil
}

// Remove 删掉图片与它的元信息。
func Remove(root, id string) error {
	if err := safeID(id); err != nil {
		return err
	}
	d := filepath.Join(root, subdir)
	err := os.Remove(filepath.Join(d, id+".png"))
	if e := os.Remove(filepath.Join(d, id+".json")); err == nil && !os.IsNotExist(e) {
		err = e
	}
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// safeID 挡住路径穿越。
//
// id 来自 HTTP 路径参数，不校验的话 `../../configs/secrets` 这种就能读到
// 输出目录之外的东西去。
func safeID(id string) error {
	if id == "" || strings.ContainsAny(id, `/\`) || strings.Contains(id, "..") {
		return fmt.Errorf("图片 id 非法: %q", id)
	}
	return nil
}
