package logbuf

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestSinceIsIncremental(t *testing.T) {
	b := New(100)
	for i := 0; i < 5; i++ {
		b.Append("INFO", "backend", "line")
	}
	all := b.Since(0, 0)
	if len(all) != 5 {
		t.Fatalf("期望 5 行，得到 %d", len(all))
	}
	if all[0].Seq != 1 || all[4].Seq != 5 {
		t.Errorf("序号应当从 1 连续到 5，得到 %d..%d", all[0].Seq, all[4].Seq)
	}
	rest := b.Since(3, 0)
	if len(rest) != 2 || rest[0].Seq != 4 {
		t.Errorf("增量拉取错了: %d 行，首条 seq=%d", len(rest), rest[0].Seq)
	}
	if n := len(b.Since(5, 0)); n != 0 {
		t.Errorf("已经拉到最新之后应当没有新行，得到 %d", n)
	}
}

// 环形缓冲：超出容量后最老的被顶掉，序号继续递增不回绕。
func TestRingOverwritesOldest(t *testing.T) {
	const cap = 8
	b := New(cap)
	for i := 0; i < cap*3; i++ {
		b.Append("INFO", "backend", "x")
	}
	all := b.Since(0, 0)
	if len(all) != cap {
		t.Fatalf("缓冲应当只留 %d 行，得到 %d", cap, len(all))
	}
	// 留下的必须是最后 cap 条，且顺序是从旧到新
	wantFirst := int64(cap*3 - cap + 1)
	if all[0].Seq != wantFirst {
		t.Errorf("最老的一条 seq 应为 %d，得到 %d", wantFirst, all[0].Seq)
	}
	for i := 1; i < len(all); i++ {
		if all[i].Seq != all[i-1].Seq+1 {
			t.Fatalf("顺序乱了：第 %d 条 seq=%d，前一条 %d", i, all[i].Seq, all[i-1].Seq)
		}
	}
	if b.LastSeq() != int64(cap*3) {
		t.Errorf("LastSeq 应为 %d，得到 %d", cap*3, b.LastSeq())
	}
}

func TestLimitKeepsNewest(t *testing.T) {
	b := New(100)
	for i := 0; i < 50; i++ {
		b.Append("INFO", "backend", "x")
	}
	got := b.Since(0, 10)
	if len(got) != 10 {
		t.Fatalf("期望 10 行，得到 %d", len(got))
	}
	// 截尾要留最新的，不是最老的——日志窗口关心的是刚发生了什么
	if got[9].Seq != 50 {
		t.Errorf("最后一条应为 seq=50，得到 %d", got[9].Seq)
	}
}

// 订阅者读得慢时绝不能把写日志的一方卡住。
func TestSlowSubscriberDoesNotBlock(t *testing.T) {
	b := New(50)
	_, cancel := b.Subscribe(1) // 只留 1 的缓冲，且从不读
	defer cancel()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 500; i++ {
			b.Append("INFO", "backend", "flood")
		}
		close(done)
	}()
	select {
	case <-done:
	case <-context.Background().Done():
	}
	if b.LastSeq() != 500 {
		t.Errorf("写入被阻塞了：只写进 %d 条", b.LastSeq())
	}
}

// slog 的 handler 要把消息和属性都抄进缓冲，同时透传给下一级。
func TestHandlerCapturesAttrs(t *testing.T) {
	b := New(20)
	var sb strings.Builder
	log := slog.New(NewHandler(slog.NewTextHandler(&sb, nil), b))
	log.With("组件", "部署").Info("开始", "步骤", "torch", "大小MB", 1800)

	lines := b.Since(0, 0)
	if len(lines) != 1 {
		t.Fatalf("期望 1 行，得到 %d", len(lines))
	}
	got := lines[0].Text
	for _, want := range []string{"开始", "组件=部署", "步骤=torch", "大小MB=1800"} {
		if !strings.Contains(got, want) {
			t.Errorf("缓冲里少了 %q：%s", want, got)
		}
	}
	if lines[0].Level != "INFO" || lines[0].Source != "backend" {
		t.Errorf("级别/来源不对: %s / %s", lines[0].Level, lines[0].Source)
	}
	// 控制台那份不能丢
	if !strings.Contains(sb.String(), "开始") {
		t.Errorf("没有透传给下一级 handler：%s", sb.String())
	}
}
