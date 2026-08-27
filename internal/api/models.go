package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/mengye/dreamtexture/internal/model"
)

// models 返回模型清单：类别、文件、占用，以及各工作流所需模型的就位情况。
func (s *Server) models(w http.ResponseWriter, r *http.Request) {
	inv := s.Models.Cached()
	if inv == nil || r.URL.Query().Get("refresh") == "1" {
		fresh, err := s.Models.Scan(r.Context())
		if err != nil {
			// 有缓存就先用缓存，让界面不至于空白。
			if inv == nil {
				writeErr(w, http.StatusBadGateway, "盘点模型失败: "+err.Error())
				return
			}
		} else {
			inv = fresh
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"inventory": inv,
		"missing":   inv.MissingCount(),
		"downloads": s.Downloads.List(),
	})
}

// startDownload 把一个缺失的模型加入下载队列。
//
// 只接受 file+dir 定位到已登记的需求，不接受任意 URL：下载地址一律来自
// 工作流声明，避免这个接口变成任人指使的下载器。
func (s *Server) startDownload(w http.ResponseWriter, r *http.Request) {
	var body struct {
		File string `json:"file"`
		Dir  string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	inv := s.Models.Cached()
	if inv == nil {
		var err error
		if inv, err = s.Models.Scan(r.Context()); err != nil {
			writeErr(w, http.StatusBadGateway, "盘点模型失败: "+err.Error())
			return
		}
	}
	var target *model.Requirement
	for i := range inv.Requirements {
		req := &inv.Requirements[i]
		if strings.EqualFold(req.File, body.File) && strings.EqualFold(req.Dir, body.Dir) {
			target = req
			break
		}
	}
	if target == nil {
		writeErr(w, http.StatusNotFound, "没有工作流声明需要这个模型")
		return
	}
	if target.Present {
		writeErr(w, http.StatusConflict, "这个模型已经就位了")
		return
	}
	dl, err := s.Downloads.Enqueue(*target)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, dl)
}

func (s *Server) cancelDownload(w http.ResponseWriter, r *http.Request) {
	if err := s.Downloads.Cancel(chi.URLParam(r, "id")); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) listDownloads(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"downloads": s.Downloads.List()})
}

// settings 返回可公开的设置状态。令牌只报告"有没有设置"，绝不回传内容。
//
// 一并返回令牌来源的元信息（标签、去哪申请），这样界面上新增一个来源
// 不用改前端——加在 model.Providers 里就够了。
func (s *Server) settings(w http.ResponseWriter, _ *http.Request) {
	type provider struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Help  string `json:"help"`
		Set   bool   `json:"set"`
		// Endpoint 系列描述"可否自定义接口地址"。
		//
		// EndpointOrigin 只给协议+主机，不给完整地址：不少第三方网关把密钥
		// 直接放在路径里，回传完整地址等于把它泄给任何能访问本接口的东西。
		// origin 足够让用户确认"我现在指着谁"。
		Endpoint        bool   `json:"endpoint"`
		EndpointOrigin  string `json:"endpoint_origin,omitempty"`
		EndpointDefault string `json:"endpoint_default,omitempty"`
		EndpointHelp    string `json:"endpoint_help,omitempty"`
	}
	status := s.Secrets.Status()
	provs := make([]provider, 0, len(model.Providers))
	for _, p := range model.Providers {
		provs = append(provs, provider{
			ID: p.ID, Label: p.Label, Help: p.Help, Set: status[p.ID],
			Endpoint:        p.Endpoint,
			EndpointOrigin:  s.Secrets.EndpointOrigin(p.ID),
			EndpointDefault: p.EndpointDefault,
			EndpointHelp:    p.EndpointHelp,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tokens":          status,
		"token_providers": provs,
		"comfy":           s.Sup.Health(),
		"output_dir":      s.OutputDir,
	})
}

func (s *Server) setToken(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
		Token    string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	if err := s.Secrets.Set(body.Provider, body.Token); err != nil {
		writeErr(w, http.StatusBadRequest, "保存失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tokens": s.Secrets.Status()})
}

// setEndpoint 设置某来源的自定义接口地址（兼容网关 / 自建中转）。
//
// 和令牌走同一个文件、同样的只写不读约定：回传的只有 origin，不含路径——
// 第三方网关经常把密钥直接放在路径里。
func (s *Server) setEndpoint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Provider string `json:"provider"`
		BaseURL  string `json:"base_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	if err := s.Secrets.SetEndpoint(body.Provider, body.BaseURL); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	origin := s.Secrets.EndpointOrigin(body.Provider)
	// 换了网关，之前那份模型清单就不作数了——立刻重问一次，顺便当成
	// 连通性验证：地址填错的话用户马上看到，不必等到点生成。
	msg := "已恢复官方地址"
	if origin != "" {
		msg = "已改为 " + origin
	}
	if p, ok := s.Imagen.Get(body.Provider); ok && p.Configured() {
		if detail, err := p.Ping(r.Context()); err != nil {
			msg += "；但当前不通：" + firstLine(err.Error())
		} else {
			msg += "；" + detail
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"origin": origin, "message": msg})
}
