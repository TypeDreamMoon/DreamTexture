package imagen

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
)

// FlattenReport 记录一次亮度场压平做了什么，用于写进 manifest 和界面提示。
type FlattenReport struct {
	// Falloff 是压平前的边缘/中心亮度比。1 表示完全均匀，越小说明暗角越重。
	Falloff float64 `json:"falloff"`
	// FalloffAfter 是压平后的同一指标，用来确认这一步真的起了作用。
	FalloffAfter float64 `json:"falloff_after"`
	// Applied 为 false 表示原图本来就够均匀，没有改动。
	Applied bool `json:"applied"`
	// Degree 是拟合亮度场用的多项式阶数。
	Degree int `json:"degree,omitempty"`
}

// flattenThreshold 是"值得动手"的门槛。
//
// 边缘比中心暗 2% 以内肉眼看不出来，也拼不出网格，动它只会平白引入一层
// 重采样噪声。实测本地 SDXL 出的图落在 0.99 上下，云端模型明显更低。
const flattenThreshold = 0.98

// FlattenLuminance 消除低频亮度不均（暗角）。
//
// 为什么必须做：云端图像模型有从中心到边缘的亮度衰减，单看一张图很自然，
// 一旦平铺，四个瓦片的暗边拼在一起就成了一张可见的网格——纹理最忌讳的就是
// 这种规律性痕迹。本地 SDXL 走循环卷积不存在这个问题，所以这一步只对
// 外部底图有意义。
//
// 做法是把图除以自身的大尺度模糊亮度场：模糊半径取得足够大（约边长的 1/4），
// 于是只有低频的明暗趋势被除掉，材质本身的纹理细节（高频）原样保留。
//
// strength 为 0~1，1 表示完全压平。运算在线性光空间做——在 sRGB 上直接乘
// 会让暗部提亮过头，边角泛灰。
func FlattenLuminance(data []byte, strength float64) ([]byte, FlattenReport, error) {
	var rep FlattenReport
	if strength <= 0 {
		return data, rep, nil
	}
	if strength > 1 {
		strength = 1
	}

	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, rep, fmt.Errorf("解码底图失败: %w", err)
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 32 || h < 32 {
		return data, rep, nil
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), src, b.Min, draw.Src)

	// 线性光下的亮度
	lum := make([]float64, w*h)
	for i, n := 0, w*h; i < n; i++ {
		p := img.Pix[i*4:]
		lum[i] = 0.2126*srgbToLinear(p[0]) + 0.7152*srgbToLinear(p[1]) + 0.0722*srgbToLinear(p[2])
	}

	rep.Falloff = edgeCenterRatio(lum, w, h)
	if rep.Falloff >= flattenThreshold {
		return data, rep, nil
	}

	field, ok := fitField(lum, w, h)
	if !ok {
		return data, rep, nil
	}
	rep.Degree = fieldDegree

	var sum float64
	for _, v := range field {
		sum += v
	}
	mean := sum / float64(len(field))
	if mean <= 1e-6 {
		return data, rep, nil // 几乎全黑，压平没有意义也不安全
	}

	// 增益上下限：亮度场里若有接近 0 的区域（大片纯黑），不设限会把噪声放大成雪花。
	const maxGain, minGain = 2.5, 0.4
	for i, n := 0, w*h; i < n; i++ {
		f := field[i]
		if f <= 1e-6 {
			continue
		}
		g := math.Pow(mean/f, strength)
		g = math.Max(minGain, math.Min(maxGain, g))
		p := img.Pix[i*4:]
		for c := 0; c < 3; c++ {
			p[c] = linearToSrgb(srgbToLinear(p[c]) * g)
		}
		lum[i] *= g
	}
	rep.FalloffAfter = edgeCenterRatio(lum, w, h)
	rep.Applied = true

	var out bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(&out, img); err != nil {
		return nil, rep, fmt.Errorf("重新编码底图失败: %w", err)
	}
	return out.Bytes(), rep, nil
}

// edgeCenterRatio 量化暗角：外圈一成宽的环带 与 中心区域 的平均亮度之比。
//
// 取环带而不是四个角，是因为暗角未必是径向的——有些模型是上下暗、
// 有些是单边偏暗，只看角落会漏判。
func edgeCenterRatio(lum []float64, w, h int) float64 {
	band := max(1, min(w, h)/10)
	var edgeSum, edgeN, ctrSum, ctrN float64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := lum[y*w+x]
			switch {
			case x < band || x >= w-band || y < band || y >= h-band:
				edgeSum, edgeN = edgeSum+v, edgeN+1
			case x >= w/3 && x < w*2/3 && y >= h/3 && y < h*2/3:
				ctrSum, ctrN = ctrSum+v, ctrN+1
			}
		}
	}
	if edgeN == 0 || ctrN == 0 || ctrSum <= 0 {
		return 1
	}
	return (edgeSum / edgeN) / (ctrSum / ctrN)
}

// fieldDegree 是拟合亮度场的多项式阶数。
//
// 4 阶是权衡后的结果：暗角本身大致是 r² 的，2 阶就能拟合径向衰减，但真实
// 模型的不均匀经常是偏心的、甚至单边的，2 阶压不住；再往上升到 6 阶就开始
// 有能力去贴合材质本身的大色块，把不该消的对比也消掉了。
const fieldDegree = 4

// fitField 用最小二乘拟合一个低阶二维多项式作为亮度场。
//
// 一开始用的是大半径高斯模糊，实测不行：三趟带边缘延拓的模糊会把边界值
// 一路拉向内部均值，512 见方的图上只能还原三分之一的暗角（场的边缘/中心比
// 0.919，而真实是 0.757）——恰恰在最需要校正的四条边上最不准。
//
// 多项式拟合没有这个毛病：它是全局拟合，边界处的取值由整体趋势外推而来，
// 不存在"窗口伸出图外"的问题。另一个好处是它天生不会伤到细节——4 阶多项式
// 根本没有表达高频的自由度，想吃掉纹理也吃不动。
func fitField(lum []float64, w, h int) ([]float64, bool) {
	return fitFieldDeg(lum, w, h, fieldDegree)
}

func fitFieldDeg(lum []float64, w, h, degree int) ([]float64, bool) {
	// 在线性空间拟合，不要在 log 空间。
	//
	// 曾经用过 log：暗角是乘性的，取对数变成加性，看着更该这么做。实测反过来
	// 咬人——log 拟合估的是几何均值，而几何均值会被局部方差压低，于是"花纹多、
	// 明暗跳动大"的区域被误判成"暗"，去校正它就等于凭空提亮。有多离谱：一张
	// 完全没有暗角的合成图（边缘/中心 0.987）过完 log 版之后变成 0.81，
	// 也就是说这一步自己造出了一圈暗角。
	//
	// 线性最小二乘估的是局部算术均值，正是我们想要的量。乘性模型在这里也成立：
	// 若底图的局部均值大致均匀，E[lum(x)] = 均值 × 暗角(x)，拟合出来的就是
	// 差一个常数倍的暗角本身，而常数倍在后面除以 mean 时约掉了。
	var sum float64
	for _, v := range lum {
		sum += v
	}
	mean := sum / float64(len(lum))
	if mean <= 1e-6 {
		return nil, false
	}

	terms := polyTerms(degree)
	n := len(terms)
	ata := make([]float64, n*n)
	atb := make([]float64, n)

	// 拟合用抽样：场是平滑的，不需要每个像素都参与。2880 见方的图有八百多万
	// 像素，全量累加 15×15 的法方程要跑上一两秒，抽到 256 见方后可以忽略不计，
	// 而拟合结果肉眼无差。
	step := max(1, max(w, h)/256)
	basis := make([]float64, n)
	for y := 0; y < h; y += step {
		v := 2*float64(y)/float64(h-1) - 1
		for x := 0; x < w; x += step {
			u := 2*float64(x)/float64(w-1) - 1
			evalBasis(terms, u, v, basis)
			t := lum[y*w+x]
			for i := 0; i < n; i++ {
				atb[i] += basis[i] * t
				for j := i; j < n; j++ {
					ata[i*n+j] += basis[i] * basis[j]
				}
			}
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			ata[i*n+j] = ata[j*n+i]
		}
	}
	coef, ok := solve(ata, atb, n)
	if !ok {
		return nil, false
	}

	field := make([]float64, w*h)
	for y := 0; y < h; y++ {
		v := 2*float64(y)/float64(h-1) - 1
		for x := 0; x < w; x++ {
			u := 2*float64(x)/float64(w-1) - 1
			evalBasis(terms, u, v, basis)
			var acc float64
			for i := 0; i < n; i++ {
				acc += coef[i] * basis[i]
			}
			field[y*w+x] = acc
		}
	}
	return field, true
}

// polyTerms 列出 i+j <= degree 的全部 (i,j) 指数对。
func polyTerms(degree int) [][2]int {
	var out [][2]int
	for d := 0; d <= degree; d++ {
		for i := 0; i <= d; i++ {
			out = append(out, [2]int{i, d - i})
		}
	}
	return out
}

func evalBasis(terms [][2]int, u, v float64, dst []float64) {
	for i, t := range terms {
		dst[i] = ipow(u, t[0]) * ipow(v, t[1])
	}
}

func ipow(x float64, n int) float64 {
	r := 1.0
	for ; n > 0; n-- {
		r *= x
	}
	return r
}

// solve 用带部分主元的高斯消元解 n 元线性方程组，就地修改 a 与 b。
func solve(a, b []float64, n int) ([]float64, bool) {
	a = append([]float64(nil), a...)
	x := append([]float64(nil), b...)
	for col := 0; col < n; col++ {
		p, best := col, math.Abs(a[col*n+col])
		for r := col + 1; r < n; r++ {
			if m := math.Abs(a[r*n+col]); m > best {
				p, best = r, m
			}
		}
		// 病态或奇异：拟合不可信，交给调用方放弃压平而不是硬算一个结果出来。
		if best < 1e-12 {
			return nil, false
		}
		if p != col {
			for k := 0; k < n; k++ {
				a[col*n+k], a[p*n+k] = a[p*n+k], a[col*n+k]
			}
			x[col], x[p] = x[p], x[col]
		}
		inv := 1 / a[col*n+col]
		for r := col + 1; r < n; r++ {
			f := a[r*n+col] * inv
			if f == 0 {
				continue
			}
			for k := col; k < n; k++ {
				a[r*n+k] -= f * a[col*n+k]
			}
			x[r] -= f * x[col]
		}
	}
	for r := n - 1; r >= 0; r-- {
		s := x[r]
		for k := r + 1; k < n; k++ {
			s -= a[r*n+k] * x[k]
		}
		x[r] = s / a[r*n+r]
	}
	return x, true
}

func srgbToLinear(v uint8) float64 {
	f := float64(v) / 255
	if f <= 0.04045 {
		return f / 12.92
	}
	return math.Pow((f+0.055)/1.055, 2.4)
}

func linearToSrgb(f float64) uint8 {
	if f <= 0 {
		return 0
	}
	if f >= 1 {
		return 255
	}
	var s float64
	if f <= 0.0031308 {
		s = f * 12.92
	} else {
		s = 1.055*math.Pow(f, 1/2.4) - 0.055
	}
	return uint8(math.Round(s * 255))
}
