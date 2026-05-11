package domain

import (
	"errors"
	"fmt"
)

// Sentinel domain errors.
var (
	ErrJobNotFound       = errors.New("cognify job not found")
	ErrJobAlreadyRunning = errors.New("cognify job is already running for this dataset")
	ErrInvalidJobStatus  = errors.New("invalid job status transition")
	ErrMissingTenantID   = errors.New("tenant_id is required")
	ErrMissingDatasetID  = errors.New("dataset_id is required")
	ErrEmptyText         = errors.New("text content is empty for chunking")
)

// ErrPipelineFailed wraps a stage failure with context.
type ErrPipelineFailed struct {
	Stage StageType
	Cause error
}

func (e *ErrPipelineFailed) Error() string {
	return fmt.Sprintf("pipeline failed at stage %s: %v", e.Stage, e.Cause)
}

func (e *ErrPipelineFailed) Unwrap() error { return e.Cause }

// ErrLLMTimeout indicates an LLM call exceeded the deadline.
type ErrLLMTimeout struct {
	Stage   StageType
	ChunkID string
}

func (e *ErrLLMTimeout) Error() string {
	return fmt.Sprintf("LLM timeout at stage %s (chunk: %s)", e.Stage, e.ChunkID)
}

// ErrEntityResolution indicates entity deduplication failed.
type ErrEntityResolution struct {
	EntityA string
	EntityB string
	Cause   error
}

func (e *ErrEntityResolution) Error() string {
	return fmt.Sprintf("entity resolution failed for %q vs %q: %v", e.EntityA, e.EntityB, e.Cause)
}

func (e *ErrEntityResolution) Unwrap() error { return e.Cause }
