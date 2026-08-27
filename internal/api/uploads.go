package api

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// maxUploadBytes 是参考图的大小上限。
//
// 参考图最终会被缩到 1K 附近参与生成，几十 MB 的原图没有意义，
// 早点拒绝比让它穿过整条管线再失败要好。
const maxUploadBytes = 32 << 20

var allowedImageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".bmp": true,
}

// upload 接收参考图并转存到 ComfyUI 的 input 目录，返回可填进 LoadImage 的文件名。
//
// 图片不落在 DreamTexture 自己的目录：ComfyUI 只认它自己 input 目录下的文件，
// 转一手比让两边共享目录更少踩坑（尤其 attach 模式下 ComfyUI 可能在另一台机器上）。
func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
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
	if header.Size > maxUploadBytes {
		writeErr(w, http.StatusRequestEntityTooLarge,
			fmt.Sprintf("图片 %.1f MB 超过上限 %d MB", float64(header.Size)/(1<<20), maxUploadBytes>>20))
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

	// 文件名由后端定，不用客户端传来的：避免路径穿越，也避免同名互相覆盖。
	name := "dt_ref_" + randToken(10) + ext
	stored, err := s.Sup.Client().UploadImage(r.Context(), name, data)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "转存到 ComfyUI 失败: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     stored,
		"bytes":    len(data),
		"original": header.Filename,
	})
}
