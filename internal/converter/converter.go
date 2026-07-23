package converter

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Params 保留兼容（自动检测，参数不再使用）
type Params struct{}
func DefaultParams() Params          { return Params{} }
func (p *Params) Validate() error    { return nil }
func (p Params) ToJSON() string      { return "{}" }

type Service struct {
	dataDir string
	timeout time.Duration
}

func New(dataDir string) *Service {
	return &Service{dataDir: dataDir, timeout: 120 * time.Second}
}

func (s *Service) Convert(inputPath string, _ Params) (string, error) {
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outPath := filepath.Join(s.dataDir, "tmp", base+".svg")
	os.MkdirAll(filepath.Dir(outPath), 0755)

	// 1. 自动统计颜色数，决定 posterize 级别
	colorCount, err := countColors(inputPath)
	if err != nil {
		return "", fmt.Errorf("统计颜色失败: %w", err)
	}

	var posterizeLevel int
	switch {
	case colorCount < 64:
		posterizeLevel = 0
	case colorCount < 256:
		posterizeLevel = 6
	case colorCount < 1024:
		posterizeLevel = 5
	default:
		posterizeLevel = 4
	}

	// 2. Posterize + median 去噪
	workPath := inputPath
	if posterizeLevel > 0 {
		pp := filepath.Join(s.dataDir, "tmp", base+"_posterized.png")
		cmd := exec.Command("convert", inputPath,
			"-posterize", strconv.Itoa(posterizeLevel),
			"-median", "2", pp,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("posterize 失败: %w, %s", err, string(out))
		}
		workPath = pp
	}

	// 3. 提取颜色 → potrace 逐层描边 → 合并
	colors, err := extractColors(workPath)
	if err != nil {
		return "", fmt.Errorf("提取颜色失败: %w", err)
	}
	if len(colors) > 64 {
		colors = colors[:64] // 最多 64 层
	}

	var layers []string
	for i, c := range colors {
		maskPath := filepath.Join(s.dataDir, "tmp", fmt.Sprintf("%s_mask_%d.pbm", base, i))
		svgPath := filepath.Join(s.dataDir, "tmp", fmt.Sprintf("%s_layer_%d.svg", base, i))
		hex := rgbToHex(c)

		// 生成黑白 mask + morphology smooth 消除锯齿
		cmd := exec.Command("convert", workPath,
			"-fill", "black", "-fuzz", "0%", "-opaque", hex,
			"-fill", "white", "+opaque", "black",
			"-morphology", "Smooth", "Octagon:3",
			maskPath,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("mask 失败: %w, %s", err, string(out))
		}

		// potrace
		cmd = exec.Command("potrace", maskPath, "-o", svgPath, "-b", "svg", "--turdsize", "20")
		out, _ := cmd.CombinedOutput()
		_ = out
		data, err := os.ReadFile(svgPath)
		os.Remove(maskPath)
		os.Remove(svgPath)
		if err != nil || len(data) == 0 {
			continue
		}

		data = bytes.ReplaceAll(data, []byte(`fill="#000000"`), []byte(fmt.Sprintf(`fill="%s"`, hex)))
		layers = append(layers, extractPathElements(string(data)))
	}

	// 4. 合并
	merged := mergeSVG(inputPath, layers)
	if err := os.WriteFile(outPath, []byte(merged), 0644); err != nil {
		return "", fmt.Errorf("写入 SVG 失败: %w", err)
	}
	return outPath, nil
}

func (s *Service) CheckDeps() error {
	if err := exec.Command("convert", "-version").Run(); err != nil {
		return fmt.Errorf("ImageMagick 不可用: %w", err)
	}
	if err := exec.Command("potrace", "--version").Run(); err != nil {
		return fmt.Errorf("potrace 不可用: %w", err)
	}
	return nil
}
