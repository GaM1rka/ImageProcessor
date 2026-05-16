package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"imageprocessor/backend/internal/models"
	"imageprocessor/backend/internal/queue"
	"imageprocessor/backend/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
)

type Handler struct {
	store    storage.Store
	producer queue.Producer
}

func NewHandler(store storage.Store, producer queue.Producer) *Handler {
	return &Handler{store: store, producer: producer}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/upload", h.upload)
	r.Get("/image/{id}", h.getImage)
	r.Delete("/image/{id}", h.deleteImage)
	r.Get("/status/{id}", h.getStatus)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	return r
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(20 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	now := time.Now().UTC()
	image := models.Image{
		ID:           uuid.NewString(),
		OriginalName: header.Filename,
		ContentType:  header.Header.Get("Content-Type"),
		Status:       models.StatusQueued,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.store.SaveOriginal(r.Context(), image, file); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save image")
		return
	}

	if err := h.producer.Publish(r.Context(), models.ProcessMessage{ID: image.ID}); err != nil {
		_ = h.store.UpdateStatus(r.Context(), image.ID, models.StatusFailed, "failed to enqueue image")
		writeError(w, http.StatusServiceUnavailable, "failed to enqueue image")
		return
	}

	writeJSON(w, http.StatusAccepted, image)
}

func (h *Handler) getImage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	image, err := h.store.Get(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "image not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get image")
		return
	}
	if image.Status != models.StatusDone {
		writeJSON(w, http.StatusAccepted, image)
		return
	}

	file, err := h.store.OpenProcessed(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "processed image not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to open image")
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeContent(w, r, id+".jpg", image.UpdatedAt, file)
}

func (h *Handler) getStatus(w http.ResponseWriter, r *http.Request) {
	image, err := h.store.Get(r.Context(), chi.URLParam(r, "id"))
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "image not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get image")
		return
	}
	writeJSON(w, http.StatusOK, image)
}

func (h *Handler) deleteImage(w http.ResponseWriter, r *http.Request) {
	if err := h.store.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete image")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
