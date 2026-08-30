package job

import (
	"sync"

	"github.com/mengye/dreamtexture/internal/store"
)

// Event 是推给前端 / UE 的一条任务事件。
type Event struct {
	Type string     `json:"type"` // job.queued | job.progress | job.done | job.failed | comfy.status
	Job  *store.Job `json:"job,omitempty"`
	Data any        `json:"data,omitempty"`
}

// Bus 是进程内的事件广播。订阅者各有一条带缓冲的通道，
// 慢订阅者只会丢自己的事件，不会拖住任务执行。
type Bus struct {
	mu   sync.RWMutex
	next int
	subs map[int]chan Event
}

func NewBus() *Bus { return &Bus{subs: map[int]chan Event{}} }

func (b *Bus) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 64)
	b.mu.Lock()
	id := b.next
	b.next++
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

func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subs {
		select {
		case ch <- ev:
		default:
		}
	}
}
