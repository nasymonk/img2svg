package preprocess

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"sort"
)

// Denoise 去噪（中值滤波 3x3），图像至少 3x3 才执行
func Denoise(img image.Image) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 3 || h < 3 {
		return img
	}
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
	sort.Ints(vals)
	return uint8(vals[len(vals)/2])
}

// Sharpen 边缘增强（拉普拉斯算子），图像至少 3x3 才执行
func Sharpen(img image.Image, strength float64) image.Image {
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 3 || h < 3 {
		return img
	}
	out := image.NewRGBA(bounds)

	for y := bounds.Min.Y + 1; y < bounds.Max.Y-1; y++ {
		for x := bounds.Min.X + 1; x < bounds.Max.X-1; x++ {
			r, g, b := laplacian(img, x, y, strength)
			out.Set(x, y, &color.RGBA{r, g, b, 255})
		}
	}
	// 边界拷贝
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

func laplacian(img image.Image, x, y int, strength float64) (uint8, uint8, uint8) {
	rc, gc, bc, _ := img.At(x, y).RGBA()
	rt, gt, bt, _ := img.At(x, y-1).RGBA()
	rb, gb, bb, _ := img.At(x, y+1).RGBA()
	rl, gl, bl, _ := img.At(x-1, y).RGBA()
	rr, gr, br, _ := img.At(x+1, y).RGBA()

	rLap := float64(rc>>8) - 0.25*(float64(rt>>8)+float64(rb>>8)+float64(rl>>8)+float64(rr>>8))
	gLap := float64(gc>>8) - 0.25*(float64(gt>>8)+float64(gb>>8)+float64(gl>>8)+float64(gr>>8))
	bLap := float64(bc>>8) - 0.25*(float64(bt>>8)+float64(bb>>8)+float64(bl>>8)+float64(br>>8))

	rSum := math.Max(0, math.Min(255, float64(rc>>8)+strength*rLap))
	gSum := math.Max(0, math.Min(255, float64(gc>>8)+strength*gLap))
	bSum := math.Max(0, math.Min(255, float64(bc>>8)+strength*bLap))

	return uint8(rSum), uint8(gSum), uint8(bSum)
}

// Quantize 简单色彩量化
func Quantize(img image.Image, maxColors int) image.Image {
	bounds := img.Bounds()
	out := image.NewRGBA(bounds)
	div := 256 / int(math.Sqrt(float64(maxColors)))
	if div < 1 {
		div = 1
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			out.Set(x, y, &color.RGBA{
				uint8((int(r>>8) / div) * div),
				uint8((int(g>>8) / div) * div),
				uint8((int(b>>8) / div) * div),
				uint8(a >> 8),
			})
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
			r, g, b, _ := out.At(x, y).RGBA()
			if r>>8 > 248 && g>>8 > 248 && b>>8 > 248 {
				out.Set(x, y, &color.RGBA{0, 0, 0, 0})
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
	Denoise    bool
	Sharpen    bool
	SharpenStr float64
	Quantize   bool
	MaxColors  int
	TransBG    bool
}

func DefaultPipeline() Pipeline {
	return Pipeline{
		Denoise:    true,
		Sharpen:    true,
		SharpenStr: 0.3,
		Quantize:   true,
		MaxColors:  32,
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
