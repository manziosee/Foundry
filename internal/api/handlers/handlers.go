package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"imgstore/internal/types"
)

type Handlers struct {
	db  *sql.DB
	svc ServiceInterface
}

type ServiceInterface interface {
	EnqueueImage(ctx context.Context, name, url, checksum string) error
	GetImageStatus(name string) (string, error)
	GetAllImages() ([]types.ImageInfo, error)
	RemoveImage(name string) error
	Cleanup() error
}



type CreateImageRequest struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func New(db *sql.DB, svc ServiceInterface) *Handlers {
	return &Handlers{db: db, svc: svc}
}

func (h *Handlers) HandleImages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listImages(w, r)
	case http.MethodPost:
		h.createImage(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) HandleImageByName(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/api/v1/images/")
	if name == "" {
		http.Error(w, "Image name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getImage(w, r, name)
	case http.MethodDelete:
		h.deleteImage(w, r, name)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) listImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.svc.GetAllImages()
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, images)
}

func (h *Handlers) createImage(w http.ResponseWriter, r *http.Request) {
	var req CreateImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.URL == "" || req.Checksum == "" {
		http.Error(w, "name, url, and checksum are required", http.StatusBadRequest)
		return
	}

	if err := h.svc.EnqueueImage(r.Context(), req.Name, req.URL, req.Checksum); err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	h.writeJSON(w, map[string]string{"status": "enqueued", "name": req.Name})
}

func (h *Handlers) getImage(w http.ResponseWriter, r *http.Request, name string) {
	state, err := h.svc.GetImageStatus(name)
	if err != nil {
		h.writeError(w, err, http.StatusNotFound)
		return
	}
	h.writeJSON(w, map[string]string{"name": name, "state": state})
}

func (h *Handlers) deleteImage(w http.ResponseWriter, r *http.Request, name string) {
	if err := h.svc.RemoveImage(name); err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple health check
	var count int
	err := h.db.QueryRow("SELECT COUNT(*) FROM images").Scan(&count)
	if err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	status := map[string]interface{}{
		"status":      "healthy",
		"image_count": count,
		"version":     "1.0.0",
	}
	h.writeJSON(w, status)
}

func (h *Handlers) HandleCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.svc.Cleanup(); err != nil {
		h.writeError(w, err, http.StatusInternalServerError)
		return
	}

	h.writeJSON(w, map[string]string{"status": "cleanup completed"})
}

func (h *Handlers) HandleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	html := `<!DOCTYPE html>
<html>
<head><title>Image Store</title></head>
<body>
<h1>Image Store API</h1>
<p>Available endpoints:</p>
<ul>
<li>GET /api/v1/images - List all images</li>
<li>POST /api/v1/images - Create new image</li>
<li>GET /api/v1/images/{name} - Get image status</li>
<li>DELETE /api/v1/images/{name} - Remove image</li>
<li>GET /api/v1/status - System status</li>
<li>POST /api/v1/cleanup - Cleanup unused blobs</li>
</ul>
</body>
</html>`
	
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
}

func (h *Handlers) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *Handlers) writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
}