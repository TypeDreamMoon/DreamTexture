package comfy

import (
	"os/exec"
	"strconv"
	"strings"
)

// 本文件把 ComfyUI 的启动参数做成可发现的控件。
//
// 为什么值得做：这些参数决定跑得快不快、爆不爆显存，但它们藏在一串命令行里，
// 谁也记不住 --use-sage-attention 和 --lowvram 能不能一起写。而互斥组里同时
// 写了两个，ComfyUI 会直接起不来——用户看到的只是"启动失败"，找不到原因。
//
// 清单对着本机 ComfyUI 0.9.2 的 comfy/cli_args.py 核过，互斥组一一对应成下拉框。
// **不认识的参数一律原样保留**（见 Parse 的 leftover）：这份目录跟不上上游是
// 早晚的事，跟不上的时候不该把用户手写的参数吃掉。

// 控件类型。
const (
	KindChoice = "choice"
	KindBool   = "bool"
)

// Choice 是一个可选项。Args 是它展开成的参数，空表示"不加任何参数"。
type Choice struct {
	Value string   `json:"value"`
	Label string   `json:"label"`
	Note  string   `json:"note,omitempty"`
	Args  []string `json:"-"`
}

// Option 是设置页上的一行。
type Option struct {
	Key     string   `json:"key"`
	Label   string   `json:"label"`
	Help    string   `json:"help"`
	Kind    string   `json:"kind"`
	Icon    string   `json:"icon,omitempty"`
	Choices []Choice `json:"choices,omitempty"`

	// 以下仅 bool 用。
	Flag string `json:"flag,omitempty"`
	// Invert 表示"开关打开 = 不加这个参数"。
	//
	// 需要它是因为 ComfyUI 的开关多是反着说的（--disable-smart-memory），
	// 而界面上正着说才好懂——"智能显存优化"开着是常态，让用户去理解一个
	// 双重否定没有必要。
	Invert bool `json:"invert,omitempty"`

	// Advanced 的项默认折叠。
	Advanced bool `json:"advanced,omitempty"`
}

// Catalog 返回参数目录。gpus 由调用方探测后传入，为空则只给"默认 / CPU"。
func Catalog(gpus []GPU) []Option {
	device := []Choice{{Value: "", Label: "由 ComfyUI 自己挑"}}
	for _, g := range gpus {
		device = append(device, Choice{
			Value: "cuda" + strconv.Itoa(g.Index),
			Label: "GPU " + strconv.Itoa(g.Index) + "：" + g.Name + g.memSuffix(),
			Args:  []string{"--cuda-device", strconv.Itoa(g.Index)},
		})
	}
	device = append(device, Choice{
		Value: "cpu", Label: "CPU（不用显卡）", Args: []string{"--cpu"},
		Note: "慢几十倍，只在显卡不可用时救急",
	})

	return []Option{
		{
			Key: "device", Label: "生成引擎", Icon: "chip", Kind: KindChoice,
			Help:    "参与计算的硬件。多卡时可以指定用哪一张",
			Choices: device,
		},
		{
			Key: "vram", Label: "显存优化", Icon: "gauge", Kind: KindChoice,
			Help: "用更长的计算时间换更低的显存上限要求。默认让 ComfyUI 按显存大小自己判断",
			Choices: []Choice{
				{Value: "", Label: "由 ComfyUI 决定"},
				{Value: "gpu-only", Label: "全部留在显存", Args: []string{"--gpu-only"},
					Note: "最快，但显存不够会直接崩"},
				{Value: "highvram", Label: "高显存", Args: []string{"--highvram"}},
				{Value: "normalvram", Label: "普通", Args: []string{"--normalvram"}},
				{Value: "lowvram", Label: "低显存", Args: []string{"--lowvram"}},
				{Value: "novram", Label: "极低显存", Args: []string{"--novram"},
					Note: "模型基本不驻留显存，很慢"},
			},
		},
		{
			Key: "attn", Label: "注意力算法", Icon: "flow", Kind: KindChoice,
			Help: "交叉注意力的实现方式，明显影响速度与显存。不是每种都在每张卡上可用——换完起不来就退回自动",
			Choices: []Choice{
				{Value: "", Label: "自动"},
				{Value: "pytorch", Label: "PyTorch 原生", Args: []string{"--use-pytorch-cross-attention"}},
				{Value: "sage", Label: "SageAttention", Args: []string{"--use-sage-attention"},
					Note: "要另装 sageattention 包"},
				{Value: "flash", Label: "FlashAttention", Args: []string{"--use-flash-attention"},
					Note: "要另装 flash-attn 包"},
				{Value: "split", Label: "Split（省显存）", Args: []string{"--use-split-cross-attention"}},
				{Value: "quad", Label: "Quad（最省显存）", Args: []string{"--use-quad-cross-attention"}},
			},
		},
		{
			Key: "preview", Label: "预览图", Icon: "image", Kind: KindChoice,
			Help: "采样过程中的预览图。DreamTexture 只取最终产物，预览纯属额外开销，所以默认关掉",
			Choices: []Choice{
				{Value: "none", Label: "不生成（推荐）", Args: []string{"--preview-method", "none"}},
				{Value: "auto", Label: "自动", Args: []string{"--preview-method", "auto"}},
				{Value: "latent2rgb", Label: "Latent2RGB（快、粗）",
					Args: []string{"--preview-method", "latent2rgb"}},
				{Value: "taesd", Label: "TAESD（慢、准）", Args: []string{"--preview-method", "taesd"}},
			},
		},
		{
			Key: "smartmem", Label: "智能显存优化", Icon: "gauge", Kind: KindBool,
			Help:   "尽量把模型留在显存里，省下反复换入换出的时间。和别的程序抢显存时可以关掉",
			Flag:   "--disable-smart-memory",
			Invert: true,
		},
		{
			Key: "cpuvae", Label: "用 CPU 跑 VAE", Icon: "chip", Kind: KindBool,
			Help: "显著降低峰值显存，代价是整体变慢。显存被别的程序占着时值得一试",
			Flag: "--cpu-vae",
		},
		{
			Key: "fast", Label: "激进提速", Icon: "wand", Kind: KindChoice,
			Help: "以一定质量代价换速度。要显卡型号与 Torch 版本支持，不支持时会被忽略",
			Choices: []Choice{
				{Value: "", Label: "关闭"},
				{Value: "fp16acc", Label: "仅 fp16 累加", Args: []string{"--fast", "fp16_accumulation"},
					Note: "最稳的一档"},
				{Value: "all", Label: "全开", Args: []string{"--fast"},
					Note: "含 fp8 矩阵乘、cublas、autotune"},
			},
		},
		{
			Key: "deterministic", Label: "稳定计算", Icon: "shield", Kind: KindBool,
			Help: "改用较慢但结果可复现的算法。要求同一种子每次出一模一样的图时才需要",
			Flag: "--deterministic",
		},

		// ── 以下默认折叠 ──
		{
			Key: "cudamalloc", Label: "显存分配器", Icon: "gauge", Kind: KindChoice, Advanced: true,
			Help: "cudaMallocAsync 分配更快，但它的池子不把显存还给驱动；和别的程序抢显存时换回原生的更稳",
			Choices: []Choice{
				{Value: "", Label: "由 ComfyUI 决定"},
				{Value: "on", Label: "cudaMallocAsync", Args: []string{"--cuda-malloc"}},
				{Value: "off", Label: "PyTorch 原生", Args: []string{"--disable-cuda-malloc"}},
			},
		},
		{
			Key: "precision", Label: "通用模型精度", Icon: "sliders", Kind: KindChoice, Advanced: true,
			Help: "一刀切地压住所有模型的精度。下面几项是分别指定，冲突时以更具体的那个为准",
			Choices: []Choice{
				{Value: "", Label: "由 ComfyUI 决定"},
				{Value: "fp16", Label: "半精度 fp16", Args: []string{"--force-fp16"}},
				{Value: "fp32", Label: "全精度 fp32", Args: []string{"--force-fp32"},
					Note: "最稳，也最吃显存"},
			},
		},
		{
			Key: "upcast", Label: "注意力精度提升", Icon: "sliders", Kind: KindChoice, Advanced: true,
			Help: "把交叉注意力升到更高精度算。能缓解某些模型的黑图，代价是慢一些",
			Choices: []Choice{
				{Value: "", Label: "由 ComfyUI 决定"},
				{Value: "on", Label: "强制提升", Args: []string{"--force-upcast-attention"}},
				{Value: "off", Label: "禁止提升", Args: []string{"--dont-upcast-attention"}},
			},
		},
		{
			Key: "xformers", Label: "允许使用 xFormers", Icon: "flow", Kind: KindBool, Advanced: true,
			Help:   "装了 xformers 时 ComfyUI 会自动用上。这里关掉可以在怀疑它出问题时排除掉",
			Flag:   "--disable-xformers",
			Invert: true,
		},
		{
			Key: "unet", Label: "UNet 精度", Icon: "sliders", Kind: KindChoice, Advanced: true,
			Help: "降精度省显存也更快，但可能出黑图。拿不准就别动",
			Choices: []Choice{
				{Value: "", Label: "自动"},
				{Value: "fp16", Label: "fp16", Args: []string{"--fp16-unet"}},
				{Value: "bf16", Label: "bf16", Args: []string{"--bf16-unet"}},
				{Value: "fp32", Label: "fp32", Args: []string{"--fp32-unet"}},
				{Value: "fp8e4", Label: "fp8 (e4m3fn)", Args: []string{"--fp8_e4m3fn-unet"}},
				{Value: "fp8e5", Label: "fp8 (e5m2)", Args: []string{"--fp8_e5m2-unet"}},
			},
		},
		{
			Key: "vae", Label: "VAE 精度", Icon: "sliders", Kind: KindChoice, Advanced: true,
			Help: "VAE 跑 fp16 出黑图是常见故障，遇到就改 fp32",
			Choices: []Choice{
				{Value: "", Label: "自动"},
				{Value: "fp16", Label: "fp16", Args: []string{"--fp16-vae"}},
				{Value: "bf16", Label: "bf16", Args: []string{"--bf16-vae"}},
				{Value: "fp32", Label: "fp32", Args: []string{"--fp32-vae"}},
			},
		},
		{
			Key: "textenc", Label: "文本编码器精度", Icon: "sliders", Kind: KindChoice, Advanced: true,
			Help: "影响提示词编码那一步，显存吃紧时可以降",
			Choices: []Choice{
				{Value: "", Label: "自动"},
				{Value: "fp16", Label: "fp16", Args: []string{"--fp16-text-enc"}},
				{Value: "bf16", Label: "bf16", Args: []string{"--bf16-text-enc"}},
				{Value: "fp32", Label: "fp32", Args: []string{"--fp32-text-enc"}},
				{Value: "fp8e4", Label: "fp8 (e4m3fn)", Args: []string{"--fp8_e4m3fn-text-enc"}},
				{Value: "fp8e5", Label: "fp8 (e5m2)", Args: []string{"--fp8_e5m2-text-enc"}},
			},
		},
		{
			Key: "cache", Label: "节点缓存", Icon: "layers", Kind: KindChoice, Advanced: true,
			Help: "缓存中间结果能让重复执行快很多，代价是内存占用",
			Choices: []Choice{
				{Value: "", Label: "由 ComfyUI 决定"},
				{Value: "classic", Label: "经典", Args: []string{"--cache-classic"}},
				{Value: "none", Label: "不缓存", Args: []string{"--cache-none"}},
			},
		},
	}
}

// GPU 是一张显卡。
type GPU struct {
	Index  int    `json:"index"`
	Name   string `json:"name"`
	TotalM int64  `json:"total_mb"`
}

func (g GPU) memSuffix() string {
	if g.TotalM <= 0 {
		return ""
	}
	return "（" + strconv.FormatInt(g.TotalM/1024, 10) + " GB）"
}

// DetectGPUs 用 nvidia-smi 列出显卡。
//
// 列不出就返回空，目录里只剩"默认 / CPU"两项，仍然可用——不是每台机器都有
// N 卡，也不是每台有 N 卡的机器都把 nvidia-smi 放进了 PATH。
func DetectGPUs() []GPU {
	if _, err := exec.LookPath("nvidia-smi"); err != nil {
		return nil
	}
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=index,name,memory.total", "--format=csv,noheader,nounits").Output()
	if err != nil {
		return nil
	}
	var gpus []GPU
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.Split(strings.TrimSpace(line), ",")
		if len(parts) < 3 {
			continue
		}
		idx, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			continue
		}
		mb, _ := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		gpus = append(gpus, GPU{Index: idx, Name: strings.TrimSpace(parts[1]), TotalM: mb})
	}
	return gpus
}
