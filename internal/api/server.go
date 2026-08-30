// Package api 提供 REST + WebSocket 接口。
//
// Web 前端、MCP（Claude）与将来的 UE 插件是三个对等客户端，共用同一套 service 层，
// 没有任何一方走私有通道。
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/mengye/dreamtexture/internal/catalog"
	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/deploy"
	"github.com/mengye/dreamtexture/internal/imagen"
	"github.com/mengye/dreamtexture/internal/job"
	"github.com/mengye/dreamtexture/internal/logbuf"
	"github.com/mengye/dreamtexture/internal/material"
	"github.com/mengye/dreamtexture/internal/model"
	"github.com/mengye/dreamtexture/internal/nodes"
	"github.com/mengye/dreamtexture/internal/settings"
	"github.com/mengye/dreamtexture/internal/store"
	"github.com/mengye/dreamtexture/internal/workflow"
)

type Server struct {
	Sup       *comfy.Supervisor
	Reg       *workflow.Registry
	Store     *store.Store
	Runner    *job.Runner
	Bus       *job.Bus
	Log       *slog.Logger
	OutputDir string
	DataDir   string
	Models    *model.Manager
	Downloads *model.Downloader
	Secrets   *model.Secrets
	Nodes     *nodes.Manager
	Catalog   *catalog.Catalog
	Imagen    *imagen.Registry
	Logs      *logbuf.Buffer
	Settings  *settings.Store
	Deploy    *deploy.Deployer
	// MCP 是挂在 /mcp 上的处理器，可为 nil。
	MCP http.Handler
	// Web 提供内嵌的前端，可为 nil。
	Web http.Handler
}

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", s.health)
		r.Get("/checks", s.checks)

		r.Get("/workflows", s.listWorkflows)
		r.Post("/workflows/reload", s.reloadWorkflows)
		r.Post("/workflows/import", s.importWorkflow)
		r.Get("/workflows/{id}/template", s.templateJSON)
		r.Post("/workflows/{id}/open-in-comfy", s.openInComfy)

		r.Post("/uploads", s.upload)
		r.Post("/generate", s.generate)
		r.Get("/jobs", s.listJobs)
		r.Get("/jobs/{id}", s.getJob)
		r.Post("/jobs/{id}/cancel", s.cancelJob)

		r.Get("/materials", s.listMaterials)
		r.Get("/materials/{id}", s.getMaterial)
		r.Post("/materials/{id}/favorite", s.favoriteMaterial)
		r.Get("/materials/{id}/files/{name}", s.materialFile)

		r.Get("/pictures", s.listPictures)
		r.Get("/pictures/{id}", s.getPicture)
		r.Get("/pictures/{id}/file", s.pictureFile)
		r.Post("/pictures/{id}/favorite", s.favoritePicture)
		r.Delete("/pictures/{id}", s.deletePicture)

		r.Get("/prompts/models", s.textModels)
		r.Post("/prompts/refine", s.refinePrompt)

		r.Get("/refs", s.listRefs)
		r.Post("/refs", s.addRef)
		r.Post("/refs/from-picture", s.refFromPicture)
		r.Get("/refs/{id}/file", s.refFile)
		r.Post("/refs/{id}/rename", s.renameRef)
		r.Post("/refs/{id}/use", s.useRef)
		r.Delete("/refs/{id}", s.deleteRef)

		r.Get("/models", s.models)
		r.Get("/models/downloads", s.listDownloads)
		r.Post("/models/downloads", s.startDownload)
		r.Post("/models/downloads/{id}/cancel", s.cancelDownload)

		r.Get("/catalog/models", s.browseModels)
		r.Get("/catalog/dirs", s.modelDirs)
		r.Post("/catalog/download", s.downloadFromCatalog)

		r.Get("/imagen/providers", s.imagenProviders)

		r.Get("/nodes", s.listNodes)
		r.Get("/nodes/manager", s.nodeManagerInfo)
		r.Get("/nodes/queue", s.nodeQueue)
		r.Post("/nodes/action", s.nodeAction)

		r.Get("/settings", s.settings)
		r.Post("/settings/tokens", s.setToken)
		r.Post("/settings/endpoint", s.setEndpoint)
		r.Get("/config", s.runtimeConfig)
		r.Post("/config", s.updateConfig)

		r.Get("/deploy", s.deployStatus)
		r.Post("/deploy", s.deployStart)
		r.Post("/deploy/cancel", s.deployCancel)
		r.Post("/deploy/apply", s.deployApply)

		r.Get("/comfy/versions", s.comfyVersions)
		r.Post("/comfy/versions/fetch", s.fetchComfyVersions)
		r.Post("/comfy/versions/switch", s.switchComfyVersion)

		r.Get("/logs", s.logs)

		r.Get("/comfy/status", s.comfyStatus)
		r.Get("/comfy/flags", s.comfyFlags)
		r.Post("/comfy/flags", s.setComfyFlags)
		r.Post("/comfy/restart", s.comfyRestart)
		r.Post("/comfy/start", s.comfyStart)
		r.Post("/comfy/stop", s.comfyStop)

		r.Get("/ws", s.websocket)

		// Phase B 的占位：UE 插件会注册会话并接收"发送到 UE"的推送。
		// 现在明确回 501，比 404 更能说明"接口规划了但还没实现"。
		r.HandleFunc("/ue/*", notImplemented)
		r.Post("/materials/{id}/send-to-ue", notImplemented)
	})

	if s.MCP != nil {
		r.Handle("/mcp", s.MCP)
		r.Handle("/mcp/*", s.MCP)
	}
	// 前端兜底放在最后：/api 与 /mcp 之外的路径都交给它。
	if s.Web != nil {
		r.NotFound(s.Web.ServeHTTP)
	}
	return r
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- 工作流 ----------

func (s *Server) listWorkflows(w http.ResponseWriter, _ *http.Request) {
	list := s.Reg.List()
	out := make([]workflow.Meta, 0, len(list))
	for _, t := range list {
		out = append(out, t.Meta)
	}
	// 段自己提交不了（出图段没有落盘节点，分解段入口悬空），但界面要用它们渲染
	// "出图模型 / 分解模型"那两个下拉，所以一并回传，省一次往返。
	//
	// 两个切片都用 make 而不是 var：nil 切片会序列化成 null，前端一个 .length
	// 就炸。这个坑在这个项目里已经踩过两次了。
	segs := s.Reg.Segments()
	segOut := make([]workflow.Meta, 0, len(segs))
	for _, t := range segs {
		segOut = append(segOut, t.Meta)
	}
	writeJSON(w, http.StatusOK, map[string]any{"workflows": out, "segments": segOut})
}

// ---------- 任务 ----------

func (s *Server) generate(w http.ResponseWriter, r *http.Request) {
	var req job.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	jobs, err := s.Runner.Submit(req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"jobs": jobs})
}

func (s *Server) listJobs(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	jobs, err := s.Store.ListJobs(store.Status(r.URL.Query().Get("status")), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if jobs == nil {
		jobs = []*store.Job{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) getJob(w http.ResponseWriter, r *http.Request) {
	j, err := s.Store.GetJob(chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if j == nil {
		writeErr(w, http.StatusNotFound, "任务不存在")
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (s *Server) cancelJob(w http.ResponseWriter, r *http.Request) {
	if err := s.Runner.Cancel(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---------- 素材库 ----------

func (s *Server) listMaterials(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, err := s.Store.SearchMaterials(store.MaterialQuery{
		Text:         q.Get("q"),
		Style:        q.Get("style"),
		FavoriteOnly: q.Get("fav") == "1" || q.Get("fav") == "true",
		Limit:        limit,
		Offset:       offset,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"materials": list, "fts": s.Store.HasFTS()})
}

func (s *Server) getMaterial(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	man, err := material.ReadManifest(s.OutputDir, id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "读取材质失败: "+err.Error())
		return
	}
	rec, _ := s.Store.GetMaterial(id)
	writeJSON(w, http.StatusOK, map[string]any{"manifest": man, "index": rec})
}

func (s *Server) favoriteMaterial(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Favorite bool `json:"favorite"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := s.Store.SetFavorite(chi.URLParam(r, "id"), body.Favorite); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// materialFile 提供套装内的贴图与预览。
func (s *Server) materialFile(w http.ResponseWriter, r *http.Request) {
	id, name := chi.URLParam(r, "id"), chi.URLParam(r, "name")
	// id 与文件名都来自 URL，必须挡住 ../ 之类的穿越尝试。
	if !safeSegment(id) || !safeSegment(name) {
		writeErr(w, http.StatusBadRequest, "非法路径")
		return
	}
	http.ServeFile(w, r, filepath.Join(s.OutputDir, id, name))
}

func safeSegment(s string) bool {
	return s != "" && s != "." && s != ".." &&
		!strings.ContainsAny(s, `/\`) && !strings.Contains(s, "..")
}

// ---------- ComfyUI ----------

func (s *Server) comfyStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Sup.Health())
}

func (s *Server) comfyRestart(w http.ResponseWriter, r *http.Request) {
	if err := s.Sup.Restart(r.Context()); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.Sup.Health())
}

func notImplemented(w http.ResponseWriter, _ *http.Request) {
	writeErr(w, http.StatusNotImplemented, "该接口属于 Phase B（UE 插件），尚未实现")
}

// randToken 生成一段随机十六进制，用于服务端自定的文件名。
func randToken(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"error": msg})
}
