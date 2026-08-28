// DreamTexture 后端：托管 ComfyUI、调度生成任务、产出标准材质套装。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mengye/dreamtexture/internal/api"
	"github.com/mengye/dreamtexture/internal/applog"
	"github.com/mengye/dreamtexture/internal/catalog"
	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/config"
	"github.com/mengye/dreamtexture/internal/deploy"
	"github.com/mengye/dreamtexture/internal/imagen"
	"github.com/mengye/dreamtexture/internal/job"
	"github.com/mengye/dreamtexture/internal/logbuf"
	"github.com/mengye/dreamtexture/internal/mcpsrv"
	"github.com/mengye/dreamtexture/internal/model"
	"github.com/mengye/dreamtexture/internal/nodes"
	"github.com/mengye/dreamtexture/internal/settings"
	"github.com/mengye/dreamtexture/internal/store"
	"github.com/mengye/dreamtexture/internal/web"
	"github.com/mengye/dreamtexture/internal/workflow"
)

var version = "0.1.0-m1"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "启动失败:", err)
		os.Exit(1)
	}
}

func run() (err error) {
	cfgPath := flag.String("config", "configs/dreamtexture.json", "配置文件路径")
	debug := flag.Bool("debug", false, "输出调试日志")
	logDir := flag.String("log-dir", "", "日志目录，留空则放在程序旁边的 logs/")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}

	// 日志有三个去处，缺一不可：
	//   标准输出 —— 从终端跑时最直接
	//   内存环形缓冲 —— 界面上的控制台页
	//   文件 —— 前两个都靠不住的时候：双击启动没有控制台，进程退了内存也就没了，
	//           而最需要日志的恰恰是"起不来"和"上次崩了"
	out := io.Writer(os.Stdout)
	logFile, logPath, ferr := applog.Open(*logDir)
	if ferr == nil {
		defer logFile.Close()
		applog.Banner(logFile, version, [2]string{"配置", *cfgPath})
		out = io.MultiWriter(os.Stdout, logFile)
	}

	logs := logbuf.New(4000)
	log := slog.New(logbuf.NewHandler(
		slog.NewTextHandler(out, &slog.HandlerOptions{Level: level}), logs))

	// 启动失败也要留在文件里。以前这类错误只往 stderr 打一行就退出，
	// 而没有控制台的时候那一行谁也看不见。
	defer func() {
		if err != nil {
			log.Error("启动失败", "err", err)
		}
	}()

	if ferr != nil {
		log.Warn("日志文件打不开，只留在控制台与界面上", "err", ferr)
	} else {
		log.Info("日志文件", "路径", logPath)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		return err
	}
	for _, d := range []string{cfg.OutputDir, cfg.DataDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	// 路径都是 finalize 之后的绝对路径，和配置文件里写的未必一样。
	// 排查"它到底在读哪个目录"时，这一行省掉一轮来回。
	log.Info("配置已载入",
		"输出", cfg.OutputDir, "数据", cfg.DataDir, "工作流", cfg.WorkflowsDir,
		"ComfyUI模式", cfg.Comfy.Mode, "ComfyUI地址", cfg.Comfy.BaseURL)

	reg := workflow.NewRegistry(cfg.WorkflowsDir)
	if err := reg.Load(); err != nil {
		return err
	}
	for _, t := range reg.List() {
		log.Info("已载入工作流", "id", t.Meta.ID, "风格", t.Meta.Style, "名称", t.Meta.Name)
	}

	st, err := store.Open(filepath.Join(cfg.DataDir, "dreamtexture.db"))
	if err != nil {
		return err
	}
	defer st.Close()
	if n, err := st.ResumeInterrupted("后端重启，该任务的执行状态已无法追踪"); err != nil {
		return err
	} else if n > 0 {
		log.Warn("清理了上次退出时残留的未完成任务", "数量", n)
	}
	if !st.HasFTS() {
		log.Warn("当前 SQLite 构建不含 FTS5，素材搜索降级为 LIKE 匹配")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sup := comfy.NewSupervisor(cfg.Comfy, log)
	sup.SetSink(logs)
	defer sup.Stop()
	log.Info("正在连接 ComfyUI", "模式", cfg.Comfy.Mode, "地址", cfg.Comfy.BaseURL)
	// ComfyUI 起不来不算致命：后端照常提供界面，只是不能生成。
	//
	// 这条是被一键部署逼出来的——全新装的人根本还没有 ComfyUI，若后端因此
	// 拒绝启动，那个"帮你把 ComfyUI 装起来"的页面就永远打不开，最需要它的人
	// 反而用不上。任务侧本来就会排队等它回来（waitComfyAvailable），
	// 健康探针和环境自检也都会如实报告，所以降级运行是安全的。
	//
	// 放进协程还有第二个理由：ComfyUI 不一定是"起来"或"起不来"两种结局，
	// 它还可能**很慢**——自定义节点首次启动会现装依赖（实测 Impact-Pack 要从
	// GitHub 拉 sam2），进程一直活着，于是 waitReady 一直等下去。同步等的话
	// 这段时间里整个界面都打不开，而控制台页恰恰是唯一能看见它在干什么的地方。
	go func() {
		if err := sup.Start(ctx); err != nil {
			log.Warn("ComfyUI 暂时不可用，后端仍会启动；去设置页部署或修正路径", "原因", err)
			return
		}
		h := sup.Health()
		log.Info("ComfyUI 就绪", "版本", h.Version, "设备", h.Device, "显存MB", h.VRAMTotalMB)
	}()

	bus := job.NewBus()

	secrets, err := model.LoadSecrets(filepath.Join(filepath.Dir(*cfgPath), "secrets.json"))
	if err != nil {
		log.Warn("读取凭据文件失败，下载受限来源时会要求先在设置页填写令牌", "err", err)
	}

	// 设置存储：界面上改的配置经它落回配置文件，代理、压平强度这类
	// 立刻生效的项也由它提供现取。
	live := settings.New(*cfgPath, cfg)

	// 外部底图来源。令牌、代理、接口地址都是现取的，在设置页改完就能用，
	// 不必重启后端。
	imagenReg := imagen.NewRegistry(
		imagen.NewOpenAI(cfg.Imagen.OpenAIBaseURL, secrets, imagen.Transport(live.Proxy)))
	if cfg.Imagen.Proxy != "" {
		log.Info("外部底图来源将走指定代理", "proxy", cfg.Imagen.Proxy)
	}

	runner := job.NewRunner(sup, reg, st, bus, log, job.Options{
		OutputDir: cfg.OutputDir,
		Imagen:    imagenReg,
		Flatten:   live.Flatten,
	})
	nodeMgr := nodes.New(cfg.Comfy.BaseURL)
	cat := catalog.New(cfg.Comfy.BaseURL)
	models := model.NewManager(sup, reg)
	downloads := model.NewDownloader(secrets, log, func(d *model.Download) {
		bus.Publish(job.Event{Type: "model.download", Data: d})
	})
	go downloads.Run(ctx)
	go func() {
		// 开机盘点一次，界面打开就有数据；失败不影响主流程。
		if inv, err := models.Scan(ctx); err != nil {
			log.Warn("模型盘点失败", "err", err)
		} else if n := inv.MissingCount(); n > 0 {
			log.Warn("有工作流所需的模型尚未就位", "缺失数", n)
		}
	}()

	// ComfyUI 的事件流用来推进度；完成判定仍以 /history 为准。
	events := comfy.NewEventStream(cfg.Comfy.BaseURL, runner.ClientID(), log)
	go events.Run(ctx)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-events.Events():
				runner.OnComfyEvent(ev)
			}
		}
	}()
	go runner.Run(ctx)

	// 健康状态变化推给前端，让 ComfyUI 掉线在界面上立刻可见。
	go watchHealth(ctx, sup, bus)

	srv := &api.Server{
		Sup: sup, Reg: reg, Store: st, Runner: runner, Bus: bus,
		Log: log, OutputDir: cfg.OutputDir, DataDir: cfg.DataDir,
		Models: models, Downloads: downloads, Secrets: secrets,
		Nodes: nodeMgr, Catalog: cat, Imagen: imagenReg, Logs: logs, Settings: live,
		Deploy: deploy.New(logs),
		MCP: mcpsrv.Handler(mcpsrv.Deps{
			Sup: sup, Reg: reg, Store: st, Runner: runner,
			Models: models, Downloads: downloads,
			OutputDir: cfg.OutputDir, Version: version,
		}),
		Web: web.Handler(),
	}
	if !web.Built() {
		log.Warn("二进制里没有前端产物，界面不可用；在 web/ 执行 pnpm build 后重新 go build")
	}
	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 15 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("DreamTexture 已启动",
			"版本", version, "监听", cfg.Addr,
			"接口", "http://"+cfg.Addr+"/api", "MCP", "http://"+cfg.Addr+"/mcp")
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		log.Info("收到退出信号，正在关闭")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return httpSrv.Shutdown(shutdownCtx)
}

func watchHealth(ctx context.Context, sup *comfy.Supervisor, bus *job.Bus) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	var lastAlive, lastReady bool
	first := true
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		h := sup.Health()
		if first || h.Alive != lastAlive || h.Ready != lastReady {
			bus.Publish(job.Event{Type: "comfy.status", Data: h})
			lastAlive, lastReady, first = h.Alive, h.Ready, false
		}
	}
}
