package api

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mengye/dreamtexture/internal/nvidia"
)

// vramWarn 是"该提醒一声"的空闲显存下限（MB）。
//
// 取值来由：SDXL 出一张 1024 图，UNet 常驻约 5 GB，解码那一下还要再要一块
// 上 GB 的连续显存。低于 3 GB 时基本注定要么换出模型重装载（慢），
// 要么触发驱动回退到内存（更慢，而且不报错）。
const vramWarn = 3072

// checkVRAM 报告显卡上还剩多少显存，以及是谁占着。
//
// 单独列一条检查，是因为这台机器的典型症状——GPU 占用 99%、显存打满、
// 进度条不动——从任何一条现成的检查里都看不出来：ComfyUI 活着、模型齐、
// 节点在、目录可写，样样正常，唯独显存被别的程序拿走了。
func (s *Server) checkVRAM(r *http.Request) Check {
	c := Check{Key: "vram", Label: "显存"}
	h := s.Sup.Health()
	if !h.Alive {
		c.Status, c.Detail = "warn", "ComfyUI 未连接，读不到显存信息"
		return c
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	st, err := s.Sup.Client().SystemStats(ctx)
	if err != nil || len(st.Devices) == 0 {
		c.Status, c.Detail = "warn", "读取显存信息失败"
		return c
	}
	d := st.Devices[0]
	free := d.VRAMFree >> 20
	total := d.VRAMTotal >> 20
	c.Detail = fmt.Sprintf("%s：空闲 %d MB / 共 %d MB", d.Name, free, total)

	// 系统内存回退是"显存不够"会不会变成假死的分水岭，所以两种情况都要说。
	fb := nvidia.Query(s.Settings.Get().Comfy.Python)
	switch {
	case !fb.Supported:
		// 读不到就不提，别拿一句"未知"去占用户的注意力。
	case fb.Value == nvidia.ValueDeny:
		c.Detail += "；系统内存回退已关闭，显存不够会直接报错而不是卡住"
	default:
		c.Detail += "；**系统内存回退开着**"
	}

	// 回退开着才是"卡住不动"的成因，所以它单独构成一个警告——
	// 哪怕此刻显存还够，下一个任务撞上别的程序涨显存就会中招。
	fallbackOn := fb.Supported && fb.Value != nvidia.ValueDeny
	if free >= vramWarn && !fallbackOn {
		c.Status = "ok"
		return c
	}

	c.Status = "warn"
	c.Fix = "关掉别的吃显存的程序再生成；也可以在设置里调大「显存余量」，" +
		"让 ComfyUI 主动把模型换到内存，而不是等驱动去悄悄降级"
	if fallbackOn {
		c.Fix = "去 NVIDIA 控制面板 →「管理 3D 设置」→「CUDA - 系统内存回退策略」" +
			"改成「优先不使用系统内存回退」。开着的话显存一旦不够，驱动会悄悄改用" +
			"内存硬算——不报错、不结束、慢几十倍，看到的就是进度条卡住不动；" +
			"关掉之后显存不够会立刻报错，至少知道发生了什么。另外：" + c.Fix
	}
	// 把占着显存的大户点名列出来。用户看到"UnrealEditor"比看到一个数字有用得多。
	if hogs := gpuHogs(ctx); len(hogs) > 0 {
		c.Items = hogs
	}
	return c
}

// gpuHogs 列出正在用显卡的其他程序。
//
// Windows 的 WDDM 驱动模型下 nvidia-smi 查不到每个进程占了多少显存（一律 N/A），
// 所以这里只报名字，不编造数值。名字足够了：看见"UnrealEditor"就知道该关谁。
func gpuHogs(ctx context.Context) []string {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "nvidia-smi",
		"--query-compute-apps=pid,process_name", "--format=csv,noheader").Output()
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ",", 2)
		if len(parts) != 2 {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		name := strings.TrimSpace(parts[1])
		if i := strings.LastIndexAny(name, `\/`); i >= 0 {
			name = name[i+1:]
		}
		if name == "" || ignoredHog(name) {
			continue
		}
		seen[fmt.Sprintf("%s (pid %d)", name, pid)] = true
	}

	list := make([]string, 0, len(seen))
	for k := range seen {
		list = append(list, k)
	}
	sort.Strings(list)
	// 桌面上零碎的 GPU 进程能有几十个，全列出来反而找不到重点。
	if len(list) > 8 {
		list = append(list[:8], fmt.Sprintf("…另有 %d 个", len(list)-8))
	}
	return list
}

// ignoredHog 过掉系统与浏览器内核那一堆各占几十兆的小进程。
//
// 只按名字过滤，不按占用量——WDDM 下根本拿不到占用量（见 gpuHogs）。
var desktopNoise = []string{
	"dwm", "explorer", "shellhost", "shellexperiencehost", "startmenuexperiencehost",
	"searchhost", "textinputhost", "applicationframehost", "systemsettings",
	"lockapp", "taskmgr", "sihost", "widgets",
}

func ignoredHog(name string) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	n := strings.ToLower(strings.TrimSuffix(name, ".exe"))
	for _, x := range desktopNoise {
		if n == x {
			return true
		}
	}
	return false
}
