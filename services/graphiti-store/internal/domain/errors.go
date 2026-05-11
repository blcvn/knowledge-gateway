package domain

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for the graphiti-store domain.
var (
	// Validation errors.
	ErrMissingUUID    = errors.New("uuid is required")
	ErrMissingName    = errors.New("name is required")
	ErrMissingGroupID = errors.New("group_id is required")
	ErrMissingNodeID  = errors.New("source_node_id and target_node_id are required")
	ErrMissingValidAt = errors.New("valid_at is required")
	ErrEmptyContent   = errors.New("content is empty")
	ErrEmptyFact      = errors.New("fact is empty")

	// Embedding errors.
	ErrEmptyEmbedding            = errors.New("embedding vector is empty")
	ErrInvalidEmbeddingDimension = errors.New("embedding dimension mismatch")

	// Node/Edge lookup errors.
	ErrNodeNotFound = errors.New("node not found")
	ErrEdgeNotFound = errors.New("edge not found")

	// Driver errors.
	ErrDriverNotSupported    = errors.New("graph driver not supported")
	ErrDriverNotImplemented  = errors.New("graph driver not yet implemented")

	// Transaction errors.
	ErrTransactionFailed = errors.New("transaction failed")

	// Index errors.
	ErrIndexNotFound = errors.New("index not found")

	// Label errors.
	ErrInvalidNodeLabel = errors.New("invalid node label")
)

// ErrInvalidTemporalRange is returned when bi-temporal constraints are violated.
type ErrInvalidTemporalRange struct {
	Field   string
	Value   time.Time
	ValidAt time.Time
	Reason  string
}

func (e *ErrInvalidTemporalRange) Error() string {
	return fmt.Sprintf("invalid temporal range: %s (%s) violates constraint against valid_at (%s): %s",
		e.Field, e.Value.Format(time.RFC3339), e.ValidAt.Format(time.RFC3339), e.Reason)
}

// ErrBulkOperationFailed wraps an error from a bulk operation with context.
type ErrBulkOperationFailed struct {
	Operation string
	EpisodeID string
	Cause     error
}

func (e *ErrBulkOperationFailed) Error() string {
	return fmt.Sprintf("bulk %s failed for episode %s: %v", e.Operation, e.EpisodeID, e.Cause)
}

func (e *ErrBulkOperationFailed) Unwrap() error { return e.Cause }
