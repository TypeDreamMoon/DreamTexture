package imagen

import (
	"bytes"
	"image"
	"image/png"
	"math"
	"math/rand/v2"
	"testing"
)

// synthTexture 造一张有细节的图：重复的花纹 + 颗粒。
//
// 所有成分的周期都远小于画幅（最低频也有十几个周期）——这是刻意的，因为
// 可平铺纹理本来就长这样：细节在小尺度上重复，整幅画面没有单调的明暗走向。
//
// 早先的版本用了周期约 1/3.5 画幅的正弦，结果多项式把那道起伏当成趋势拟合
// 了进去，测试一路失败。那是 fixture 的问题不是算法的问题：真有那么低频的
// 亮度走向时，它平铺起来本来就会露馅，压掉才是对的。
func synthTexture(w, h int, seed uint64) *image.NRGBA {
	rng := rand.New(rand.NewPCG(seed, 0x9E3779B9))
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	fx, fy := float64(w), float64(h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			u, v := float64(x)/fx, float64(y)/fy
			base := 128 +
				34*math.Sin(2*math.Pi*u*13) +
				26*math.Cos(2*math.Pi*v*11) +
				20*math.Sin(2*math.Pi*(u+v)*23)
			grain := (rng.Float64() - 0.5) * 42
			val := clamp255(base + grain)
			p := img.Pix[(y*w+x)*4:]
			p[0], p[1], p[2], p[3] = val, uint8(float64(val)*0.92), uint8(float64(val)*0.78), 255
		}
	}
	return img
}

// applyVignette 乘一个径向衰减，模拟云端模型的暗角。
func applyVignette(img *image.NRGBA, depth float64) *image.NRGBA {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(b)
	cx, cy := float64(w-1)/2, float64(h-1)/2
	maxR := math.Hypot(cx, cy)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r := math.Hypot(float64(x)-cx, float64(y)-cy) / maxR
			g := 1 - depth*r*r
			src := img.Pix[(y*w+x)*4:]
			dst := out.Pix[(y*w+x)*4:]
			for c := 0; c < 3; c++ {
				dst[c] = linearToSrgb(srgbToLinear(src[c]) * g)
			}
			dst[3] = 255
		}
	}
	return out
}

func clamp255(f float64) uint8 {
	if f < 0 {
		return 0
	}
	if f > 255 {
		return 255
	}
	return uint8(f)
}

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("编码失败: %v", err)
	}
	return buf.Bytes()
}

func lumaOf(t *testing.T, data []byte) ([]float64, int, int) {
	t.Helper()
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("解码失败: %v", err)
	}
	b := im.Bounds()
	w, h := b.Dx(), b.Dy()
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bb, _ := im.At(b.Min.X+x, b.Min.Y+y).RGBA()
			out[y*w+x] = 0.2126*srgbToLinear(uint8(r>>8)) +
				0.7152*srgbToLinear(uint8(g>>8)) +
				0.0722*srgbToLinear(uint8(bb>>8))
		}
	}
	return out, w, h
}

// detailEnergy 是相邻像素差的均方根，用来衡量高频细节还剩多少。
func detailEnergy(lum []float64, w, h int) float64 {
	var sum float64
	var n int
	for y := 0; y < h; y++ {
		for x := 1; x < w; x++ {
			d := lum[y*w+x] - lum[y*w+x-1]
			sum += d * d
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return math.Sqrt(sum / float64(n))
}

func TestFlattenRemovesVignette(t *testing.T) {
	const size = 512
	clean := synthTexture(size, size, 7)
	cleanPNG := encode(t, clean)
	cleanLum, _, _ := lumaOf(t, cleanPNG)
	cleanRatio := edgeCenterRatio(cleanLum, size, size)
	if math.Abs(cleanRatio-1) > 0.03 {
		t.Fatalf("合成图本身就不均匀（比值 %.4f），测试前提不成立", cleanRatio)
	}

	for _, depth := range []float64{0.15, 0.30, 0.45} {
		vig := applyVignette(clean, depth)
		vigPNG := encode(t, vig)

		before, _, _ := lumaOf(t, vigPNG)
		beforeRatio := edgeCenterRatio(before, size, size)
		if beforeRatio >= flattenThreshold {
			t.Fatalf("depth=%.2f 没造出可检出的暗角（比值 %.4f）", depth, beforeRatio)
		}

		fixed, rep, err := FlattenLuminance(vigPNG, 1)
		if err != nil {
			t.Fatalf("depth=%.2f 压平失败: %v", depth, err)
		}
		if !rep.Applied {
			t.Fatalf("depth=%.2f 应该被判定为需要压平，实际没有", depth)
		}

		after, _, _ := lumaOf(t, fixed)
		afterRatio := edgeCenterRatio(after, size, size)

		// 正确的目标是回到**原图自身**的边缘/中心比，不是回到 1.0。
		//
		// 这一条是实测逼出来的：桩服务那张图的正弦周期在采样窗口里不整除，
		// 自身比值就有 1.054，压平后落在 1.047 看着像过冲了 5%，其实离它
		// 自己的固有值只差 0.7%。拿 1.0 当靶子会把"纹理本来就不匀"误判成
		// 算法有问题，反过来也会诱使人去多校正一点，把真实结构抹掉。
		if d := math.Abs(afterRatio - cleanRatio); d > 0.02 {
			t.Errorf("depth=%.2f 压平后没有回到原图自身的比值: %.4f vs %.4f（差 %.4f）",
				depth, afterRatio, cleanRatio, d)
		}
		if math.Abs(afterRatio-cleanRatio) > math.Abs(beforeRatio-cleanRatio)/3 {
			t.Errorf("depth=%.2f 改善不足: %.4f -> %.4f（原图 %.4f）",
				depth, beforeRatio, afterRatio, cleanRatio)
		}

		// 细节必须留下来——把整张图糊平也能让比值变成 1，那不是我们要的。
		//
		// 基准取"打了暗角之后"的图而不是原图：暗角本身就是乘性压暗，暗下去的
		// 区域在 8 位量化里已经丢掉了一部分精度，那部分谁也捡不回来。这里要
		// 验的是压平**没有额外**抹掉细节，而且提亮暗区之后细节应当有所回升。
		cleanDetail := detailEnergy(cleanLum, size, size)
		vigDetail := detailEnergy(before, size, size)
		afterDetail := detailEnergy(after, size, size)

		// 2% 的余量留给一次 8 位重新量化；实测损失在 0.2%~0.6%。
		if afterDetail < vigDetail*0.98 {
			t.Errorf("depth=%.2f 压平额外抹掉了细节: %.5f -> %.5f", depth, vigDetail, afterDetail)
		}
		// 相对原图的 0.7：压平是归一化到**均值**，中心会被压暗、边缘被提亮，
		// 暗角越深整体细节强度就越靠近均值水平。这是预期行为，不是模糊。
		if afterDetail < cleanDetail*0.7 {
			t.Errorf("depth=%.2f 细节相对原图损失过多，像是被糊平了: %.5f vs %.5f",
				depth, afterDetail, cleanDetail)
		}
		if afterDetail > cleanDetail*1.15 {
			t.Errorf("depth=%.2f 细节被放大过头，可能是增益失控: %.5f > %.5f",
				depth, afterDetail, cleanDetail)
		}
		t.Logf("depth=%.2f  边缘/中心 %.4f -> %.4f   细节 原图 %.5f / 加暗角 %.5f / 压平后 %.5f",
			depth, beforeRatio, afterRatio, cleanDetail, vigDetail, afterDetail)
	}
}

// 已经均匀的图不该被动——多一次重编码就是白白多一层损失。
func TestFlattenSkipsCleanImage(t *testing.T) {
	const size = 384
	data := encode(t, synthTexture(size, size, 11))
	out, rep, err := FlattenLuminance(data, 1)
	if err != nil {
		t.Fatalf("失败: %v", err)
	}
	if rep.Applied {
		t.Errorf("均匀的图不该被压平（比值 %.4f）", rep.Falloff)
	}
	if !bytes.Equal(out, data) {
		t.Errorf("跳过时应当原样返回")
	}
}

func TestFlattenStrengthZeroIsNoop(t *testing.T) {
	data := encode(t, applyVignette(synthTexture(256, 256, 3), 0.4))
	out, rep, err := FlattenLuminance(data, 0)
	if err != nil {
		t.Fatalf("失败: %v", err)
	}
	if rep.Applied || !bytes.Equal(out, data) {
		t.Errorf("强度为 0 应当什么都不做")
	}
}

func TestSizeOK(t *testing.T) {
	ok := []string{"", "auto", "1024x1024", "1536x1024", "2048x2048", "2880x2880", "3840x2160"}
	for _, s := range ok {
		if err := sizeOK(s); err != nil {
			t.Errorf("%q 应当合法: %v", s, err)
		}
	}
	// 3:1 恰好允许，所以取 3088x1024（3.016:1）来验越界
	bad := map[string]string{
		"1000x1000": "16 的倍数",
		"4096x4096": "长边",
		"3088x1024": "宽高比",
		"2896x2896": "总像素", // 比 2880 只大一档就超总像素上限
		"512x512":   "总像素",
		"abc":       "宽x高",
	}
	if err := sizeOK("3072x1024"); err != nil {
		t.Errorf("3:1 应当恰好允许: %v", err)
	}
	for s, want := range bad {
		err := sizeOK(s)
		if err == nil {
			t.Errorf("%q 应当被拒（期望提到 %s）", s, want)
			continue
		}
		t.Logf("%-12s -> %s", s, err)
	}
}
