// Package job 实现任务队列与执行器。
//
// 单 GPU 就只跑一个 ComfyUI 实例、只开一个 worker——同一张卡上并发跑多个任务
// 会互相挤爆显存。排队在这里做，ComfyUI 那边始终只有一个在执行。
package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	mrand "math/rand/v2"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/imagen"
	"github.com/mengye/dreamtexture/internal/material"
	"github.com/mengye/dreamtexture/internal/store"
	"github.com/mengye/dreamtexture/internal/workflow"
)

// ErrQueueFull 表示排队数已达上限。
var ErrQueueFull = errors.New("任务队列已满")

type Runner struct {
	sup    *comfy.Supervisor
	reg    *workflow.Registry
	st     *store.Store
	bus    *Bus
	log    *slog.Logger
	imagen *imagen.Registry
	// flatten 现取而不是存值：用户在设置页改完压平强度，下一个任务就该按
	// 新值来，不该要求重启后端。
	flatten   func() float64
	outputDir string
	clientID  string

	queue chan string // 只放任务 id，任务本体以数据库为准

	mu       sync.Mutex
	current  *store.Job
	rendered map[string]*workflow.Rendered // jobID -> 渲染结果，执行期用
	sources  map[string]*SourceInfo        // jobID -> 外部底图的出处，落 manifest 用
	cancel   context.CancelFunc
	// lastWorkflow 记录上一个执行过的工作流，用于判断要不要先腾显存。
	lastWorkflow string

	// beat 是卡死看门狗的心跳，见 stall.go。
	beat beat
	// node 是 ComfyUI 正在执行的节点 id，卡死时用来指认停在哪一步。
	node atomic.Value
}

type Options struct {
	OutputDir string
	QueueSize int
	// Imagen 是外部底图来源；为 nil 时声明了 source 的工作流会直接报错。
	Imagen *imagen.Registry
	// Flatten 现取底图亮度场压平强度；为 nil 时按 1（全压平）。
	Flatten func() float64
}

func NewRunner(sup *comfy.Supervisor, reg *workflow.Registry, st *store.Store, bus *Bus,
	log *slog.Logger, opt Options) *Runner {
	if opt.QueueSize <= 0 {
		opt.QueueSize = 64
	}
	if opt.Imagen == nil {
		opt.Imagen = imagen.NewRegistry()
	}
	if opt.Flatten == nil {
		opt.Flatten = func() float64 { return 1 }
	}
	return &Runner{
		sup: sup, reg: reg, st: st, bus: bus, log: log,
		imagen:    opt.Imagen,
		flatten:   opt.Flatten,
		outputDir: opt.OutputDir,
		clientID:  "dreamtexture-" + randHex(8),
		queue:     make(chan string, opt.QueueSize),
		rendered:  map[string]*workflow.Rendered{},
		sources:   map[string]*SourceInfo{},
	}
}

func (r *Runner) ClientID() string { return r.clientID }

// Request 是一次生成请求。
type Request struct {
	WorkflowID string         `json:"workflow_id"`
	Params     map[string]any `json:"params"`
	// Variants 是要生成的变体数。一次多出几张再挑，比反复重 roll 体验好得多。
	Variants int `json:"variants"`
	// Name 是材质展示名，留空则取提示词。
	Name string `json:"name"`
}

// Submit 把一次请求拆成 N 个任务入队，返回创建的任务。
func (r *Runner) Submit(req Request) ([]*store.Job, error) {
	tpl, ok := r.reg.Get(req.WorkflowID)
	if !ok {
		return nil, fmt.Errorf("工作流 %q 不存在", req.WorkflowID)
	}
	if req.Variants <= 0 {
		req.Variants = 1
	}
	if req.Variants > 8 {
		req.Variants = 8
	}

	// 提前跑一次仅为校验参数：宁可在提交时就报错，也不要等排到了才失败。
	//
	// 直出的模板没有节点图，只能校验到参数这一层——拿 Render 去校验它会被
	// 一句"没有节点图可渲染"挡在门外，整条云端出图就此提交不了。
	if err := validateParams(tpl, req.Params); err != nil {
		return nil, err
	}
	if err := r.checkSource(tpl, req.Params); err != nil {
		return nil, err
	}

	batch := randHex(6)
	now := time.Now()
	jobs := make([]*store.Job, 0, req.Variants)
	for i := 0; i < req.Variants; i++ {
		params := map[string]any{}
		for k, v := range req.Params {
			params[k] = v
		}
		params["seed"] = resolveSeed(req.Params["seed"], i)
		if req.Name != "" {
			params["__name"] = req.Name
		}

		j := &store.Job{
			ID:         "job_" + now.Format("20060102_150405") + "_" + randHex(4),
			MaterialID: "mat_" + now.Format("20060102_150405") + "_" + randHex(4),
			WorkflowID: req.WorkflowID,
			BatchID:    batch,
			Params:     params,
			Status:     store.StatusQueued,
			CreatedAt:  time.Now(),
		}
		if err := r.st.CreateJob(j); err != nil {
			return nil, err
		}
		select {
		case r.queue <- j.ID:
		default:
			j.Status = store.StatusFailed
			j.Error = ErrQueueFull.Error()
			_ = r.st.UpdateJob(j)
			return jobs, ErrQueueFull
		}
		jobs = append(jobs, j)
		r.bus.Publish(Event{Type: "job.queued", Job: j})
	}
	return jobs, nil
}

// validateParams 在提交时先把参数验一遍。
//
// 有节点图的照旧整张渲染，顺带能查出连线断掉之类的模板问题；直出的只验参数。
func validateParams(tpl *workflow.Template, params map[string]any) error {
	if tpl.Meta.Direct() {
		_, err := tpl.Resolve(params)
		return err
	}
	_, err := tpl.Render(params, "probe")
	return err
}

// maxSafeSeed 是 JSON 数字能无损往返的上界（2^53-1）。
//
// 种子要穿过好几层 JSON：入库、出库、写进 manifest、发给前端与 MCP。
// JSON 没有整数类型，超过 2^53 的值在任何一环被解成 float64 都会被悄悄改写，
// 于是 manifest 里记的种子复现不出原图。索性只在安全区间取值。
const maxSafeSeed = 1<<53 - 1

// resolveSeed 把 -1（或缺省）解释为随机；同一批次的变体之间种子必须不同。
func resolveSeed(raw any, variant int) int64 {
	base := int64(-1)
	switch v := raw.(type) {
	case float64:
		base = int64(v)
	case int:
		base = int64(v)
	case int64:
		base = v
	}
	if base < 0 {
		return int64(mrand.Uint64N(maxSafeSeed))
	}
	if base > maxSafeSeed {
		base = base % maxSafeSeed
	}
	return base + int64(variant)
}

// Run 是 worker 主循环，阻塞到 ctx 取消。
func (r *Runner) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-r.queue:
			r.runOne(ctx, id)
		}
	}
}

// Cancel 取消一个任务：排队中的直接判负，正在跑的连带打断 ComfyUI。
func (r *Runner) Cancel(ctx context.Context, id string) error {
	j, err := r.st.GetJob(id)
	if err != nil {
		return err
	}
	if j == nil {
		return fmt.Errorf("任务 %s 不存在", id)
	}
	if j.Status.Terminal() {
		return fmt.Errorf("任务 %s 已结束（%s），无法取消", id, j.Status)
	}

	r.mu.Lock()
	isCurrent := r.current != nil && r.current.ID == id
	cancel := r.cancel
	r.mu.Unlock()

	if isCurrent {
		if err := r.sup.Client().Interrupt(ctx); err != nil {
			r.log.Warn("打断 ComfyUI 失败", "err", err)
		}
		if cancel != nil {
			cancel()
		}
		return nil
	}

	now := time.Now()
	j.Status, j.Error, j.FinishedAt = store.StatusCanceled, "已取消", &now
	if err := r.st.UpdateJob(j); err != nil {
		return err
	}
	r.bus.Publish(Event{Type: "job.failed", Job: j})
	return nil
}

func (r *Runner) runOne(parent context.Context, id string) {
	j, err := r.st.GetJob(id)
	if err != nil || j == nil {
		r.log.Error("取任务失败", "job", id, "err", err)
		return
	}
	// 排队期间可能已被取消。
	if j.Status != store.StatusQueued {
		return
	}

	ctx, cancel := context.WithCancel(parent)
	r.mu.Lock()
	r.current, r.cancel = j, cancel
	r.mu.Unlock()
	defer func() {
		cancel()
		r.mu.Lock()
		r.current, r.cancel = nil, nil
		delete(r.rendered, j.ID)
		delete(r.sources, j.ID)
		r.mu.Unlock()
	}()

	now := time.Now()
	j.Status, j.StartedAt, j.Stage = store.StatusRunning, &now, "提交中"
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.progress", Job: j})

	if err := r.execute(ctx, j); err != nil {
		fin := time.Now()
		j.FinishedAt = &fin
		if ctx.Err() != nil && parent.Err() == nil {
			j.Status, j.Error = store.StatusCanceled, "已取消"
		} else {
			j.Status, j.Error = store.StatusFailed, err.Error()
			r.log.Error("任务失败", "job", j.ID, "err", err)
		}
		_ = r.st.UpdateJob(j)
		r.bus.Publish(Event{Type: "job.failed", Job: j})
		return
	}

	fin := time.Now()
	j.Status, j.FinishedAt, j.Progress, j.Stage = store.StatusSucceeded, &fin, 1, "完成"
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.done", Job: j})
	r.log.Info("任务完成", "job", j.ID, "材质", j.MaterialID, "耗时", fin.Sub(now).Round(time.Second))
}

func (r *Runner) execute(ctx context.Context, j *store.Job) error {
	tpl, ok := r.reg.Get(j.WorkflowID)
	if !ok {
		return fmt.Errorf("工作流 %q 不存在", j.WorkflowID)
	}

	// 纯云端直出：整条链路上没有 ComfyUI 的事，连等它都不必等。
	//
	// 这一点很要紧——否则"想用云端出张图"就得先有一个装好的 ComfyUI，
	// 而那正是新用户还没有的东西。
	if tpl.Meta.Direct() {
		return r.executeDirect(ctx, j, tpl)
	}

	if err := r.waitComfyAvailable(ctx, j); err != nil {
		return err
	}

	// 底图来自外部服务时，先把图取回来上传给 ComfyUI，再把文件名当成
	// 普通参数注入——这样云端来源就复用了既有的整条注入与产出路径。
	params := j.Params
	if src := tpl.Meta.Source; src != nil {
		resolved, err := tpl.Resolve(params)
		if err != nil {
			return err
		}
		name, info, err := r.fetchSource(ctx, j, tpl, resolved)
		if err != nil {
			return err
		}
		params = make(map[string]any, len(j.Params)+1)
		for k, v := range j.Params {
			params[k] = v
		}
		params[src.ImageParam] = name
		r.mu.Lock()
		r.sources[j.ID] = info
		r.mu.Unlock()
	}

	// 每个任务用互不相同的输出前缀：既防不同任务互相覆盖，也让 ComfyUI 的
	// 节点缓存不会把上一个任务的文件名当成这次的产物报回来。
	rendered, err := tpl.Render(params, "dreamtexture/"+j.MaterialID)
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.rendered[j.ID] = rendered
	r.mu.Unlock()

	cli := r.sup.Client()
	r.freeIfSwitching(ctx, j.WorkflowID)

	resp, err := cli.Submit(ctx, rendered.Graph, r.clientID)
	if err != nil {
		return err
	}
	// 校验期错误只出现在这里。ComfyUI 会把没问题的分支照常跑完，
	// 整体状态仍报 success —— 不看这里就会把残缺产物当成功。
	if s := resp.NodeErrorSummary(rendered.TitleOf); s != "" {
		return fmt.Errorf("工作流校验未通过:\n%s", s)
	}

	j.PromptID = resp.PromptID
	j.Stage = "生成中"
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.progress", Job: j})

	entry, err := r.waitDone(ctx, j)
	if err != nil {
		return err
	}
	if msg := entry.ExecutionError(); msg != "" {
		return errors.New(msg)
	}

	// 判定成败的真正依据：期望的输出节点必须一个不少。
	if missing := missingOutputs(rendered.Expect, entry); len(missing) > 0 {
		return fmt.Errorf("产物不完整，缺少 %s（ComfyUI 报告的状态是 %s）",
			strings.Join(missing, "、"), entry.Status.StatusStr)
	}

	j.Stage = "落盘中"
	j.Progress = 0.95
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.progress", Job: j})

	if tpl.Meta.Kind == workflow.KindImage {
		return r.collectImage(ctx, j, tpl, rendered, entry)
	}
	return r.collect(ctx, j, tpl, rendered, entry)
}

// comfyWaitTimeout 是等待 ComfyUI 回来的上限。
//
// 给得比较宽松是有原因的：ComfyUI 重启后可能在装缺失的依赖（自定义节点的
// requirements，个别还要从源码编译），这段时间端口根本不开，实测能持续好几分钟。
const comfyWaitTimeout = 15 * time.Minute

// waitComfyAvailable 等 ComfyUI 可用，而不是发现不可用就把任务判死。
//
// 用户点了生成，任务却因为后台正在重启 ComfyUI 而瞬间失败，是很糟的体验——
// 尤其重启往往就是后端自己发起的。排着等回来才对，等不到再报错。
func (r *Runner) waitComfyAvailable(ctx context.Context, j *store.Job) error {
	if r.sup.Health().Alive {
		return nil
	}

	j.Stage = "等待 ComfyUI"
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.progress", Job: j})
	r.log.Info("ComfyUI 当前不可用，任务排队等待", "job", j.ID)

	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	deadline := time.Now().Add(comfyWaitTimeout)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
		if r.sup.Health().Alive {
			r.log.Info("ComfyUI 已恢复，继续执行", "job", j.ID)
			j.Stage = "提交中"
			_ = r.st.UpdateJob(j)
			r.bus.Publish(Event{Type: "job.progress", Job: j})
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等待 ComfyUI 恢复超过 %s 仍不可用: %s",
				comfyWaitTimeout, r.sup.Health().Reason)
		}
	}
}

// freeIfSwitching 在换工作流时先让 ComfyUI 卸载已加载的模型。
//
// 起因是实测：16GB 卡上，如果显存已经被上一条管线的模型占着（底模 + LoRA 等），
// 再跑 CHORD 分解会慢一个数量级——日志里能看到 "Unloaded partially"，
// 显存逼近上限时 Python 循环密集的泊松求解器会被分配器拖死。
// 同一个工作流连着跑不受影响，所以只在切换时腾一次，代价是重新加载模型的几秒钟。
func (r *Runner) freeIfSwitching(ctx context.Context, workflowID string) {
	r.mu.Lock()
	prev := r.lastWorkflow
	r.lastWorkflow = workflowID
	r.mu.Unlock()

	if prev == "" || prev == workflowID {
		return
	}
	if err := r.sup.Client().Free(ctx, true, true); err != nil {
		r.log.Debug("切换工作流前卸载模型失败，继续执行", "err", err)
		return
	}
	r.log.Info("切换工作流，已卸载上一条管线的模型", "从", prev, "到", workflowID)
}

// waitDone 等任务执行完。
//
// 以 WebSocket 事件推进度，但完成判定一律以 /history 为准：事件可能因为
// 断线或缓冲丢弃而漏掉，history 是唯一可靠的事实来源。
func (r *Runner) waitDone(ctx context.Context, j *store.Job) (*comfy.HistoryEntry, error) {
	cli := r.sup.Client()
	poll := time.NewTicker(2 * time.Second)
	defer poll.Stop()

	r.beat.touch()
	r.node.Store("")

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-poll.C:
		}
		entry, ok, err := cli.History(ctx, j.PromptID)
		if err != nil {
			// ComfyUI 可能正在重启，让健康巡检去处理，这里继续等。
			// 连不上也不该推进心跳：真断了要让看门狗算数。
			r.log.Debug("查询 history 失败，稍后重试", "job", j.ID, "err", err)
			continue
		}
		if ok && entry.Status.Completed {
			return entry, nil
		}
		// 没跑完，看看是"在跑"还是"死在那儿"。
		if idle := r.beat.idle(); idle > stallTimeout {
			node, _ := r.node.Load().(string)
			reason := r.stallReason(ctx, idle, node)
			r.log.Warn("判定任务卡死，正在打断 ComfyUI", "job", j.ID, "空转", idle, "节点", node)
			// 必须打断：否则 ComfyUI 会一直磨这张图，后面排队的任务全跟着陪葬。
			if err := cli.Interrupt(context.WithoutCancel(ctx)); err != nil {
				r.log.Warn("打断 ComfyUI 失败", "err", err)
			}
			return nil, errors.New(reason)
		}
	}
}

// OnComfyEvent 把 ComfyUI 的进度事件折算到当前任务上。
func (r *Runner) OnComfyEvent(ev comfy.Event) {
	r.mu.Lock()
	j := r.current
	r.mu.Unlock()
	if j == nil || (ev.PromptID != "" && ev.PromptID != j.PromptID) {
		return
	}

	// executing 也算动静：采样之外的环节（解码、分解、落盘）没有 progress
	// 事件，只有换节点时吭一声。不把它算进心跳，看门狗会误杀正常的慢步骤。
	if ev.Type == "executing" {
		r.beat.touch()
		if ev.Node != "" {
			r.node.Store(ev.Node)
		}
		return
	}
	if ev.Type != "progress" || ev.Max <= 0 {
		return
	}
	r.beat.touch()
	// 采样占大头，但后面还有分解和落盘，所以留出尾部空间。
	p := float64(ev.Value) / float64(ev.Max) * 0.9
	if p <= j.Progress {
		return
	}
	j.Progress = p
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.progress", Job: j})
}

func missingOutputs(expect map[string]string, entry *comfy.HistoryEntry) []string {
	var missing []string
	for nodeID, channel := range expect {
		out, ok := entry.Outputs[nodeID]
		if !ok || len(out.Images) == 0 {
			missing = append(missing, channel)
		}
	}
	sort.Strings(missing)
	return missing
}

func (r *Runner) collect(ctx context.Context, j *store.Job, tpl *workflow.Template,
	rendered *workflow.Rendered, entry *comfy.HistoryEntry) error {

	dir, err := material.NewDir(r.outputDir, j.MaterialID)
	if err != nil {
		return err
	}
	// 中途失败就把半成品清掉，避免素材库里出现残缺套装。
	ok := false
	defer func() {
		if !ok {
			dir.Remove()
		}
	}()

	cli := r.sup.Client()
	maps := map[string]material.Map{}
	var basecolor []byte

	for nodeID, channel := range rendered.Expect {
		out := entry.Outputs[nodeID]
		ref := out.Images[0]
		data, err := cli.View(ctx, ref)
		if err != nil {
			return fmt.Errorf("取回 %s: %w", channel, err)
		}
		spec := tpl.Meta.Outputs[channel]
		y := spec.Y
		if spec.YFromParam != "" {
			if v, exists := rendered.Values[spec.YFromParam]; exists {
				y = strings.ToLower(fmt.Sprint(v))
			}
		}
		m, err := dir.WriteMap(channel, data, spec.Colorspace, y, spec.Packing, spec.Role)
		if err != nil {
			return err
		}
		maps[channel] = m
		if channel == "basecolor" {
			basecolor = data
		}
	}

	preview := ""
	if basecolor != nil {
		if png, err := material.TilePreview(basecolor, 340); err == nil {
			if err := dir.WriteFile("preview.png", png); err == nil {
				preview = "preview.png"
			}
		} else {
			r.log.Warn("生成平铺预览失败", "job", j.ID, "err", err)
		}
	}

	man := r.buildManifest(j, tpl, rendered, maps, preview)
	if err := dir.WriteManifest(man); err != nil {
		return err
	}
	if err := r.st.IndexMaterial(&store.Material{
		ID:         man.ID,
		Name:       man.Name,
		Style:      man.Style,
		WorkflowID: man.Workflow.ID,
		Prompt:     man.Prompt,
		Negative:   man.Negative,
		Seed:       man.Seed,
		Resolution: man.Resolution,
		CreatedAt:  man.CreatedAt,
	}); err != nil {
		return err
	}
	ok = true
	return nil
}

func (r *Runner) buildManifest(j *store.Job, tpl *workflow.Template,
	rendered *workflow.Rendered, maps map[string]material.Map, preview string) *material.Manifest {

	str := func(k string) string {
		if v, ok := rendered.Values[k]; ok && v != nil {
			return fmt.Sprint(v)
		}
		return ""
	}
	name, _ := j.Params["__name"].(string)
	if name == "" {
		name = str("prompt")
	}

	var seed int64
	if v, ok := rendered.Values["seed"]; ok {
		if n, ok := v.(int64); ok {
			seed = n
		}
	}
	res := tpl.Meta.Resolution
	if v, ok := rendered.Values["resolution"]; ok {
		if f, ok := v.(float64); ok {
			res = int(f)
		} else if n, ok := v.(int64); ok {
			res = int(n)
		}
	}

	params := map[string]any{}
	for k, v := range rendered.Values {
		params[k] = v
	}

	var ref *material.Reference
	if v, ok := rendered.Values["reference"]; ok && strings.TrimSpace(fmt.Sprint(v)) != "" {
		ref = &material.Reference{File: fmt.Sprint(v)}
		if d, ok := j.Params["__ref_origin"].(string); ok {
			ref.Origin = d
		}
		if dn, ok := rendered.Values["denoise"].(float64); ok {
			ref.Denoise = dn
		}
	}

	r.mu.Lock()
	srcInfo := r.sources[j.ID]
	r.mu.Unlock()
	var src *material.Source
	if srcInfo != nil {
		src = &material.Source{
			Provider: srcInfo.Provider, Model: srcInfo.Model, Prompt: srcInfo.Prompt,
			Size: srcInfo.Size, Quality: srcInfo.Quality, RevisedPrompt: srcInfo.Revised,
			ElapsedMS: srcInfo.ElapsedMS,
		}
		if u := srcInfo.Usage; u != nil {
			src.InputTokens, src.OutputTokens, src.CostUSD = u.InputTokens, u.OutputTokens, u.CostUSD
		}
		if f := srcInfo.Flatten; f != nil {
			src.Flattened, src.FalloffBefore, src.FalloffAfter = f.Applied, f.Falloff, f.FalloffAfter
		}
	}

	man := &material.Manifest{
		ID:         j.MaterialID,
		Name:       name,
		Style:      tpl.Meta.Style,
		Workflow:   material.WorkflowRef{ID: tpl.Meta.ID, Version: tpl.Meta.Version},
		Prompt:     str("prompt"),
		Negative:   str("negative"),
		Seed:       seed,
		Resolution: res,
		Tileable:   tileable(tpl, rendered),
		Reference:  ref,
		Source:     src,
		Maps:       maps,
		Preview:    preview,
		CreatedAt:  time.Now(),
		Params:     params,
		Generator: material.Generator{
			ComfyUI:   r.sup.Health().Version,
			NodePacks: tpl.Meta.NodePacks,
		},
	}
	// 提示词经过了前后缀拼接，这里存的是真正送进 ComfyUI 的完整文本。
	for _, p := range tpl.Meta.AllParams() {
		if p.Key == "prompt" && (p.Prefix != "" || p.Suffix != "") {
			man.Prompt = p.Prefix + str("prompt") + p.Suffix
		}
	}
	if len(tpl.Meta.Licenses) > 0 {
		ok, reason := tpl.Meta.Commercial()
		man.LicenseFlags = &material.LicenseFlags{CommercialUse: ok, Reason: reason}
	}
	// 只记这一次**真的用到**的模型。
	//
	// 不能照抄模板声明的 model_requirements：条件接线会让整条支路被删掉——
	// 云端底图直出时 SDXL 那一段根本不在提交的图里，底模一次都没加载过。
	// 照抄的话 manifest 会写上一个从未参与的底模，排障时能把人带得很远。
	loaded := map[string]bool{}
	for _, n := range rendered.Graph {
		for _, v := range n.Inputs {
			if s, ok := v.(string); ok {
				loaded[s] = true
			}
		}
	}
	for _, req := range tpl.Meta.ModelRequirements {
		if req.Kind != "checkpoint" || !loaded[req.File] {
			continue
		}
		if strings.Contains(strings.ToLower(req.File), "chord") {
			man.Generator.PBREstimator = req.File
		} else if man.Generator.Checkpoint == "" {
			man.Generator.Checkpoint = req.File
		}
	}
	return man
}

// tileable 判定这一次的产物到底无不无缝。
//
// 不能只看模板上那个静态的 tileable：云端底图管线的无缝性取决于用户有没有
// 开本地重整，同一份模板两种结果。manifest 里这个字段是给三维预览和 UE 用的，
// 报错了就是骗下游。
func tileable(tpl *workflow.Template, rendered *workflow.Rendered) bool {
	key := tpl.Meta.TileableWhenPositive
	if key == "" {
		return tpl.Meta.Tileable
	}
	v, ok := rendered.Values[key]
	if !ok {
		return tpl.Meta.Tileable
	}
	switch n := v.(type) {
	case float64:
		return n > 0
	case int64:
		return n > 0
	}
	return tpl.Meta.Tileable
}

func randHex(n int) string {
	b := make([]byte, (n+1)/2)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:n]
}
