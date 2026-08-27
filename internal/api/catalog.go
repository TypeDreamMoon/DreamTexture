package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mengye/dreamtexture/internal/catalog"
)

// browseModels 在模型库里检索。
func (s *Server) browseModels(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	entries, warns, err := s.Catalog.Search(r.Context(), catalog.Query{
		Text:   q.Get("q"),
		Kind:   q.Get("kind"),
		Source: q.Get("source"),
		Limit:  limit,
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	// 标出已经装了的，免得用户重复下几个 GB。
	if inv := s.Models.Cached(); inv != nil {
		have := map[string]bool{}
		for _, f := range inv.Files {
			have[strings.ToLower(f.Name)] = true
		}
		for i := range entries {
			if have[strings.ToLower(entries[i].Filename)] {
				entries[i].Installed = true
			}
		}
	}
	if entries == nil {
		entries = []catalog.Entry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "warnings": warns})
}

// downloadFromCatalog 下载模型库里的一条模型。
//
// 只接受来源与 id：下载地址从刚才的检索结果里重新取，不由客户端指定，
// 免得这个接口变成任人指使的下载器。
func (s *Server) downloadFromCatalog(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Source string `json:"source"`
		ID     string `json:"id"`
		Query  string `json:"query"`
		Kind   string `json:"kind"`
		// Dir 允许用户改目标目录——启发式判断不一定对，猜错会让模型静默失效。
		Dir string `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}

	entries, _, err := s.Catalog.Search(r.Context(), catalog.Query{
		Text: body.Query, Kind: body.Kind, Source: body.Source, Limit: 100,
	})
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	var hit *catalog.Entry
	for i := range entries {
		if entries[i].Source == body.Source && entries[i].ID == body.ID {
			hit = &entries[i]
			break
		}
	}
	if hit == nil {
		writeErr(w, http.StatusNotFound, "在检索结果里找不到这条模型，请重新搜索后再下载")
		return
	}

	dir := strings.TrimSpace(body.Dir)
	if dir == "" {
		dir = hit.Dir
	}
	if dir == "" {
		writeErr(w, http.StatusBadRequest,
			"无法判断这个模型该放进哪个目录，请在界面上选一个")
		return
	}

	target, err := s.resolveModelTarget(r, dir, hit.Filename)
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	dl, err := s.Downloads.EnqueueDirect(hit.Filename, dir, target, hit.URL, hit.SizeBytes)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, dl)
}

// resolveModelTarget 把"目录名 + 文件名"解析成绝对路径。
//
// 目录的真实位置问 ComfyUI 要，不能硬编码——用户很可能通过
// extra_model_paths.yaml 把模型挂在别的盘上。
func (s *Server) resolveModelTarget(r *http.Request, dir, filename string) (string, error) {
	// dir 可能是 controlnet/SDXL 这种带子路径的形式。
	top, sub, _ := strings.Cut(filepath.ToSlash(dir), "/")
	folders, err := s.Sup.Client().ModelFolders(r.Context())
	if err != nil {
		return "", err
	}
	for _, f := range folders {
		if !strings.EqualFold(f.Name, top) || len(f.Paths) == 0 {
			continue
		}
		root := f.Paths[0]
		if len(f.Paths) > 1 {
			// 与模型盘点同样的取舍：优先用户通过 extra_model_paths 挂的目录，
			// ComfyUI 自带的 models/ 往往和程序同盘、余量紧张。
			root = f.Paths[1]
		}
		if sub != "" {
			return filepath.Join(root, filepath.FromSlash(sub), filename), nil
		}
		return filepath.Join(root, filename), nil
	}
	return "", &dirNotFound{dir: top}
}

type dirNotFound struct{ dir string }

func (e *dirNotFound) Error() string {
	return "ComfyUI 没有名为 " + e.dir + " 的模型目录"
}

// modelDirs 列出 ComfyUI 认得的模型目录，供界面上的目标目录选择。
func (s *Server) modelDirs(w http.ResponseWriter, r *http.Request) {
	folders, err := s.Sup.Client().ModelFolders(r.Context())
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	names := make([]string, 0, len(folders))
	for _, f := range folders {
		names = append(names, f.Name)
	}
	writeJSON(w, http.StatusOK, map[string]any{"dirs": names})
}
