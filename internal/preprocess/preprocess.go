package preprocess

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
)

// Preprocess 科研图预处理：去噪、边缘增强、色彩量化
// 提升 vtracer 矢量化质量

// Denoise 去噪（中值滤波 3x3）
func Denoise(img image.Image) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)

	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			out.Set(x, y, medianPixel(img, x, y))
		}
	}
	// 边界保持不变
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		out.Set(x, bounds.Min.Y, img.At(x, bounds.Min.Y))
		out.Set(x, bounds.Max.Y-1, img.At(x, bounds.Max.Y-1))
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		out.Set(bounds.Min.X, y, img.At(bounds.Min.X, y))
		out.Set(bounds.Max.X-1, y, img.At(bounds.Max.X-1, y))
	}
	return out
}

func medianPixel(img image.Image, x, y int) color.Color {
	var rs, gs, bs []int
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			r, g, b, _ := img.At(x+dx, y+dy).RGBA()
			rs = append(rs, int(r>>8))
			gs = append(gs, int(g>>8))
			bs = append(bs, int(b>>8))
		}
	}
	return color.RGBA{median(rs), median(gs), median(bs), 255}
}

func median(vals []int) uint8 {
	for i := 0; i < len(vals); i++ {
		for j := i + 1; j < len(vals); j++ {
			if vals[i] > vals[j] {
				vals[i], vals[j] = vals[j], vals[i]
			}
		}
	}
	return uint8(vals[len(vals)/2])
}

// Sharpen 边缘增强（拉普拉斯算子）
func Sharpen(img image.Image, strength float64) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	// 拉普拉斯核: [0, -1, 0; -1, 4, -1; 0, -1, 0]
	// 增强 = 原图 + strength * 拉普拉斯

	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			r, g, b := laplacian(img, x, y, strength)
			out.Set(x, y, &color.RGBA{r, g, b, 255})
		}
	}
	return out
}

func laplacian(img image.Image, x, y int, strength float64) (uint8, uint8, uint8) {
	var rSum, gSum, bSum float64

	// 原中心像素
	rc, gc, bc, _ := img.At(x, y).RGBA()

	// 上下左右
	rt, gt, bt, _ := img.At(x, y-1).RGBA()
	rb, gb, bb, _ := img.At(x, y+1).RGBA()
	rl, gl, bl, _ := img.At(x-1, y).RGBA()
	rr, gr, br, _ := img.At(x+1, y).RGBA()

	rLap := float64(rc>>8) - 0.25*(float64(rt>>8)+float64(rb>>8)+float64(rl>>8)+float64(rr>>8))
	gLap := float64(gc>>8) - 0.25*(float64(gt>>8)+float64(gb>>8)+float64(gl>>8)+float64(gr>>8))
	bLap := float64(bc>>8) - 0.25*(float64(bt>>8)+float64(bb>>8)+float64(bl>>8)+float64(br>>8))

	rSum = float64(rc>>8) + strength*rLap
	gSum = float64(gc>>8) + strength*gLap
	bSum = float64(bc>>8) + strength*bLap

	rSum = math.Max(0, math.Min(255, rSum))
	gSum = math.Max(0, math.Min(255, gSum))
	bSum = math.Max(0, math.Min(255, bSum))

	return uint8(rSum), uint8(gSum), uint8(bSum)
}

// Quantize 简单色彩量化（减少颜色数）
func Quantize(img image.Image, maxColors int) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)

	// 将每个像素颜色除以下取整再乘回，减少颜色数
	div := 256 / int(math.Sqrt(float64(maxColors)))
	if div < 1 {
		div = 1
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			rr := uint8((int(r>>8) / div) * div)
			gg := uint8((int(g>>8) / div) * div)
			bb := uint8((int(b>>8) / div) * div)
			out.Set(x, y, &color.RGBA{rr, gg, bb, uint8(a >> 8)})
		}
	}
	return out
}

// WhiteToTransparent 白底转透明
func WhiteToTransparent(img image.Image) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	draw.Draw(out, bounds, img, bounds.Min, draw.Src)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := out.At(x, y).RGBA()
			// 白色或接近白色 → 透明
			if r>>8 > 248 && g>>8 > 248 && b>>8 > 248 {
				out.Set(x, y, &color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0})
				_ = a
			}
		}
	}
	return out
}

// SavePNG 保存为 PNG
func SavePNG(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Pipeline 预处理流水线
type Pipeline struct {
	Denoise  bool
	Sharpen    bool
	SharpenStr float64
	Quantize   bool
	MaxColors  int
	TransBG    bool // 透明背景
}

func DefaultPipeline() Pipeline {
	return Pipeline{
		Denoise:    true,
		Sharpen:    true,
		SharpenStr: 0.3,
		Quantize:   false,
		MaxColors:  16,
		TransBG:    false,
	}
}

func (pl Pipeline) Process(img image.Image) image.Image {
	result := img
	if pl.Denoise {
		result = Denoise(result)
	}
	if pl.Sharpen {
		result = Sharpen(result, pl.SharpenStr)
	}
	if pl.Quantize {
		result = Quantize(result, pl.MaxColors)
	}
	if pl.TransBG {
		result = WhiteToTransparent(result)
	}
	return result
}
