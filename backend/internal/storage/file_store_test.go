package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"imageprocessor/backend/internal/models"
)

func TestFileStoreLifecycle(t *testing.T) {
	t.Parallel()

	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	ctx := context.Background()
	now := time.Now().UTC()
	image := models.Image{
		ID:           "image-1",
		OriginalName: "photo.png",
		ContentType:  "image/png",
		Status:       models.StatusQueued,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := store.SaveOriginal(ctx, image, bytes.NewBufferString("original")); err != nil {
		t.Fatalf("SaveOriginal: %v", err)
	}
	if err := store.SaveProcessed(ctx, image.ID, bytes.NewBufferString("processed")); err != nil {
		t.Fatalf("SaveProcessed: %v", err)
	}
	if err := store.SaveThumbnail(ctx, image.ID, bytes.NewBufferString("thumbnail")); err != nil {
		t.Fatalf("SaveThumbnail: %v", err)
	}

	got, err := store.Get(ctx, image.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != image.ID || got.Status != models.StatusQueued {
		t.Fatalf("unexpected metadata: %+v", got)
	}

	images, err := store.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(images) != 1 || images[0].ID != image.ID {
		t.Fatalf("unexpected list: %+v", images)
	}

	reader, err := store.OpenOriginal(ctx, image.ID)
	assertContent(t, reader, err, "original")
	reader, err = store.OpenProcessed(ctx, image.ID)
	assertContent(t, reader, err, "processed")
	reader, err = store.OpenThumbnail(ctx, image.ID)
	assertContent(t, reader, err, "thumbnail")

	if err := store.UpdateStatus(ctx, image.ID, models.StatusDone, ""); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	got, err = store.Get(ctx, image.ID)
	if err != nil {
		t.Fatalf("Get after status update: %v", err)
	}
	if got.Status != models.StatusDone {
		t.Fatalf("status was not updated: %s", got.Status)
	}

	if err := store.Delete(ctx, image.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, image.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func assertContent(t *testing.T, reader io.ReadCloser, err error, expected string) {
	t.Helper()
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(data) != expected {
		t.Fatalf("expected %q, got %q", expected, string(data))
	}
}
