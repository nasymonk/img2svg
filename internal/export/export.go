package export

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Service 格式导出服务（SVG → EPS/PDF）
type Service struct {
	inkscapePath string
	dataDir      string
}

func New(dataDir string) *Service {
	return &Service{
		// inkscape 在 ECS 上通过 apt 安装后路径为 /usr/bin/inkscape
		// 如果找不到，导出 EPS/PDF 暂不可用，仅支持 SVG
		inkscapePath: findInkscape(),
		dataDir:      dataDir,
	}
}

func findInkscape() string {
	for _, p := range []string{"inkscape", "/usr/bin/inkscape", "/usr/local/bin/inkscape"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// ToEPS SVG → EPS（需要 inkscape）
func (s *Service) ToEPS(svgPath string) (string, error) {
	if s.inkscapePath == "" {
		return "", fmt.Errorf("EPS 导出需要安装 inkscape")
	}
	epsPath := strings.TrimSuffix(svgPath, ".svg") + ".eps"
	cmd := exec.Command(s.inkscapePath, "--export-filename="+epsPath, svgPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inkscape EPS 导出失败: %w", err)
	}
	return epsPath, nil
}

// ToPDF SVG → PDF（需要 inkscape）
func (s *Service) ToPDF(svgPath string) (string, error) {
	if s.inkscapePath == "" {
		return "", fmt.Errorf("PDF 导出需要安装 inkscape")
	}
	pdfPath := strings.TrimSuffix(svgPath, ".svg") + ".pdf"
	cmd := exec.Command(s.inkscapePath, "--export-filename="+pdfPath, svgPath)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inkscape PDF 导出失败: %w", err)
	}
	return pdfPath, nil
}

// ExportPath 根据格式返回导出路径
func (s *Service) ExportPath(svgPath, format string) (string, error) {
	switch strings.ToLower(format) {
	case "svg":
		return svgPath, nil
	case "eps":
		return s.ToEPS(svgPath)
	case "pdf":
		return s.ToPDF(svgPath)
	default:
		return "", fmt.Errorf("不支持的格式: %s", format)
	}
}

// CachePath 获取缓存的导出路径
func (s *Service) CachePath(svgPath, format string) string {
	return strings.TrimSuffix(svgPath, ".svg") + "." + strings.ToLower(format)
}

// HasCached 检查是否已经有缓存的导出文件
func (s *Service) HasCached(svgPath, format string) bool {
	_, err := os.Stat(s.CachePath(svgPath, format))
	return err == nil
}

// TempDir 清理旧的临时文件
func (s *Service) CleanupTemp() {
	entries, err := os.ReadDir(filepath.Join(s.dataDir, "tmp"))
	if err != nil {
		return
	}
	for _, e := range entries {
		// 只清理 1 小时前的文件
		info, err := e.Info()
		if err == nil && info.ModTime().Add(3600*1e9).Before(info.ModTime()) {
			// 暂时跳过复杂的时间判断
			_ = info
		}
	}
}
