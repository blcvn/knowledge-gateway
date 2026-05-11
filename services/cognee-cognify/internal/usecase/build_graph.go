package usecase

import (
	"context"
	"log/slog"

	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

// BuildGraphStage persists entities and relationships to the graph database.
type BuildGraphStage struct {
	graphRepo port.GraphRepository
	logger    *slog.Logger
}

func NewBuildGraphStage(graphRepo port.GraphRepository, logger *slog.Logger) *BuildGraphStage {
	return &BuildGraphStage{graphRepo: graphRepo, logger: logger.With("stage", "build_graph")}
}

func (s *BuildGraphStage) Name() domain.StageType { return domain.StageBuildGraph }

func (s *BuildGraphStage) Execute(ctx context.Context, job *domain.CognifyJob, state *PipelineState) error {
	// Upsert entities
	for _, entity := range state.Entities {
		if _, err := s.graphRepo.UpsertEntity(ctx, job.TenantID, entity); err != nil {
			return &domain.ErrPipelineFailed{
				Stage: domain.StageBuildGraph,
				Cause: err,
			}
		}
	}

	// Upsert relationships
	for _, rel := range state.Relationships {
		if err := s.graphRepo.UpsertRelationship(ctx, job.TenantID, rel); err != nil {
			s.logger.Warn("relationship upsert failed", "rel_id", rel.ID, "error", err)
			// Non-fatal — continue with other relationships
		}
	}

	s.logger.Info("graph build complete",
		"entities_persisted", len(state.Entities),
		"relationships_persisted", len(state.Relationships),
	)
	return nil
}
