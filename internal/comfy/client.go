// Package comfy 封装 ComfyUI 的 HTTP/WebSocket API 与子进程监管。
package comfy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	base string
	http *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: 2 * time.Minute},
	}
}

func (c *Client) BaseURL() string { return c.base }

// NodeError 是 ComfyUI 在提交阶段校验失败时给出的单条错误。
type NodeError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// PromptResponse 是 POST /prompt 的响应。
//
// 注意 NodeErrors：某些输出分支校验失败时，ComfyUI 仍会执行其余分支并把整体状态报为
// success，失败信息只出现在这里。调用方必须检查，不能只看 /history 的 status。
type PromptResponse struct {
	PromptID   string `json:"prompt_id"`
	Number     int    `json:"number"`
	NodeErrors map[string]struct {
		Errors []NodeError `json:"errors"`
	} `json:"node_errors"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Details string `json:"details"`
	} `json:"error"`
}

// NodeErrorSummary 把校验错误拍平成人能读的多行文本，空字符串表示没有错误。
func (r *PromptResponse) NodeErrorSummary(titleOf func(nodeID string) string) string {
	if len(r.NodeErrors) == 0 {
		return ""
	}
	var sb strings.Builder
	for id, ne := range r.NodeErrors {
		name := id
		if titleOf != nil {
			if t := titleOf(id); t != "" {
				name = t
			}
		}
		for _, e := range ne.Errors {
			fmt.Fprintf(&sb, "节点 %s: %s (%s)\n", name, e.Message, e.Details)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// Submit 提交一张 API-format 工作流图。graph 会被直接序列化，通常是
// workflow.Render 产出的节点表。
func (c *Client) Submit(ctx context.Context, graph any, clientID string) (*PromptResponse, error) {
	body, err := json.Marshal(map[string]any{"prompt": graph, "client_id": clientID})
	if err != nil {
		return nil, err
	}
	var out PromptResponse
	if err := c.doJSON(ctx, http.MethodPost, "/prompt", bytes.NewReader(body), &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return &out, fmt.Errorf("ComfyUI 拒绝了任务: %s - %s", out.Error.Message, out.Error.Details)
	}
	return &out, nil
}

// ImageRef 指向 ComfyUI 输出目录里的一个文件。
type ImageRef struct {
	Filename  string `json:"filename"`
	Subfolder string `json:"subfolder"`
	Type      string `json:"type"`
}

type HistoryEntry struct {
	Status struct {
		StatusStr string            `json:"status_str"`
		Completed bool              `json:"completed"`
		Messages  []json.RawMessage `json:"messages"`
	} `json:"status"`
	Outputs map[string]struct {
		Images []ImageRef `json:"images"`
	} `json:"outputs"`
}

// History 取一个任务的执行记录；第二个返回值表示记录是否已经存在。
func (c *Client) History(ctx context.Context, promptID string) (*HistoryEntry, bool, error) {
	var m map[string]HistoryEntry
	if err := c.doJSON(ctx, http.MethodGet, "/history/"+promptID, nil, &m); err != nil {
		return nil, false, err
	}
	e, ok := m[promptID]
	if !ok {
		return nil, false, nil
	}
	return &e, true, nil
}

// ExecutionError 从 history 的 messages 里挖出执行期错误，没有则返回空串。
func (e *HistoryEntry) ExecutionError() string {
	for _, raw := range e.Status.Messages {
		var msg []json.RawMessage
		if json.Unmarshal(raw, &msg) != nil || len(msg) < 2 {
			continue
		}
		var kind string
		if json.Unmarshal(msg[0], &kind) != nil {
			continue
		}
		if kind != "execution_error" {
			continue
		}
		var d struct {
			NodeID           string `json:"node_id"`
			NodeType         string `json:"node_type"`
			ExceptionType    string `json:"exception_type"`
			ExceptionMessage string `json:"exception_message"`
		}
		if json.Unmarshal(msg[1], &d) != nil {
			continue
		}
		return fmt.Sprintf("节点 %s (%s) 执行失败: %s: %s",
			d.NodeID, d.NodeType, d.ExceptionType, strings.TrimSpace(d.ExceptionMessage))
	}
	return ""
}

func (c *Client) View(ctx context.Context, ref ImageRef) ([]byte, error) {
	q := url.Values{}
	q.Set("filename", ref.Filename)
	q.Set("subfolder", ref.Subfolder)
	q.Set("type", ref.Type)
	if q.Get("type") == "" {
		q.Set("type", "output")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/view?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("取回 %s 失败: HTTP %d", ref.Filename, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// UploadImage 上传参考图，返回 ComfyUI 侧的文件名，可直接填进 LoadImage 节点。
func (c *Client) UploadImage(ctx context.Context, name string, data []byte) (string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", name)
	if err != nil {
		return "", err
	}
	if _, err := fw.Write(data); err != nil {
		return "", err
	}
	_ = w.WriteField("overwrite", "true")
	if err := w.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/upload/image", &buf)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("上传失败 HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		Name      string `json:"name"`
		Subfolder string `json:"subfolder"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Subfolder != "" {
		return out.Subfolder + "/" + out.Name, nil
	}
	return out.Name, nil
}

type SystemStats struct {
	System struct {
		ComfyUIVersion string `json:"comfyui_version"`
		PythonVersion  string `json:"python_version"`
	} `json:"system"`
	Devices []struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		VRAMTotal int64  `json:"vram_total"`
		VRAMFree  int64  `json:"vram_free"`
	} `json:"devices"`
}

func (c *Client) SystemStats(ctx context.Context) (*SystemStats, error) {
	var s SystemStats
	if err := c.doJSON(ctx, http.MethodGet, "/system_stats", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// QueueDepth 返回运行中 + 等待中的任务数。
func (c *Client) QueueDepth(ctx context.Context) (int, error) {
	var q struct {
		Running []json.RawMessage `json:"queue_running"`
		Pending []json.RawMessage `json:"queue_pending"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/queue", nil, &q); err != nil {
		return 0, err
	}
	return len(q.Running) + len(q.Pending), nil
}

// ObjectInfo 返回全部节点的自省信息，用于启动自检节点包是否齐全。
func (c *Client) ObjectInfo(ctx context.Context) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if err := c.doJSON(ctx, http.MethodGet, "/object_info", nil, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *Client) Interrupt(ctx context.Context) error {
	return c.doJSON(ctx, http.MethodPost, "/interrupt", nil, nil)
}

// Free 卸载模型释放显存。切换底座或与 UE 抢显存时调用。
func (c *Client) Free(ctx context.Context, unloadModels, freeMemory bool) error {
	body, _ := json.Marshal(map[string]bool{"unload_models": unloadModels, "free_memory": freeMemory})
	return c.doJSON(ctx, http.MethodPost, "/free", bytes.NewReader(body), nil)
}

func (c *Client) doJSON(ctx context.Context, method, path string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("%s %s: HTTP %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(b)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
