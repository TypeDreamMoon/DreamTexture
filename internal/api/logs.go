package api

import (
	"net/http"
	"strconv"
)

// logs 增量返回日志。
//
// 走轮询 + since 游标而不是把每一行都塞进 WebSocket：ComfyUI 采样时会刷出
// 大量进度行，逐条推送等于把事件流拿去当日志管道用，正经的任务事件反而
// 可能被挤掉。轮询这边天然合批，界面上看着一样是实时滚动。
func (s *Server) logs(w http.ResponseWriter, r *http.Request) {
	if s.Logs == nil {
		writeJSON(w, http.StatusOK, map[string]any{"lines": []any{}, "last": 0})
		return
	}
	q := r.URL.Query()
	after, _ := strconv.ParseInt(q.Get("since"), 10, 64)
	limit := 500
	if v, err := strconv.Atoi(q.Get("limit")); err == nil && v > 0 && v <= 5000 {
		limit = v
	}
	// Since 内部已经按 limit 截过尾了，首次拉取（since=0）也不会把整个缓冲倒出来。
	writeJSON(w, http.StatusOK, map[string]any{
		"lines": s.Logs.Since(after, limit),
		"last":  s.Logs.LastSeq(),
	})
}

// comfyStart / comfyStop 是界面上的启停按钮。
//
// 停止走 StopByUser 而不是 Stop：后者是后端自己退出时用的，会把巡检一并
// 结束；这里只是把进程停住，并记下"用户不想让它跑"，免得自动重启立刻
// 又把它拉起来。
func (s *Server) comfyStop(w http.ResponseWriter, _ *http.Request) {
	if err := s.Sup.StopByUser(); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "health": s.Sup.Health()})
}

func (s *Server) comfyStart(w http.ResponseWriter, r *http.Request) {
	// 冷启动要等 ComfyUI 加载完节点、还要等启动期后台任务安定，
	// 几分钟很正常，所以这里不设自己的超时，跟着请求上下文走。
	if err := s.Sup.StartByUser(r.Context()); err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "health": s.Sup.Health()})
}
