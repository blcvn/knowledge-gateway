package dto

import (
	"encoding/json"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/domain/model"
)

type InsertBlobRequest struct {
	UserID     string          `json:"user_id"`
	ProjectID  string          `json:"project_id"`
	BlobType   model.BlobType  `json:"blob_type"`
	BlobData   json.RawMessage `json:"blob_data"`
	Persistent bool            `json:"persistent"`
}

type BufferStatusRequest struct {
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
}

type FlushBufferRequest struct {
	UserID    string `json:"user_id"`
	ProjectID string `json:"project_id"`
}

type DeleteBlobRequest struct {
	ProjectID string `json:"project_id"`
	BlobID    string `json:"blob_id"`
}
