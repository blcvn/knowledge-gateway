package model

// BufferReadyEvent represents the event published when buffer reaches threshold
type BufferReadyEvent struct {
	UserID    string   `json:"user_id"`
	ProjectID string   `json:"project_id"`
	BufferIDs []string `json:"buffer_ids"`
	BlobType  BlobType `json:"blob_type"`
}
