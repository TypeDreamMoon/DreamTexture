package api

import (
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// TestHeartbeatMatchesFrontend 盯住一对跨语言的约定：后端的心跳间隔必须明显
// 小于前端判定连接已死的超时。
//
// 这两个值分处 Go 和 TypeScript，编译器谁也管不着谁。把间隔调大过了头，症状
// 是所有客户端每 40 秒集体断开重连一次——看着像网络抖动，没人会想到是这里。
func TestHeartbeatMatchesFrontend(t *testing.T) {
	b, err := os.ReadFile("../../web/src/store.ts")
	if err != nil {
		t.Skipf("读不到前端源码，跳过：%v", err)
	}
	m := regexp.MustCompile(`(?m)^const STALE_AFTER = (\d+)$`).FindSubmatch(b)
	if m == nil {
		t.Fatal("在 web/src/store.ts 里找不到 STALE_AFTER；改名了就把这个测试一起改")
	}
	ms, err := strconv.Atoi(string(m[1]))
	if err != nil {
		t.Fatalf("STALE_AFTER 不是数字：%v", err)
	}
	stale := time.Duration(ms) * time.Millisecond

	// 至少要容得下两拍。只留一拍的话，一次 GC 停顿就能让前端误判。
	if min := 2 * HeartbeatInterval; stale < min {
		t.Errorf("STALE_AFTER=%v 太紧，至少要 %v（心跳 %v 的两倍）", stale, min, HeartbeatInterval)
	}
	// 反过来也不能太松：超时期间界面显示的是过期状态，用户看着是"好的"。
	if max := 6 * HeartbeatInterval; stale > max {
		t.Errorf("STALE_AFTER=%v 太松，界面会挂着过期状态太久（心跳 %v）", stale, HeartbeatInterval)
	}
}
