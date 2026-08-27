package job

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// stallTimeout 是"多久没动静就判定卡死"的阈值。
//
// 定得这么宽是因为没有进度事件的环节确实存在且合法：VAEDecode、CHORD 分解、
// 大图落盘都是一声不吭跑几十秒到一两分钟。真出问题时表现是彻底不动，
// 差着一个数量级，不用卡得太紧。
const stallTimeout = 8 * time.Minute

// beat 记录当前任务最近一次有动静的时刻。
//
// 用原子量而不是 mu 保护的字段：事件是从 WebSocket 的读循环里进来的，
// 那条协程不该为了打个时间戳去抢执行路径的锁。
type beat struct {
	at atomic.Int64
}

func (b *beat) touch() { b.at.Store(time.Now().UnixNano()) }
func (b *beat) idle() time.Duration {
	n := b.at.Load()
	if n == 0 {
		return 0
	}
	return time.Since(time.Unix(0, n))
}

// stallReason 在判定卡死时组一段能照着做事的说明。
//
// 光说"超时了"没用——用户看完还是不知道该干嘛。这里现场读一次显存，
// 把"是不是显存被别的程序占了"这个最常见的原因摆出来。
func (r *Runner) stallReason(ctx context.Context, idle time.Duration, node string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "任务卡住了：%s 没有任何进展", idle.Round(time.Second))
	if node != "" {
		fmt.Fprintf(&b, "（停在节点 %s）", node)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if s, err := r.sup.Client().SystemStats(ctx); err == nil && len(s.Devices) > 0 {
		d := s.Devices[0]
		freeMB := d.VRAMFree / (1 << 20)
		totalMB := d.VRAMTotal / (1 << 20)
		fmt.Fprintf(&b, "。显存 %d/%d MB 空闲", freeMB, totalMB)
		// 阈值取 SDXL 出一张 1024 图在解码环节的大致峰值需求。
		if totalMB > 0 && freeMB < 2048 {
			b.WriteString("——显存已经被占满。Windows 上显卡驱动默认允许回退到内存，" +
				"于是不报错，改用内存硬算，慢几十倍，看起来就是卡在这儿不动。" +
				"关掉别的吃显存的程序（虚幻编辑器、游戏、开着硬件加速的浏览器）再试")
		}
	}
	return b.String()
}
