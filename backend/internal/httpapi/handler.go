package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
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
	r.Use(cors)

	r.Post("/upload", h.upload)
	r.Get("/images", h.listImages)
	r.Get("/image/{id}", h.getImage)
	r.Get("/image/{id}/thumbnail", h.getThumbnail)
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

	reader, contentType, err := validateImage(file)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	image := models.Image{
		ID:           uuid.NewString(),
		OriginalName: header.Filename,
		ContentType:  contentType,
		Status:       models.StatusQueued,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := h.store.SaveOriginal(r.Context(), image, reader); err != nil {
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

func (h *Handler) listImages(w http.ResponseWriter, r *http.Request) {
	images, err := h.store.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list images")
		return
	}
	sort.Slice(images, func(i, j int) bool {
		return images[i].CreatedAt.After(images[j].CreatedAt)
	})
	writeJSON(w, http.StatusOK, images)
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

	serveJPEG(w, file)
}

func (h *Handler) getThumbnail(w http.ResponseWriter, r *http.Request) {
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

	file, err := h.store.OpenThumbnail(r.Context(), id)
	if errors.Is(err, storage.ErrNotFound) {
		writeError(w, http.StatusNotFound, "thumbnail not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to open thumbnail")
		return
	}
	defer file.Close()

	serveJPEG(w, file)
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

func serveJPEG(w http.ResponseWriter, reader io.Reader) {
	w.Header().Set("Content-Type", "image/jpeg")
	_, _ = io.Copy(w, reader)
}

func validateImage(reader io.Reader) (io.Reader, string, error) {
	head := make([]byte, 512)
	n, err := io.ReadFull(reader, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, "", err
	}
	head = head[:n]

	contentType := http.DetectContentType(head)
	switch contentType {
	case "image/jpeg", "image/png", "image/gif":
		return io.MultiReader(bytes.NewReader(head), reader), contentType, nil
	default:
		return nil, "", errors.New("only jpg, png and gif images are supported")
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
