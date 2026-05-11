package model

import (
	"encoding/json"
	"time"
)

// BlobType represents the type of a blob
type BlobType string

const (
	BlobTypeChat    BlobType = "chat"
	BlobTypeDoc     BlobType = "doc"
	BlobTypeSummary BlobType = "summary"
)

// GeneralBlob represents the general blob entity
type GeneralBlob struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	ProjectID string          `json:"project_id"`
	BlobType  BlobType        `json:"blob_type"`
	BlobData  json.RawMessage `json:"blob_data"`
	AddFields json.RawMessage `json:"add_fields,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
