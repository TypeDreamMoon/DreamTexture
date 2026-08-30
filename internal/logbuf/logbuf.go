// Package logbuf 把日志同时留一份在内存里，供界面实时显示。
//
// 有了它，用户不必去翻控制台窗口——尤其部署这种要跑十几分钟的操作，
// 看不见输出就只能干等，出错了也不知道卡在哪一步。
package logbuf

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
)

// Line 是一条日志。
type Line struct {
	// Seq 单调递增，客户端据此增量拉取，不必每次要全量。
	Seq   int64     `json:"seq"`
	At    time.Time `json:"at"`
	Level string    `json:"level"`
	// Source 区分是后端自己的日志还是 ComfyUI 的输出。
	Source string `json:"source"`
	Text   string `json:"text"`
}

// Buffer 是一个定长环形缓冲。
//
// 定长是刻意的：ComfyUI 跑起来能刷出成千上万行（采样进度、节点加载），
// 无上限地留在内存里迟早把进程撑爆。看历史应当去翻日志文件。
type Buffer struct {
	mu    sync.RWMutex
	lines []Line
	next  int
	full  bool
	seq   int64

	subs map[int]chan Line
	subN int
}

func New(capacity int) *Buffer {
	if capacity <= 0 {
		capacity = 2000
	}
	return &Buffer{lines: make([]Line, capacity), subs: map[int]chan Line{}}
}

// Append 记一行。source 为空时记作 backend。
func (b *Buffer) Append(level, source, text string) {
	if source == "" {
		source = "backend"
	}
	if level == "" {
		level = "INFO"
	}
	b.mu.Lock()
	b.seq++
	ln := Line{Seq: b.seq, At: time.Now(), Level: level, Source: source, Text: text}
	b.lines[b.next] = ln
	b.next = (b.next + 1) % len(b.lines)
	if b.next == 0 {
		b.full = true
	}
	subs := make([]chan Line, 0, len(b.subs))
	for _, c := range b.subs {
		subs = append(subs, c)
	}
	b.mu.Unlock()

	// 推给订阅者时绝不阻塞：某个客户端读得慢不该把整个后端的日志写卡住。
	// 塞不下就丢，客户端自己会用 seq 发现断档并补拉。
	for _, c := range subs {
		select {
		case c <- ln:
		default:
		}
	}
}

// Since 返回 seq 大于 after 的日志。after 传 0 拿全部。
func (b *Buffer) Since(after int64, limit int) []Line {
	b.mu.RLock()
	defer b.mu.RUnlock()

	n := len(b.lines)
	count := b.next
	start := 0
	if b.full {
		count, start = n, b.next
	}
	out := make([]Line, 0, count)
	for i := 0; i < count; i++ {
		ln := b.lines[(start+i)%n]
		if ln.Seq > after {
			out = append(out, ln)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out
}

// LastSeq 返回当前最大序号。
func (b *Buffer) LastSeq() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.seq
}

// Subscribe 订阅后续日志，返回取消函数。
func (b *Buffer) Subscribe(bufSize int) (<-chan Line, func()) {
	if bufSize <= 0 {
		bufSize = 256
	}
	ch := make(chan Line, bufSize)
	b.mu.Lock()
	id := b.subN
	b.subN++
	b.subs[id] = ch
	b.mu.Unlock()

	return ch, func() {
		b.mu.Lock()
		if c, ok := b.subs[id]; ok {
			delete(b.subs, id)
			close(c)
		}
		b.mu.Unlock()
	}
}

// Handler 把 slog 的输出复制一份进缓冲，同时透传给下一级 handler。
//
// 包一层而不是替换：控制台输出仍然要有，出了事故还得靠它。
type Handler struct {
	next  slog.Handler
	buf   *Buffer
	attrs []slog.Attr
	group string
}

func NewHandler(next slog.Handler, buf *Buffer) *Handler {
	return &Handler{next: next, buf: buf}
}

func (h *Handler) Enabled(ctx context.Context, l slog.Level) bool { return h.next.Enabled(ctx, l) }

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var sb strings.Builder
	sb.WriteString(r.Message)
	write := func(a slog.Attr) {
		if a.Equal(slog.Attr{}) {
			return
		}
		sb.WriteString("  ")
		sb.WriteString(a.Key)
		sb.WriteString("=")
		sb.WriteString(fmt.Sprint(a.Value.Any()))
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(func(a slog.Attr) bool { write(a); return true })

	h.buf.Append(r.Level.String(), "backend", sb.String())
	return h.next.Handle(ctx, r)
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &Handler{next: h.next.WithAttrs(attrs), buf: h.buf,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...), group: h.group}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{next: h.next.WithGroup(name), buf: h.buf, attrs: h.attrs, group: name}
}
