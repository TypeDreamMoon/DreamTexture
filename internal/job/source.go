package job

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/imagen"
	"github.com/mengye/dreamtexture/internal/store"
	"github.com/mengye/dreamtexture/internal/workflow"
)

// SourceInfo 记录一次外部底图是怎么来的，写进 manifest。
//
// 云端底图没有可复现的种子（接口不支持），所以"怎么复现"只能靠把请求本身
// 完整记下来：模型、提示词、尺寸、画质。这些字段就是这份材质的出身证明。
type SourceInfo struct {
	Provider string        `json:"provider"`
	Model    string        `json:"model"`
	Prompt   string        `json:"prompt"`
	Size     string        `json:"size,omitempty"`
	Quality  string        `json:"quality,omitempty"`
	Revised  string        `json:"revised_prompt,omitempty"`
	Usage    *imagen.Usage `json:"usage,omitempty"`
	// Flatten 记录亮度场压平的结果，没做就是 nil。
	Flatten *imagen.FlattenReport `json:"flatten,omitempty"`
	// ElapsedMS 是云端那一步的耗时，和本地耗时分开看才有意义。
	ElapsedMS int64 `json:"elapsed_ms,omitempty"`
	// Reproducible 恒为 false：留着这个字段是为了让 manifest 自己说清楚
	// 这份材质重跑一遍不会一样，而不是让人从"seed 为空"去猜。
	Reproducible bool `json:"reproducible"`
}

// checkSource 在提交时就把"肯定跑不通"的情况拦下来。
//
// 云端底图这条路上，等排到才失败的代价比本地大得多：用户可能一次提交好几个
// 变体，每个都要等前面的跑完才轮到，全军覆没之后才知道令牌没填。
func (r *Runner) checkSource(tpl *workflow.Template, params map[string]any) error {
	src := tpl.Meta.Source
	if src == nil {
		return nil
	}
	prov, ok := r.imagen.Get(src.Provider)
	if !ok {
		return fmt.Errorf("工作流 %s 需要底图来源 %q，但它没有注册", tpl.Meta.ID, src.Provider)
	}
	if !prov.Configured() {
		return fmt.Errorf("%s 还没配置访问令牌。去「模型 → 设置」里填上 %s 的 API Key",
			prov.Label(), prov.Label())
	}
	resolved, err := tpl.Resolve(params)
	if err != nil {
		return err
	}
	if key := src.Roles["model"]; key != "" {
		if v, _ := resolved[key].(string); strings.TrimSpace(v) == "" {
			return fmt.Errorf("请先选一个云端模型")
		}
	}
	return nil
}

// fetchSource 调外部接口取底图，压平亮度场，上传给 ComfyUI，返回文件名。
//
// 放在任务执行里同步做，而不是提交时预取：外部接口是按量计费的，用户取消
// 排队中的任务时不该已经把钱花掉了。代价是这段时间 GPU 闲着——对单人单卡
// 的本地工具来说，这个取舍比省几十秒重要。
func (r *Runner) fetchSource(ctx context.Context, j *store.Job, tpl *workflow.Template,
	resolved map[string]any) (string, *SourceInfo, error) {

	src := tpl.Meta.Source
	prov, ok := r.imagen.Get(src.Provider)
	if !ok {
		return "", nil, fmt.Errorf("底图来源 %q 未注册", src.Provider)
	}
	if !prov.Configured() {
		return "", nil, fmt.Errorf("%s 还没配置访问令牌，去「模型 → 设置」里填上", prov.Label())
	}

	role := func(name string) string {
		key := src.Roles[name]
		if key == "" {
			return ""
		}
		v, ok := resolved[key]
		if !ok || v == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(v))
	}

	req := imagen.Request{
		Model:      role("model"),
		Prompt:     r.sourcePrompt(tpl, src, resolved),
		Size:       role("size"),
		Quality:    role("quality"),
		Background: role("background"),
	}
	if req.Model == "" {
		return "", nil, fmt.Errorf("没有选择云端模型")
	}

	// 参考图走图生图：它此刻在 ComfyUI 的 input 目录里（前端上传时就转存过去了），
	// 从那儿取回来再转发。多绕一手是为了让上传通道只有一条——两边各存一份迟早会不一致。
	if name := role("reference"); name != "" {
		data, err := r.sup.Client().View(ctx, comfy.ImageRef{Filename: name, Type: "input"})
		if err != nil {
			return "", nil, fmt.Errorf("取回参考图 %s 失败: %w", name, err)
		}
		req.Reference, req.ReferenceName = data, name
	}

	j.Stage = "云端出图"
	j.Progress = 0.05
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.progress", Job: j})
	r.log.Info("向云端请求底图", "job", j.ID, "provider", src.Provider,
		"model", req.Model, "quality", req.Quality, "size", req.Size, "图生图", len(req.Reference) > 0)

	res, err := prov.Generate(ctx, req)
	if err != nil {
		var refusal *imagen.Refusal
		if errors.As(err, &refusal) {
			return "", nil, fmt.Errorf("云端拒绝了这个提示词：%s\n"+
				"换个说法再试，或改用本地管线——本地没有内容审核", refusal.Reason)
		}
		return "", nil, err
	}

	info := &SourceInfo{
		Provider: src.Provider, Model: res.Model, Prompt: req.Prompt,
		Size: req.Size, Quality: req.Quality, Revised: res.Revised,
		Usage: &res.Usage, ElapsedMS: res.Elapsed.Milliseconds(),
	}
	r.log.Info("云端底图已返回", "job", j.ID, "耗时", res.Elapsed.Round(time.Second),
		"字节", len(res.Image), "花费USD", fmt.Sprintf("%.4f", res.Usage.CostUSD))

	img := res.Image
	if flat, rep, err := imagen.FlattenLuminance(img, r.flatten()); err != nil {
		// 压平失败不该让整个任务垮掉：底图本身是好的，大不了带着暗角继续。
		r.log.Warn("底图亮度场压平失败，按原图继续", "job", j.ID, "err", err)
	} else {
		img = flat
		info.Flatten = &rep
		if rep.Applied {
			r.log.Info("已压平底图亮度场", "job", j.ID,
				"边缘中心比", fmt.Sprintf("%.4f→%.4f", rep.Falloff, rep.FalloffAfter))
		}
	}

	name := "dt_src_" + j.MaterialID + ".png"
	stored, err := r.sup.Client().UploadImage(ctx, name, img)
	if err != nil {
		return "", nil, fmt.Errorf("把底图转存到 ComfyUI 失败: %w", err)
	}
	return stored, info, nil
}

// sourcePrompt 拼出真正送给云端的提示词。
//
// 必须带上模板声明的前后缀。那不是装饰——CHORD 对平光输入很敏感，
// "flat even lighting, no shadows" 这类约束一旦丢掉，分解出来的
// 粗糙度和法线就会把烘焙进底图的阴影当成真实起伏。
func (r *Runner) sourcePrompt(tpl *workflow.Template, src *workflow.Source, resolved map[string]any) string {
	key := src.Roles["prompt"]
	body := ""
	if v, ok := resolved[key]; ok && v != nil {
		body = fmt.Sprint(v)
	}
	for _, p := range tpl.Meta.AllParams() {
		if p.Key == key {
			return p.Prefix + body + p.Suffix
		}
	}
	return body
}
