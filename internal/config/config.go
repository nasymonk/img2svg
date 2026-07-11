package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port        string
	DataDir     string
	TransDBPath string // trans app.db 路径，用于共享用户认证
	VtracerPath string // vtracer 二进制路径
	CookieSecret string
}

func Load() *Config {
	cfg := &Config{
		Port:        envOrDefault("PORT", "4003"),
		DataDir:     envOrDefault("DATA_DIR", "./data"),
		TransDBPath: envOrDefault("TRANS_DB_PATH", "./data/app.db"),
		VtracerPath: envOrDefault("VTRACER_PATH", "./bin/vtracer"),
		CookieSecret: envOrDefault("COOKIE_SECRET", ""),
	}
	// 确保数据目录存在
	os.MkdirAll(filepath.Join(cfg.DataDir, "tmp"), 0755)
	return cfg
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
