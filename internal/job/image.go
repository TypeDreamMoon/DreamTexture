package job

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/picture"
	"github.com/mengye/dreamtexture/internal/store"
	"github.com/mengye/dreamtexture/internal/workflow"
)

// executeDirect 跑纯云端出图：拿回来就是成品，不经过 ComfyUI。
//
// 这条路刻意绕开了 waitComfyAvailable —— 想用云端出张图不该先要求本机
// 装好一整套 ComfyUI，而那恰恰是新用户还没有的东西。
func (r *Runner) executeDirect(ctx context.Context, j *store.Job, tpl *workflow.Template) error {
	resolved, err := tpl.Resolve(j.Params)
	if err != nil {
		return err
	}
	res, info, err := r.callSource(ctx, j, tpl, resolved)
	if err != nil {
		return err
	}

	j.Stage = "落盘中"
	j.Progress = 0.9
	_ = r.st.UpdateJob(j)
	r.bus.Publish(Event{Type: "job.progress", Job: j})

	// 直出不做亮度场压平：那是给"要拿去平铺"的底图准备的，
	// 单张图片按原样保留模型给的明暗才对。
	return r.savePicture(j, tpl, resolved, res.Image, info)
}

// collectImage 从 ComfyUI 的产物里取出单张图并落盘。
// 本地工作流出图（kind=image 但有节点图）走这条。
func (r *Runner) collectImage(ctx context.Context, j *store.Job, tpl *workflow.Template,
	rendered *workflow.Rendered, entry *comfy.HistoryEntry) error {

	// kind=image 的模板只声明一路产物，取它就是。
	var nodeID string
	for id := range rendered.Expect {
		nodeID = id
		break
	}
	out := entry.Outputs[nodeID]
	if len(out.Images) == 0 {
		return fmt.Errorf("ComfyUI 没有返回图片")
	}
	data, err := r.sup.Client().View(ctx, out.Images[0])
	if err != nil {
		return fmt.Errorf("取回图片: %w", err)
	}
	r.mu.Lock()
	info := r.sources[j.ID]
	r.mu.Unlock()
	return r.savePicture(j, tpl, rendered.Values, data, info)
}

// savePicture 把一张图连同它的来历写进输出目录并建索引。
func (r *Runner) savePicture(j *store.Job, tpl *workflow.Template,
	values map[string]any, data []byte, info *SourceInfo) error {

	str := func(k string) string {
		if v, ok := values[k]; ok && v != nil {
			return fmt.Sprint(v)
		}
		return ""
	}
	name, _ := j.Params["__name"].(string)
	if name == "" {
		name = str("prompt")
	}
	if name == "" {
		name = "未命名"
	}

	// 提示词经过了模板的前后缀拼接，存的要是真正送出去的那一份。
	prompt := str("prompt")
	for _, p := range tpl.Meta.AllParams() {
		if p.Key == "prompt" && (p.Prefix != "" || p.Suffix != "") {
			prompt = p.Prefix + prompt + p.Suffix
		}
	}

	var seed int64
	if v, ok := values["seed"].(int64); ok {
		seed = v
	}

	m := &picture.Meta{
		ID: j.MaterialID, Name: trimName(name), WorkflowID: tpl.Meta.ID,
		Prompt: prompt, Negative: str("negative"), Seed: seed,
		Params: values, CreatedAt: time.Now(),
	}
	if ref := str("reference"); ref != "" {
		m.Reference = ref
	}
	if info != nil {
		m.Source = &picture.Source{
			Provider: info.Provider, Model: info.Model, Size: info.Size,
			Quality: info.Quality, RevisedPrompt: info.Revised, ElapsedMS: info.ElapsedMS,
		}
		if u := info.Usage; u != nil {
			m.Source.InputTokens, m.Source.OutputTokens = u.InputTokens, u.OutputTokens
			m.Source.CostUSD = u.CostUSD
		}
		// 云端不支持种子，别在 manifest 里留一个假的。
		m.Seed = 0
	}

	if err := picture.Write(r.outputDir, m, data); err != nil {
		return err
	}
	rec := &store.Picture{
		ID: m.ID, Name: m.Name, WorkflowID: m.WorkflowID, Prompt: m.Prompt,
		Negative: m.Negative, Seed: m.Seed, Width: m.Width, Height: m.Height,
		CreatedAt: m.CreatedAt,
	}
	if m.Source != nil {
		rec.Provider, rec.Model, rec.CostUSD = m.Source.Provider, m.Source.Model, m.Source.CostUSD
	}
	if err := r.st.IndexPicture(rec); err != nil {
		// 索引写不进去就把文件也撤掉：留下一张检索不到的图，
		// 用户只会觉得"生成成功了但列表里没有"。
		_ = picture.Remove(r.outputDir, m.ID)
		return err
	}
	r.log.Info("图片已落盘", "job", j.ID, "图片", m.ID,
		"尺寸", fmt.Sprintf("%dx%d", m.Width, m.Height))
	return nil
}

// trimName 把提示词裁成能当标题的长度。
func trimName(s string) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	rs := []rune(s)
	if len(rs) > 40 {
		return string(rs[:40]) + "…"
	}
	return s
}
