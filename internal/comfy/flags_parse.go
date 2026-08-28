package comfy

import "strings"

// managedFlags 是由后端自己填、不该出现在用户参数里的。
//
// --listen/--port 由 BaseURL 推出来（见 spawn），--reserve-vram 有独立的
// 「显存余量」设置项。让它们也能在这儿写一遍，就会出现"两个地方说了不同的话，
// 而后写的那个赢"——排查起来很费劲。这里只负责认出来并如实告知。
var managedFlags = map[string]bool{
	"--listen": true, "--port": true, "--reserve-vram": true,
}

// Parse 把一串参数拆成"目录里认识的取值"和"剩下的原样保留"。
//
// leftover 是这个函数存在的理由：目录必然跟不上上游的 cli_args.py，
// 也拦不住自定义节点自带的参数。跟不上的时候原样留着，而不是悄悄丢掉——
// 用户手写的 --whitelist-custom-nodes 被界面吃掉，是不可接受的。
func Parse(args []string, catalog []Option) (values map[string]string, leftover []string) {
	toks := normalize(args)
	used := make([]bool, len(toks))
	values = map[string]string{}

	for _, opt := range catalog {
		switch opt.Kind {
		case KindBool:
			if i := findSeq(toks, used, []string{opt.Flag}); i >= 0 {
				markUsed(used, i, 1)
				values[opt.Key] = boolStr(!opt.Invert)
			} else {
				values[opt.Key] = boolStr(opt.Invert)
			}

		case KindChoice:
			// 长的先匹配：--fast fp16_accumulation 必须先于光秃秃的 --fast 试，
			// 否则前者会被后者截胡，界面上显示成"全开"。
			best, bestLen, bestAt := "", -1, -1
			for _, c := range opt.Choices {
				if len(c.Args) == 0 {
					continue
				}
				if i := findSeq(toks, used, c.Args); i >= 0 && len(c.Args) > bestLen {
					best, bestLen, bestAt = c.Value, len(c.Args), i
				}
			}
			if bestAt >= 0 {
				markUsed(used, bestAt, bestLen)
				values[opt.Key] = best
			} else {
				values[opt.Key] = defaultChoice(opt)
			}
		}
	}

	for i, t := range toks {
		if !used[i] {
			leftover = append(leftover, t)
		}
	}
	return values, leftover
}

// Build 把取值和"其他参数"合成一串完整的启动参数。
//
// 顺序：目录里的在前、用户手写的在后。后写的赢是 argparse 的规矩，
// 于是"我明明在其他参数里写了它却不生效"这种事不会发生。
func Build(values map[string]string, leftover []string, catalog []Option) []string {
	var out []string
	for _, opt := range catalog {
		v, ok := values[opt.Key]
		if !ok {
			continue
		}
		switch opt.Kind {
		case KindBool:
			// 普通项：开着才加。Invert 的项反过来——界面开着是常态，
			// 关掉才需要那个 --disable-xxx。异或正好表达这件事。
			if (v == "true") != opt.Invert {
				out = append(out, opt.Flag)
			}
		case KindChoice:
			for _, c := range opt.Choices {
				if c.Value == v {
					out = append(out, c.Args...)
					break
				}
			}
		}
	}
	return append(out, normalize(leftover)...)
}

// ManagedIn 挑出用户参数里那些本该由后端管的，供界面提醒。
//
// 返回空切片而不是 nil：这个值要进 JSON，nil 会变成 null，前端一句
// `managed.length` 就炸，而且炸的是整个组件——设置页上这一整块直接不见了，
// 控制台之外没有任何线索。凡是"一个列表"的字段都别让它变成 null。
func ManagedIn(args []string) []string {
	found := []string{}
	seen := map[string]bool{}
	for _, t := range normalize(args) {
		if managedFlags[t] && !seen[t] {
			seen[t] = true
			found = append(found, t)
		}
	}
	return found
}

// normalize 把 --k=v 拆成两个词，并丢掉空白项。
//
// argparse 两种写法都吃，不统一的话 "--preview-method=none" 会匹配不上
// ["--preview-method","none"]，界面显示"自动"而实际跑的是 none。
func normalize(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if strings.HasPrefix(a, "--") {
			if k, v, ok := strings.Cut(a, "="); ok {
				out = append(out, k, v)
				continue
			}
		}
		out = append(out, a)
	}
	return out
}

// findSeq 找一段连续且都还没被认领的词，返回起始下标。
func findSeq(toks []string, used []bool, seq []string) int {
	if len(seq) == 0 || len(seq) > len(toks) {
		return -1
	}
	for i := 0; i+len(seq) <= len(toks); i++ {
		ok := true
		for j, s := range seq {
			if used[i+j] || toks[i+j] != s {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func markUsed(used []bool, at, n int) {
	for i := at; i < at+n; i++ {
		used[i] = true
	}
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// defaultChoice 是没匹配上任何选项时显示的值：取第一个不带参数的选项。
//
// 不能一律取空串——「预览图」的默认是 none 而不是"不设置"，它带着参数。
func defaultChoice(opt Option) string {
	for _, c := range opt.Choices {
		if len(c.Args) == 0 {
			return c.Value
		}
	}
	if len(opt.Choices) > 0 {
		return opt.Choices[0].Value
	}
	return ""
}
