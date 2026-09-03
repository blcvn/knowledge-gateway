package dto

type InsertBlobResponse struct {
	BlobID           string `json:"blob_id"`
	BufferFlushed    bool   `json:"buffer_flushed"`
	BufferTokenCount int    `json:"buffer_token_count"`
}

type BufferStatus struct {
	UserID          string `json:"user_id"`
	ProjectID       string `json:"project_id"`
	IdleCount       int    `json:"idle_count"`
	ProcessingCount int    `json:"processing_count"`
	FailedCount     int    `json:"failed_count"`
	TotalTokens     int    `json:"total_tokens"`
	Threshold       int    `json:"threshold"`
}

type FlushResponse struct {
	FlushedCount int `json:"flushed_count"`
}
