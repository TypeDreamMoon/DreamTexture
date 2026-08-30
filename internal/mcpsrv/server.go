// Package mcpsrv 把 DreamTexture 的能力暴露成 MCP 工具。
//
// 这不是"操作裸 ComfyUI"，而是操作整个 DreamTexture：提交生成、查任务、
// 翻素材库、看 ComfyUI 状态。与 REST 共用同一套 service 层，所以 Claude
// 与 Web 前端看到的是同一份世界。
package mcpsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/job"
	"github.com/mengye/dreamtexture/internal/material"
	"github.com/mengye/dreamtexture/internal/model"
	"github.com/mengye/dreamtexture/internal/store"
	"github.com/mengye/dreamtexture/internal/workflow"
)

type Deps struct {
	Sup       *comfy.Supervisor
	Reg       *workflow.Registry
	Store     *store.Store
	Runner    *job.Runner
	Models    *model.Manager
	Downloads *model.Downloader
	OutputDir string
	Version   string
}

// Handler 返回可挂载到 /mcp 的 HTTP 处理器。
func Handler(d Deps) http.Handler {
	srv := mcp.NewServer(&mcp.Implementation{Name: "dreamtexture", Version: d.Version}, nil)
	register(srv, d)
	return mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return srv },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
}

func register(srv *mcp.Server, d Deps) {
	mcp.AddTool(srv, &mcp.Tool{
		Name: "list_workflows",
		Description: "列出可用的材质生成工作流及其可调参数。生成前先看这个，" +
			"才知道有哪些风格预设和参数键名。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		type paramInfo struct {
			Key     string `json:"key"`
			Label   string `json:"label"`
			Type    string `json:"type"`
			Default any    `json:"default,omitempty"`
			Options []any  `json:"options,omitempty"`
			Note    string `json:"note,omitempty"`
		}
		type wfInfo struct {
			ID          string      `json:"id"`
			Name        string      `json:"name"`
			Style       string      `json:"style"`
			Description string      `json:"description"`
			Resolution  int         `json:"resolution"`
			Channels    []string    `json:"channels"`
			Params      []paramInfo `json:"params"`
			Advanced    []paramInfo `json:"advanced"`
			Commercial  *bool       `json:"commercial_use,omitempty"`
		}
		conv := func(ps []workflow.Param) []paramInfo {
			out := make([]paramInfo, 0, len(ps))
			for _, p := range ps {
				out = append(out, paramInfo{p.Key, p.Label, p.Type, p.Default, p.Options, p.Note})
			}
			return out
		}
		var list []wfInfo
		for _, t := range d.Reg.List() {
			channels := make([]string, 0, len(t.Meta.Outputs))
			for ch := range t.Meta.Outputs {
				channels = append(channels, ch)
			}
			info := wfInfo{
				ID: t.Meta.ID, Name: t.Meta.Name, Style: t.Meta.Style,
				Description: t.Meta.Description, Resolution: t.Meta.Resolution,
				Channels: channels, Params: conv(t.Meta.Params), Advanced: conv(t.Meta.Advanced),
			}
			if len(t.Meta.Licenses) > 0 {
				c, _ := t.Meta.Commercial()
				info.Commercial = &c
			}
			list = append(list, info)
		}
		return jsonResult(map[string]any{"workflows": list})
	})

	type generateArgs struct {
		WorkflowID string         `json:"workflow_id" jsonschema:"工作流 id，来自 list_workflows"`
		Prompt     string         `json:"prompt" jsonschema:"材质内容描述，只写主体即可，平铺与平光等约束由模板自动追加"`
		Variants   int            `json:"variants,omitempty" jsonschema:"变体数量，默认 1，最多 8"`
		Name       string         `json:"name,omitempty" jsonschema:"材质展示名，留空则取提示词"`
		Params     map[string]any `json:"params,omitempty" jsonschema:"其余参数，键名来自 list_workflows，例如 seed、resolution、steps"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name: "generate_material",
		Description: "提交一次 PBR 材质生成。立即返回任务 id，不等待完成；" +
			"要拿结果请接着用 wait_job。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a generateArgs) (*mcp.CallToolResult, any, error) {
		params := map[string]any{}
		for k, v := range a.Params {
			params[k] = v
		}
		if a.Prompt != "" {
			params["prompt"] = a.Prompt
		}
		jobs, err := d.Runner.Submit(job.Request{
			WorkflowID: a.WorkflowID, Params: params, Variants: a.Variants, Name: a.Name,
		})
		if err != nil {
			return errResult(err)
		}
		ids := make([]string, 0, len(jobs))
		for _, j := range jobs {
			ids = append(ids, j.ID)
		}
		return jsonResult(map[string]any{
			"job_ids": ids,
			"hint":    "用 wait_job 等待完成并拿到材质 id",
		})
	})

	type jobArgs struct {
		JobID string `json:"job_id"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_job",
		Description: "查询单个生成任务的当前状态与进度，不阻塞。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a jobArgs) (*mcp.CallToolResult, any, error) {
		j, err := d.Store.GetJob(a.JobID)
		if err != nil {
			return errResult(err)
		}
		if j == nil {
			return errResult(fmt.Errorf("任务 %s 不存在", a.JobID))
		}
		return jsonResult(j)
	})

	type waitArgs struct {
		JobID          string `json:"job_id"`
		TimeoutSeconds int    `json:"timeout_seconds,omitempty" jsonschema:"等待上限，默认 300 秒"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name: "wait_job",
		Description: "阻塞等待任务结束，返回最终状态。成功时附带材质 id、" +
			"各通道贴图的本地路径与 manifest 摘要。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a waitArgs) (*mcp.CallToolResult, any, error) {
		timeout := time.Duration(a.TimeoutSeconds) * time.Second
		if timeout <= 0 {
			timeout = 5 * time.Minute
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		t := time.NewTicker(time.Second)
		defer t.Stop()
		for {
			j, err := d.Store.GetJob(a.JobID)
			if err != nil {
				return errResult(err)
			}
			if j == nil {
				return errResult(fmt.Errorf("任务 %s 不存在", a.JobID))
			}
			if j.Status.Terminal() {
				out := map[string]any{"job": j}
				if j.Status == store.StatusSucceeded {
					if man, err := material.ReadManifest(d.OutputDir, j.MaterialID); err == nil {
						out["material_id"] = man.ID
						out["material_dir"] = d.OutputDir + "/" + man.ID
						files := map[string]string{}
						for ch, m := range man.Maps {
							files[ch] = d.OutputDir + "/" + man.ID + "/" + m.File
						}
						out["files"] = files
						out["seed"] = man.Seed
						out["tileable"] = man.Tileable
					}
				}
				return jsonResult(out)
			}
			select {
			case <-ctx.Done():
				return jsonResult(map[string]any{
					"job":     j,
					"timeout": true,
					"hint":    "任务仍在进行，可以稍后再用 get_job 查询",
				})
			case <-t.C:
			}
		}
	})

	type searchArgs struct {
		Query        string `json:"query,omitempty" jsonschema:"按名称或提示词搜索，留空返回最近的"`
		Style        string `json:"style,omitempty" jsonschema:"realistic 或 stylized"`
		FavoriteOnly bool   `json:"favorite_only,omitempty"`
		Limit        int    `json:"limit,omitempty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "search_materials",
		Description: "在素材库中搜索已生成的材质。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a searchArgs) (*mcp.CallToolResult, any, error) {
		list, err := d.Store.SearchMaterials(store.MaterialQuery{
			Text: a.Query, Style: a.Style, FavoriteOnly: a.FavoriteOnly, Limit: a.Limit,
		})
		if err != nil {
			return errResult(err)
		}
		return jsonResult(map[string]any{"materials": list, "count": len(list)})
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name: "list_models",
		Description: "盘点 ComfyUI 的模型：各类别的文件与磁盘占用，以及各工作流" +
			"所需模型是否就位。生成报缺模型时先看这个。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		inv, err := d.Models.Scan(ctx)
		if err != nil {
			return errResult(err)
		}
		type reqInfo struct {
			File        string   `json:"file"`
			Dir         string   `json:"dir"`
			Present     bool     `json:"present"`
			SizeBytes   int64    `json:"size_bytes,omitempty"`
			Auth        string   `json:"auth,omitempty"`
			Source      string   `json:"source,omitempty"`
			WorkflowIDs []string `json:"workflow_ids"`
			Note        string   `json:"note,omitempty"`
		}
		reqs := make([]reqInfo, 0, len(inv.Requirements))
		for _, r := range inv.Requirements {
			size := r.ActualBytes
			if size == 0 {
				size = r.SizeBytes
			}
			reqs = append(reqs, reqInfo{r.File, r.Dir, r.Present, size, r.Auth, r.Source, r.WorkflowIDs, r.Note})
		}
		return jsonResult(map[string]any{
			"folders":      inv.Folders,
			"total_bytes":  inv.TotalBytes,
			"file_count":   len(inv.Files),
			"requirements": reqs,
			"missing":      inv.MissingCount(),
		})
	})

	type dlArgs struct {
		File string `json:"file" jsonschema:"模型文件名，来自 list_models 的 requirements"`
		Dir  string `json:"dir" jsonschema:"所属类别目录，例如 checkpoints、loras"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name: "download_model",
		Description: "下载一个缺失的模型。只能下载工作流声明里登记过的模型；" +
			"受限来源（需要登录的）会提示改由用户自行下载。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, a dlArgs) (*mcp.CallToolResult, any, error) {
		inv := d.Models.Cached()
		if inv == nil {
			var err error
			if inv, err = d.Models.Scan(ctx); err != nil {
				return errResult(err)
			}
		}
		for i := range inv.Requirements {
			r := &inv.Requirements[i]
			if !strings.EqualFold(r.File, a.File) || !strings.EqualFold(r.Dir, a.Dir) {
				continue
			}
			if r.Present {
				return jsonResult(map[string]any{"ok": true, "note": "这个模型已经就位了"})
			}
			dl, err := d.Downloads.Enqueue(*r)
			if err != nil {
				return errResult(err)
			}
			return jsonResult(dl)
		}
		return errResult(fmt.Errorf("没有工作流声明需要 %s/%s", a.Dir, a.File))
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name: "comfy_status",
		Description: "查看 ComfyUI 的存活状态、显存占用与队列深度。" +
			"生成失败时先看这个。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return jsonResult(d.Sup.Health())
	})

	mcp.AddTool(srv, &mcp.Tool{
		Name: "comfy_restart",
		Description: "重启 ComfyUI 子进程。仅在 managed 模式下可用；" +
			"attach 模式下 ComfyUI 由用户自己管理。",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		if err := d.Sup.Restart(ctx); err != nil {
			return errResult(err)
		}
		return jsonResult(d.Sup.Health())
	})
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return errResult(err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, nil, nil
}

func errResult(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: strings.TrimSpace(err.Error())}},
	}, nil, nil
}
