package storage

import (
	"context"
	"io"

	"imageprocessor/backend/internal/models"
)

type Store interface {
	SaveOriginal(ctx context.Context, image models.Image, reader io.Reader) error
	OpenOriginal(ctx context.Context, id string) (io.ReadCloser, error)
	SaveProcessed(ctx context.Context, id string, reader io.Reader) error
	OpenProcessed(ctx context.Context, id string) (io.ReadCloser, error)
	SaveThumbnail(ctx context.Context, id string, reader io.Reader) error
	OpenThumbnail(ctx context.Context, id string) (io.ReadCloser, error)
	Get(ctx context.Context, id string) (models.Image, error)
	List(ctx context.Context) ([]models.Image, error)
	UpdateStatus(ctx context.Context, id string, status models.Status, message string) error
	Delete(ctx context.Context, id string) error
}
