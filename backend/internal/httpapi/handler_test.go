package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"imageprocessor/backend/internal/models"
	"imageprocessor/backend/internal/storage"
)

type fakeProducer struct {
	messages []models.ProcessMessage
}

func (p *fakeProducer) Publish(_ context.Context, message models.ProcessMessage) error {
	p.messages = append(p.messages, message)
	return nil
}

func (p *fakeProducer) Close() error {
	return nil
}

func TestUploadStoresImageAndPublishesMessage(t *testing.T) {
	t.Parallel()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	producer := &fakeProducer{}
	handler := NewHandler(store, producer).Routes()

	body, contentType := multipartImage(t)
	request := httptest.NewRequest(http.MethodPost, "/upload", body)
	request.Header.Set("Content-Type", contentType)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d: %s", http.StatusAccepted, response.Code, response.Body.String())
	}

	var got models.Image
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID == "" {
		t.Fatal("expected generated id")
	}
	if got.Status != models.StatusQueued {
		t.Fatalf("unexpected status: %s", got.Status)
	}
	if got.ContentType != "image/png" {
		t.Fatalf("unexpected content type: %s", got.ContentType)
	}
	if len(producer.messages) != 1 || producer.messages[0].ID != got.ID {
		t.Fatalf("unexpected producer messages: %+v", producer.messages)
	}

	saved, err := store.Get(context.Background(), got.ID)
	if err != nil {
		t.Fatalf("stored image not found: %v", err)
	}
	if saved.OriginalName != "upload.png" {
		t.Fatalf("unexpected original name: %s", saved.OriginalName)
	}
}

func TestUploadRejectsUnsupportedFile(t *testing.T) {
	t.Parallel()

	store, err := storage.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}
	handler := NewHandler(store, &fakeProducer{}).Routes()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("image", "notes.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := file.Write([]byte("not an image")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, response.Code)
	}
}

func multipartImage(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("image", "upload.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.RGBA{R: 60, G: 120, B: 180, A: 255})
		}
	}
	if err := png.Encode(file, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return &body, writer.FormDataContentType()
}
