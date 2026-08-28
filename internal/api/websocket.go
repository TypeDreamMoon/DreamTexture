package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

// HeartbeatInterval 是心跳间隔。前端按它的倍数设超时，改这里前端也要跟着改
// （web/src/store.ts 的 STALE_AFTER）。
const HeartbeatInterval = 15 * time.Second

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

	beat := time.NewTicker(HeartbeatInterval)
	defer beat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-beat.C:
			// 心跳发的是**文本帧**，不是协议层的 ping。
			//
			// 协议层 ping 浏览器是在网络层自动回 pong 的，JS 一个事件都收不到——
			// 也就是说前端没有任何办法凭它判断对端还活着。而 close 事件并不保证
			// 会来：标签页被冻结、机器休眠再唤醒、半开连接，都会让它丢掉，
			// 于是界面挂着一份过期状态一直显示"已连接"（实测把后端进程杀掉，
			// 页面照旧显示 ComfyUI 就绪）。给前端一个看得见的节拍，它才有
			// 判据超时重连。
			hctx, hcancel := context.WithTimeout(ctx, 10*time.Second)
			werr := conn.Write(hctx, websocket.MessageText, []byte(`{"type":"hb"}`))
			hcancel()
			if werr != nil {
				return
			}
			// 协议层 ping 仍然要发：写成功只说明塞进了发送缓冲区，
			// 而 ping 等的是真正的往返，这是**服务端**发现死客户端的手段。
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
