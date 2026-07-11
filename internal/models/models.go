package models

import "time"

// User 对应 trans 的 users 表（只读）
type User struct {
	ID             string
	Username       string
	PasswordHash   string
	IsAdmin        bool
	IsActive       bool
	CreatedAt      time.Time
}

// ConvertTask 矢量化任务
type ConvertTask struct {
	ID           string
	UserID       string
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
