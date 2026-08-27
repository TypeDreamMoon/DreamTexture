package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Check 是一项环境自检结果。
type Check struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Status string `json:"status"` // ok | warn | fail
	Detail string `json:"detail"`
	// Fix 是可执行的修复动作标识，前端据此显示按钮；空表示只能人工处理。
	Fix string `json:"fix,omitempty"`
	// Items 是造成不通过的具体条目（缺失的节点类、模型名等）。
	Items []string `json:"items,omitempty"`
}

// checks 做一轮环境自检：ComfyUI 是否可用、工作流用到的节点是否齐全、
// 所需模型是否就位、输出目录是否可写。
//
// 这是从竞品尸检里学到的最一致的一课：装不上、跑不起来是同类工具最大的劝退点，
// 与其让用户点了生成才看到一句 "node not found"，不如一进门就告诉他缺什么。
func (s *Server) checks(w http.ResponseWriter, r *http.Request) {
	out := []Check{s.checkComfy(), s.checkVRAM(r), s.checkNodes(r), s.checkModels(r), s.checkOutputDir()}
	// 云端底图只在真有工作流用到它时才检查，免得给不用这功能的人添一条无关警告。
	if c, used := s.checkImagen(r); used {
		out = append(out, c)
	}
	worst := "ok"
	for _, c := range out {
		if c.Status == "fail" {
			worst = "fail"
			break
		}
		if c.Status == "warn" {
			worst = "warn"
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": worst, "checks": out})
}

func (s *Server) checkComfy() Check {
	h := s.Sup.Health()
	c := Check{Key: "comfy", Label: "ComfyUI 连接"}
	switch {
	case !h.Alive:
		c.Status, c.Detail = "fail", h.Reason
		if h.Mode == "managed" {
			c.Fix = "restart-comfy"
		} else {
			c.Detail += "（attach 模式：请自行启动 ComfyUI，或把配置改成 managed）"
		}
	case !h.Ready:
		c.Status, c.Detail = "warn", h.Reason
	default:
		c.Status = "ok"
		c.Detail = "ComfyUI " + h.Version
		if h.Device != "" {
			c.Detail += " · " + h.Device
		}
	}
	return c
}

// checkNodes 比对工作流实际用到的节点类与 ComfyUI 已注册的节点。
//
// 不去比对"节点包名"：包名和它注册出来的节点类并非一一对应，装了包也可能因为
// 依赖缺失而没注册成功。直接看节点类在不在，才是真的能不能跑。
func (s *Server) checkNodes(r *http.Request) Check {
	c := Check{Key: "nodes", Label: "工作流节点"}
	info, err := s.Sup.Client().ObjectInfo(r.Context())
	if err != nil {
		c.Status, c.Detail = "fail", "无法读取 ComfyUI 节点列表: "+err.Error()
		return c
	}
	missing := map[string][]string{} // 节点类 -> 用到它的工作流
	for _, t := range s.Reg.List() {
		for _, class := range t.UsedClasses() {
			if _, ok := info[class]; !ok {
				missing[class] = append(missing[class], t.Meta.ID)
			}
		}
	}
	if len(missing) == 0 {
		c.Status = "ok"
		c.Detail = "全部工作流所需的节点都已注册"
		return c
	}
	c.Status = "fail"
	for class, users := range missing {
		c.Items = append(c.Items, class+"（"+strings.Join(users, "、")+"）")
	}
	sort.Strings(c.Items)
	c.Detail = "有节点没有注册，通常是自定义节点包没装或加载失败；" +
		"装好后需重启 ComfyUI 才会生效"
	return c
}

func (s *Server) checkModels(r *http.Request) Check {
	c := Check{Key: "models", Label: "所需模型"}
	inv := s.Models.Cached()
	if inv == nil {
		var err error
		if inv, err = s.Models.Scan(r.Context()); err != nil {
			c.Status, c.Detail = "warn", "盘点模型失败: "+err.Error()
			return c
		}
	}
	for _, req := range inv.Requirements {
		if !req.Present {
			c.Items = append(c.Items, req.File)
		}
	}
	if len(c.Items) == 0 {
		c.Status, c.Detail = "ok", "全部就位"
		return c
	}
	// 缺模型只影响用到它的那条管线，别的照样能跑，所以是 warn 不是 fail。
	c.Status = "warn"
	c.Detail = "缺少的模型会让对应的风格预设无法生成，其余预设不受影响"
	c.Fix = "open-models"
	return c
}

func (s *Server) checkOutputDir() Check {
	c := Check{Key: "output", Label: "输出目录"}
	if err := os.MkdirAll(s.OutputDir, 0o755); err != nil {
		c.Status, c.Detail = "fail", "无法创建 "+s.OutputDir+": "+err.Error()
		return c
	}
	probe := filepath.Join(s.OutputDir, ".dt-write-probe")
	if err := os.WriteFile(probe, []byte("ok"), 0o644); err != nil {
		c.Status, c.Detail = "fail", "目录不可写: "+err.Error()
		return c
	}
	_ = os.Remove(probe)
	c.Status, c.Detail = "ok", s.OutputDir
	return c
}
