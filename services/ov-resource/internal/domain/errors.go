package domain

import "errors"

var (
	ErrUnsupportedFormat = errors.New("unsupported file format")
	ErrParseFailed       = errors.New("failed to parse resource")
	ErrIngestFailed      = errors.New("failed to ingest resource")
	ErrWatchTaskNotFound = errors.New("watch task not found")
	ErrResourceExhausted = errors.New("resource limits exhausted")
)
