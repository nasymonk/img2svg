package main

import (
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/nasymonk/img2svg/internal/api"
	"github.com/nasymonk/img2svg/internal/auth"
	"github.com/nasymonk/img2svg/internal/config"
	"github.com/nasymonk/img2svg/internal/converter"
	"github.com/nasymonk/img2svg/internal/export"
	"github.com/nasymonk/img2svg/internal/middleware"
	"github.com/nasymonk/img2svg/internal/storage"
)

//go:embed web/dist/*
var frontend embed.FS

func main() {
	cfg := config.Load()

	// 存储
	store, err := storage.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}
	defer store.Close()

	// 认证（只读 trans 数据库）
	if cfg.CookieSecret == "" {
		log.Fatal("请设置 COOKIE_SECRET 环境变量（至少 32 字符）")
	}
	authSvc, err := auth.New(cfg.TransDBPath, cfg.CookieSecret)
	if err != nil {
		log.Fatalf("初始化认证失败: %v", err)
	}
	defer authSvc.Close()

	// 转换器
	convSvc := converter.New(cfg.VtracerPath, cfg.DataDir)
	if err := convSvc.CheckVtracer(); err != nil {
		log.Printf("⚠ vtracer 不可用: %v", err)
	}

	// 导出
	exportSvc := export.New(cfg.DataDir)

	// API 路由
	h := api.NewHandler(cfg, store, authSvc, convSvc, exportSvc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 静态文件（前端 SPA）
	spa := spaHandler{fs: http.FS(frontend)}
	mux.HandleFunc("/", spa.ServeHTTP)

	// 中间件链
	handler := middleware.Logger(middleware.Secure(mux))

	addr := ":" + cfg.Port
	log.Printf("img2svg 启动于 http://0.0.0.0%s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

// spaHandler SPA fallback：找不到文件时返回 index.html
type spaHandler struct {
	fs http.FileSystem
}

func (s *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// API 路由不 fallback
	if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
		http.NotFound(w, r)
		return
	}

	f, err := s.fs.Open(r.URL.Path)
	if os.IsNotExist(err) {
		// SPA fallback
		index, err := s.fs.Open("/index.html")
		if err != nil {
			http.Error(w, "index.html not found", http.StatusInternalServerError)
			return
		}
		defer index.Close()
		stat, _ := index.Stat()
		http.ServeContent(w, r, "index.html", stat.ModTime(), index)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	f.Close()

	http.FileServer(s.fs).ServeHTTP(w, r)
}
