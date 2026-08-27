package imagen

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"math"
	"math/rand/v2"
	"testing"
)

// srgbVignetteTexture 在 **sRGB 空间**乘暗角，而不是线性空间。
//
// 真实的图像模型输出更接近这一种：暗角是在最终 8 位像素上体现的，
// 而不是在物理线性光上。两者不等价（sRGB 上乘 g 相当于线性上乘约 g^2.2），
// 所以值得单独验一遍——线性空间那条路径已经由 TestFlattenRemovesVignette 覆盖。
func srgbVignetteTexture(size int, depth float64, seed uint64) *image.NRGBA {
	rnd := rand.New(rand.NewPCG(seed, 99))
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	cx, cy := float64(size-1)/2, float64(size-1)/2
	maxr := math.Hypot(cx, cy)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			u, v := float64(x)/float64(size), float64(y)/float64(size)
			base := 128 +
				34*math.Sin(2*math.Pi*u*17) +
				26*math.Cos(2*math.Pi*v*13) +
				18*math.Sin(2*math.Pi*(u+v)*29)
			base += (rnd.Float64() - 0.5) * 36
			r := math.Hypot(float64(x)-cx, float64(y)-cy) / maxr
			c := clamp255(base * (1 - depth*r*r))
			p := img.Pix[(y*size+x)*4:]
			p[0], p[1], p[2], p[3] = c, uint8(float64(c)*0.93), uint8(float64(c)*0.80), 255
		}
	}
	return img
}

func decodeNRGBA(t *testing.T, data []byte) *image.NRGBA {
	t.Helper()
	im, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	out := image.NewNRGBA(im.Bounds())
	draw.Draw(out, out.Bounds(), im, im.Bounds().Min, draw.Src)
	return out
}

func TestFlattenSRGBVignette(t *testing.T) {
	const size = 512
	const seed = 4242

	// 校正的靶子是纹理自身的比值，不是 1.0。
	cleanLum, _, _ := lumaOf(t, encode(t, srgbVignetteTexture(size, 0, seed)))
	target := edgeCenterRatio(cleanLum, size, size)
	t.Logf("纹理自身 边缘/中心 = %.4f（校正目标）", target)

	for _, depth := range []float64{0.15, 0.28, 0.45} {
		raw := encode(t, srgbVignetteTexture(size, depth, seed))
		beforeLum, _, _ := lumaOf(t, raw)
		before := edgeCenterRatio(beforeLum, size, size)

		out, rep, err := FlattenLuminance(raw, 1)
		if err != nil {
			t.Fatalf("depth=%.2f: %v", depth, err)
		}
		if !rep.Applied {
			t.Fatalf("depth=%.2f 没有被判定为需要压平（%.4f）", depth, rep.Falloff)
		}
		afterLum, _, _ := lumaOf(t, out)
		after := edgeCenterRatio(afterLum, size, size)

		// 报告值必须和落盘 PNG 实测的对得上——写进 manifest 的是报告值，
		// 它是从内部浮点数组算的，中间还隔着一次 8 位量化与截断。
		if d := math.Abs(rep.FalloffAfter - after); d > 0.005 {
			t.Errorf("depth=%.2f 报告 %.4f 与实测 %.4f 不符（差 %.4f）",
				depth, rep.FalloffAfter, after, d)
		}
		if d := math.Abs(after - target); d > 0.02 {
			t.Errorf("depth=%.2f 压平后 %.4f 偏离纹理自身值 %.4f 太多", depth, after, target)
		}

		// 高光不该被顶到 255 上：截顶会把亮部细节永久压平。
		im := decodeNRGBA(t, out)
		clipped := 0
		for i := 0; i < size*size; i++ {
			if p := im.Pix[i*4:]; p[0] == 255 {
				clipped++
			}
		}
		if pct := 100 * float64(clipped) / float64(size*size); pct > 0.5 {
			t.Errorf("depth=%.2f 有 %.2f%% 的像素被截顶", depth, pct)
		}
		t.Logf("depth=%.2f  %.4f -> %.4f（报告 %.4f，目标 %.4f）  截顶 %d",
			depth, before, after, rep.FalloffAfter, target, clipped)
	}
}
