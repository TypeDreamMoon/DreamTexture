package comfy

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Event 是从 ComfyUI WebSocket 收到的一条事件。
type Event struct {
	Type     string
	PromptID string
	// Node 是当前执行的节点 id，executing 事件里为空表示该任务执行完毕。
	Node string
	// Value/Max 仅 progress 事件有效。
	Value, Max int
	// QueueRemaining 仅 status 事件有效。
	QueueRemaining int
	// Raw 保留原始 data，供调用方按需深挖。
	Raw json.RawMessage
}

// EventStream 维持一条到 ComfyUI 的 WebSocket，断线自动重连。
type EventStream struct {
	base     string
	clientID string
	log      *slog.Logger
	out      chan Event
	// connected 在每次连接成功时收到一个信号，用于判断是否需要补一次 history 对账。
	reconnected chan struct{}
}

func NewEventStream(baseURL, clientID string, log *slog.Logger) *EventStream {
	return &EventStream{
		base:        strings.TrimRight(baseURL, "/"),
		clientID:    clientID,
		log:         log,
		out:         make(chan Event, 256),
		reconnected: make(chan struct{}, 8),
	}
}

func (s *EventStream) Events() <-chan Event       { return s.out }
func (s *EventStream) Reconnected() <-chan struct{} { return s.reconnected }

// Run 阻塞直到 ctx 取消，期间持续接收事件并在断线后重连。
func (s *EventStream) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := s.once(ctx); err != nil && ctx.Err() == nil {
			s.log.Debug("ComfyUI 事件流断开，准备重连", "err", err, "backoff", backoff)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
}

func (s *EventStream) once(ctx context.Context) error {
	u := s.wsURL()
	conn, _, err := websocket.Dial(ctx, u, nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	// ComfyUI 的预览是二进制帧，可能不小。
	conn.SetReadLimit(16 << 20)

	select {
	case s.reconnected <- struct{}{}:
	default:
	}

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		if typ != websocket.MessageText {
			continue // 预览图等二进制帧，当前用不上
		}
		var env struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if json.Unmarshal(data, &env) != nil {
			continue
		}
		ev := Event{Type: env.Type, Raw: env.Data}
		var d struct {
			PromptID string          `json:"prompt_id"`
			Node     json.RawMessage `json:"node"`
			Value    int             `json:"value"`
			Max      int             `json:"max"`
			Status   struct {
				ExecInfo struct {
					QueueRemaining int `json:"queue_remaining"`
				} `json:"exec_info"`
			} `json:"status"`
		}
		if json.Unmarshal(env.Data, &d) == nil {
			ev.PromptID = d.PromptID
			ev.Value, ev.Max = d.Value, d.Max
			ev.QueueRemaining = d.Status.ExecInfo.QueueRemaining
			// node 可能是字符串或 null
			var node string
			if json.Unmarshal(d.Node, &node) == nil {
				ev.Node = node
			}
		}
		select {
		case s.out <- ev:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 宁可丢事件也不阻塞读循环；进度靠 history 兜底对账。
		}
	}
}

func (s *EventStream) wsURL() string {
	u := s.base
	switch {
	case strings.HasPrefix(u, "https://"):
		u = "wss://" + strings.TrimPrefix(u, "https://")
	case strings.HasPrefix(u, "http://"):
		u = "ws://" + strings.TrimPrefix(u, "http://")
	}
	return u + "/ws?clientId=" + url.QueryEscape(s.clientID)
}
