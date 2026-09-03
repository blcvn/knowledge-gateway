package usecase

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/domain/repository"
	"vnp-memory/services/memobase-engine/internal/usecase/dto"
	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type processBufferUseCase struct {
	llmClient        port.LLMClient
	profileExtractor port.ProfileExtractor
	profileMerger    port.ProfileMerger
	eventProcessor   port.EventProcessor
	blobRepo         repository.BlobRepository
	profileRepo      repository.ProfileRepository
	eventPublisher   port.EventPublisher
}

// NewProcessBufferUseCase creates the main orchestrator for the pipeline.
func NewProcessBufferUseCase(
	llmClient port.LLMClient,
	profileExtractor port.ProfileExtractor,
	profileMerger port.ProfileMerger,
	eventProcessor port.EventProcessor,
	blobRepo repository.BlobRepository,
	profileRepo repository.ProfileRepository,
	eventPublisher port.EventPublisher,
) port.BufferProcessor {
	return &processBufferUseCase{
		llmClient:        llmClient,
		profileExtractor: profileExtractor,
		profileMerger:    profileMerger,
		eventProcessor:   eventProcessor,
		blobRepo:         blobRepo,
		profileRepo:      profileRepo,
		eventPublisher:   eventPublisher,
	}
}

func (u *processBufferUseCase) ProcessBuffer(ctx context.Context, req dto.ProcessBufferRequest) (dto.ProcessBufferResponse, error) {
	// Step 1: Fetch blobs using blobRepo
	// Step 2: LLM #1 -> entry_chat_summary
	// Step 3: Parallel processing
	//   - LLM #2 -> extract_topics (via profileExtractor)
	//   - Event Processing (via eventProcessor)
	// Step 4: Fetch existing profiles from profileRepo
	// Step 5: LLM #3 -> merge_yolo (via profileMerger)
	// Step 6: Post-processing (organize_profiles, re_summary)
	// Step 7: Persist changes via profileRepo
	// Step 8: Publish events via eventPublisher
	
	// Stub implementation
	return dto.ProcessBufferResponse{}, nil
}
