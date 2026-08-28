package api

import (
	"encoding/json"
	"net/http"

	"github.com/mengye/dreamtexture/internal/comfyver"
	"github.com/mengye/dreamtexture/internal/deploy"
)

// comfyVersions 列出 ComfyUI 可切换的版本。
//
// 仓库位置由配置里的 main_py 推出来，不接受客户端指定——那个值决定了我们要去
// 哪个目录跑 git checkout，让客户端说了算等于开了个后门。
func (s *Server) comfyVersions(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind != comfyver.KindDev {
		kind = comfyver.KindStable
	}

	out := map[string]any{
		"kind":     kind,
		"versions": []comfyver.Version{},
		"status":   comfyver.Status{},
	}
	repo, err := comfyver.Open(s.Settings.Get().Comfy.MainPy)
	if err != nil {
		out["status"] = comfyver.Status{Reason: err.Error()}
		writeJSON(w, http.StatusOK, out)
		return
	}

	st := repo.Status(r.Context())
	out["status"] = st

	// 浅克隆列不出东西，但这不是错误——如实说清楚，让界面去引导补一次历史。
	if st.Shallow {
		writeJSON(w, http.StatusOK, out)
		return
	}
	list, err := repo.List(r.Context(), kind)
	if err != nil {
		out["error"] = firstLine(err.Error())
	} else if list != nil {
		out["versions"] = list
	}
	writeJSON(w, http.StatusOK, out)
}

// fetchComfyVersions 把远端历史与 tag 拉全。
//
// 单独一个动作而不是列表时顺手做：补历史要联网、可能几十秒，
// 而"看一眼当前版本"不该每次都等这个。
func (s *Server) fetchComfyVersions(w http.ResponseWriter, r *http.Request) {
	repo, err := comfyver.Open(s.Settings.Get().Comfy.MainPy)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := repo.Fetch(r.Context()); err != nil {
		writeErr(w, http.StatusBadGateway, firstLine(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":     true,
		"status": repo.Status(r.Context()),
	})
}

// switchComfyVersion 切到指定版本，走部署那套步骤（切换 → 重装依赖 → 验证）。
func (s *Server) switchComfyVersion(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ref    string `json:"ref"`
		Mirror bool   `json:"mirror"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	cfg := s.Settings.Get()
	err := s.Deploy.Switch(r.Context(), deploy.SwitchOptions{
		MainPy: cfg.Comfy.MainPy,
		Python: cfg.Comfy.Python,
		Ref:    body.Ref,
		Mirror: body.Mirror,
		Proxy:  cfg.Imagen.Proxy,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
