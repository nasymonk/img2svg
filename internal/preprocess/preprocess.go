package preprocess

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
)

// Posterize 用 ImageMagick 做专业的颜色量化和去抗锯齿
// levels: 颜色层级 (2-8, 越小越少颜色)
// 内部调用: convert input -posterize levels -median 2 output
func Posterize(inputPath, outputPath string, levels int) error {
	// ImageMagick 的 posterize 做了正确的颜色聚类，median 消除抗锯齿
	medianRadius := 2
	if levels <= 2 {
		medianRadius = 3 // 更激进去锯齿
	}
	cmd := exec.Command("convert",
		inputPath,
		"-posterize", fmt.Sprintf("%d", levels),
		"-median", fmt.Sprintf("%d", medianRadius),
		outputPath,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ImageMagick 处理失败: %w\n输出: %s", err, string(output))
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
