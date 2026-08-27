package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/mengye/dreamtexture/internal/imagen"
	"github.com/mengye/dreamtexture/internal/settings"
)

// runtimeConfig 返回设置页要显示的全部配置。
//
// 只给可改的和有助于排障的，不给全量结构体——配置里有一堆内部时长阈值，
// 摆到界面上只会让人以为那些也该调。
func (s *Server) runtimeConfig(w http.ResponseWriter, _ *http.Request) {
	c := s.Settings.Get()
	h := s.Sup.Health()

	pyOK, pyDetail := probePath(c.Comfy.Python)
	mainOK, mainDetail := probePath(c.Comfy.MainPy)

	writeJSON(w, http.StatusOK, map[string]any{
		"config_path": s.Settings.Path(),
		"imagen": map[string]any{
			"proxy":                c.Imagen.Proxy,
			"openai_base_url":      c.Imagen.OpenAIBaseURL,
			"flatten":              c.Imagen.Flatten,
			"refine_model":         c.Imagen.RefineModel,
			"refine_model_default": imagen.DefaultRefineModel,
		},
		"comfy": map[string]any{
			"mode":            string(c.Comfy.Mode),
			"base_url":        c.Comfy.BaseURL,
			"python":          c.Comfy.Python,
			"main_py":         c.Comfy.MainPy,
			"extra_args":      c.Comfy.ExtraArgs,
			"auto_restart":    c.Comfy.AutoRestart,
			"reserve_vram_gb": c.Comfy.ReserveVRAM,
			// 路径存不存在直接告诉用户，别等启动失败才发现打错一个字母
			"python_exists":  pyOK,
			"python_detail":  pyDetail,
			"main_py_exists": mainOK,
			"main_py_detail": mainDetail,
			"alive":          h.Alive,
			"user_stopped":   s.Sup.UserStopped(),
		},
		"paths": map[string]any{
			"output":    c.OutputDir,
			"data":      c.DataDir,
			"workflows": c.WorkflowsDir,
			"root":      workingDir(),
		},
		// 环境变量里的代理如实报出来：配置留空时走的就是它，
		// 而"我明明设了 HTTPS_PROXY 却不生效"是很常见的困惑。
		"env_proxy": firstNonEmpty(os.Getenv("HTTPS_PROXY"), os.Getenv("https_proxy"),
			os.Getenv("HTTP_PROXY"), os.Getenv("http_proxy")),
		"os": runtime.GOOS,
	})
}

// updateConfig 应用设置页的改动。
func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) {
	var p settings.Patch
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	if p.Proxy != nil {
		if err := imagen.CheckProxy(*p.Proxy); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	needRestart, err := s.Settings.Apply(p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"need_restart": needRestart,
	})
}

// probePath 报告一个路径存不存在，并说清是文件还是目录。
func probePath(p string) (bool, string) {
	if p == "" {
		return false, "未配置"
	}
	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return false, "文件不存在"
		}
		return false, err.Error()
	}
	if fi.IsDir() {
		return false, "这是个目录，需要指到具体文件"
	}
	return true, ""
}

func workingDir() string {
	d, err := os.Getwd()
	if err != nil {
		return ""
	}
	abs, err := filepath.Abs(d)
	if err != nil {
		return d
	}
	return abs
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
