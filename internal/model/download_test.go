package model

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/mengye/dreamtexture/internal/workflow"
)

type noCreds struct{}

func (noCreds) Header(string) (string, string) { return "", "" }

func newTestDownloader(t *testing.T) *Downloader {
	t.Helper()
	d := NewDownloader(noCreds{}, slog.New(slog.DiscardHandler), nil)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go d.Run(ctx)
	return d
}

// waitFor 轮询到下载进入终态，返回最终记录。
func waitFor(t *testing.T, d *Downloader, id string) *Download {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, x := range d.List() {
			if x.ID != id {
				continue
			}
			switch x.State {
			case DownloadDone, DownloadFailed, DownloadCanceled:
				return x
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("下载 %s 迟迟没有结束", id)
	return nil
}

func req(dir, file, url, target string) Requirement {
	return Requirement{
		ModelRequirement: workflow.ModelRequirement{
			Dir: dir, File: file, DownloadURL: url, Source: "https://example.test/page",
		},
		Target: target,
	}
}

func TestDownloadWritesFileAndRemovesPart(t *testing.T) {
	body := strings.Repeat("m", 5000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		io.WriteString(w, body)
	}))
	defer srv.Close()

	d := newTestDownloader(t)
	target := filepath.Join(t.TempDir(), "sub", "model.safetensors")
	dl, err := d.Enqueue(req("loras", "model.safetensors", srv.URL, target))
	if err != nil {
		t.Fatal(err)
	}
	got := waitFor(t, d, dl.ID)
	if got.State != DownloadDone {
		t.Fatalf("状态 = %s，期望 done（err=%s）", got.State, got.Error)
	}
	b, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != body {
		t.Fatalf("落盘内容长度 %d，期望 %d", len(b), len(body))
	}
	// .part 必须已经改名掉，否则半截文件会被 ComfyUI 当成可用模型。
	if _, err := os.Stat(target + ".part"); !os.IsNotExist(err) {
		t.Fatal(".part 临时文件没有清理")
	}
}

func TestDownloadResumesFromPartialFile(t *testing.T) {
	body := strings.Repeat("abcdefghij", 500) // 5000 字节
	var gotRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotRange = r.Header.Get("Range")
		w.Header().Set("Content-Type", "application/octet-stream")
		if gotRange == "" {
			io.WriteString(w, body)
			return
		}
		off, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(gotRange, "bytes="), "-"))
		w.Header().Set("Content-Range", "bytes "+strconv.Itoa(off)+"-4999/5000")
		w.WriteHeader(http.StatusPartialContent)
		io.WriteString(w, body[off:])
	}))
	defer srv.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "model.safetensors")
	// 先造一个下了一半的 .part
	if err := os.WriteFile(target+".part", []byte(body[:2000]), 0o644); err != nil {
		t.Fatal(err)
	}

	d := newTestDownloader(t)
	dl, err := d.Enqueue(req("loras", "model.safetensors", srv.URL, target))
	if err != nil {
		t.Fatal(err)
	}
	if got := waitFor(t, d, dl.ID); got.State != DownloadDone {
		t.Fatalf("状态 = %s，期望 done（err=%s）", got.State, got.Error)
	}
	if gotRange != "bytes=2000-" {
		t.Fatalf("Range 头 = %q，期望 bytes=2000-", gotRange)
	}
	b, _ := os.ReadFile(target)
	if string(b) != body {
		t.Fatalf("续传后内容不完整：%d 字节，期望 %d", len(b), len(body))
	}
}

func TestDownloadRejectsHTMLResponse(t *testing.T) {
	// 有些站点不返 401 而是 200 一张登录页，存成 .safetensors 之后
	// 只会让 ComfyUI 在加载时报看不懂的错。
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, "<html><body>请先登录</body></html>")
	}))
	defer srv.Close()

	d := newTestDownloader(t)
	target := filepath.Join(t.TempDir(), "model.safetensors")
	dl, _ := d.Enqueue(req("loras", "model.safetensors", srv.URL, target))
	got := waitFor(t, d, dl.ID)
	if got.State != DownloadFailed {
		t.Fatalf("状态 = %s，期望 failed", got.State)
	}
	if !strings.Contains(got.Error, "网页") {
		t.Fatalf("错误信息没有说明是网页：%s", got.Error)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("不该把网页落盘成模型文件")
	}
}

func TestDownloadUnauthorizedGivesActionableMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	d := newTestDownloader(t)
	target := filepath.Join(t.TempDir(), "model.safetensors")
	dl, _ := d.Enqueue(req("loras", "model.safetensors", srv.URL, target))
	got := waitFor(t, d, dl.ID)
	if got.State != DownloadFailed {
		t.Fatalf("状态 = %s，期望 failed", got.State)
	}
	// 报错要说清楚下一步怎么办，而不只是丢一个状态码。
	for _, want := range []string{"401", "令牌", filepath.Dir(target)} {
		if !strings.Contains(got.Error, want) {
			t.Fatalf("错误信息缺少 %q：%s", want, got.Error)
		}
	}
}

func TestEnqueueRefusesWithoutDirectURL(t *testing.T) {
	d := newTestDownloader(t)
	r := req("loras", "model.safetensors", "", filepath.Join(t.TempDir(), "model.safetensors"))
	_, err := d.Enqueue(r)
	if err == nil {
		t.Fatal("没有直链时应当拒绝入队，而不是去下模型主页")
	}
	if !strings.Contains(err.Error(), "手动下载") {
		t.Fatalf("拒绝理由不够明确：%v", err)
	}
}
