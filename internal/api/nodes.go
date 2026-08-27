package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mengye/dreamtexture/internal/nodes"
)

// listNodes 搜索可安装的自定义节点包。
func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	if limit <= 0 || limit > 200 {
		limit = 40
	}
	hits, total, err := s.Nodes.Search(r.Context(), nodes.Query{
		Text:   q.Get("q"),
		State:  q.Get("state"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	if hits == nil {
		hits = []nodes.Pack{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"packs": hits, "total": total})
}

// nodeAction 执行安装 / 卸载 / 启停 / 更新。
//
// 只接受包 id：具体要传给 Manager 的字段从我们自己缓存的列表里取，
// 不让客户端指定仓库地址，免得这个接口变成任人指使的 git clone。
func (s *Server) nodeAction(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID      string `json:"id"`
		Action  string `json:"action"` // install | uninstall | enable | disable | update
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	all, err := s.Nodes.List(r.Context(), false)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	var pack *nodes.Pack
	for i := range all {
		if all[i].ID == body.ID {
			pack = &all[i]
			break
		}
	}
	if pack == nil {
		writeErr(w, http.StatusNotFound, "节点库里没有 "+body.ID)
		return
	}

	switch body.Action {
	case "install":
		err = s.Nodes.Install(r.Context(), *pack, body.Version)
	case "uninstall":
		err = s.Nodes.Uninstall(r.Context(), *pack)
	case "enable":
		err = s.Nodes.SetEnabled(r.Context(), *pack, true)
	case "disable":
		err = s.Nodes.SetEnabled(r.Context(), *pack, false)
	case "update":
		err = s.Nodes.Update(r.Context(), *pack)
	default:
		writeErr(w, http.StatusBadRequest, "未知操作 "+body.Action)
		return
	}
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"ok": true,
		"hint": "已交给 ComfyUI-Manager 处理。装好后需要重启 ComfyUI 才会生效，" +
			"可以在这里点「重启 ComfyUI」。",
	})
}

func (s *Server) nodeQueue(w http.ResponseWriter, r *http.Request) {
	st, err := s.Nodes.QueueStatus(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// nodeManagerInfo 报告 ComfyUI-Manager 的可用性与接口世代。
//
// 界面据此决定要不要显示节点页，以及在 Manager 不提供节点目录时给出说明。
func (s *Server) nodeManagerInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Nodes.Detect(r.Context(), r.URL.Query().Get("refresh") == "1"))
}
