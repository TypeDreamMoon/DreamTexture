package comfy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mengye/dreamtexture/internal/config"
)

// Health 是对外暴露的 ComfyUI 状态快照。
//
// Alive 与 Ready 是分开的：进程活着但队列积压时不该继续派活。
type Health struct {
	Mode           string    `json:"mode"`
	Alive          bool      `json:"alive"`
	Ready          bool      `json:"ready"`
	Reason         string    `json:"reason,omitempty"`
	BaseURL        string    `json:"base_url"`
	PID            int       `json:"pid,omitempty"`
	Version        string    `json:"comfyui_version,omitempty"`
	Device         string    `json:"device,omitempty"`
	VRAMTotalMB    int64     `json:"vram_total_mb,omitempty"`
	VRAMFreeMB     int64     `json:"vram_free_mb,omitempty"`
	QueueDepth    int       `json:"queue_depth"`
	Restarts      int       `json:"restarts"`
	LastCheckedAt time.Time `json:"last_checked_at"`
	// UserStopped 区分"我停的"和"它挂了"。
	//
	// 给界面一个正经字段，而不是让它去比对 Reason 里的中文——那样改一下措辞
	// 就会悄悄把界面判断弄坏，而且还没有任何报错。
	UserStopped bool `json:"user_stopped"`
}

// Supervisor 负责让 ComfyUI 处于可用状态。
//
// managed 模式下它拉起并看护子进程；attach 模式下它只做健康巡检，
// 生命周期归用户自己（例如开着 Comfy Desktop）。
type Supervisor struct {
	cfg config.Comfy
	cli *Client
	log *slog.Logger

	mu       sync.Mutex
	cmd      *exec.Cmd
	group    *procGroup
	restarts int
	health   Health
	logPath  string
	// procExited 在子进程真正退出时被关闭。
	//
	// 判断"要不要重启"必须看进程死没死，不能看 HTTP 通不通：ComfyUI 启动期
	// 可能在装自定义节点的依赖（个别还要从源码编译），这期间进程活着但端口不开，
	// 按可达性判断就会把它杀掉重启，而重启又打断安装——死循环，永远起不来。
	procExited chan struct{}

	stopped chan struct{}
	once    sync.Once

	// userStopped 表示用户从界面上主动停掉了 ComfyUI。
	// 自动重启必须尊重这个意图，否则那个停止按钮看起来就是坏的。
	userStopped bool

	// sink 收 ComfyUI 的每一行输出，供界面实时显示；可为 nil。
	sink Sink
}

// Sink 接收一行输出。用接口而不是直接依赖 logbuf，免得 comfy 包
// 为了打日志反过来依赖上层。
type Sink interface {
	Append(level, source, text string)
}

func NewSupervisor(cfg config.Comfy, log *slog.Logger) *Supervisor {
	return &Supervisor{
		cfg:     cfg,
		cli:     NewClient(cfg.BaseURL),
		log:     log,
		stopped: make(chan struct{}),
		health:  Health{Mode: string(cfg.Mode), BaseURL: cfg.BaseURL},
	}
}

// SetSink 指定 ComfyUI 输出的去处。须在 Start 之前调用。
func (s *Supervisor) SetSink(k Sink) { s.sink = k }

// lineSink 把字节流按行切开送进 Sink。
//
// ComfyUI 的进度条是用 \r 原地刷新的（tqdm），不切开的话一整条进度会攒成
// 一行几万字符的巨块；按 \r 也切，界面上就是正常滚动的进度。
type lineSink struct {
	sink Sink
	buf  []byte
}

func newLineSink(k Sink) *lineSink { return &lineSink{sink: k} }

func (w *lineSink) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexAny(w.buf, "\n\r")
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:i]), " \t")
		w.buf = w.buf[i+1:]
		if line != "" {
			w.sink.Append(levelOf(line), "comfyui", line)
		}
	}
	// 单行超长（例如某个节点打印了一大坨 JSON）就先冲出去，别无限攒着。
	if len(w.buf) > 8192 {
		w.sink.Append("INFO", "comfyui", string(w.buf))
		w.buf = w.buf[:0]
	}
	return len(p), nil
}

// levelOf 从行首的关键词猜一个级别，纯为了界面上能把报错标红。
// 猜错了无非是颜色不对，不影响任何判断逻辑。
func levelOf(line string) string {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "traceback"), strings.HasPrefix(l, "error"),
		strings.Contains(l, "[error]"), strings.Contains(l, "exception"):
		return "ERROR"
	case strings.HasPrefix(l, "warning"), strings.Contains(l, "[warn"):
		return "WARN"
	}
	return "INFO"
}

func (s *Supervisor) Client() *Client { return s.cli }

func (s *Supervisor) Health() Health {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.health
	h.UserStopped = s.userStopped
	return h
}

// Start 让 ComfyUI 可用，并阻塞到首次就绪（或超时）。
//
// 返回错误不代表后端该退出——调用方可以降级运行（界面照开，只是不能生成）。
// 所以无论成败都会把巡检协程带起来：失败之后正是最需要它去自愈的时候，
// 何况用户可能正打算去设置页把环境装上。
func (s *Supervisor) Start(ctx context.Context) error {
	defer func() { go s.watch(ctx) }()

	if s.cfg.Mode == config.ModeManaged {
		if err := s.spawn(ctx); err != nil {
			return err
		}
	}
	if err := s.waitReady(ctx, s.cfg.StartTimeout.D()); err != nil {
		return err
	}
	s.waitSettled(ctx)
	return nil
}

// pathsReady 检查 managed 模式下必需的两个路径。
func (s *Supervisor) pathsReady() error {
	for _, p := range []struct{ what, path string }{
		{"Python 解释器", s.cfg.Python},
		{"ComfyUI 主程序", s.cfg.MainPy},
	} {
		if p.path == "" {
			return fmt.Errorf("还没有配置%s。去设置页做一次一键部署，或手动填好路径", p.what)
		}
		if _, err := os.Stat(p.path); err != nil {
			return fmt.Errorf("%s不存在：%s。去设置页做一次一键部署，或改成正确的路径",
				p.what, p.path)
		}
	}
	return nil
}

// waitSettled 等 ComfyUI 的启动期后台工作安定下来。
//
// 能应答 HTTP 不等于能好好干活：自定义节点的启动任务（ComfyUI-Manager 抓取
// registry 是典型）会在后台持续占用 Python 侧的 GIL，此时探针依然是毫秒级，
// 但落在这个窗口里的生成任务会慢一个数量级（实测 878s vs 34.7s）。
//
// 判据只看日志有没有停止增长，不认任何具体插件；没有额外启动任务的环境
// 日志会立刻安静，基本不产生等待。仅 managed 模式可用——只有这时日志归我们管。
func (s *Supervisor) waitSettled(ctx context.Context) {
	quiet := s.cfg.SettleQuiet.D()
	s.mu.Lock()
	path := s.logPath
	s.mu.Unlock()
	if quiet <= 0 || path == "" {
		return
	}

	deadline := time.Now().Add(s.cfg.SettleTimeout.D())
	var lastSize int64 = -1
	lastChange := time.Now()
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
		size := int64(-1)
		if fi, err := os.Stat(path); err == nil {
			size = fi.Size()
		}
		if size != lastSize {
			lastSize, lastChange = size, time.Now()
		}
		if time.Since(lastChange) >= quiet {
			if waited := time.Since(start); waited > 2*time.Second {
				s.log.Info("ComfyUI 启动期后台任务已安定", "等待", waited.Round(time.Second))
			}
			return
		}
		if time.Now().After(deadline) {
			s.log.Warn("等待 ComfyUI 安定超时，继续放行；首个任务可能明显偏慢",
				"上限", s.cfg.SettleTimeout.D())
			return
		}
	}
}

// Stop 终止 managed 模式下的子进程（含整棵进程树）。用于后端自身退出。
func (s *Supervisor) Stop() {
	s.once.Do(func() { close(s.stopped) })
	s.mu.Lock()
	defer s.mu.Unlock()
	s.killLocked()
}

// StopByUser 是用户从界面上主动停掉 ComfyUI。
//
// 与 Stop 的区别是它不结束巡检，只把"停在这儿别动"的意图记下来——
// 否则自动重启会立刻把它拉起来，用户会觉得那个停止按钮坏了。
func (s *Supervisor) StopByUser() error {
	if s.cfg.Mode != config.ModeManaged {
		return fmt.Errorf("attach 模式下后端不管理 ComfyUI 生命周期，请到它自己的窗口里停止")
	}
	s.mu.Lock()
	s.userStopped = true
	running := s.cmd != nil
	s.killLocked()
	s.health.Alive, s.health.Ready = false, false
	s.health.Reason = "已被手动停止"
	s.health.PID = 0
	s.mu.Unlock()
	if running {
		s.log.Info("ComfyUI 已按用户要求停止")
	}
	return nil
}

// StartByUser 重新拉起被手动停掉的 ComfyUI。
func (s *Supervisor) StartByUser(ctx context.Context) error {
	if s.cfg.Mode != config.ModeManaged {
		return fmt.Errorf("attach 模式下后端不管理 ComfyUI 生命周期，请自行启动")
	}
	s.mu.Lock()
	s.userStopped = false
	alreadyUp := s.cmd != nil && s.procExited != nil
	s.mu.Unlock()
	if alreadyUp && s.processRunning() {
		return fmt.Errorf("ComfyUI 已经在运行")
	}
	if err := s.spawn(ctx); err != nil {
		return err
	}
	if err := s.waitReady(ctx, s.cfg.StartTimeout.D()); err != nil {
		return err
	}
	s.waitSettled(ctx)
	return nil
}

// UserStopped 报告是不是被用户主动停住的。
func (s *Supervisor) UserStopped() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.userStopped
}

// Restart 重启子进程。attach 模式下无从重启，返回错误。
func (s *Supervisor) Restart(ctx context.Context) error {
	if s.cfg.Mode != config.ModeManaged {
		return fmt.Errorf("attach 模式下后端不管理 ComfyUI 生命周期，请手动重启")
	}
	s.mu.Lock()
	s.killLocked()
	s.mu.Unlock()
	// 等端口真正释放，避免立刻重启撞上 TIME_WAIT 或残留监听。
	if err := s.waitPortFree(ctx, 20*time.Second); err != nil {
		return err
	}
	if err := s.spawn(ctx); err != nil {
		return err
	}
	if err := s.waitReady(ctx, s.cfg.StartTimeout.D()); err != nil {
		return err
	}
	s.waitSettled(ctx)
	return nil
}

func (s *Supervisor) spawn(ctx context.Context) error {
	// 路径不对就别白等超时了：全新安装的人还没部署过环境，让他对着三分钟的
	// 进度条干等只会以为程序挂了。这里同时也挡住了 watch 的反复重启。
	if err := s.pathsReady(); err != nil {
		s.setHealth(Health{Mode: string(s.cfg.Mode), BaseURL: s.cfg.BaseURL,
			Reason: err.Error(), LastCheckedAt: time.Now()})
		return err
	}
	if err := s.waitPortFree(ctx, 10*time.Second); err != nil {
		return fmt.Errorf("%s 端口未释放，可能有残留的 ComfyUI 进程: %w", s.cfg.BaseURL, err)
	}
	host, port, err := hostPort(s.cfg.BaseURL)
	if err != nil {
		return err
	}
	args := append([]string{"-s", s.cfg.MainPy, "--listen", host, "--port", port}, s.cfg.ExtraArgs...)
	cmd := exec.Command(s.cfg.Python, args...)
	cmd.Dir = filepath.Dir(s.cfg.MainPy)
	cmd.Env = append(os.Environ(), "PYTHONIOENCODING=utf-8")

	group, err := newProcGroup()
	if err != nil {
		return err
	}
	group.prepare(cmd)

	logPath := filepath.Join(filepath.Dir(s.cfg.MainPy), "dreamtexture-comfyui.log")
	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err == nil {
		var out io.Writer = lf
		// 同时抄一份给界面。文件那份不能省：waitSettled 靠它的增长判断
		// 启动期后台任务有没有安定，出了事故也还得靠它翻历史。
		if s.sink != nil {
			out = io.MultiWriter(lf, newLineSink(s.sink))
		}
		cmd.Stdout, cmd.Stderr = out, out
	}

	if err := cmd.Start(); err != nil {
		group.close()
		return fmt.Errorf("启动 ComfyUI: %w", err)
	}
	if err := group.assign(cmd); err != nil {
		s.log.Warn("进程组绑定失败，退出时可能留下残留进程", "err", err)
	}

	exited := make(chan struct{})
	s.mu.Lock()
	s.cmd, s.group, s.logPath, s.procExited = cmd, group, logPath, exited
	s.health.PID = cmd.Process.Pid
	s.mu.Unlock()

	// 单独收割子进程，这样 watch 能凭"进程是否退出"判断该不该重启，
	// 而不是凭端口通不通。
	go func() {
		err := cmd.Wait()
		close(exited)
		if err != nil {
			s.log.Warn("ComfyUI 进程已退出", "pid", cmd.Process.Pid, "err", err)
		} else {
			s.log.Info("ComfyUI 进程已退出", "pid", cmd.Process.Pid)
		}
	}()

	s.log.Info("已拉起 ComfyUI", "pid", cmd.Process.Pid, "日志", logPath)
	return nil
}

// processRunning 报告 managed 模式下的子进程是否还活着。
// attach 模式没有子进程可言，一律当作"活着"，交由健康探针判断。
func (s *Supervisor) processRunning() bool {
	if s.cfg.Mode != config.ModeManaged {
		return true
	}
	s.mu.Lock()
	ch := s.procExited
	s.mu.Unlock()
	if ch == nil {
		return false
	}
	select {
	case <-ch:
		return false
	default:
		return true
	}
}

func (s *Supervisor) killLocked() {
	if s.group != nil {
		s.group.close() // Job Object 关闭即连坐终止整棵进程树
		s.group = nil
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		// 不在这里 Wait —— 收割由 spawn 起的协程负责，两处同时 Wait 会互相抢。
		// 等它把退出信号送出来即可。
		if ch := s.procExited; ch != nil {
			select {
			case <-ch:
			case <-time.After(5 * time.Second):
			}
		}
		s.cmd = nil
	}
	s.procExited = nil
	s.health.PID = 0
}

// hardStartCap 是即便进程还活着也不再等下去的上限。
const hardStartCap = 30 * time.Minute

func (s *Supervisor) waitReady(ctx context.Context, timeout time.Duration) error {
	start := time.Now()
	deadline := start.Add(timeout)
	warned := false
	var exitedAt time.Time
	for {
		if s.probe(ctx); s.Health().Alive {
			if !exitedAt.IsZero() {
				// 我们启的那个进程没了，但端口起来了——ComfyUI-Manager 装完依赖后
				// 会自己 re-exec 一个新进程顶上（日志里的 "Restarting to reapply
				// dependency installation"）。服务可用就算就绪。
				s.log.Info("ComfyUI 由 Manager 自行重启后已就绪")
			}
			return nil
		}
		if !s.processRunning() {
			// 别一看到进程退出就判死：Manager 的自我重启会让原进程退出，
			// 新进程要几十秒才把端口开起来。给它一个宽限窗口。
			if exitedAt.IsZero() {
				exitedAt = time.Now()
				s.log.Info("ComfyUI 进程已退出，观察是否有新进程接管（Manager 装完依赖会自行重启）")
			} else if time.Since(exitedAt) > 90*time.Second {
				return fmt.Errorf("ComfyUI 进程已退出且没有新进程接管；日志见 %s", s.logPath)
			}
		}
		if time.Now().After(deadline) {
			// attach 模式没有我们启的进程，"它在忙"这个理由不成立——
			// 多半就是根本没开。到点就放心返回，让后端降级启动、界面照开；
			// 用户什么时候把 ComfyUI 开起来，巡检自然会接上。
			if s.cfg.Mode != config.ModeManaged {
				return fmt.Errorf("等待 %s 超过 %s 仍连不上；attach 模式下请确认 ComfyUI 已启动",
					s.cfg.BaseURL, timeout)
			}
			// managed 模式下进程还活着说明它在忙（装自定义节点依赖时甚至要
			// 从源码编译，实测能拖好几分钟），超时就放弃反而会让它永远起不来。
			if !warned {
				s.log.Warn("ComfyUI 启动超过预期但进程仍在运行，继续等待",
					"已等待", time.Since(start).Round(time.Second),
					"日志", s.logPath)
				warned = true
			}
			if time.Since(start) > hardStartCap {
				return fmt.Errorf("等待 ComfyUI 就绪超过 %s 仍未完成，请查看日志 %s",
					hardStartCap, s.logPath)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// watch 定期巡检，managed 模式下发现进程死了就按配置自动重启。
func (s *Supervisor) watch(ctx context.Context) {
	t := time.NewTicker(s.cfg.HealthInterval.D())
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopped:
			return
		case <-t.C:
		}
		// 用户主动停掉的就别探了：探针会把 health 覆盖成"连不上"，
		// 界面上就分不清"我停的"和"它挂了"。
		if s.UserStopped() {
			continue
		}
		s.probe(ctx)
		if s.Health().Alive || s.cfg.Mode != config.ModeManaged || !s.cfg.AutoRestart {
			continue
		}
		// 进程还活着就只是没准备好（多半在装依赖或加载模型），耐心等着，
		// 这时候重启只会打断它、并且下一轮还会重来。
		if s.processRunning() {
			s.log.Debug("ComfyUI 尚未就绪但进程仍在运行，继续等待")
			continue
		}
		// 环境根本还没装好时别一遍遍试着重启——那只会每十秒往日志里
		// 刷一条同样的失败，把真正有用的信息淹掉。
		if err := s.pathsReady(); err != nil {
			s.log.Debug("ComfyUI 环境尚未就绪，暂不重启", "原因", err)
			continue
		}
		s.mu.Lock()
		s.restarts++
		n := s.restarts
		s.health.Restarts = n
		s.mu.Unlock()
		s.log.Warn("ComfyUI 不可达，尝试重启", "第几次", n)
		if err := s.Restart(ctx); err != nil {
			s.log.Error("重启 ComfyUI 失败", "err", err)
		} else {
			s.log.Info("ComfyUI 已恢复")
		}
	}
}

func (s *Supervisor) probe(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	h := s.Health()
	h.LastCheckedAt = time.Now()
	h.Alive, h.Ready, h.Reason = false, false, ""

	stats, err := s.cli.SystemStats(ctx)
	if err != nil {
		h.Reason = "无法连接 ComfyUI: " + err.Error()
		s.setHealth(h)
		return
	}
	h.Alive = true
	h.Version = stats.System.ComfyUIVersion
	if len(stats.Devices) > 0 {
		d := stats.Devices[0]
		h.Device = d.Name
		h.VRAMTotalMB = d.VRAMTotal / (1 << 20)
		h.VRAMFreeMB = d.VRAMFree / (1 << 20)
	}

	depth, err := s.cli.QueueDepth(ctx)
	if err != nil {
		h.Reason = "队列查询失败: " + err.Error()
		s.setHealth(h)
		return
	}
	h.QueueDepth = depth
	if s.cfg.MaxQueueDepth > 0 && depth > s.cfg.MaxQueueDepth {
		h.Reason = fmt.Sprintf("队列积压 %d 超过阈值 %d", depth, s.cfg.MaxQueueDepth)
		s.setHealth(h)
		return
	}
	h.Ready = true
	s.setHealth(h)
}

func (s *Supervisor) setHealth(h Health) {
	s.mu.Lock()
	h.Mode, h.BaseURL, h.Restarts = string(s.cfg.Mode), s.cfg.BaseURL, s.restarts
	if s.cmd != nil && s.cmd.Process != nil {
		h.PID = s.cmd.Process.Pid
	}
	s.health = h
	s.mu.Unlock()
}

func (s *Supervisor) waitPortFree(ctx context.Context, timeout time.Duration) error {
	host, port, err := hostPort(s.cfg.BaseURL)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)
	for {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err != nil {
			return nil // 连不上即端口空闲
		}
		c.Close()
		if time.Now().After(deadline) {
			return fmt.Errorf("%s 仍被占用", addr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func hostPort(base string) (string, string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", "", fmt.Errorf("解析 comfy.base_url %q: %w", base, err)
	}
	host, port := u.Hostname(), u.Port()
	if host == "" {
		host = "127.0.0.1"
	}
	if port == "" {
		port = "8188"
	}
	return host, port, nil
}
