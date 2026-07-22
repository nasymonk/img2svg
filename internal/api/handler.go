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

	"github.com/nasymonk/img2svg/internal/config"
	"github.com/nasymonk/img2svg/internal/converter"
	"github.com/nasymonk/img2svg/internal/export"
	"github.com/nasymonk/img2svg/internal/models"
	"github.com/nasymonk/img2svg/internal/storage"
)

type Handler struct {
	cfg       *config.Config
	store     *storage.Store
	converter *converter.Service
	exporter  *export.Service
}

func NewHandler(cfg *config.Config, store *storage.Store, conv *converter.Service, exp *export.Service) *Handler {
	return &Handler{cfg: cfg, store: store, converter: conv, exporter: exp}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/convert", h.UploadConvert)
	mux.HandleFunc("GET /api/convert/{id}/status", h.ConvertStatus)
	mux.HandleFunc("GET /api/convert/{id}/preview", h.ConvertPreview)
	mux.HandleFunc("GET /api/convert/{id}/source", h.ConvertSource)
	mux.HandleFunc("GET /api/export/{id}/{format}", h.ExportDownload)
	mux.HandleFunc("GET /api/history", h.History)
	mux.HandleFunc("GET /api/health", h.Health)
}

func (h *Handler) UploadConvert(w http.ResponseWriter, r *http.Request) {
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

	params := converter.DefaultParams()
	task := &models.ConvertTask{
		ID:           taskID,
		OriginalName: header.Filename,
		InputPath:    inputPath,
		Status:       "running",
		Progress:     0,
		Params:       params.ToJSON(),
	}
	if err := h.store.CreateTask(task); err != nil {
		log.Printf("create task error: %v", err)
	}

	go h.doConvert(taskID, inputPath, params)
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
	tasks, err := h.store.ListTasks(20)
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
	ok := h.converter.CheckVtracer() == nil
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"ready":  ok,
		"time":   time.Now().Format(time.RFC3339),
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
