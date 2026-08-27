package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mengye/dreamtexture/internal/picture"
	"github.com/mengye/dreamtexture/internal/store"
)

func (s *Server) listPictures(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	items, err := s.Store.SearchPictures(store.PictureQuery{
		Q:     q.Get("q"),
		Fav:   q.Get("fav") == "1" || q.Get("fav") == "true",
		Limit: limit,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pictures": items})
}

// getPicture 返回索引记录与磁盘上的元信息。
//
// 两者都给：索引里有收藏这类可变状态，元信息里有出处与用量这些落盘时就定死的
// 东西。以磁盘为准的那部分读文件，可变的那部分读库。
func (s *Server) getPicture(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, err := s.Store.GetPicture(id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rec == nil {
		writeErr(w, http.StatusNotFound, "图片 "+id+" 不存在")
		return
	}
	meta, err := picture.Read(s.OutputDir, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "读取元信息失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"picture": rec, "meta": meta})
}

func (s *Server) pictureFile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := picture.Path(s.OutputDir, id)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	// 内容带哈希的资源可以长缓存，但图片是按 id 取的、内容不会变，
	// 给个中等缓存即可；删了重生成会换新 id。
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, p)
}

func (s *Server) favoritePicture(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Favorite bool `json:"favorite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "请求体不是合法 JSON: "+err.Error())
		return
	}
	if err := s.Store.SetPictureFavorite(chi.URLParam(r, "id"), body.Favorite); err != nil {
		writeErr(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// deletePicture 删图。文件先删，索引后删。
//
// 顺序是有讲究的：反过来的话，文件删失败会留下一张检索不到、也删不掉的孤儿图；
// 而先删文件、索引删失败，下次列表里点开会 404，重试删除还能收拾干净。
func (s *Server) deletePicture(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := picture.Remove(s.OutputDir, id); err != nil {
		writeErr(w, http.StatusInternalServerError, "删除文件失败: "+err.Error())
		return
	}
	if err := s.Store.DeletePicture(id); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
