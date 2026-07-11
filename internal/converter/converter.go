package converter

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Params vtracer 转换参数
type Params struct {
	ColorCount  int    `json:"color_count"`   // 颜色数量 (2-64)
	PathPrecision int  `json:"path_precision"` // 路径精度 (0-100, vtracer 的 filter_speckle 阈值)
	CornerThreshold int `json:"corner_threshold"` // 角点阈值 (0-100, 越大线条越平滑)
	Mode         string `json:"mode"`          // "color" 彩色, "binary" 黑白
	LayerMode    string `json:"layer_mode"`    // "split" 分层, "flat" 单层
	OutputScale  float64 `json:"output_scale"` // 输出缩放
}

func DefaultParams() Params {
	return Params{
		ColorCount:      16,
		PathPrecision:   4,
		CornerThreshold: 60,
		Mode:            "color",
		LayerMode:       "split",
		OutputScale:     1.0,
	}
}

// Service vtracer 调用封装
type Service struct {
	vtracerPath string
	dataDir     string
}

func New(vtracerPath, dataDir string) *Service {
	return &Service{
		vtracerPath: vtracerPath,
		dataDir:     dataDir,
	}
}

// Convert 执行矢量化，输入 PNG 路径，返回输出 SVG 路径
func (s *Service) Convert(inputPath string, p Params) (string, error) {
	// 生成输出路径
	base := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outPath := filepath.Join(s.dataDir, "tmp", base+".svg")

	os.MkdirAll(filepath.Dir(outPath), 0755)

	args := []string{
		"--input", inputPath,
		"--output", outPath,
	}

	// 模式
	if p.Mode == "binary" {
		args = append(args, "--mode", "binary")
	} else {
		args = append(args, "--mode", "color")
		args = append(args, "--color_precision", fmt.Sprintf("%d", p.ColorCount))
	}

	// 路径精度
	if p.PathPrecision > 0 {
		args = append(args, "--path_precision", fmt.Sprintf("%d", p.PathPrecision))
	}

	// 角点阈值
	if p.CornerThreshold > 0 {
		args = append(args, "--corner_threshold", fmt.Sprintf("%d", p.CornerThreshold))
	}

	// 分层模式
	if p.LayerMode == "split" {
		args = append(args, "--layer_mode", "split")
	}

	// 输出缩放
	if p.OutputScale != 1.0 && p.OutputScale > 0 {
		args = append(args, "--output_scale", fmt.Sprintf("%.1f", p.OutputScale))
	}

	cmd := exec.Command(s.vtracerPath, args...)
	cmd.Stderr = nil
	output, err := cmd.CombinedOutput()
	if err != nil {
		// vtracer 返回非零时仍然可能生成了文件
		if _, statErr := os.Stat(outPath); os.IsNotExist(statErr) {
			return "", fmt.Errorf("vtracer 执行失败: %w, 输出: %s", err, string(output))
		}
	}
	return outPath, nil
}

// CheckVtracer 检查 vtracer 是否可用
func (s *Service) CheckVtracer() error {
	cmd := exec.Command(s.vtracerPath, "--version")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("vtracer 不可用: %w", err)
	}
	_ = out
	return nil
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
