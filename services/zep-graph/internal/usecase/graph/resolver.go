package graph

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-memory/services/zep-graph/internal/domain/fact"
)

type FactRepository interface {
	FindContradictingFacts(ctx context.Context, newEdge fact.Edge) ([]fact.Edge, error)
	InvalidateEdge(ctx context.Context, edgeUUID uuid.UUID, invalidAt time.Time) error
	SaveEdge(ctx context.Context, edge fact.Edge) error
}

type TemporalResolver struct {
	repo FactRepository
}

func NewTemporalResolver(repo FactRepository) *TemporalResolver {
	return &TemporalResolver{repo: repo}
}

// Resolve processes an incoming new fact, finding prior contradictions
// and marking them as invalid instead of performing hard deletes.
func (tr *TemporalResolver) Resolve(ctx context.Context, newFact fact.Edge, occurrenceTime time.Time) error {
	// 1. Find existing facts that contradict this new fact.
	// E.g., if User is currently LOCATED_AT "New York", but newFact says "Paris".
	contradictions, err := tr.repo.FindContradictingFacts(ctx, newFact)
	if err != nil {
		return fmt.Errorf("failed to query contradictions: %w", err)
	}

	// 2. Invalidate contradicting facts
	for _, oldFact := range contradictions {
		if oldFact.InvalidAt == nil {
			if err := tr.repo.InvalidateEdge(ctx, oldFact.UUID, occurrenceTime); err != nil {
				return fmt.Errorf("failed to invalidate old fact %s: %w", oldFact.UUID, err)
			}
		}
	}

	// 3. Set the Temporal bounds for the new fact
	newFact.ValidAt = &occurrenceTime
	newFact.CreatedAt = time.Now()

	// 4. Persist the new fact
	if err := tr.repo.SaveEdge(ctx, newFact); err != nil {
		return fmt.Errorf("failed to save new fact: %w", err)
	}

	return nil
}
