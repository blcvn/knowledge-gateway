// Package usecase defines the StepHandler interface for Chain-of-Responsibility pipeline.
// TASK-CE-010: Custom Pipelines Orchestration
package usecase

import (
	"context"

	"vnp-memory/services/cognee-cognify/internal/domain"
)

// StepHandler is the interface every pipeline step must implement.
// Steps are chained: each receives PipelineState and returns a (possibly mutated) state.
type StepHandler interface {
	Name() domain.PipelineStep
	Execute(ctx context.Context, state *PipelineState) (*PipelineState, error)
}
