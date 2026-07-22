package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port        string
	DataDir     string
	VtracerPath string // vtracer 二进制路径
}

func Load() *Config {
	cfg := &Config{
		Port:        envOrDefault("PORT", "4003"),
		DataDir:     envOrDefault("DATA_DIR", "./data"),
		VtracerPath: envOrDefault("VTRACER_PATH", "./bin/vtracer"),
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
