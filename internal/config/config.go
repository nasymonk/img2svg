package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Port    string
	DataDir string
}

func Load() *Config {
	cfg := &Config{
		Port:    envOrDefault("PORT", "4003"),
		DataDir: envOrDefault("DATA_DIR", "./data"),
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
