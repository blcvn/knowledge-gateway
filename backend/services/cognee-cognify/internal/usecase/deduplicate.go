package usecase

import (
	"context"
	"log/slog"
	"strings"

	"vnp-memory/services/cognee-cognify/internal/domain"
	"vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// DeduplicateStage resolves duplicate entities via name normalization + optional LLM verification.
type DeduplicateStage struct {
	llm    port.LLMClient
	logger *slog.Logger
}

func NewDeduplicateStage(llm port.LLMClient, logger *slog.Logger) *DeduplicateStage {
	return &DeduplicateStage{llm: llm, logger: logger.With("stage", "deduplicate")}
}

func (s *DeduplicateStage) Name() domain.StageType { return domain.StageDeduplicate }

func (s *DeduplicateStage) Execute(ctx context.Context, job *domain.CognifyJob, state *CognifyPipelineState) error {
	if job.Config.SkipDedup {
		s.logger.Info("deduplication skipped (config)")
		return nil
	}

	if len(state.Entities) == 0 {
		return nil
	}

	// Phase 1: Exact name normalization (case-insensitive)
	seen := make(map[string]*domain.Entity)    // normalized_name → canonical entity
	mergeMap := make(map[string]string)         // old_id → canonical_id (for relationship remapping)
	var deduplicated []*domain.Entity
	dedupCount := 0

	for _, entity := range state.Entities {
		key := normalizeEntityName(entity.Name)
		if canonical, exists := seen[key]; exists {
			mergeMap[entity.ID] = canonical.ID
			dedupCount++
		} else {
			seen[key] = entity
			deduplicated = append(deduplicated, entity)
		}
	}

	// Phase 2: Remap relationships to canonical entities
	if len(mergeMap) > 0 {
		for _, rel := range state.Relationships {
			if newID, ok := mergeMap[rel.SourceID]; ok {
				if canonical, found := findEntityByIDStr(deduplicated, newID); found {
					rel.SourceID = canonical.ID
				}
			}
			if newID, ok := mergeMap[rel.TargetID]; ok {
				if canonical, found := findEntityByIDStr(deduplicated, newID); found {
					rel.TargetID = canonical.ID
				}
			}
		}
	}

	state.Entities = deduplicated
	job.Metrics.EntitiesDeduplicated = dedupCount
	s.logger.Info("deduplication complete", "original", len(state.Entities)+dedupCount, "deduplicated", dedupCount, "remaining", len(deduplicated))
	return nil
}

// normalizeEntityName normalizes entity names for dedup comparison.
func normalizeEntityName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

// findEntityByIDStr finds entity by string ID in a slice.
func findEntityByIDStr(entities []*domain.Entity, idStr string) (*domain.Entity, bool) {
	for _, e := range entities {
		if e.ID == idStr {
			return e, true
		}
	}
	return nil, false
}
