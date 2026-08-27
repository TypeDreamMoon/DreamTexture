// Package web 把前端构建产物嵌进二进制，实现"一个 exe 就能跑"。
//
// dist 由 `pnpm build` 生成（web/vite.config.ts 的 outDir）。为了让仓库在没有
// 构建过前端时也能编译，这里放了一个占位 index.html —— 真正构建过之后
// go:embed 会连同 assets 一起嵌入。
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var dist embed.FS

// Handler 返回前端的静态文件处理器。
//
// 前端用 hash 路由，所有路径都由同一个 index.html 承载，因此找不到的文件
// 一律回落到 index.html，而不是 404。
func Handler() http.Handler {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		return http.NotFoundHandler()
	}
	files := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		if _, err := fs.Stat(sub, p); err != nil {
			r = r.Clone(r.Context())
			r.URL.Path = "/"
		}
		// 带内容哈希的资源可以长期缓存；index.html 不能，否则升级后打不开新版。
		if strings.HasPrefix(p, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		files.ServeHTTP(w, r)
	})
}

// Built 表示二进制里是否带着真正构建过的前端。
func Built() bool {
	b, err := dist.ReadFile("dist/index.html")
	return err == nil && !strings.Contains(string(b), "dt-placeholder")
}
