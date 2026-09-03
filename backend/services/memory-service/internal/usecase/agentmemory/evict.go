package agentmemory

import (
	"context"
	"math"
	"sort"
	"time"

	"vnp-memory/services/memory-service/internal/domain/agentmemory"
	"vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

// EvictRequest is the input for the eviction use case.
type EvictRequest struct {
	TenantID    string
	Project     string
	MaxMemories int
	DryRun      bool
}

// EvictResult is the result of the eviction use case.
type EvictResult struct {
	EvictedIDs []string
	DryRun     bool
}

type scoredMemory struct {
	Memory agentmemory.AgentMemory
	Score  float64
}

// EvictUseCase implements memory eviction based on scoring formula.
type EvictUseCase struct {
	repo      port.IMemoryRepo
	publisher port.IEventPublisher
}

func NewEvictUseCase(repo port.IMemoryRepo, publisher port.IEventPublisher) *EvictUseCase {
	return &EvictUseCase{repo: repo, publisher: publisher}
}

// Execute runs the eviction policy.
// Eviction score = importance × recency × frequency
func (uc *EvictUseCase) Execute(ctx context.Context, req EvictRequest) (*EvictResult, error) {
	memories, err := uc.repo.ListLatestByProject(ctx, req.TenantID, req.Project)
	if err != nil {
		return nil, err
	}
	if len(memories) <= req.MaxMemories {
		return &EvictResult{DryRun: req.DryRun}, nil
	}

	scored := make([]scoredMemory, len(memories))
	for i, m := range memories {
		daysSince := time.Since(m.UpdatedAt).Hours() / 24
		recency := math.Exp(-daysSince / 30.0)
		frequency := math.Log1p(float64(len(m.SessionIDs)))
		scored[i] = scoredMemory{Memory: m, Score: m.Strength * recency * frequency}
	}

	// Sort ascending: lowest score evicted first
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score < scored[j].Score })

	toEvict := scored[:len(memories)-req.MaxMemories]
	var evictedIDs []string

	for _, sm := range toEvict {
		if sm.Memory.Strength >= 1.0 {
			continue // Pinned memories are immune
		}
		evictedIDs = append(evictedIDs, sm.Memory.ID)
		if !req.DryRun {
			_ = uc.repo.Delete(ctx, sm.Memory.ID)
			_ = uc.publisher.Publish(ctx, "agentmemory.memory.expired", map[string]any{
				"memory_id": sm.Memory.ID,
				"reason":    "eviction",
			})
		}
	}

	return &EvictResult{EvictedIDs: evictedIDs, DryRun: req.DryRun}, nil
}
