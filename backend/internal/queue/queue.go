package queue

import (
	"context"

	"imageprocessor/backend/internal/models"
)

type Producer interface {
	Publish(ctx context.Context, message models.ProcessMessage) error
	Close() error
}

type Handler func(context.Context, models.ProcessMessage) error
