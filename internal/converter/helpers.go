package converter

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// RGB 颜色
type RGB struct {
	R, G, B uint8
}

// countColors 用 ImageMagick 统计唯一颜色数
func countColors(path string) (int, error) {
	cmd := exec.Command("identify", "-format", "%k", path)
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

// extractColors 提取 posterize 后图片的颜色直方图，过滤小色斑
func extractColors(path string) ([]RGB, error) {
	// 直接统计当前图片颜色，不做二次量化
	cmd := exec.Command("convert", path,
		"-depth", "8",
		"-format", "%c", "histogram:info:",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("histogram: %w", err)
	}

	type colorCount struct {
		c RGB
		n int
	}
	var colors []colorCount
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		idx := strings.Index(line, "#")
		if idx < 0 || idx+7 > len(line) {
			continue
		}
		// 解析像素计数
		fields := strings.Fields(line[:idx])
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		hex := line[idx : idx+7]
		c := hexToRGB(hex)

		// 跳过白色背景和极小色斑 (< 200 像素)
		if isNearWhite(c) || n < 200 {
			continue
		}
		colors = append(colors, colorCount{c, n})
	}

	// 按像素数降序排列（大面积色块优先）
	for i := 0; i < len(colors); i++ {
		for j := i + 1; j < len(colors); j++ {
			if colors[j].n > colors[i].n {
				colors[i], colors[j] = colors[j], colors[i]
			}
		}
	}

	// 最多 16 层
	var result []RGB
	for i, cc := range colors {
		if i >= 16 {
			break
		}
		result = append(result, cc.c)
	}
	return result, nil
}

func rgbToHex(c RGB) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

func hexToRGB(hex string) RGB {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return RGB{}
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return RGB{uint8(r), uint8(g), uint8(b)}
}

func isNearWhite(c RGB) bool {
	return c.R > 245 && c.G > 245 && c.B > 245
}

// extractPathElements 从 potrace SVG 中提取 <path> 元素
func extractPathElements(svg string) string {
	var paths []string
	lines := strings.Split(svg, "\n")
	inPath := false
	var pathBuf strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "<path ") {
			inPath = true
			pathBuf.Reset()
			pathBuf.WriteString(trimmed)
			if strings.Contains(trimmed, "/>") {
				paths = append(paths, pathBuf.String())
				inPath = false
			}
		} else if inPath {
			pathBuf.WriteString(trimmed)
			if strings.Contains(trimmed, "/>") || strings.Contains(trimmed, "</path>") {
				paths = append(paths, pathBuf.String())
				inPath = false
			}
		}
	}
	return strings.Join(paths, "\n")
}

// mergeSVG 合并多个图层为完整 SVG
func mergeSVG(inputPath string, layers []string) string {
	var buf bytes.Buffer

	// 从原图获取尺寸
	width, height := 800, 600
	cmd := exec.Command("identify", "-format", "%w %h", inputPath)
	out, err := cmd.Output()
	if err == nil {
		parts := strings.Fields(string(out))
		if len(parts) == 2 {
			w, _ := strconv.Atoi(parts[0])
			h, _ := strconv.Atoi(parts[1])
			width, height = w, h
		}
	}

	buf.WriteString(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<svg version="1.1" xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">
`, width, height, width, height))

	// 白色背景
	buf.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="white"/>`+"\n", width, height))

	for _, layer := range layers {
		if layer != "" {
			buf.WriteString(layer)
			buf.WriteString("\n")
		}
	}

	buf.WriteString("</svg>\n")
	return buf.String()
}
