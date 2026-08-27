// Package model 管理 ComfyUI 侧的模型：清单、占用、缺失检测与下载。
//
// 清单一律经 ComfyUI 的 /experiment/models 接口取得，不直接扫文件系统——
// 这样 attach 模式（用户自己开着 ComfyUI）同样可用，也天然认得
// extra_model_paths.yaml 挂进来的目录。
package model

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/mengye/dreamtexture/internal/comfy"
	"github.com/mengye/dreamtexture/internal/workflow"
)

type File struct {
	Name     string    `json:"name"`
	Dir      string    `json:"dir"`
	Path     string    `json:"path"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	// UsedBy 是引用了这个文件的工作流 id。空表示当前没有工作流用到它。
	UsedBy []string `json:"used_by,omitempty"`
}

type Folder struct {
	Name       string   `json:"name"`
	Paths      []string `json:"paths"`
	Count      int      `json:"count"`
	TotalBytes int64    `json:"total_bytes"`
}

// Requirement 是某个工作流声明所需的一个模型，附带它当前是否就位。
type Requirement struct {
	workflow.ModelRequirement
	WorkflowIDs []string `json:"workflow_ids"`
	Present     bool     `json:"present"`
	ActualBytes int64    `json:"actual_bytes,omitempty"`
	// Target 是缺失时建议写入的绝对路径。
	Target string `json:"target,omitempty"`
}

type Inventory struct {
	Folders      []Folder      `json:"folders"`
	Files        []File        `json:"files"`
	Requirements []Requirement `json:"requirements"`
	TotalBytes   int64         `json:"total_bytes"`
	ScannedAt    time.Time     `json:"scanned_at"`
}

// MissingCount 返回尚未就位的模型数量。
func (inv *Inventory) MissingCount() int {
	n := 0
	for _, r := range inv.Requirements {
		if !r.Present {
			n++
		}
	}
	return n
}

type Manager struct {
	sup *comfy.Supervisor
	reg *workflow.Registry

	mu     sync.RWMutex
	cached *Inventory
}

func NewManager(sup *comfy.Supervisor, reg *workflow.Registry) *Manager {
	return &Manager{sup: sup, reg: reg}
}

// Cached 返回上次扫描结果，没有则返回 nil。
func (m *Manager) Cached() *Inventory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cached
}

// Scan 重新盘点全部模型，并与各工作流的声明比对。
//
// 只扫工作流真正用得到的类别：ComfyUI 的类别有二十多个，全扫一遍很慢，
// 而绝大多数与我们无关。
func (m *Manager) Scan(ctx context.Context) (*Inventory, error) {
	cli := m.sup.Client()
	folders, err := cli.ModelFolders(ctx)
	if err != nil {
		return nil, err
	}

	wanted := m.wantedDirs()
	inv := &Inventory{ScannedAt: time.Now()}
	// name -> dir -> 引用它的工作流
	usedBy := m.requirementUsage()
	// dir -> 该类别的绝对路径列表，供下载时选目标
	pathsByDir := map[string][]string{}

	for _, f := range folders {
		pathsByDir[f.Name] = f.Paths
		if !wanted[f.Name] {
			continue
		}
		files, err := cli.ModelFiles(ctx, f.Name)
		if err != nil {
			// 单个类别失败不该让整次盘点失败（例如目录被挪走）。
			continue
		}
		fold := Folder{Name: f.Name, Paths: f.Paths}
		for _, x := range files {
			p := x.Name
			if x.PathIndex >= 0 && x.PathIndex < len(f.Paths) {
				p = filepath.Join(f.Paths[x.PathIndex], x.Name)
			}
			file := File{
				Name: x.Name, Dir: f.Name, Path: p,
				Size: x.Size, Modified: x.ModTime(),
				UsedBy: usedBy[key(f.Name, x.Name)],
			}
			inv.Files = append(inv.Files, file)
			fold.Count++
			fold.TotalBytes += x.Size
		}
		inv.TotalBytes += fold.TotalBytes
		inv.Folders = append(inv.Folders, fold)
	}

	present := map[string]int64{}
	for _, f := range inv.Files {
		present[key(f.Dir, f.Name)] = f.Size
	}

	for _, req := range m.mergedRequirements() {
		k := key(req.Dir, req.File)
		if size, ok := present[k]; ok {
			req.Present, req.ActualBytes = true, size
		} else if paths := pathsByDir[req.Dir]; len(paths) > 0 {
			req.Target = filepath.Join(preferredPath(paths), req.File)
		}
		inv.Requirements = append(inv.Requirements, req)
	}

	sort.Slice(inv.Folders, func(i, j int) bool { return inv.Folders[i].Name < inv.Folders[j].Name })
	sort.Slice(inv.Files, func(i, j int) bool {
		if inv.Files[i].Dir != inv.Files[j].Dir {
			return inv.Files[i].Dir < inv.Files[j].Dir
		}
		return inv.Files[i].Name < inv.Files[j].Name
	})
	sort.Slice(inv.Requirements, func(i, j int) bool {
		if inv.Requirements[i].Present != inv.Requirements[j].Present {
			return !inv.Requirements[i].Present // 缺失的排前面
		}
		return inv.Requirements[i].File < inv.Requirements[j].File
	})

	m.mu.Lock()
	m.cached = inv
	m.mu.Unlock()
	return inv, nil
}

// preferredPath 从多个候选路径里挑一个下载目标。
//
// ComfyUI 自带的 models 目录常常和程序装在同一个盘、余量紧张，而用户通过
// extra_model_paths.yaml 额外挂上来的目录通常正是为放模型准备的，所以优先用
// 后者（列表里排在自带目录之后）。
func preferredPath(paths []string) string {
	if len(paths) > 1 {
		return paths[1]
	}
	return paths[0]
}

func (m *Manager) wantedDirs() map[string]bool {
	dirs := map[string]bool{}
	for _, t := range m.reg.List() {
		for _, r := range t.Meta.ModelRequirements {
			dirs[r.Dir] = true
		}
	}
	// 这几类即便当前没被声明也值得展示：用户手上多半有存货，占的盘也最多。
	for _, d := range []string{"checkpoints", "loras", "vae", "upscale_models", "controlnet"} {
		dirs[d] = true
	}
	return dirs
}

// mergedRequirements 把各工作流的声明去重合并，同一个文件被多个工作流引用时合成一条。
func (m *Manager) mergedRequirements() []Requirement {
	byKey := map[string]*Requirement{}
	var order []string
	for _, t := range m.reg.List() {
		for _, r := range t.Meta.ModelRequirements {
			k := key(r.Dir, r.File)
			if e, ok := byKey[k]; ok {
				e.WorkflowIDs = append(e.WorkflowIDs, t.Meta.ID)
				continue
			}
			byKey[k] = &Requirement{ModelRequirement: r, WorkflowIDs: []string{t.Meta.ID}}
			order = append(order, k)
		}
	}
	out := make([]Requirement, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	return out
}

func (m *Manager) requirementUsage() map[string][]string {
	usage := map[string][]string{}
	for _, t := range m.reg.List() {
		for _, r := range t.Meta.ModelRequirements {
			k := key(r.Dir, r.File)
			usage[k] = append(usage[k], t.Meta.ID)
		}
	}
	return usage
}

func key(dir, file string) string {
	// ComfyUI 在 Windows 上大小写不敏感，比对时统一小写免得误判缺失。
	return strings.ToLower(dir + "/" + filepath.ToSlash(file))
}
