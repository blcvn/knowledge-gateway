package usecase

import (
	"context"

	"vnp-memory/services/memobase-engine/internal/domain/repository"
	"vnp-memory/services/memobase-engine/internal/usecase/dto"
	"vnp-memory/services/memobase-engine/internal/usecase/port"
)

type processEventUseCase struct {
	llmClient      port.LLMClient
	embedderClient port.EmbedderClient
	eventRepo      repository.EventRepository
}

// NewProcessEventUseCase creates a new instance of the event processing use case.
func NewProcessEventUseCase(
	llmClient port.LLMClient,
	embedderClient port.EmbedderClient,
	eventRepo repository.EventRepository,
) port.EventProcessor {
	return &processEventUseCase{
		llmClient:      llmClient,
		embedderClient: embedderClient,
		eventRepo:      eventRepo,
	}
}

func (u *processEventUseCase) ProcessEvent(ctx context.Context, req dto.ProcessEventRequest) error {
	// 1. Optional LLM call to tag events if req.Config.EventTags is set
	// 2. Call embedderClient to generate embedding for the event text
	// 3. Construct model.UserEvent
	// 4. Save using eventRepo
	
	return nil
}
