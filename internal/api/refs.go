package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mengye/dreamtexture/internal/picture"
	"github.com/mengye/dreamtexture/internal/store"
)

// refsDir 是参考图库的落盘位置。
//
// 放 data/ 而不是 output/：output 是"产出的东西"，参考图是输入素材，
// 混在一起会让"把 output 整个拷走"这件事变得不干净。
func (s *Server) refsDir() (string, error) {
	d := filepath.Join(s.DataDir, "references")
	return d, os.MkdirAll(d, 0o755)
}

func (s *Server) listRefs(w http.ResponseWriter, r *http.Request) {
	items, err := s.Store.ListRefs(0)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"refs": items})
}

// addRef 收一张上传的参考图存进库。
//
// 与既有的 POST /api/uploads 分工：那个是"传一张、这次用完就散"，
// 直接丢进 ComfyUI 的 input 目录；这个是要长期留着、能反复用的。
func (s *Server) addRef(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeErr(w, http.StatusBadRequest, "解析上传内容失败: "+err.Error())
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "请求里没有名为 image 的文件字段")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedImageExt[ext] {
		writeErr(w, http.StatusBadRequest,
			fmt.Sprintf("不支持的图片格式 %q，请用 PNG / JPG / WebP / BMP", ext))
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "读取图片失败: "+err.Error())
		return
	}
	if len(data) > maxUploadBytes {
		writeErr(w, http.StatusRequestEntityTooLarge, "图片超过大小上限")
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = strings.TrimSuffix(header.Filename, ext)
	}
	s.saveRef(w, r, data, ext, name, "upload")
}

// refFromPicture 把一张生成出来的图提升成参考图。
//
// 这条路是参考图库最有用的地方：出了张满意的，直接拿它当下一轮的参考，
// 不必先下载到本地再传回来。
func (s *Server) refFromPicture(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PictureID string `json:"picture_id"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	p, err := picture.Path(s.OutputDir, body.PictureID)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	data, err := os.ReadFile(p)
	if err != nil {
		writeErr(w, http.StatusNotFound, "读取图片失败: "+err.Error())
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		if rec, _ := s.Store.GetPicture(body.PictureID); rec != nil {
			name = rec.Name
		} else {
			name = body.PictureID
		}
	}
	s.saveRef(w, r, data, ".png", name, "picture:"+body.PictureID)
}

func (s *Server) saveRef(w http.ResponseWriter, _ *http.Request,
	data []byte, ext, name, origin string) {

	dir, err := s.refsDir()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	id := "ref_" + randToken(10)
	// 文件名由后端定，不用客户端传来的：避免路径穿越，也避免同名互相覆盖。
	fname := id + ext
	if err := os.WriteFile(filepath.Join(dir, fname), data, 0o644); err != nil {
		writeErr(w, http.StatusInternalServerError, "落盘失败: "+err.Error())
		return
	}

	rec := &store.Ref{
		ID: id, Name: name, File: fname, Bytes: int64(len(data)),
		Origin: origin, CreatedAt: time.Now(),
	}
	if cfg, _, err := image.DecodeConfig(bytes.NewReader(data)); err == nil {
		rec.Width, rec.Height = cfg.Width, cfg.Height
	}
	if err := s.Store.AddRef(rec); err != nil {
		_ = os.Remove(filepath.Join(dir, fname))
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ref": rec})
}

func (s *Server) refFile(w http.ResponseWriter, r *http.Request) {
	rec, err := s.Store.GetRef(chi.URLParam(r, "id"))
	if err != nil || rec == nil {
		writeErr(w, http.StatusNotFound, "参考图不存在")
		return
	}
	dir, err := s.refsDir()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, filepath.Join(dir, rec.File))
}

func (s *Server) renameRef(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	name := strings.TrimSpace(body.Name)
	if name == "" {
		writeErr(w, http.StatusBadRequest, "名字不能为空")
		return
	}
	if err := s.Store.RenameRef(chi.URLParam(r, "id"), name); err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) deleteRef(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, err := s.Store.GetRef(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rec != nil {
		if dir, err := s.refsDir(); err == nil {
			_ = os.Remove(filepath.Join(dir, rec.File))
		}
	}
	if err := s.Store.DeleteRef(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// useRef 把库里的参考图送进 ComfyUI 的 input 目录，返回可填进 LoadImage 的文件名。
//
// 每次用都重新上传一遍，而不是记住第一次的结果就一直用：ComfyUI 可能被清空、
// 重装、或者换成了另一个实例，那时候记着的文件名就是个失效的名字，而失败会
// 发生在提交工作流之后——报错是"LoadImage 找不到文件"，看不出跟参考图库有关。
func (s *Server) useRef(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, err := s.Store.GetRef(id)
	if err != nil || rec == nil {
		writeErr(w, http.StatusNotFound, "参考图不存在")
		return
	}
	dir, err := s.refsDir()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	data, err := os.ReadFile(filepath.Join(dir, rec.File))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取参考图失败: "+err.Error())
		return
	}
	stored, err := s.Sup.Client().UploadImage(r.Context(), rec.File, data)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "转存到 ComfyUI 失败: "+err.Error())
		return
	}
	_ = s.Store.SetRefComfyName(id, stored)
	writeJSON(w, http.StatusOK, map[string]any{"name": stored, "ref": rec})
}
