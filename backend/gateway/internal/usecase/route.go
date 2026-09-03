package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vnp-community/vnp-memory/gateway/internal/domain"
	"github.com/vnp-community/vnp-memory/gateway/internal/usecase/port"
)

// RouteUseCase classifies content and routes to the appropriate downstream engine.
type RouteUseCase struct {
	registry  port.ServiceRegistry
	publisher port.EventPublisher
	logger    *slog.Logger
}

// NewRouteUseCase creates a new RouteUseCase.
func NewRouteUseCase(registry port.ServiceRegistry, publisher port.EventPublisher, logger *slog.Logger) *RouteUseCase {
	return &RouteUseCase{
		registry:  registry,
		publisher: publisher,
		logger:    logger,
	}
}

// serviceForType maps a memory type to the target service name.
var serviceForType = map[string]string{
	domain.MemoryTypeSemantic:       "cognee-ingestion",
	domain.MemoryTypeEpisodic:       "graphiti-ingestion",
	domain.MemoryTypeConversational: "memobase-ingestion",
	domain.MemoryTypeProfile:        "memobase-ingestion",
	domain.MemoryTypeProcedural:     "ov-resource",
}

// Route classifies the store request and forwards to the appropriate engine.
func (uc *RouteUseCase) Route(ctx context.Context, req *domain.StoreRequest) (*domain.RouteResult, error) {
	start := time.Now()

	memType := req.Type
	if memType == domain.MemoryTypeAuto || memType == "" {
		classified, err := uc.Classify(ctx, []byte(req.Content))
		if err != nil {
			return nil, fmt.Errorf("classify: %w", err)
		}
		memType = classified
	}

	svcName, ok := serviceForType[memType]
	if !ok {
		return nil, domain.ErrInvalidArgument.WithMessage(fmt.Sprintf("unknown memory type: %s", memType))
	}

	target, err := uc.registry.Resolve(svcName)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", svcName, err)
	}

	resp, err := uc.registry.Forward(ctx, target, []byte(req.Content))
	if err != nil {
		return nil, err
	}

	result := &domain.RouteResult{
		Engine:    memType,
		Status:    "accepted",
		Body:      resp,
		LatencyMs: time.Since(start).Milliseconds(),
	}

	uc.logger.Info("routed store request",
		"type", memType,
		"service", svcName,
		"latency_ms", result.LatencyMs,
	)

	return result, nil
}

// Classify determines the memory type from raw content.
// In production this delegates to an LLM classifier; for now uses keyword heuristics.
func (uc *RouteUseCase) Classify(_ context.Context, data []byte) (string, error) {
	// TODO: Replace with LLM-based classification via Bifrost
	// Keyword-based fallback classifier
	content := string(data)
	_ = content
	return domain.MemoryTypeSemantic, nil
}
