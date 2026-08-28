package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/settings"
)

// 参数目录只在进程内算一次。显卡列表要 fork 一个 nvidia-smi，
// 而它在整个进程生命周期里不会变。
var (
	catalogOnce []comfy.Option
)

func (s *Server) flagCatalog() []comfy.Option {
	if catalogOnce == nil {
		catalogOnce = comfy.Catalog(comfy.DetectGPUs())
	}
	return catalogOnce
}

// comfyFlags 把当前的启动参数拆成界面能渲染的形状。
func (s *Server) comfyFlags(w http.ResponseWriter, _ *http.Request) {
	cat := s.flagCatalog()
	args := s.Settings.Get().Comfy.ExtraArgs
	values, leftover := comfy.Parse(args, cat)

	writeJSON(w, http.StatusOK, map[string]any{
		"catalog": cat,
		"values":  values,
		// extra 用空格连起来给界面当一行文本编辑。参数里带空格的极少
		// （路径类的在这儿基本用不到），为此上一套引号解析不值得。
		"extra":   strings.Join(leftover, " "),
		"managed": comfy.ManagedIn(args),
		"raw":     strings.Join(args, " "),
	})
}

// setComfyFlags 把界面上的取值合成参数写回配置。
//
// 合成放在后端而不是前端：Build 与 Parse 必须是同一份逻辑的两半，
// 分家之后"存进去的和读出来的不一样"这种 bug 会非常难查。
func (s *Server) setComfyFlags(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Values map[string]string `json:"values"`
		Extra  string            `json:"extra"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}

	args := comfy.Build(body.Values, strings.Fields(body.Extra), s.flagCatalog())
	if bad := comfy.ManagedIn(args); len(bad) > 0 {
		writeErr(w, http.StatusBadRequest,
			strings.Join(bad, "、")+" 由 DreamTexture 自己填，写在这里只会和它打架。"+
				"监听地址在「ComfyUI 地址」，显存余量有单独一项")
		return
	}

	needRestart, err := s.Settings.Apply(settings.Patch{ComfyExtraArgs: &args})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"need_restart": needRestart,
		"args":         strings.Join(args, " "),
	})
}
