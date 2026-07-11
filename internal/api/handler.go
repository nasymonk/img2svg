package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/nasymonk/img2svg/internal/auth"
	"github.com/nasymonk/img2svg/internal/config"
	"github.com/nasymonk/img2svg/internal/converter"
	"github.com/nasymonk/img2svg/internal/export"
	"github.com/nasymonk/img2svg/internal/models"
	"github.com/nasymonk/img2svg/internal/preprocess"
	"github.com/nasymonk/img2svg/internal/storage"
)

type Handler struct {
	cfg       *config.Config
	store     *storage.Store
	authSvc   *auth.Service
	converter *converter.Service
	exporter  *export.Service
}

func NewHandler(cfg *config.Config, store *storage.Store, authSvc *auth.Service, conv *converter.Service, exp *export.Service) *Handler {
	return &Handler{cfg: cfg, store: store, authSvc: authSvc, converter: conv, exporter: exp}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/auth/me", h.Me)
	mux.HandleFunc("POST /api/convert", h.authWrap(h.UploadConvert))
	mux.HandleFunc("GET /api/convert/{id}/status", h.authWrap(h.ConvertStatus))
	mux.HandleFunc("GET /api/convert/{id}/preview", h.authWrap(h.ConvertPreview))
	mux.HandleFunc("GET /api/convert/{id}/source", h.authWrap(h.ConvertSource))
	mux.HandleFunc("GET /api/export/{id}/{format}", h.authWrap(h.ExportDownload))
	mux.HandleFunc("GET /api/history", h.authWrap(h.History))
	mux.HandleFunc("GET /api/health", h.Health)
}

func (h *Handler) authWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.authSvc.ValidateSession(r); err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
			return
		}
		fn(w, r)
	}
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}
	user, err := h.authSvc.VerifyPassword(req.Username, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	if err := h.authSvc.SetSessionCookie(w, user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session 设置失败"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"username": user.Username,
		"is_admin": user.IsAdmin,
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authSvc.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "已登出"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.authSvc.ValidateSession(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
}

func (h *Handler) UploadConvert(w http.ResponseWriter, r *http.Request) {
	user, _ := h.authSvc.ValidateSession(r)

	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "文件过大，最大 10MB"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请上传文件"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" && ext != ".bmp" && ext != ".gif" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "不支持的格式"})
		return
	}

	// 解析矢量化参数
	params := converter.DefaultParams()
	if v := r.FormValue("color_count"); v != "" {
		fmt.Sscanf(v, "%d", &params.ColorCount)
	}
	if v := r.FormValue("mode"); v == "binary" {
		params.Mode = "binary"
	}

	// 解析 posterize 层级（映射 UI 色彩简化为 ImageMagick posterize levels）
	posterizeLevels := 4 // 默认
	if v := r.FormValue("simplify_colors"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		switch {
		case n == 0:
			posterizeLevels = 0
		case n <= 8:
			posterizeLevels = 2
		case n <= 16:
			posterizeLevels = 3
		case n <= 32:
			posterizeLevels = 4
		default:
			posterizeLevels = 5
		}
	}

	if err := params.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	taskID := genID()
	inputPath := filepath.Join(h.cfg.DataDir, "tmp", taskID+".png")
	os.MkdirAll(filepath.Dir(inputPath), 0755)

	dst, err := os.Create(inputPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存文件失败"})
		return
	}
	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存文件失败"})
		return
	}
	dst.Close()

	// 验证图片可解码
	f, err := os.Open(inputPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取文件失败"})
		return
	}
	_, _, err = image.Decode(f)
	f.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无法解码图像"})
		return
	}

	// ImageMagick: posterize + median 消除 AI 图杂色和抗锯齿
	workPath := inputPath
	if posterizeLevels > 0 && preprocess.IsImageMagickAvailable() {
		cleanPath := filepath.Join(h.cfg.DataDir, "tmp", taskID+"_clean.png")
		if err := preprocess.Posterize(inputPath, cleanPath, posterizeLevels); err != nil {
			log.Printf("ImageMagick warning: %v", err)
		} else {
			workPath = cleanPath
		}
	}

	task := &models.ConvertTask{
		ID:           taskID,
		UserID:       user.Username,
		OriginalName: header.Filename,
		InputPath:    inputPath,
		Status:       "running",
		Progress:     0,
		Params:       params.ToJSON(),
	}
	if err := h.store.CreateTask(task); err != nil {
		log.Printf("create task error: %v", err)
	}

	go h.doConvert(taskID, workPath, params)
	writeJSON(w, http.StatusCreated, map[string]string{"id": taskID, "status": "running"})
}

func (h *Handler) doConvert(taskID, inputPath string, params converter.Params) {
	h.store.UpdateTaskStatus(taskID, "running", 30, "", "")

	svgPath, err := h.converter.Convert(inputPath, params)
	if err != nil {
		log.Printf("convert %s failed: %v", taskID, err)
		h.store.UpdateTaskStatus(taskID, "failed", 0, "", err.Error())
		return
	}

	h.store.UpdateTaskStatus(taskID, "succeeded", 100, svgPath, "")
	log.Printf("convert %s succeeded: %s", taskID, svgPath)
}

func (h *Handler) ConvertStatus(w http.ResponseWriter, r *http.Request) {
	task, err := h.store.GetTask(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "任务不存在"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id": task.ID, "status": task.Status, "progress": task.Progress,
		"error": task.ErrorMessage, "created_at": task.CreatedAt,
	})
}

func (h *Handler) ConvertPreview(w http.ResponseWriter, r *http.Request) {
	task, err := h.store.GetTask(r.PathValue("id"))
	if err != nil || task.OutputPath == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	http.ServeFile(w, r, task.OutputPath)
}

func (h *Handler) ConvertSource(w http.ResponseWriter, r *http.Request) {
	task, err := h.store.GetTask(r.PathValue("id"))
	if err != nil || task.InputPath == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, task.InputPath)
}

func (h *Handler) ExportDownload(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	format := r.PathValue("format")

	task, err := h.store.GetTask(taskID)
	if err != nil || task.OutputPath == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	cachePath := h.exporter.CachePath(task.OutputPath, format)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		exportPath, err := h.exporter.ExportPath(task.OutputPath, format)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		cachePath = exportPath
	}

	switch format {
	case "svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case "eps":
		w.Header().Set("Content-Type", "application/postscript")
	case "pdf":
		w.Header().Set("Content-Type", "application/pdf")
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.%s"`,
		strings.TrimSuffix(task.OriginalName, filepath.Ext(task.OriginalName)), format))
	http.ServeFile(w, r, cachePath)
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	user, _ := h.authSvc.ValidateSession(r)
	tasks, err := h.store.ListTasks(user.Username, 20)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if tasks == nil {
		tasks = []models.ConvertTask{}
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	vtracerOK := h.converter.CheckVtracer() == nil
	imOK := preprocess.IsImageMagickAvailable()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "ok",
		"vtracer":   vtracerOK,
		"imagemagick": imOK,
		"time":      time.Now().Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func genID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}
