package preprocess

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
)

// RemoveAntialiasing 用 ImageMagick median 滤波消除抗锯齿
// radius: 滤波半径 (1-3)，越大越激进；不改变颜色，只消除过渡像素
func RemoveAntialiasing(inputPath, outputPath string, radius int) error {
	cmd := exec.Command("convert", inputPath,
		"-median", fmt.Sprintf("%d", radius),
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ImageMagick median 失败: %w\n输出: %s", err, string(output))
	}
	return nil
}

// Posterize 减少颜色层级（可选），用于大量杂色时需要减少颜色
// 不要和 median 一起用，分两步调
func Posterize(inputPath, outputPath string, levels int) error {
	cmd := exec.Command("convert", inputPath,
		"-posterize", fmt.Sprintf("%d", levels),
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ImageMagick posterize 失败: %w\n输出: %s", err, string(output))
	}
	return nil
}

// IsImageMagickAvailable 检查 ImageMagick 是否可用
func IsImageMagickAvailable() bool {
	return exec.Command("convert", "-version").Run() == nil
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

// Pipeline 预处理配置（保留用于参数传递，实际处理走 ImageMagick）
type Pipeline struct {
	PosterizeLevels int  // ImageMagick posterize 层级 (2-8)
	TransBG         bool // 透明背景
}

func DefaultPipeline() Pipeline {
	return Pipeline{
		PosterizeLevels: 4,
		TransBG:         false,
	}
}
