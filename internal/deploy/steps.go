package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// run 依次执行各步。任何一步失败即整体失败——后面的步骤都建立在前面的
// 产物上，硬着头皮往下跑只会得到一堆更难看懂的错误。
func (d *Deployer) run(ctx context.Context, opt Options) error {
	if err := os.MkdirAll(opt.Dir, 0o755); err != nil {
		return err
	}
	d.say("目标目录：%s", opt.Dir)
	if opt.Mirror {
		d.say("已开启国内镜像：PyPI 走清华，ComfyUI 源走 jihulab")
	}

	venvDir := filepath.Join(opt.Dir, "venv")
	comfyDir := filepath.Join(opt.Dir, "ComfyUI")
	python := filepath.Join(venvDir, "Scripts", "python.exe")
	mainPy := filepath.Join(comfyDir, "main.py")

	var uvPath, gitPath string

	if err := d.step(ctx, "tools", "检查 uv 与 git", func() (bool, string, error) {
		var err error
		if uvPath, err = findUV(); err != nil {
			return false, "", err
		}
		if gitPath, err = exec.LookPath("git"); err != nil {
			return false, "", fmt.Errorf("找不到 git。装一个（git-scm.com）再回来，" +
				"ComfyUI 与节点包都要靠它拉取和更新")
		}
		v, _ := d.capture(ctx, uvPath, "--version")
		gv, _ := d.capture(ctx, gitPath, "--version")
		return false, strings.TrimSpace(v) + " / " + strings.TrimSpace(gv), nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "python", "准备 Python "+opt.PyVersion, func() (bool, string, error) {
		// uv 自己管 Python 发行版，装的是完整的 CPython，
		// 不是那种缺 stdlib、要手改 ._pth 的嵌入式版。
		if err := d.exec(ctx, opt, "", uvPath, "python", "install", opt.PyVersion); err != nil {
			return false, "", err
		}
		return false, "CPython " + opt.PyVersion, nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "venv", "创建虚拟环境", func() (bool, string, error) {
		if fileExists(python) {
			return true, "已存在 " + rel(opt.Dir, python), nil
		}
		if err := d.exec(ctx, opt, "", uvPath, "venv", "--python", opt.PyVersion, venvDir); err != nil {
			return false, "", err
		}
		if !fileExists(python) {
			return false, "", fmt.Errorf("虚拟环境建好了但没有 %s", python)
		}
		return false, rel(opt.Dir, python), nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "torch", "安装 PyTorch ("+opt.Torch+")", func() (bool, string, error) {
		if opt.SkipTorch {
			return true, "按设置跳过", nil
		}
		// "已装则跳过"必须连档位一起看：只验能不能 import 的话，
		// 一个 CPU 版的 torch 也会被当成装好了，于是永远装不上 CUDA 版。
		if v, ok := d.torchVersion(ctx, python); ok && d.torchMatches(ctx, opt, python) {
			return true, "已装 " + v, nil
		} else if ok {
			d.warn("  已装的 %s 与所选档位 %s 不符，将重装", v, opt.Torch)
		}
		// PyTorch 的 CUDA 轮子只在它自己的源上，不能走 PyPI 镜像；
		// 这一步是整个部署里最大的一笔下载（2~3GB），也最容易被网络打断。
		idx := "https://download.pytorch.org/whl/" + opt.Torch
		args := []string{"pip", "install", "--python", python,
			"--index-url", idx, "--reinstall-package", "torch",
			"torch", "torchvision", "torchaudio"}
		d.say("  从 %s 下载，约 2~3GB，慢是正常的", idx)
		if err := d.exec(ctx, opt, "", uvPath, args...); err != nil {
			return false, "", err
		}
		v, _ := d.torchVersion(ctx, python)
		return false, v, nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "comfyui", "获取 ComfyUI", func() (bool, string, error) {
		if fileExists(mainPy) {
			return true, "已存在 " + rel(opt.Dir, comfyDir), nil
		}
		repo := comfyRepoGitHub
		if opt.Mirror {
			repo = comfyRepoMirror
		}
		if err := d.exec(ctx, opt, "", gitPath, "clone", "--depth", "1", repo, comfyDir); err != nil {
			// 镜像失败就回退官方源：镜像是加速手段，不该成为单点。
			if opt.Mirror {
				d.warn("  镜像拉取失败，改用官方源重试")
				if err2 := d.exec(ctx, opt, "", gitPath, "clone", "--depth", "1",
					comfyRepoGitHub, comfyDir); err2 != nil {
					return false, "", err2
				}
			} else {
				return false, "", err
			}
		}
		return false, rel(opt.Dir, comfyDir), nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "deps", "安装 ComfyUI 依赖", func() (bool, string, error) {
		req := filepath.Join(comfyDir, "requirements.txt")
		if !fileExists(req) {
			return false, "", fmt.Errorf("没有找到 %s", req)
		}
		if err := d.pipInstall(ctx, opt, uvPath, python, "-r", req); err != nil {
			return false, "", err
		}
		// ComfyUI 的 requirements.txt 里列着 torch，而 PyPI 上的 torch 是
		// **CPU 版**。装依赖这一步很可能把前面辛苦下的 CUDA 版顶掉——
		// 实测跳过 torch 步骤时，这一步自己拉来了 2.13.0+cpu。
		//
		// 这个故障是静默的：装完一切正常，只是生成慢十几倍，用户根本
		// 想不到该往这儿查。所以装完立刻回头验一遍，被顶掉就补回来。
		if fixed, err := d.ensureCUDATorch(ctx, opt, uvPath, python); err != nil {
			return false, "", err
		} else if fixed {
			return false, "装依赖时 torch 被换成了 CPU 版，已重新装回 CUDA 版", nil
		}
		return false, "", nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "nodes", "安装所需节点包", func() (bool, string, error) {
		cnDir := filepath.Join(comfyDir, "custom_nodes")
		if err := os.MkdirAll(cnDir, 0o755); err != nil {
			return false, "", err
		}
		var added, kept []string
		for _, p := range nodePacks {
			dst := filepath.Join(cnDir, p.Name)
			if dirExists(dst) {
				kept = append(kept, p.Name)
				continue
			}
			d.say("  拉取 %s（%s）", p.Name, p.Why)
			if err := d.exec(ctx, opt, "", gitPath, "clone", "--depth", "1", p.Repo, dst); err != nil {
				return false, "", fmt.Errorf("拉取 %s 失败: %w", p.Name, err)
			}
			added = append(added, p.Name)
			// 节点包的依赖要装进同一个虚拟环境。没有 requirements.txt 的
			// 直接跳过——那多半是纯 Python 实现，不需要额外依赖。
			if req := filepath.Join(dst, "requirements.txt"); fileExists(req) {
				if err := d.pipInstall(ctx, opt, uvPath, python, "-r", req); err != nil {
					// 单个节点包的依赖装不上不该让整个部署垮掉，
					// 但必须说清楚是哪个，否则用户只会看到"某节点没注册"。
					d.warn("  %s 的依赖没装全：%v（该节点可能无法注册）", p.Name, err)
				}
			}
		}
		detail := fmt.Sprintf("新增 %d 个", len(added))
		if len(kept) > 0 {
			detail += fmt.Sprintf("，已有 %d 个", len(kept))
		}
		return false, detail, nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "dtnodes", "接入 DreamTexture 自有节点", func() (bool, string, error) {
		src, err := selfNodesDir()
		if err != nil {
			return false, "", err
		}
		dst := filepath.Join(comfyDir, "custom_nodes", "ComfyUI-DreamTexture")
		if dirExists(dst) {
			return true, "已接入", nil
		}
		// 用目录联接而不是拷贝：改了 comfy_nodes 下的代码，重启 ComfyUI 就生效，
		// 不必记得同步两份。
		if err := d.exec(ctx, opt, "", "cmd", "/c", "mklink", "/J", dst, src); err != nil {
			d.warn("  建目录联接失败，改为复制（以后改了代码要重跑这一步）")
			if err := copyDir(src, dst); err != nil {
				return false, "", err
			}
			return false, "已复制", nil
		}
		return false, "已联接到 " + src, nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "modelpaths", "接上已有模型库", func() (bool, string, error) {
		if strings.TrimSpace(opt.ModelBasePath) == "" {
			return true, "未指定模型库目录，新环境会用自己的 models/", nil
		}
		if !dirExists(opt.ModelBasePath) {
			return false, "", fmt.Errorf("模型库目录不存在: %s", opt.ModelBasePath)
		}
		p := filepath.Join(comfyDir, "extra_model_paths.yaml")
		if err := os.WriteFile(p, []byte(modelPathsYAML(opt.ModelBasePath)), 0o644); err != nil {
			return false, "", err
		}
		return false, "复用 " + opt.ModelBasePath, nil
	}); err != nil {
		return err
	}

	if err := d.step(ctx, "verify", "验证环境", func() (bool, string, error) {
		if !fileExists(mainPy) {
			return false, "", fmt.Errorf("没有找到 %s", mainPy)
		}
		pyVer, err := d.capture(ctx, python, "-c", "import sys;print(sys.version.split()[0])")
		if err != nil {
			return false, "", fmt.Errorf("虚拟环境跑不起来: %w", err)
		}
		detail := "Python " + strings.TrimSpace(pyVer)

		// 跳过了 torch 就别去验它：那不是失败，是用户明确选了自己装。
		if _, ok := d.torchVersion(ctx, python); !ok && opt.SkipTorch {
			d.warn("  按设置跳过了 PyTorch，装上之前 ComfyUI 起不来")
			return false, detail + " / 无 PyTorch（按设置跳过）", nil
		}
		out, err := d.capture(ctx, python, "-c",
			"import torch;print(torch.__version__,torch.cuda.is_available())")
		if err != nil {
			return false, "", fmt.Errorf("PyTorch 装了但导入失败: %w", err)
		}
		f := strings.Fields(strings.TrimSpace(out))
		if len(f) < 2 {
			return false, "", fmt.Errorf("验证输出不对劲: %q", out)
		}
		if f[1] != "True" {
			// 不当成失败：CPU 也能跑，只是慢。但必须显眼地说出来，
			// 否则用户会以为装好了，跑一张图要十分钟才起疑。
			d.warn("  PyTorch 认不到 CUDA，将只能用 CPU 计算（会非常慢）")
		}
		return false, fmt.Sprintf("%s / torch %s / CUDA %s", detail, f[0], f[1]), nil
	}); err != nil {
		return err
	}

	d.mu.Lock()
	d.status.Python, d.status.MainPy = python, mainPy
	d.mu.Unlock()
	d.say("环境已就绪。到设置页把 Python 与主程序路径切过去，或直接点「应用到配置」：")
	d.say("  python  = %s", python)
	d.say("  main.py = %s", mainPy)
	return nil
}

// Detect 看某个运行时目录里是否已经躺着一套装好的环境。
//
// 部署状态是内存态，后端一重启就没了；但磁盘上的东西还在。没有这个探测的话，
// 用户重启一次后端就会发现"应用到配置"的按钮不见了，而环境明明装好了。
func Detect(dir string) (python, mainPy string, ok bool) {
	if strings.TrimSpace(dir) == "" {
		return "", "", false
	}
	python = filepath.Join(dir, "venv", "Scripts", "python.exe")
	mainPy = filepath.Join(dir, "ComfyUI", "main.py")
	if fileExists(python) && fileExists(mainPy) {
		return python, mainPy, true
	}
	return "", "", false
}

// pipInstall 走 uv 装包，按需带上国内镜像。
func (d *Deployer) pipInstall(ctx context.Context, opt Options, uvPath, python string,
	args ...string) error {

	full := []string{"pip", "install", "--python", python}
	if opt.Mirror {
		full = append(full, "--index-url", pypiMirror)
	}
	full = append(full, args...)
	return d.exec(ctx, opt, "", uvPath, full...)
}

// ensureCUDATorch 确认装着的 torch 还是 CUDA 版，被换成 CPU 版就装回来。
//
// 判据用 torch.version.cuda 而不是版本号里有没有 "+cu"：本地编译的轮子
// 未必带那个后缀，而 version.cuda 是运行时真实值。
func (d *Deployer) ensureCUDATorch(ctx context.Context, opt Options,
	uvPath, python string) (bool, error) {

	if opt.Torch == "cpu" {
		return false, nil // 用户就是要 CPU 版，不用管
	}
	out, err := d.capture(ctx, python, "-c", "import torch;print(torch.version.cuda or 'none')")
	if err != nil {
		// torch 还没装上（比如 SkipTorch），不属于"被顶掉"，交给 verify 去说。
		return false, nil
	}
	if strings.TrimSpace(out) != "none" {
		return false, nil
	}
	d.warn("  检测到 torch 变成了 CPU 版，正在换回 %s（否则生成会慢十几倍）", opt.Torch)
	idx := "https://download.pytorch.org/whl/" + opt.Torch
	err = d.exec(ctx, opt, "", uvPath, "pip", "install", "--python", python,
		"--index-url", idx, "--reinstall-package", "torch",
		"torch", "torchvision", "torchaudio")
	if err != nil {
		return false, fmt.Errorf("重装 CUDA 版 torch 失败: %w", err)
	}
	return true, nil
}

// torchMatches 判断装着的 torch 与所选档位是否相符。
func (d *Deployer) torchMatches(ctx context.Context, opt Options, python string) bool {
	out, err := d.capture(ctx, python, "-c", "import torch;print(torch.version.cuda or 'none')")
	if err != nil {
		return false
	}
	got := strings.TrimSpace(out)
	if opt.Torch == "cpu" {
		return got == "none"
	}
	// cu130 → 13.0，cu128 → 12.8
	want := opt.Torch
	if strings.HasPrefix(want, "cu") && len(want) >= 4 {
		digits := want[2:]
		want = digits[:len(digits)-1] + "." + digits[len(digits)-1:]
	}
	return got == want
}

func (d *Deployer) torchVersion(ctx context.Context, python string) (string, bool) {
	if !fileExists(python) {
		return "", false
	}
	out, err := d.capture(ctx, python, "-c", "import torch;print(torch.__version__)")
	if err != nil {
		return "", false
	}
	v := strings.TrimSpace(out)
	return v, v != ""
}

func modelPathsYAML(base string) string {
	// ComfyUI 的 YAML 解析对反斜杠不友好，统一写成正斜杠。
	b := strings.ReplaceAll(base, `\`, "/")
	if !strings.HasSuffix(b, "/") {
		b += "/"
	}
	dirs := []string{"checkpoints", "loras", "vae", "vae_approx", "upscale_models",
		"controlnet", "clip_vision", "style_models", "diffusion_models", "text_encoders",
		"embeddings", "clip"}
	var sb strings.Builder
	sb.WriteString("# 由 DreamTexture 一键部署生成：复用已有的模型库，避免重复下载\n")
	sb.WriteString("dreamtexture:\n")
	sb.WriteString("    base_path: " + b + "\n")
	for _, d := range dirs {
		sb.WriteString("    " + d + ": " + d + "\n")
	}
	return sb.String()
}

// selfNodesDir 找到本仓库的 comfy_nodes 目录。
func selfNodesDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	p := filepath.Join(wd, "comfy_nodes")
	if dirExists(p) {
		return p, nil
	}
	// 从可执行文件旁边再找一次：双击 exe 启动时工作目录未必是仓库根。
	if exe, err := os.Executable(); err == nil {
		p = filepath.Join(filepath.Dir(exe), "comfy_nodes")
		if dirExists(p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("找不到 comfy_nodes 目录（DreamTexture 自有节点）")
}

// findUV 定位 uv。PATH 里没有就去它的默认安装位置看看——
// uv 的安装脚本装到 ~/.local/bin，那个目录常常没进 PATH。
func findUV() (string, error) {
	if p, err := exec.LookPath("uv"); err == nil {
		return p, nil
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, c := range []string{
			filepath.Join(home, ".local", "bin", "uv.exe"),
			filepath.Join(home, ".cargo", "bin", "uv.exe"),
		} {
			if fileExists(c) {
				return c, nil
			}
		}
	}
	return "", fmt.Errorf("找不到 uv。它负责装 Python 和依赖，先装上再回来：\n" +
		`  PowerShell 里执行  irm https://astral.sh/uv/install.ps1 | iex`)
}

func fileExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && !fi.IsDir()
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func rel(base, p string) string {
	if r, err := filepath.Rel(base, p); err == nil && !strings.HasPrefix(r, "..") {
		return r
	}
	return p
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		r, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, r)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}
