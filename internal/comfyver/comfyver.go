// Package comfyver 管 ComfyUI 本体的版本：看当前装的是哪个、能切到哪些、切过去。
//
// 只做 git 那一层的事。切版本之后还要重装依赖（ComfyUI 的 requirements.txt
// 在版本之间是会变的），那部分在 internal/deploy 里——它本来就有装依赖和
// 兜住 CUDA torch 的那套步骤，不该在这儿抄一遍。
package comfyver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Kind 区分两个来源。对应界面上的两个页签。
const (
	// KindStable 是打了 tag 的发布版。
	KindStable = "stable"
	// KindDev 是主分支上的提交，还没发版。
	KindDev = "dev"
)

// Version 是一个可切换的目标。
type Version struct {
	// Ref 是完整提交号，切换时用它。
	Ref string `json:"ref"`
	// Short 是界面上显示的短号。
	Short string `json:"short"`
	// Name 是版本名（tag）或提交标题（dev）。
	Name string    `json:"name"`
	Date time.Time `json:"date"`
	Kind string    `json:"kind"`
	// Current 表示当前就停在这个提交上。
	Current bool `json:"current"`
}

// Status 是当前仓库的状态。
type Status struct {
	// Available 为 false 时 Reason 说明为什么管不了（不是 git 仓库、没装 git…）。
	Available bool   `json:"available"`
	Reason    string `json:"reason,omitempty"`

	Dir    string `json:"dir,omitempty"`
	Remote string `json:"remote,omitempty"`
	Branch string `json:"branch,omitempty"`
	// Ref/Short/Name/Date 描述当前所在的提交。
	Ref   string    `json:"ref,omitempty"`
	Short string    `json:"short,omitempty"`
	Name  string    `json:"name,omitempty"`
	Date  time.Time `json:"date,omitempty"`

	// Shallow 为 true 时历史不全，列不出版本，得先补一次。
	Shallow bool `json:"shallow"`
	// Dirty 表示工作区有改动。切版本会失败或者把改动冲掉，所以要拦住。
	Dirty bool `json:"dirty"`
	// DirtyFiles 列几个改动的文件，让用户知道是什么挡住了。
	DirtyFiles []string `json:"dirty_files,omitempty"`
}

// Repo 是一个 ComfyUI 的 git 仓库。
type Repo struct{ dir string }

// Open 由 main.py 的路径定位仓库——配置里存的就是它。
func Open(mainPy string) (*Repo, error) {
	if strings.TrimSpace(mainPy) == "" {
		return nil, fmt.Errorf("还没有配置 ComfyUI 主程序")
	}
	dir := filepath.Dir(mainPy)
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return nil, fmt.Errorf("%s 不是 git 仓库，管不了版本。"+
			"整合包如果是解压出来的就没有版本信息，用设置页的一键部署装一份就有了", dir)
	}
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("找不到 git，装一个再来")
	}
	return &Repo{dir: dir}, nil
}

func (r *Repo) git(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = r.dir
	// 别让 git 弹凭据框把我们卡死——拉公开仓库本来也不需要。
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never")
	out, err := cmd.CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		return s, fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, s)
	}
	return s, nil
}

// Status 读当前状态。
func (r *Repo) Status(ctx context.Context) Status {
	st := Status{Available: true, Dir: r.dir}

	if v, err := r.git(ctx, "log", "-1", "--format=%H%x1f%h%x1f%cI%x1f%s"); err == nil {
		f := strings.Split(v, "\x1f")
		if len(f) == 4 {
			st.Ref, st.Short, st.Name = f[0], f[1], f[3]
			st.Date, _ = time.Parse(time.RFC3339, f[2])
		}
	}
	// 停在 tag 上就用 tag 名，比提交标题好认。
	if t, err := r.git(ctx, "describe", "--tags", "--exact-match"); err == nil && t != "" {
		st.Name = t
	}
	if b, err := r.git(ctx, "rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		st.Branch = b
	}
	if u, err := r.git(ctx, "remote", "get-url", "origin"); err == nil {
		st.Remote = u
	}
	if s, err := r.git(ctx, "rev-parse", "--is-shallow-repository"); err == nil {
		st.Shallow = s == "true"
	}
	if p, err := r.git(ctx, "status", "--porcelain"); err == nil && p != "" {
		st.Dirty = true
		for i, line := range strings.Split(p, "\n") {
			if i >= 8 {
				st.DirtyFiles = append(st.DirtyFiles, "…")
				break
			}
			st.DirtyFiles = append(st.DirtyFiles, strings.TrimSpace(line))
		}
	}
	return st
}

// Fetch 把远端的历史与 tag 拉全。
//
// 一键部署用的是 --filter=blob:none 的部分克隆，历史是全的；但如果这份仓库是
// 别处来的浅克隆（--depth 1），就得先补齐，否则一个版本都列不出来。
// 补的时候同样带 blob 过滤，只要提交图不要老文件——ComfyUI 的完整历史几百 MB，
// 而我们只是要一份版本清单。
func (r *Repo) Fetch(ctx context.Context) error {
	st := r.Status(ctx)
	if st.Shallow {
		if _, err := r.git(ctx, "fetch", "--filter=blob:none", "--unshallow", "--tags"); err == nil {
			return nil
		}
		// 服务端不支持部分克隆就老实全拉。
		if _, err := r.git(ctx, "fetch", "--unshallow", "--tags"); err != nil {
			return err
		}
		return nil
	}
	_, err := r.git(ctx, "fetch", "--filter=blob:none", "--tags", "--prune")
	if err != nil {
		_, err = r.git(ctx, "fetch", "--tags", "--prune")
	}
	return err
}

// devLimit 是开发版列多少条。
//
// 主分支上的提交每天都有好几个，全列出来没人看得完；用户真要某个具体提交，
// 那多半是照着别处给的提交号来的，会走"手动输入"那条路。
const devLimit = 60

// List 列出可切换的版本。
func (r *Repo) List(ctx context.Context, kind string) ([]Version, error) {
	cur, _ := r.git(ctx, "rev-parse", "HEAD")

	if kind == KindDev {
		// 优先看远端分支：本地 HEAD 可能已经切到某个老版本上了，
		// 拿它去列"最新的提交"会漏掉后面的。
		src := "origin/HEAD"
		if _, err := r.git(ctx, "rev-parse", "--verify", "origin/HEAD"); err != nil {
			src = "origin/master"
			if _, err := r.git(ctx, "rev-parse", "--verify", src); err != nil {
				src = "HEAD"
			}
		}
		out, err := r.git(ctx, "log", src,
			fmt.Sprintf("-n%d", devLimit), "--format=%H%x1f%h%x1f%cI%x1f%s")
		if err != nil {
			return nil, err
		}
		list := parseLines(out, KindDev, cur)
		// git log 的默认顺序是拓扑序，遇到合并会把时间穿插开。界面上"最新的在
		// 上面"是硬预期，所以自己按日期排一遍。
		sort.SliceStable(list, func(i, j int) bool { return list[i].Date.After(list[j].Date) })
		return list, nil
	}

	out, err := r.git(ctx, "for-each-ref", "--sort=-creatordate",
		"--format=%(objectname)%1f%(objectname:short)%1f%(creatordate:iso-strict)%1f%(refname:short)",
		"refs/tags")
	if err != nil {
		return nil, err
	}
	// tag 指向的是 commit 还是 tag 对象，取决于是不是附注标签；
	// 统一解引用一次，否则"当前"那一列永远对不上。
	list := parseLines(out, KindStable, cur)
	for i := range list {
		if c, err := r.git(ctx, "rev-list", "-n1", list[i].Ref); err == nil && c != "" {
			list[i].Ref = c
			list[i].Current = c == cur
			if len(c) >= 7 {
				list[i].Short = c[:7]
			}
		}
	}
	return list, nil
}

func parseLines(out, kind, cur string) []Version {
	var list []Version
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(strings.TrimSpace(line), "\x1f")
		if len(f) != 4 || f[0] == "" {
			continue
		}
		t, _ := time.Parse(time.RFC3339, f[2])
		list = append(list, Version{
			Ref: f[0], Short: f[1], Date: t, Name: f[3],
			Kind: kind, Current: f[0] == cur,
		})
	}
	return list
}

// Resolve 把用户给的东西（tag、短号、完整提交号）解析成完整提交号。
func (r *Repo) Resolve(ctx context.Context, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("没有指定版本")
	}
	out, err := r.git(ctx, "rev-list", "-n1", ref)
	if err != nil || out == "" {
		return "", fmt.Errorf("找不到版本 %q。可能是历史还没拉全，先点一下刷新", ref)
	}
	return out, nil
}

// Checkout 切到某个提交。
//
// 用游离头指针（detached HEAD）而不是建分支：我们要的是"钉在这个版本上"，
// 建一堆本地分支只会让用户之后自己用 git 时一头雾水。
func (r *Repo) Checkout(ctx context.Context, ref string) error {
	if st := r.Status(ctx); st.Dirty {
		return fmt.Errorf("ComfyUI 目录里有未提交的改动，切版本会把它们冲掉。"+
			"先自己处理一下：%s", strings.Join(st.DirtyFiles, "、"))
	}
	_, err := r.git(ctx, "checkout", "--detach", ref)
	return err
}

// Dir 返回仓库目录，给调用方拼 requirements.txt 之类的路径用。
func (r *Repo) Dir() string { return r.dir }
