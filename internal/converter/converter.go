package converter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Params vtracer 转换参数
// UI 颜色数 2-64 映射到 vtracer color_precision (每通道位数, 1-8)
type Params struct {
	ColorCount      int    `json:"color_count"`      // UI 颜色数 (2-64)
	PathPrecision   int    `json:"path_precision"`   // 路径精度 (1-10)
	CornerThreshold int    `json:"corner_threshold"` // 角点阈值 (1-100)
	Mode            string `json:"mode"`             // "color" 彩色, "binary" 黑白
}

// colorPrecision 将 UI 颜色数映射为 vtracer color_precision (1-8)
func colorPrecision(uiCount int) int {
	// ceil(log2(count)), clamped to [1,8]
	p := int(math.Ceil(math.Log2(float64(uiCount))))
	if p < 1 {
		p = 1
	}
	if p > 8 {
		p = 8
	}
	return p
}

// Validate 校验参数合法性
func (p *Params) Validate() error {
	if p.ColorCount < 2 || p.ColorCount > 64 {
		return fmt.Errorf("颜色数必须在 2-64 之间，当前: %d", p.ColorCount)
	}
	if p.PathPrecision < 1 || p.PathPrecision > 10 {
		return fmt.Errorf("路径精度必须在 1-10 之间")
	}
	if p.CornerThreshold < 1 || p.CornerThreshold > 100 {
		return fmt.Errorf("角点阈值必须在 1-100 之间")
	}
	if p.Mode != "color" && p.Mode != "binary" {
		return fmt.Errorf("模式必须是 color 或 binary")
	}
	return nil
}

func DefaultParams() Params {
	return Params{
		ColorCount:      16,
		PathPrecision:   4,
		CornerThreshold: 60,
		Mode:            "color",
	}
}

// Service vtracer 调用封装
type Service struct {
	vtracerPath string
	dataDir     string
	timeout     time.Duration
}

func New(vtracerPath, dataDir string) *Service {
	return &Service{
		vtracerPath: vtracerPath,
		dataDir:     dataDir,
		timeout:     120 * time.Second,
	}
}

// Convert 执行矢量化
func (s *Service) Convert(inputPath string, p Params) (string, error) {
	if err := p.Validate(); err != nil {
		return "", err
	}

	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outPath := filepath.Join(s.dataDir, "tmp", base+".svg")
	os.MkdirAll(filepath.Dir(outPath), 0755)

	args := []string{
		"--input", inputPath,
		"--output", outPath,
		"--mode", "spline",
		"--path_precision", fmt.Sprintf("%d", p.PathPrecision),
		"--corner_threshold", fmt.Sprintf("%d", p.CornerThreshold),
	}

	if p.Mode == "binary" {
		args = append(args, "--colormode", "bw")
	} else {
		args = append(args, "--colormode", "color")
		args = append(args, "--color_precision", fmt.Sprintf("%d", colorPrecision(p.ColorCount)))
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.vtracerPath, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("矢量化超时（>%v），请尝试降低颜色数或使用更小的图片", s.timeout)
	}
	if err != nil {
		if _, statErr := os.Stat(outPath); os.IsNotExist(statErr) {
			return "", fmt.Errorf("vtracer 执行失败: %w\n输出: %s", err, string(output))
		}
	}
	return outPath, nil
}

// CheckVtracer 检查 vtracer 是否可用
func (s *Service) CheckVtracer() error {
	cmd := exec.Command(s.vtracerPath, "--version")
	return cmd.Run()
}

func (p Params) ToJSON() string {
	b, _ := json.Marshal(p)
	return string(b)
}

func ParamsFromJSON(s string) Params {
	p := DefaultParams()
	json.Unmarshal([]byte(s), &p)
	return p
}
