package dto

import "openviking.com/ov-resource/internal/domain/model"

type IngestRequest struct {
	Content     []byte
	Filename    string
	Path        string
	AccountID   string
	ForceParser string
}

type IngestResponse struct {
	ChunksCount     int
	TotalTokens     int
	Path            string
	ParseDurationMs int
}

type ParseRequest struct {
	Content      []byte
	Filename     string
	ChunkSize    int32
	ChunkOverlap int32
}

type ParseResponse struct {
	Chunks []model.Chunk
}

type WatchRequest struct {
	AccountID      string
	SourcePath     string
	TargetPath     string
	PollIntervalMs int64
	Patterns       []string
}

type RefreshRequest struct {
	AccountID string
	Paths     []string
	Force     bool
}

type RefreshResponse struct {
	Refreshed int
	Failed    int
}
