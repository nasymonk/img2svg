package models

import "time"

// ConvertTask 矢量化任务
type ConvertTask struct {
	ID           string
	OriginalName string
	InputPath    string
	OutputPath   string // SVG 输出路径
	Status       string // pending, running, succeeded, failed
	Progress     int    // 0-100
	Params       string // JSON: 转换参数
	ErrorMessage string
	CreatedAt    time.Time
	FinishedAt   *time.Time
}
