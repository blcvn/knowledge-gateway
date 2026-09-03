package engine

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/engine"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/ingestion"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/port"
)

// Service implements port.EngineUseCase enforcing exactly 3 LLM calls.
type Service struct {
	llm      port.LLMService
	profiles port.ProfileRepository
	gists    port.GistRepository
	pub      port.EventPublisher
}

func NewService(llm port.LLMService, profiles port.ProfileRepository, gists port.GistRepository, pub port.EventPublisher) *Service {
	return &Service{llm: llm, profiles: profiles, gists: gists, pub: pub}
}

// YOLOMerge performs the 3-LLM-call merge operation.
func (s *Service) YOLOMerge(ctx context.Context, tenantID, userID uuid.UUID, blobs []ingestion.Blob) (*engine.MergeResult, error) {
	// Consolidate blob content
	var fullContent string
	for _, b := range blobs {
		fullContent += b.Content + "\n"
	}

	// Step 1 & 2a: Extract Topics (LLM Call 1 & 2 logic may be wrapped in Bifrost)
	// According to Architecture: 
	// 1: Entry Summary -> 2: Topic Extraction
	topics, err := s.llm.ExtractTopics(ctx, fullContent)
	if err != nil {
		return nil, fmt.Errorf("extract topics: %w", err)
	}

	// Fetch existing profile
	profile, err := s.profiles.FindByUser(ctx, tenantID, userID)
	if err != nil {
		// If not found, we create a new empty one
		profile = &engine.Profile{
			ID:       uuid.New(),
			TenantID: tenantID,
			UserID:   userID,
			Traits:   make(map[string]any),
		}
	}

	// Step 2b: YOLO Merge (LLM Call 3)
	updatedTraits, err := s.llm.MergeTraits(ctx, profile.Traits, fullContent)
	if err != nil {
		return nil, fmt.Errorf("merge traits: %w", err)
	}

	profile.Topics = append(profile.Topics, topics...) // simplify topic addition for YOLO
	profile.Traits = updatedTraits
	
	if err := s.profiles.Upsert(ctx, profile); err != nil {
		return nil, fmt.Errorf("upsert profile: %w", err)
	}

	// Step 3 & 4: Event Tagging & Generation
	gist, err := s.llm.GenerateGist(ctx, fullContent)
	if err != nil {
		return nil, fmt.Errorf("generate gist: %w", err)
	}
	gist.TenantID = tenantID
	gist.UserID = userID

	if err := s.gists.Create(ctx, gist); err != nil {
		return nil, fmt.Errorf("create gist: %w", err)
	}

	return &engine.MergeResult{
		NewTopics:     topics,
		UpdatedTraits: updatedTraits,
		Gist:          gist,
		LLMCalls:      3, // Exact count guaranteed by design
	}, nil
}

func (s *Service) GetProfile(ctx context.Context, tenantID, userID uuid.UUID) (*engine.Profile, error) {
	return s.profiles.FindByUser(ctx, tenantID, userID)
}

func (s *Service) GetEventGists(ctx context.Context, tenantID, userID uuid.UUID, limit int) ([]engine.EventGist, error) {
	return s.gists.FindByUser(ctx, tenantID, userID, limit)
}
