// Package ingestion implements the saga-orchestrated episode ingestion.
package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/graphiti-pipeline/internal/domain/ingestion"
	"github.com/vnp-community/vnp-memory/services/graphiti-pipeline/internal/usecase/port"
)

// Service implements port.IngestionUseCase with saga orchestration.
type Service struct {
	episodes  port.EpisodeRepository
	knowledge port.KnowledgeUseCase // LOCAL call — consolidated
	pub       port.EventPublisher
}

func NewService(ep port.EpisodeRepository, know port.KnowledgeUseCase, pub port.EventPublisher) *Service {
	return &Service{episodes: ep, knowledge: know, pub: pub}
}

// IngestEpisode runs the 5-step saga pipeline.
// Knowledge extraction is called LOCALLY (was cross-service gRPC).
func (s *Service) IngestEpisode(ctx context.Context, tenantID uuid.UUID, groupID, content, speaker, source string, validAt time.Time) (*ingestion.Episode, error) {
	ep := &ingestion.Episode{
		ID: uuid.New(), TenantID: tenantID, GroupID: groupID,
		Content: content, Speaker: speaker, Source: source,
		ValidAt: validAt, CreatedAt: time.Now(),
	}
	if err := s.episodes.CreateEpisode(ctx, ep); err != nil {
		return nil, fmt.Errorf("create episode: %w", err)
	}

	saga := &ingestion.SagaState{
		ID: uuid.New(), EpisodeID: ep.ID,
		CurrentStep: "extract_entities", Status: "running",
		CompletedAt: make(map[string]time.Time),
	}
	_ = s.episodes.CreateSaga(ctx, saga)

	// Step 1: Extract entities (LOCAL call to knowledge usecase)
	entities, err := s.knowledge.ExtractEntities(ctx, tenantID, content, validAt)
	if err != nil {
		_ = s.episodes.UpdateSagaStep(ctx, saga.ID, "extract_entities", "failed")
		return ep, fmt.Errorf("extract entities: %w", err)
	}
	_ = s.episodes.UpdateSagaStep(ctx, saga.ID, "extract_entities", "completed")

	// Step 2: Extract edges (LOCAL call)
	_, err = s.knowledge.ExtractEdges(ctx, tenantID, entities, content)
	if err != nil {
		_ = s.episodes.UpdateSagaStep(ctx, saga.ID, "extract_edges", "failed")
		return ep, fmt.Errorf("extract edges: %w", err)
	}
	_ = s.episodes.UpdateSagaStep(ctx, saga.ID, "extract_edges", "completed")

	// Step 3: Detect communities (LOCAL call)
	_, _ = s.knowledge.DetectCommunities(ctx, tenantID)
	_ = s.episodes.UpdateSagaStep(ctx, saga.ID, "detect_communities", "completed")

	// Publish completion event
	_ = s.pub.PublishPipelineCompleted(ctx, tenantID, ep.ID)

	return ep, nil
}

func (s *Service) GetEpisode(ctx context.Context, id uuid.UUID) (*ingestion.Episode, error) {
	return s.episodes.FindEpisodeByID(ctx, id)
}

func (s *Service) GetSagaState(ctx context.Context, episodeID uuid.UUID) (*ingestion.SagaState, error) {
	return s.episodes.FindSagaByEpisode(ctx, episodeID)
}
