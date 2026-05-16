package models

import "time"

type Status string

const (
	StatusQueued     Status = "queued"
	StatusProcessing Status = "processing"
	StatusDone       Status = "done"
	StatusFailed     Status = "failed"
)

type Image struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	Status       Status    `json:"status"`
	Error        string    `json:"error,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ProcessMessage struct {
	ID string `json:"id"`
}
