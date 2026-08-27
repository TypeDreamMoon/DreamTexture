package material

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
)

func bytesReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }

// TilePreview 把基础色拼成 3×3，用于列表缩略与接缝验收。
//
// 无缝是这套工具的硬指标，而接缝只有拼起来才看得见——所以预览图不是装饰，
// 是验收工具，每个套装都必须有。
func TilePreview(basecolor []byte, cell int) ([]byte, error) {
	src, err := png.Decode(bytesReader(basecolor))
	if err != nil {
		return nil, err
	}
	small := downscale(src, cell)
	out := image.NewRGBA(image.Rect(0, 0, cell*3, cell*3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			draw.Draw(out, image.Rect(x*cell, y*cell, (x+1)*cell, (y+1)*cell),
				small, image.Point{}, draw.Src)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// downscale 用盒式滤波缩图。贴图预览不追求重采样质量，但必须做面积平均——
// 最近邻会把高频细节采成摩尔纹，反而让人误以为有接缝。
func downscale(src image.Image, size int) image.Image {
	b := src.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	sw, sh := b.Dx(), b.Dy()
	for y := 0; y < size; y++ {
		y0, y1 := b.Min.Y+y*sh/size, b.Min.Y+(y+1)*sh/size
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < size; x++ {
			x0, x1 := b.Min.X+x*sw/size, b.Min.X+(x+1)*sw/size
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, bl, n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					cr, cg, cb, _ := src.At(sx, sy).RGBA()
					r += uint64(cr >> 8)
					g += uint64(cg >> 8)
					bl += uint64(cb >> 8)
					n++
				}
			}
			if n == 0 {
				continue
			}
			i := dst.PixOffset(x, y)
			dst.Pix[i] = uint8(r / n)
			dst.Pix[i+1] = uint8(g / n)
			dst.Pix[i+2] = uint8(bl / n)
			dst.Pix[i+3] = 255
		}
	}
	return dst
}
