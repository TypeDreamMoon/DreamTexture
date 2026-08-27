package deploy

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// exec 跑一条外部命令，把它的输出逐行送进日志。
//
// 逐行转发而不是跑完再一把倒出来：torch 那一步要下几个 GB、跑十几分钟，
// 不实时给进度的话界面上就是一片死寂，用户分不清是在下载还是卡死了。
func (d *Deployer) exec(ctx context.Context, opt Options, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = deployEnv(opt)

	d.say("  $ %s", brief(name, args))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 %s 失败: %w", name, err)
	}

	// 留最后几行给失败时的报错用：外部工具的关键信息通常在末尾，
	// 而整段输出可能有几千行，全塞进错误里没法看。
	var tail tailBuf
	var last atomicTime
	last.set(time.Now())

	var wg sync.WaitGroup
	pump := func(r io.Reader, level string) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := strings.TrimRight(sc.Text(), " \t\r")
			if line == "" {
				continue
			}
			last.set(time.Now())
			tail.push(line)
			d.log.Append(level, "deploy", "  "+line)
		}
	}
	wg.Add(2)
	go pump(stdout, "INFO")
	// stderr 一律记 INFO：pip / git 把进度也往 stderr 写，
	// 按 ERROR 记会让整屏飘红，真的错误反而看不出来。
	go pump(stderr, "INFO")

	// 心跳。
	//
	// uv 的进度条是靠 \r 原地重绘的，转发到日志里只会变成一坨乱码，所以关掉了
	// （UV_NO_PROGRESS）。代价是下 1.8GB 的 torch 时，控制台会足足安静十几分钟——
	// 而"看不见输出"正是这套日志要解决的问题。所以自己补一条：只报还活着和
	// 已经等了多久，不假装知道进度。
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(20 * time.Second)
		defer t.Stop()
		start := time.Now()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				if quiet := time.Since(last.get()); quiet >= 20*time.Second {
					d.say("  …仍在进行，已 %s（大文件下载时没有输出是正常的）",
						time.Since(start).Round(time.Second))
				}
			}
		}
	}()

	wg.Wait()
	close(done)

	if err := cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("已取消")
		}
		if t := tail.String(); t != "" {
			return fmt.Errorf("%w\n%s", err, t)
		}
		return err
	}
	return nil
}

// capture 跑一条命令并取回它的标准输出，不往日志里刷。
// 用于探测版本号这类只要结果的场合。
func (d *Deployer) capture(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errb.String())
		if msg == "" {
			msg = strings.TrimSpace(out.String())
		}
		if msg != "" {
			return "", fmt.Errorf("%w: %s", err, firstLines(msg, 3))
		}
		return "", err
	}
	return out.String(), nil
}

// deployEnv 给子进程配好环境。
//
// 代理必须显式传下去：uv 和 git 都读 HTTPS_PROXY，而后端自己可能是从
// 一个没有这些变量的环境里起来的（服务、计划任务、双击 exe）。
func deployEnv(opt Options) []string {
	env := os.Environ()
	env = append(env, "PYTHONIOENCODING=utf-8", "PYTHONUNBUFFERED=1")
	// 让 uv 别用颜色转义码，否则日志里全是 ANSI 乱码。
	env = append(env, "NO_COLOR=1", "UV_NO_PROGRESS=1")
	if p := strings.TrimSpace(opt.Proxy); p != "" {
		// 大小写两套都给：不同工具认的不一样，git 认小写、uv 两个都认。
		env = append(env,
			"HTTPS_PROXY="+p, "https_proxy="+p,
			"HTTP_PROXY="+p, "http_proxy="+p,
			// 本地地址别走代理，否则连自己都连不上。
			"NO_PROXY=127.0.0.1,localhost", "no_proxy=127.0.0.1,localhost")
	}
	return env
}

// brief 把命令行缩短到能看的长度，长路径只留文件名。
func brief(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shortenPath(name))
	for _, a := range args {
		parts = append(parts, shortenPath(a))
	}
	return strings.Join(parts, " ")
}

func shortenPath(s string) string {
	if len(s) < 48 || !strings.ContainsAny(s, `\/`) {
		return s
	}
	// 保留最后两段，前面用 … 顶替
	sep := `\`
	if strings.Count(s, "/") > strings.Count(s, `\`) {
		sep = "/"
	}
	p := strings.Split(s, sep)
	if len(p) <= 2 {
		return s
	}
	return "…" + sep + strings.Join(p[len(p)-2:], sep)
}

// atomicTime 是心跳与输出泵之间共享的"最后一次有输出"的时刻。
type atomicTime struct {
	mu sync.Mutex
	t  time.Time
}

func (a *atomicTime) set(t time.Time) { a.mu.Lock(); a.t = t; a.mu.Unlock() }
func (a *atomicTime) get() time.Time  { a.mu.Lock(); defer a.mu.Unlock(); return a.t }

// tailBuf 只保留最后 N 行。
type tailBuf struct {
	mu    sync.Mutex
	lines []string
}

const tailKeep = 12

func (t *tailBuf) push(s string) {
	t.mu.Lock()
	t.lines = append(t.lines, s)
	if len(t.lines) > tailKeep {
		t.lines = t.lines[len(t.lines)-tailKeep:]
	}
	t.mu.Unlock()
}

func (t *tailBuf) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return strings.Join(t.lines, "\n")
}

func firstLines(s string, n int) string {
	parts := strings.SplitN(s, "\n", n+1)
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, "\n")
}
