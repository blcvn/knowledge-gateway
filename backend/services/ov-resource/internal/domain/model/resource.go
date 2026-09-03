package model

import "time"

type ResourceType string

const (
	ResourceTypeCode     ResourceType = "code"
	ResourceTypeDocument ResourceType = "document"
	ResourceTypeMarkdown ResourceType = "markdown"
	ResourceTypeDefault  ResourceType = "default"
)

type ResourceStatus string

const (
	StatusPending    ResourceStatus = "pending"
	StatusProcessing ResourceStatus = "processing"
	StatusCompleted  ResourceStatus = "completed"
	StatusFailed     ResourceStatus = "failed"
)

type Resource struct {
	ID              string
	AccountID       string
	SourcePath      string
	TargetPath      string
	Filename        string
	MimeType        string
	ParserType      ResourceType
	ChunkCount      int
	TotalTokens     int
	ContentHash     string
	Status          ResourceStatus
	ErrorMessage    string
	ParseDurationMs int
	IngestedAt      time.Time
	CreatedAt       time.Time
}

type IngestionResult struct {
	ResourceID      string
	ChunksCount     int
	TotalTokens     int
	ParseDurationMs int
	TargetPath      string
}
