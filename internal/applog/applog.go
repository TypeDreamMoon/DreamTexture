// Package applog 把后端日志同时落到磁盘上。
//
// 为什么需要：日志本来只有两个去处——标准输出，和给控制台页用的内存环形缓冲。
// 两个都靠不住：
//
//   - 双击 exe 启动、或者以后被壳程序拉起来时，根本没有控制台，标准输出直接丢掉
//   - 内存缓冲只留最近 4000 行，而且进程一退就没了
//
// 而最需要日志的恰恰是"起不来"和"上次崩了"这两种情形——那时候界面打不开，
// 控制台页也就无从看起。
package applog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// maxSize 超过就轮转。8 MB 大约是几十万行，够看很久了。
const maxSize = 8 << 20

// Open 打开日志文件，返回文件与它的真实路径。
//
// dir 为空时放在可执行文件旁边的 logs/ 里——那是用户找得到的地方。
// 那儿写不了（装到 Program Files 之类）就退到用户缓存目录，而不是干脆不记。
func Open(dir string) (*os.File, string, error) {
	if dir == "" {
		dir = defaultDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		if alt, ok := fallbackDir(); ok && alt != dir {
			dir = alt
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, "", err
			}
		} else {
			return nil, "", err
		}
	}

	path := filepath.Join(dir, "dreamtexture.log")
	rotate(path)

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		// 目录能建但文件开不了（多半是权限），再试一次缓存目录。
		if alt, ok := fallbackDir(); ok && alt != dir {
			if err2 := os.MkdirAll(alt, 0o755); err2 == nil {
				path = filepath.Join(alt, "dreamtexture.log")
				rotate(path)
				if f2, err2 := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); err2 == nil {
					return f2, path, nil
				}
			}
		}
		return nil, "", err
	}
	return f, path, nil
}

// rotate 在文件太大时把它挪成 .1，只留一代。
//
// 只留一代是刻意的：日志是用来查"刚才发生了什么"的，不是留档。
// 攒一堆 .1 .2 .3 只会让人不知道该看哪个，还悄悄吃盘。
func rotate(path string) {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < maxSize {
		return
	}
	_ = os.Remove(path + ".1")
	_ = os.Rename(path, path+".1")
}

func defaultDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "logs"
	}
	return filepath.Join(filepath.Dir(exe), "logs")
}

func fallbackDir() (string, bool) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(base, "DreamTexture", "logs"), true
}

// Banner 往日志开头写一段分隔与环境信息。
//
// 一个文件里会攒好几次启动，没有分隔就分不清哪段是这次的。而版本、路径这些，
// 是别人把日志发过来时你第一个想知道的东西——与其到时候反复追问，
// 不如每次启动都写清楚。
func Banner(w io.Writer, version string, kv ...[2]string) {
	var b strings.Builder
	b.WriteString("\n" + strings.Repeat("=", 72) + "\n")
	fmt.Fprintf(&b, "DreamTexture %s  启动于 %s\n",
		version, time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "  %-12s %s/%s\n", "系统", runtime.GOOS, runtime.GOARCH)
	if exe, err := os.Executable(); err == nil {
		fmt.Fprintf(&b, "  %-12s %s\n", "程序", exe)
	}
	if wd, err := os.Getwd(); err == nil {
		fmt.Fprintf(&b, "  %-12s %s\n", "工作目录", wd)
	}
	for _, p := range kv {
		if p[1] != "" {
			fmt.Fprintf(&b, "  %-12s %s\n", p[0], p[1])
		}
	}
	b.WriteString(strings.Repeat("=", 72) + "\n")
	_, _ = io.WriteString(w, b.String())
}
