package api

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nasymonk/img2svg/internal/auth"
	"github.com/nasymonk/img2svg/internal/config"
	"github.com/nasymonk/img2svg/internal/converter"
	"github.com/nasymonk/img2svg/internal/export"
	"github.com/nasymonk/img2svg/internal/models"
	"github.com/nasymonk/img2svg/internal/preprocess"
	"github.com/nasymonk/img2svg/internal/storage"

	"crypto/rand"
	"encoding/hex"
)

type Handler struct {
	cfg       *config.Config
	store     *storage.Store
	authSvc   *auth.Service
	converter *converter.Service
	exporter  *export.Service
}

func NewHandler(cfg *config.Config, store *storage.Store, authSvc *auth.Service, conv *converter.Service, exp *export.Service) *Handler {
	return &Handler{
		cfg:       cfg,
		store:     store,
		authSvc:   authSvc,
		converter: conv,
		exporter:  exp,
	}
}

// RegisterRoutes 注册所有路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 认证
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/auth/me", h.Me)

	// 矢量化
	mux.HandleFunc("POST /api/convert", h.authWrap(h.UploadConvert))
	mux.HandleFunc("GET /api/convert/{id}/status", h.authWrap(h.ConvertStatus))
	mux.HandleFunc("GET /api/convert/{id}/preview", h.authWrap(h.ConvertPreview))
	mux.HandleFunc("GET /api/convert/{id}/source", h.authWrap(h.ConvertSource))

	// 导出
	mux.HandleFunc("GET /api/export/{id}/{format}", h.authWrap(h.ExportDownload))

	// 历史
	mux.HandleFunc("GET /api/history", h.authWrap(h.History))

	// 健康检查
	mux.HandleFunc("GET /api/health", h.Health)
}

// authWrap 认证包装器
func (h *Handler) authWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := h.authSvc.ValidateSession(r)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
			return
		}
		fn(w, r)
	}
}

// Login 登录
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

// Logout 登出
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.authSvc.ClearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"message": "已登出"})
}

// Me 当前用户
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, err := h.authSvc.ValidateSession(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "未登录"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"username": user.Username})
}

// UploadConvert 上传并转换
func (h *Handler) UploadConvert(w http.ResponseWriter, r *http.Request) {
	user, _ := h.authSvc.ValidateSession(r)

	// 限制上传大小 10MB
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

	// 校验格式
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" && ext != ".bmp" && ext != ".gif" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "不支持的格式，请上传 PNG/JPG/WEBP/BMP/GIF"})
		return
	}

	// 解析预处理参数
	prepipe := preprocess.DefaultPipeline()
	if v := r.FormValue("simplify_colors"); v != "" {
		var n int
		fmt.Sscanf(v, "%d", &n)
		if n == 0 {
			prepipe.Quantize = false
		} else if n >= 2 && n <= 256 {
			prepipe.Quantize = true
			prepipe.MaxColors = n
		}
	}
	if v := r.FormValue("denoise"); v == "false" {
		prepipe.Denoise = false
	}
	if v := r.FormValue("sharpen"); v == "false" {
		prepipe.Sharpen = false
	}
	if v := r.FormValue("transparent_bg"); v == "true" {
		prepipe.TransBG = true
	}

	// 解析转换参数
	params := converter.DefaultParams()
	if v := r.FormValue("color_count"); v != "" {
		fmt.Sscanf(v, "%d", &params.ColorCount)
	}
	if v := r.FormValue("mode"); v == "binary" {
		params.Mode = "binary"
	}

	// 校验参数
	if err := params.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// 生成任务 ID 和文件路径
	taskID := genID()
	inputPath := filepath.Join(h.cfg.DataDir, "tmp", taskID+".png")
	os.MkdirAll(filepath.Dir(inputPath), 0755)

	// 保存上传文件
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

	// 重新打开解码图像
	dst, err = os.Open(inputPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "读取文件失败"})
		return
	}
	img, _, err := image.Decode(dst)
	dst.Close()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "无法解码图像"})
		return
	}

	// 预处理
	processed := prepipe.Process(img)
	if err := preprocess.SavePNG(processed, inputPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "预处理失败"})
		return
	}

	// 创建任务记录
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

	// 异步执行矢量化
	go h.doConvert(taskID, inputPath, params)

	writeJSON(w, http.StatusCreated, map[string]string{"id": taskID, "status": "running"})
}

func (h *Handler) doConvert(taskID, inputPath string, params converter.Params) {
	// 更新状态为 running
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

// ConvertStatus 查询转换状态
func (h *Handler) ConvertStatus(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	task, err := h.store.GetTask(taskID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "任务不存在"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         task.ID,
		"status":     task.Status,
		"progress":   task.Progress,
		"error":      task.ErrorMessage,
		"created_at": task.CreatedAt,
	})
}

// ConvertPreview 获取 SVG 预览
func (h *Handler) ConvertPreview(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	task, err := h.store.GetTask(taskID)
	if err != nil || task.OutputPath == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	http.ServeFile(w, r, task.OutputPath)
}

// ConvertSource 获取原图
func (h *Handler) ConvertSource(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	task, err := h.store.GetTask(taskID)
	if err != nil || task.InputPath == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, task.InputPath)
}

// ExportDownload 导出下载
func (h *Handler) ExportDownload(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	format := r.PathValue("format")

	task, err := h.store.GetTask(taskID)
	if err != nil || task.OutputPath == "" {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// 检查缓存
	cachePath := h.exporter.CachePath(task.OutputPath, format)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		// 执行导出
		exportPath, err := h.exporter.ExportPath(task.OutputPath, format)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		cachePath = exportPath
	}

	// 设置下载头
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

// History 历史记录
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

// Health 健康检查
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	vtracerOK := h.converter.CheckVtracer() == nil
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "ok",
		"vtracer": vtracerOK,
		"time":    time.Now().Format(time.RFC3339),
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
