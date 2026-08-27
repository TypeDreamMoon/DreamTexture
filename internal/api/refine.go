package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mengye/dreamtexture/internal/imagen"
)

// refineProvider 是扩写那套凭据的 id，与 model.Providers 里的条目对应。
const refineProvider = "openai-text"

// probeRefine 真扩写一次，用来验证文本接口配对了没有。
func probeRefine(ctx context.Context, s *Server) string {
	p, ok := s.Imagen.Get("openai")
	if !ok {
		return "无法验证：openai 来源未注册"
	}
	rf, ok := p.(imagen.Refiner)
	if !ok {
		return "无法验证：该来源不支持扩写"
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	out, err := rf.Refine(ctx, imagen.RefineRequest{
		Prompt:  "red brick",
		Model:   s.Settings.Get().Imagen.RefineModel,
		Purpose: "image",
	})
	if err != nil {
		return "但扩写不通：" + firstLine(err.Error())
	}
	return fmt.Sprintf("扩写可用（%s，%dms）", out.Model, out.Elapsed)
}

// refinePrompt 让文本模型把提示词扩写开。
//
// 只返回结果、不替用户改：扩写完让他自己看一眼再决定用不用。模型有时会自作
// 主张加东西，直接盖掉输入框会让人莫名其妙地丢掉自己写的要求。
func (s *Server) refinePrompt(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prompt   string `json:"prompt"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		// Purpose：texture 会带上正交平光可平铺那套硬约束，image 不带。
		Purpose string `json:"purpose"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	if body.Provider == "" {
		body.Provider = "openai"
	}
	p, ok := s.Imagen.Get(body.Provider)
	if !ok {
		writeErr(w, http.StatusBadRequest, "来源 "+body.Provider+" 未注册")
		return
	}
	rf, ok := p.(imagen.Refiner)
	if !ok {
		writeErr(w, http.StatusBadRequest, p.Label()+" 不支持提示词扩写")
		return
	}
	// 不在这里查 p.Configured()：扩写可以有自己的令牌与地址（出图网关不提供
	// 对话接口时就得这么配）。凭据是否齐全由 Refine 自己判断，它才知道
	// 该看哪一套。

	model := body.Model
	if model == "" {
		model = s.Settings.Get().Imagen.RefineModel
	}
	out, err := rf.Refine(r.Context(), imagen.RefineRequest{
		Prompt: body.Prompt, Model: model, Purpose: body.Purpose,
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// TextLister 是能列出文本模型的来源。单开一个接口而不是塞进 Provider：
// 出图和扩写可以是两家服务，不是每家都两样都做。
type TextLister interface {
	TextModels(ctx context.Context) ([]string, error)
}

// textModels 给设置页的「提示词扩写模型」下拉框供数。
//
// 列不出来不算错——网关可能不提供 /models，令牌也可能只有调用权限没有列举权限。
// 这时返回空清单加一句原因，界面上照样能手输模型名。
func (s *Server) textModels(w http.ResponseWriter, r *http.Request) {
	out := map[string]any{
		"models":  []string{},
		"default": imagen.DefaultRefineModel,
	}
	p, ok := s.Imagen.Get("openai")
	if !ok {
		out["error"] = "openai 来源未注册"
		writeJSON(w, http.StatusOK, out)
		return
	}
	lister, ok := p.(TextLister)
	if !ok {
		out["error"] = p.Label() + " 不支持列举文本模型"
		writeJSON(w, http.StatusOK, out)
		return
	}
	ids, err := lister.TextModels(r.Context())
	if err != nil {
		out["error"] = firstLine(err.Error())
	} else {
		out["models"] = ids
	}
	writeJSON(w, http.StatusOK, out)
}
