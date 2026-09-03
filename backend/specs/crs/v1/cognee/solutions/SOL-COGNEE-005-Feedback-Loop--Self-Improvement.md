# SOL-COGNEE-005 — Solution: Feedback Loop & Self-Improvement

| Field | Value |
|---|---|
| **Solution ID** | SOL-COGNEE-005 |
| **CR** | [CR-COGNEE-005](../../../../docs/crs/v1/cognee/CR-COGNEE-005*.md) |
| **TDD ref** | [02-cognee-services.md](../../../tdd/architecture/02-cognee-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |

---
## 1. Giải pháp

Track search result relevance → retrain entity extraction prompt over time.

### 1.1 `services/cognee-search/internal/usecase/feedback.go` [NEW]

```go
type FeedbackUseCase struct {
    feedbackRepo port.FeedbackRepository
    promptRepo   port.PromptRepository
}

// POST /v1/cognee/feedback
func (u *FeedbackUseCase) RecordFeedback(ctx context.Context, req *FeedbackRequest) error {
    return u.feedbackRepo.Store(ctx, &Feedback{
        QueryID:   req.QueryID,
        ResultID:  req.ResultID,
        Relevant:  req.Relevant,      // true/false
        Comment:   req.Comment,
        UserID:    req.UserID,
        TenantID:  req.TenantID,
    })
}

// Background job: aggregate feedback → update extraction prompts
func (u *FeedbackUseCase) RunPromptOptimization(ctx context.Context) {
    lowScorePatterns := u.feedbackRepo.GetLowRelevancePatterns(ctx)
    newPrompt := u.llm.OptimizePrompt(ctx, currentPrompt, lowScorePatterns)
    u.promptRepo.UpdateExtractionPrompt(ctx, newPrompt)
}
```

## 2. Acceptance Criteria

- [ ] Users can rate search results (relevant/not relevant)
- [ ] Feedback stored per query/result pair
- [ ] Weekly batch updates extraction prompts based on feedback
