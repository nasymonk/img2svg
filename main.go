package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/nasymonk/img2svg/internal/api"
	"github.com/nasymonk/img2svg/internal/config"
	"github.com/nasymonk/img2svg/internal/converter"
	"github.com/nasymonk/img2svg/internal/export"
	"github.com/nasymonk/img2svg/internal/middleware"
	"github.com/nasymonk/img2svg/internal/storage"
)

//go:embed web/dist
var frontend embed.FS

func main() {
	cfg := config.Load()

	// 存储
	store, err := storage.New(cfg.DataDir)
	if err != nil {
		log.Fatalf("初始化存储失败: %v", err)
	}
	defer store.Close()

	// 转换器
	convSvc := converter.New(cfg.VtracerPath, cfg.DataDir)
	if err := convSvc.CheckVtracer(); err != nil {
		log.Printf("⚠ vtracer 不可用: %v", err)
	}

	// 导出
	exportSvc := export.New(cfg.DataDir)

	// API 路由
	h := api.NewHandler(cfg, store, convSvc, exportSvc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// 静态文件（前端 SPA）
	frontendFS, err := fs.Sub(frontend, "web/dist")
	if err != nil {
		log.Fatalf("加载前端静态文件失败: %v", err)
	}
	spa := spaHandler{fs: http.FS(frontendFS)}
	mux.HandleFunc("/", spa.ServeHTTP)

	// 清理旧临时文件（24 小时前的）
	go cleanupTempFiles(cfg.DataDir)

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

// cleanupTempFiles 清理超过 24 小时的临时文件
func cleanupTempFiles(dataDir string) {
	tmpDir := filepath.Join(dataDir, "tmp")
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	n := 0
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(tmpDir, e.Name()))
			n++
		}
	}
	if n > 0 {
		log.Printf("清理了 %d 个过期临时文件", n)
	}
}
