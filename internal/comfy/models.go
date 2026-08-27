package comfy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ModelFolder 是一类模型（checkpoints / loras / ...）及其全部搜索路径。
//
// Paths 可能有多个：ComfyUI 自身的 models 目录，加上 extra_model_paths.yaml 里
// 挂进来的。下载新模型时要在这些路径里挑一个写入。
type ModelFolder struct {
	Name  string   `json:"name"`
	Paths []string `json:"folders"`
}

// ModelFile 是某类模型下的一个文件。
type ModelFile struct {
	Name string `json:"name"`
	// PathIndex 指向所属 ModelFolder.Paths 的下标。
	PathIndex int     `json:"pathIndex"`
	Size      int64   `json:"size"`
	Modified  float64 `json:"modified"`
	Created   float64 `json:"created"`
}

func (f ModelFile) ModTime() time.Time {
	sec := int64(f.Modified)
	return time.Unix(sec, int64((f.Modified-float64(sec))*1e9))
}

// ModelFolders 列出全部模型类别及其绝对路径。
func (c *Client) ModelFolders(ctx context.Context) ([]ModelFolder, error) {
	var out []ModelFolder
	if err := c.doJSON(ctx, http.MethodGet, "/experiment/models", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SaveUserWorkflow 把一份 UI-format 工作流写进 ComfyUI 的用户目录，
// 之后它就会出现在 ComfyUI 界面左侧的工作流列表里。
//
// 走 /userdata 接口而不是直接写文件：attach 模式下 ComfyUI 可能不在本机，
// 而且它自己知道用户目录在哪。
func (c *Client) SaveUserWorkflow(ctx context.Context, name string, data []byte) error {
	path := "/userdata/" + url.PathEscape("workflows/"+name) + "?overwrite=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("写入 ComfyUI 工作流失败 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}

// UserWorkflows 列出 ComfyUI 用户目录下的工作流文件名。
func (c *Client) UserWorkflows(ctx context.Context) ([]string, error) {
	var out []string
	if err := c.doJSON(ctx, http.MethodGet, "/userdata?dir=workflows&recurse=true", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ModelFiles 列出某一类模型下的文件。
func (c *Client) ModelFiles(ctx context.Context, folder string) ([]ModelFile, error) {
	var out []ModelFile
	path := "/experiment/models/" + url.PathEscape(folder)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
