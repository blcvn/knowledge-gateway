package agentmemory

import (
    "context"
    "math"
    "time"

    "vnp-memory/services/memory-service/internal/usecase/agentmemory/port"
)

type RetentionUseCase struct{ repo port.IMemoryRepo }

type RetentionScore struct {
    Score            float64
    RecencyFactor    float64
    FrequencyFactor  float64
    ImportanceFactor float64
    DaysSinceAccess  float64
    RecommendAction  string  // "keep" | "review" | "evict"
}

func (uc *RetentionUseCase) GetScore(ctx context.Context, memID string) (*RetentionScore, error) {
    mem, err := uc.repo.Get(ctx, memID)
    if err != nil { return nil, err }

    daysSince := time.Since(mem.UpdatedAt).Hours() / 24
    recency   := math.Exp(-daysSince / 30.0)
    frequency := math.Log1p(float64(len(mem.SessionIDs)))
    score     := mem.Strength * recency * frequency

    action := "keep"
    if score < 0.1 { action = "evict" } else if score < 0.3 { action = "review" }

    return &RetentionScore{
        Score: score, RecencyFactor: recency, FrequencyFactor: frequency,
        ImportanceFactor: mem.Strength, DaysSinceAccess: daysSince,
        RecommendAction: action,
    }, nil
}
