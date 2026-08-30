package model

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mengye/dreamtexture/internal/workflow"
)

type DownloadState string

const (
	DownloadQueued   DownloadState = "queued"
	DownloadRunning  DownloadState = "running"
	DownloadDone     DownloadState = "done"
	DownloadFailed   DownloadState = "failed"
	DownloadCanceled DownloadState = "canceled"
)

type Download struct {
	ID     string        `json:"id"`
	File   string        `json:"file"`
	Dir    string        `json:"dir"`
	Target string        `json:"target"`
	Source string        `json:"source"`
	State  DownloadState `json:"state"`
	// Received / Total 单位是字节；Total 为 0 表示服务端没给长度。
	Received  int64      `json:"received"`
	Total     int64      `json:"total"`
	Error     string     `json:"error,omitempty"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

func (d *Download) Percent() float64 {
	if d.Total <= 0 {
		return 0
	}
	return float64(d.Received) / float64(d.Total)
}

// Credentials 提供下载所需的凭据。
//
// 凭据由用户自己在设置页填写、存在本地，后端只在发起请求时取用，
// 从不在任何接口里回传。
type Credentials interface {
	// Header 返回某个下载源需要的鉴权头，没有则返回空。
	Header(sourceURL string) (name, value string)
}

// Downloader 串行下载模型文件。
//
// 串行是刻意的：模型动辄几个 GB，并行下载既抢带宽又让进度难以判断，
// 而且同时写多个大文件对机械盘很不友好。
type Downloader struct {
	creds Credentials
	log   *slog.Logger
	emit  func(*Download)

	queue chan string

	mu   sync.RWMutex
	list map[string]*Download
	// cancels 保存运行中任务的取消函数。
	cancels map[string]context.CancelFunc
	seq     int
}

func NewDownloader(creds Credentials, log *slog.Logger, emit func(*Download)) *Downloader {
	if emit == nil {
		emit = func(*Download) {}
	}
	return &Downloader{
		creds: creds, log: log, emit: emit,
		queue:   make(chan string, 64),
		list:    map[string]*Download{},
		cancels: map[string]context.CancelFunc{},
	}
}

func (d *Downloader) List() []*Download {
	d.mu.RLock()
	defer d.mu.RUnlock()
	out := make([]*Download, 0, len(d.list))
	for _, x := range d.list {
		cp := *x
		out = append(out, &cp)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartedAt.After(out[j].StartedAt) })
	return out
}

// Enqueue 把一个模型加入下载队列。
func (d *Downloader) Enqueue(req Requirement) (*Download, error) {
	url := req.DownloadURL
	if url == "" {
		// 没有直链就别硬下：模型主页下下来是 HTML，存成 .safetensors 只会让
		// ComfyUI 加载时报一堆看不懂的错。
		return nil, fmt.Errorf("这个模型没有可直接下载的地址，请到 %s 手动下载后放进 %s",
			req.Source, filepath.Dir(req.Target))
	}
	if req.Target == "" {
		return nil, errors.New("无法确定写入目录，请检查 ComfyUI 的模型路径配置")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return nil, fmt.Errorf("下载地址不是 http(s)：%s", url)
	}

	d.mu.Lock()
	for _, x := range d.list {
		if x.Target == req.Target && (x.State == DownloadQueued || x.State == DownloadRunning) {
			d.mu.Unlock()
			return nil, fmt.Errorf("%s 已在下载队列中", req.File)
		}
	}
	d.seq++
	dl := &Download{
		ID:     fmt.Sprintf("dl_%d_%d", time.Now().Unix(), d.seq),
		File:   req.File,
		Dir:    req.Dir,
		Target: req.Target,
		Source: url,
		State:  DownloadQueued,
		Total:  req.SizeBytes,
	}
	d.list[dl.ID] = dl
	d.mu.Unlock()

	select {
	case d.queue <- dl.ID:
	default:
		d.finish(dl, DownloadFailed, errors.New("下载队列已满"))
		return dl, errors.New("下载队列已满")
	}
	d.emit(dl)
	return dl, nil
}

// EnqueueDirect 下载一个不来自工作流声明的模型（模型库里挑的）。
//
// 与 Enqueue 分开是刻意的：那条路径只允许下载模板登记过的模型，地址不由调用方
// 指定；这条路径的地址来自模型库检索结果，调用方仍不能随便传 URL——
// API 层会先在检索结果里核对过才调这里。
func (d *Downloader) EnqueueDirect(name, dir, target, source string, size int64) (*Download, error) {
	return d.Enqueue(Requirement{
		ModelRequirement: workflow.ModelRequirement{
			File: name, Dir: dir, DownloadURL: source, Source: source, SizeBytes: size,
		},
		Target: target,
	})
}

func (d *Downloader) Cancel(id string) error {
	d.mu.Lock()
	dl, ok := d.list[id]
	if !ok {
		d.mu.Unlock()
		return fmt.Errorf("下载任务 %s 不存在", id)
	}
	cancel := d.cancels[id]
	if dl.State == DownloadQueued {
		dl.State = DownloadCanceled
		now := time.Now()
		dl.EndedAt = &now
	}
	d.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	d.emit(dl)
	return nil
}

// Run 是下载 worker，阻塞到 ctx 取消。
func (d *Downloader) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case id := <-d.queue:
			d.runOne(ctx, id)
		}
	}
}

func (d *Downloader) runOne(parent context.Context, id string) {
	d.mu.Lock()
	dl, ok := d.list[id]
	if !ok || dl.State != DownloadQueued {
		d.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	d.cancels[id] = cancel
	dl.State, dl.StartedAt = DownloadRunning, time.Now()
	d.mu.Unlock()
	defer func() {
		cancel()
		d.mu.Lock()
		delete(d.cancels, id)
		d.mu.Unlock()
	}()

	d.emit(dl)
	if err := d.fetch(ctx, dl); err != nil {
		if ctx.Err() != nil && parent.Err() == nil {
			d.finish(dl, DownloadCanceled, errors.New("已取消"))
		} else {
			d.finish(dl, DownloadFailed, err)
		}
		return
	}
	d.finish(dl, DownloadDone, nil)
}

func (d *Downloader) fetch(ctx context.Context, dl *Download) error {
	if err := os.MkdirAll(filepath.Dir(dl.Target), 0o755); err != nil {
		return err
	}
	// 先写 .part，完整了再改名：中断留下的半截文件不会被 ComfyUI 当成可用模型。
	part := dl.Target + ".part"

	var offset int64
	if fi, err := os.Stat(part); err == nil {
		offset = fi.Size()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dl.Source, nil)
	if err != nil {
		return err
	}
	if name, value := d.creds.Header(dl.Source); name != "" {
		req.Header.Set(name, value)
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}

	client := &http.Client{Timeout: 0} // 大文件，不设整体超时
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		offset = 0 // 服务端不支持续传，从头来
	case http.StatusPartialContent:
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("下载被拒绝（HTTP %d）：这个来源需要登录凭据，请在设置页填写对应的访问令牌，或自行在浏览器中下载后放到 %s",
			resp.StatusCode, filepath.Dir(dl.Target))
	case http.StatusRequestedRangeNotSatisfiable:
		// 已经下完了。
		_ = os.Rename(part, dl.Target)
		return nil
	default:
		return fmt.Errorf("下载失败：HTTP %d", resp.StatusCode)
	}

	// 有些站点不返 401，而是 200 一张登录页。存成 .safetensors 之后 ComfyUI
	// 会在加载时报一堆看不懂的错，不如在这里就拦下来。
	if ct := resp.Header.Get("Content-Type"); strings.HasPrefix(ct, "text/html") {
		return fmt.Errorf("下载地址返回的是网页而不是模型文件（可能需要先登录）；请到 %s 手动下载后放进 %s",
			dl.Source, filepath.Dir(dl.Target))
	}

	if resp.ContentLength > 0 {
		d.update(dl, offset, offset+resp.ContentLength)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return err
	}

	received := offset
	buf := make([]byte, 1<<20)
	lastEmit := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				return werr
			}
			received += int64(n)
			// 一秒一次进度就够了，再密只是刷屏。
			if time.Since(lastEmit) > time.Second {
				d.update(dl, received, dl.Total)
				lastEmit = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	d.update(dl, received, dl.Total)
	return os.Rename(part, dl.Target)
}

func (d *Downloader) update(dl *Download, received, total int64) {
	d.mu.Lock()
	dl.Received = received
	if total > 0 {
		dl.Total = total
	}
	d.mu.Unlock()
	d.emit(dl)
}

func (d *Downloader) finish(dl *Download, state DownloadState, err error) {
	d.mu.Lock()
	now := time.Now()
	dl.State, dl.EndedAt = state, &now
	if err != nil {
		dl.Error = err.Error()
	}
	d.mu.Unlock()
	if err != nil && state == DownloadFailed {
		d.log.Warn("模型下载失败", "文件", dl.File, "err", err)
	}
	d.emit(dl)
}
