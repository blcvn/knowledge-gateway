package usecase

import (
    "context"
    "fmt"
    "math"
    "strings"
    "time"

    "vnp-memory/services/search-service/internal/domain"
    "vnp-memory/services/search-service/internal/usecase/port"
)

type BuildContextUseCase struct {
    obsStore    port.IObservationStore
    memClient   port.IAgentMemoryClient
    smartSearch *SmartSearchUseCase
}

type ContextRequest struct {
    TenantID    string
    Project     string
    SessionID   string
    Query       string
    TokenBudget int
}

type ContextResponse struct {
    Blocks      []domain.ContextBlock
    TotalTokens int
    Formatted   string
}

func (uc *BuildContextUseCase) Execute(ctx context.Context, req ContextRequest) (*ContextResponse, error) {
    if req.TokenBudget == 0 { req.TokenBudget = 2000 }
    budget := req.TokenBudget
    var blocks []domain.ContextBlock

    // P1: Recent high-strength memories (strength > 0.5, last 30 days)
    memories, _ := uc.memClient.ListLatest(ctx, req.TenantID, req.Project, 30)
    for _, m := range memories {
        tokens := len(m.Content) / 4
        if budget-tokens < 0 { break }
        recency := math.Exp(-float64(time.Since(m.UpdatedAt).Hours()/24) / 7)
        blocks = append(blocks, domain.ContextBlock{
            Type:    "memory",
            Content: fmt.Sprintf("[%s] %s: %s", m.Type, m.Title, m.Content[:min(200, len(m.Content))]),
            Tokens:  tokens,
            Recency: recency,
            Source:  m.ID,
        })
        budget -= tokens
    }

    // P2: Last 3 session summaries
    summaries, _ := uc.obsStore.GetRecentSummaries(ctx, req.TenantID, req.Project, 3)
    for _, s := range summaries {
        tokens := len(s.Narrative) / 4
        if budget-tokens < 0 { break }
        blocks = append(blocks, domain.ContextBlock{
            Type: "summary", Content: s.Narrative, Tokens: tokens,
        })
        budget -= tokens
    }

    // P3: Relevant observations via search (if query provided)
    if req.Query != "" && budget > 100 {
        searchResp, _ := uc.smartSearch.Execute(ctx, SmartSearchRequest{
            Query: req.Query, TenantID: req.TenantID, Limit: 5,
        })
        if searchResp != nil {
            for _, r := range searchResp.Results {
                content := r.Title + ": " + r.Narrative
                tokens := len(content) / 4
                if budget-tokens < 0 { break }
                blocks = append(blocks, domain.ContextBlock{
                    Type: "observation", Content: content, Tokens: tokens,
                })
                budget -= tokens
            }
        }
    }

    formatted := formatBlocks(blocks)
    return &ContextResponse{
        Blocks:      blocks,
        TotalTokens: req.TokenBudget - budget,
        Formatted:   formatted,
    }, nil
}

func formatBlocks(blocks []domain.ContextBlock) string {
    var sb strings.Builder
    for _, b := range blocks {
        sb.WriteString(fmt.Sprintf("[%s] %s\n\n", strings.ToUpper(b.Type), b.Content))
    }
    return sb.String()
}

func min(a, b int) int { if a < b { return a }; return b }
