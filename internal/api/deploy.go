package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mengye/dreamtexture/internal/deploy"
	"github.com/mengye/dreamtexture/internal/settings"
)

// deployStatus 返回部署进度，供界面轮询。
//
// 顺带给出默认参数：目标目录、可复用的模型库路径。让用户从一个已经填好的
// 表单开始，而不是对着几个空框猜该填什么。
func (s *Server) deployStatus(w http.ResponseWriter, _ *http.Request) {
	c := s.Settings.Get()
	dir := filepath.Join(workingDir(), "runtime")
	st := s.Deploy.Status()

	// 内存里没有部署记录时（后端重启过），去默认目录看一眼——环境可能早就
	// 装好了，只是那次部署的状态随进程一起没了。
	if st.Python == "" && !st.Running {
		if py, main, ok := deploy.Detect(dir); ok {
			st.Python, st.MainPy = py, main
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status": st,
		"defaults": map[string]any{
			"dir":        dir,
			"py_version": "3.13",
			"torch":      "cu130",
			"mirror":     true,
			// 已有的模型库直接复用，几十 GB 不用再下一遍。
			"model_base_path": guessModelBase(s),
		},
		"current": map[string]any{
			"python":  c.Comfy.Python,
			"main_py": c.Comfy.MainPy,
		},
	})
}

func (s *Server) deployStart(w http.ResponseWriter, r *http.Request) {
	var opt deploy.Options
	if err := json.NewDecoder(r.Body).Decode(&opt); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	// 代理由后端按设置填，不收客户端传来的：这个值会被塞进子进程的环境变量，
	// 让请求方指定等于给了它一个"让服务器把流量导去哪儿"的开关。
	opt.Proxy = s.Settings.Get().Imagen.Proxy

	// 部署要跑十几分钟，绝不能挂在这个 HTTP 请求的生命周期上——
	// 浏览器一刷新、连接一断，ctx 就取消了，装到一半的环境比没装还麻烦。
	if err := s.Deploy.Start(context.WithoutCancel(r.Context()), opt); err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": s.Deploy.Status()})
}

func (s *Server) deployCancel(w http.ResponseWriter, _ *http.Request) {
	s.Deploy.Cancel()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// deployApply 把刚装好的环境写进配置。
//
// 做成单独一步而不是部署完自动切：切过去要重启后端，正在跑的任务会断。
// 什么时候切该由用户定。
func (s *Server) deployApply(w http.ResponseWriter, _ *http.Request) {
	st := s.Deploy.Status()
	if st.Python == "" || st.MainPy == "" {
		// 同 deployStatus：内存里没记录不代表磁盘上没装。
		if py, main, ok := deploy.Detect(filepath.Join(workingDir(), "runtime")); ok {
			st.Python, st.MainPy = py, main
		} else {
			writeErr(w, http.StatusBadRequest, "还没有装好的环境可用；先跑一次部署")
			return
		}
	}
	// 能写相对路径就写相对：装出来的运行时就在程序目录下，写死绝对路径的话
	// 整个 DreamTexture 文件夹换个盘符就失效了。
	mode := "managed"
	python, mainPy := portablePath(st.Python), portablePath(st.MainPy)
	needRestart, err := s.Settings.Apply(settings.Patch{
		ComfyPython: &python,
		ComfyMainPy: &mainPy,
		ComfyMode:   &mode,
	})
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":           true,
		"need_restart": needRestart,
		"python":       python,
		"main_py":      mainPy,
	})
}

// portablePath 把程序目录之内的路径转成相对路径，之外的原样返回。
//
// 只在确实位于程序目录之下时才转：跑出 ".." 的相对路径既不好读，
// 也没有可移植性可言，那种情况老实写绝对路径。
func portablePath(p string) string {
	if p == "" {
		return p
	}
	root := workingDir()
	if root == "" {
		return p
	}
	rel, err := filepath.Rel(root, p)
	if err != nil || strings.HasPrefix(rel, "..") {
		return p
	}
	return rel
}

// guessModelBase 猜一个可复用的模型库目录。
//
// 优先看现有 ComfyUI 的 extra_model_paths.yaml —— 用户已经在那儿声明过
// 模型放哪了，照抄比让他再填一遍靠谱。
func guessModelBase(s *Server) string {
	c := s.Settings.Get()
	if c.Comfy.MainPy == "" {
		return ""
	}
	y := filepath.Join(filepath.Dir(c.Comfy.MainPy), "extra_model_paths.yaml")
	b, err := os.ReadFile(y)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		const key = "base_path:"
		i := strings.Index(line, key)
		if i < 0 {
			continue
		}
		v := strings.TrimSpace(line[i+len(key):])
		if v == "" {
			continue
		}
		if fi, err := os.Stat(v); err == nil && fi.IsDir() {
			return v
		}
	}
	return ""
}
