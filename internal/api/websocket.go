package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// websocket 把任务事件推给前端。
//
// 只推不收：客户端要做什么都走 REST，这样状态变更只有一条路径，不会出现
// "有的操作走 WS 有的走 HTTP"导致的时序问题。
func (s *Server) websocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// 单机自用，前端与后端同源或来自 localhost 的其他端口。
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		s.Log.Debug("WebSocket 握手失败", "err", err)
		return
	}
	defer conn.CloseNow()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	events, unsubscribe := s.Bus.Subscribe()
	defer unsubscribe()

	// 一上来先给一份当前状态，前端不必再单独拉一次。
	if b, err := json.Marshal(map[string]any{"type": "comfy.status", "data": s.Sup.Health()}); err == nil {
		_ = conn.Write(ctx, websocket.MessageText, b)
	}

	// 客户端断开只有在读的时候才发现得了，所以起一个读协程盯着。
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.Read(ctx); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ping.C:
			pctx, pcancel := context.WithTimeout(ctx, 10*time.Second)
			err := conn.Ping(pctx)
			pcancel()
			if err != nil {
				return
			}
		case ev, ok := <-events:
			if !ok {
				return
			}
			b, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
			err = conn.Write(wctx, websocket.MessageText, b)
			wcancel()
			if err != nil {
				return
			}
		}
	}
}
