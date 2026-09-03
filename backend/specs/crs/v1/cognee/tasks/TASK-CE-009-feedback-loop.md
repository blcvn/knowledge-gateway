# TASK-CE-009 — Feedback Loop (Interaction Logging + Edge Weight Reinforcement)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-CE-009 |
| **Wave** | 3 |
| **Component** | `services/cognee-search/` + `services/cognee-memory/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-005 §2.1 → §2.8 |
| **Priority** | Medium |
| **Depends On** | TASK-CE-001, TASK-CE-011 |
| **Estimated** | 6h |

---

## Context

Feedback Loop = self-improving retrieval quality:
1. **Interaction logging** — track search calls với `save_interaction: true`
2. **Feedback submission** — `FEEDBACK` strategy + `feedback_for` (interaction UUID)
3. **Edge weight adjustment** — positive feedback → boost Neo4j edge weights × 1.1, negative → ×0.9
4. **MemoryFact** — save Q&A pair to `cognee-memory`

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/cognee-search/internal/adapter/retriever/feedback.go` |
| CREATE | `services/cognee-search/internal/adapter/repository/postgres/interaction_repo.go` |
| CREATE | `services/cognee-search/internal/adapter/repository/postgres/feedback_repo.go` |
| MODIFY | `services/cognee-search/internal/domain/entity.go` |
| MODIFY | `services/cognee-search/internal/usecase/search.go` |
| MODIFY | `services/cognee-search/internal/usecase/port/output.go` |
| MODIFY | `services/cognee-search/internal/adapter/retriever/registry.go` |
| MODIFY | `services/cognee-search/internal/adapter/grpc/handler.go` |
| MODIFY | `api/proto/cognee/search/v1/search.proto` | (already done in TASK-CE-001)
| MODIFY | `services/cognee-memory/internal/domain/entity.go` |
| MODIFY | `services/cognee-memory/internal/usecase/remember.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

### MODIFY `domain/entity.go` — Add Interaction + FeedbackRecord

```go
// services/cognee-search/internal/domain/entity.go

// Interaction — logged search call (when save_interaction=true)
type Interaction struct {
    ID           uuid.UUID
    TenantID     string
    SessionID    *string
    DatasetID    *uuid.UUID
    Query        string
    Strategy     string
    ResultIDs    []string
    ResultScores []float64
    NodeSets     []string
    Timestamp    time.Time
}

// FeedbackRecord — user feedback on a specific search interaction
type FeedbackRecord struct {
    ID            uuid.UUID
    InteractionID uuid.UUID
    TenantID      string
    Score         float64    // -1.0 to 1.0
    Text          string
    AffectedNodes []string
    CreatedAt     time.Time
}
```

### MODIFY `usecase/search.go` — Interaction Logging + Feedback Handler

```go
// services/cognee-search/internal/usecase/search.go

type SearchUseCase struct {
    retrievers        map[SearchStrategy]Retriever
    interactionRepo   port.InteractionRepository   // [NEW]
    feedbackRepo      port.FeedbackRepository      // [NEW]
    feedbackRetriever *retriever.FeedbackRetriever // [NEW]
    memoryClient      port.MemoryClient            // [NEW]
    eventPub          port.EventPublisher
}

func (uc *SearchUseCase) Execute(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
    // ── Handle FEEDBACK strategy ──────────────────────────────────────
    if req.FeedbackFor != nil && *req.FeedbackFor != "" {
        return uc.handleFeedback(ctx, req)
    }

    // ── Normal Search ─────────────────────────────────────────────────
    results, err := uc.executeStrategies(ctx, req)
    if err != nil { return nil, err }

    resp := &SearchResponse{Results: results}

    // ── Save Interaction (optional) ───────────────────────────────────
    if req.SaveInteraction {
        interactionID := uuid.New()
        interaction := domain.Interaction{
            ID:           interactionID,
            TenantID:     req.TenantID,
            SessionID:    req.SessionID,
            DatasetID:    req.DatasetID,
            Query:        req.Query,
            Strategy:     firstStrategy(req.Strategies),
            ResultIDs:    extractResultIDs(results),
            ResultScores: extractResultScores(results),
            NodeSets:     req.NodeSets,
            Timestamp:    time.Now(),
        }
        uc.interactionRepo.Save(ctx, interaction)
        idStr := interactionID.String()
        resp.InteractionID = &idStr
    }

    return resp, nil
}

// handleFeedback processes FEEDBACK strategy
func (uc *SearchUseCase) handleFeedback(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
    interactionID, err := uuid.Parse(*req.FeedbackFor)
    if err != nil { return nil, fmt.Errorf("invalid feedback_for UUID: %w", err) }

    // Load original interaction to get result node IDs
    interaction, err := uc.interactionRepo.GetByID(ctx, interactionID)
    if err != nil { return nil, fmt.Errorf("interaction not found: %w", err) }

    // Apply Neo4j edge weight adjustment
    var affectedNodes []string
    if req.FeedbackScore != nil && len(interaction.ResultIDs) > 0 {
        affectedNodes, _ = uc.feedbackRetriever.ApplyWeightAdjustment(ctx, retriever.ApplyWeightRequest{
            NodeIDs:  interaction.ResultIDs,
            TenantID: req.TenantID,
            Score:    *req.FeedbackScore,
        })
    }

    // Save FeedbackRecord
    score := 0.0
    if req.FeedbackScore != nil { score = *req.FeedbackScore }
    feedback := domain.FeedbackRecord{
        ID:            uuid.New(),
        InteractionID: interactionID,
        TenantID:      req.TenantID,
        Score:         score,
        Text:          req.FeedbackText,
        AffectedNodes: affectedNodes,
        CreatedAt:     time.Now(),
    }
    uc.feedbackRepo.Save(ctx, feedback)

    // NATS event
    uc.eventPub.Publish(ctx, "cognee.search.feedback.applied", map[string]any{
        "interaction_id": interactionID.String(),
        "score":          score,
        "affected_nodes": affectedNodes,
        "tenant_id":      req.TenantID,
    })

    // Save to cognee-memory as MemoryFact (non-fatal)
    if req.FeedbackText != "" {
        go uc.memoryClient.Remember(context.Background(), port.RememberRequest{
            TenantID: req.TenantID,
            Content:  fmt.Sprintf("User rated query '%s' as %.1f: %s", interaction.Query, score, req.FeedbackText),
            Source:   "feedback",
            Type:     "feedback",
        })
    }

    return &SearchResponse{
        Metadata: map[string]any{
            "applied":        true,
            "affected_nodes": len(affectedNodes),
        },
    }, nil
}
```

### File 1 (NEW): `retriever/feedback.go`

```go
package retriever

import (
    "context"
    "fmt"

    "github.com/vnp-memory/services/cognee-search/internal/usecase/port"
)

const (
    BoostFactor   = 1.1   // positive feedback: edge weights × 1.1
    PenaltyFactor = 0.9   // negative feedback: edge weights × 0.9
    MinWeight     = 0.01  // prevent weights → 0
    MaxWeight     = 10.0  // cap to prevent runaway reinforcement
)

type ApplyWeightRequest struct {
    NodeIDs  []string
    TenantID string
    Score    float64
}

type FeedbackRetriever struct {
    graphRepo port.GraphRepository
}

func NewFeedbackRetriever(graphRepo port.GraphRepository) *FeedbackRetriever {
    return &FeedbackRetriever{graphRepo: graphRepo}
}

// ApplyWeightAdjustment updates Neo4j edge weights for all edges from result nodes
func (r *FeedbackRetriever) ApplyWeightAdjustment(ctx context.Context, req ApplyWeightRequest) ([]string, error) {
    if len(req.NodeIDs) == 0 { return nil, nil }

    factor := BoostFactor
    if req.Score < 0 { factor = PenaltyFactor }

    // Cypher: UPDATE edge weights, clamped to [MinWeight, MaxWeight]
    cypher := `
        MATCH (n)-[r]->(m)
        WHERE n.id IN $node_ids AND n.tenant_id = $tenant_id
        SET r.weight = CASE
            WHEN (coalesce(r.weight, 1.0) * $factor) < $min_weight THEN $min_weight
            WHEN (coalesce(r.weight, 1.0) * $factor) > $max_weight THEN $max_weight
            ELSE (coalesce(r.weight, 1.0) * $factor)
        END
        RETURN n.id AS node_id
    `
    params := map[string]any{
        "node_ids":   req.NodeIDs,
        "tenant_id":  req.TenantID,
        "factor":     factor,
        "min_weight": MinWeight,
        "max_weight": MaxWeight,
    }

    affectedIDs, err := r.graphRepo.RunCypherReturnIDs(ctx, cypher, params)
    if err != nil { return nil, fmt.Errorf("weight adjustment: %w", err) }
    return affectedIDs, nil
}
```

### File 2 (NEW): `repository/postgres/interaction_repo.go`

```go
package postgres

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/google/uuid"
    "github.com/lib/pq"
    "github.com/vnp-memory/services/cognee-search/internal/domain"
)

type InteractionRepo struct { db *pgxpool.Pool }
func NewInteractionRepo(db *pgxpool.Pool) *InteractionRepo { return &InteractionRepo{db: db} }

func (r *InteractionRepo) Save(ctx context.Context, i domain.Interaction) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO cognee_interactions
            (id, tenant_id, session_id, dataset_id, query, strategy, result_ids, result_scores, node_sets, timestamp)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
    `, i.ID, i.TenantID, i.SessionID, i.DatasetID, i.Query, i.Strategy,
        pq.Array(i.ResultIDs), pq.Array(i.ResultScores), pq.Array(i.NodeSets), i.Timestamp)
    return err
}

func (r *InteractionRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Interaction, error) {
    row := r.db.QueryRow(ctx, `
        SELECT id, tenant_id, session_id, dataset_id, query, strategy, result_ids, node_sets, timestamp
        FROM cognee_interactions WHERE id = $1
    `, id)
    var i domain.Interaction
    err := row.Scan(&i.ID, &i.TenantID, &i.SessionID, &i.DatasetID, &i.Query, &i.Strategy,
        pq.Array(&i.ResultIDs), pq.Array(&i.NodeSets), &i.Timestamp)
    if err != nil { return nil, err }
    return &i, nil
}

func (r *InteractionRepo) List(ctx context.Context, tenantID, sessionID string, limit, offset int) ([]domain.Interaction, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, query, strategy, result_ids, timestamp
        FROM cognee_interactions
        WHERE tenant_id = $1 AND ($2 = '' OR session_id = $2)
        ORDER BY timestamp DESC LIMIT $3 OFFSET $4
    `, tenantID, sessionID, limit, offset)
    if err != nil { return nil, err }
    defer rows.Close()

    var result []domain.Interaction
    for rows.Next() {
        var i domain.Interaction
        rows.Scan(&i.ID, &i.Query, &i.Strategy, pq.Array(&i.ResultIDs), &i.Timestamp)
        result = append(result, i)
    }
    return result, nil
}
```

### File 3 (NEW): `repository/postgres/feedback_repo.go`

```go
package postgres

import (
    "context"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/lib/pq"
    "github.com/vnp-memory/services/cognee-search/internal/domain"
)

type FeedbackRepo struct { db *pgxpool.Pool }
func NewFeedbackRepo(db *pgxpool.Pool) *FeedbackRepo { return &FeedbackRepo{db: db} }

func (r *FeedbackRepo) Save(ctx context.Context, f domain.FeedbackRecord) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO cognee_feedback_records
            (id, interaction_id, tenant_id, score, text, affected_nodes, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, f.ID, f.InteractionID, f.TenantID, f.Score, f.Text, pq.Array(f.AffectedNodes), f.CreatedAt)
    return err
}
```

### MODIFY `retriever/registry.go` — Add StrategyFeedback

```go
// SearchStrategy constants
const (
    // ... existing ...
    StrategyFeedback SearchStrategy = "FEEDBACK"  // [NEW]
)

// In NewRetrieverRegistry():
registry["FEEDBACK"] = func() Retriever { return NewFeedbackRetriever(graphRepo) }
```

### MODIFY `grpc/handler.go` — Add ListInteractions RPC

```go
func (h *SearchHandler) ListInteractions(ctx context.Context, req *searchpb.ListInteractionsRequest) (*searchpb.ListInteractionsResponse, error) {
    interactions, err := h.interactionRepo.List(ctx, req.TenantId, req.SessionId, int(req.Limit), int(req.Offset))
    if err != nil { return nil, status.Errorf(codes.Internal, "list interactions: %v", err) }

    items := make([]*searchpb.InteractionInfo, 0, len(interactions))
    for _, i := range interactions {
        items = append(items, &searchpb.InteractionInfo{
            Id:        i.ID.String(),
            Query:     i.Query,
            Strategy:  i.Strategy,
            ResultIds: i.ResultIDs,
            Timestamp: i.Timestamp.Format(time.RFC3339),
        })
    }
    return &searchpb.ListInteractionsResponse{Interactions: items}, nil
}
```

### MODIFY `cognee-memory/domain/entity.go` — Add FeedbackScore field

```go
type MemoryFact struct {
    ID            uuid.UUID
    TenantID      string
    Content       string
    Source        string       // "feedback" | "extraction" | "user_input" | "interaction"
    Type          string       // "qa" | "feedback" | "fact"
    InteractionID *uuid.UUID   // [NEW] link to search interaction
    FeedbackScore *float64     // [NEW] associated feedback score
    CreatedAt     time.Time
}
```

### MODIFY `gateway/router.go` — Add interactions route

```go
r.Get("/api/v1/cognee/interactions", h.ForwardTo("cognee-search", "SearchService/ListInteractions"))
// Query params: ?session_id=...&limit=10&offset=0
```

---

## Verification

```bash
cd services/cognee-search
go build ./...
go test ./internal/usecase/... -run TestFeedback -v
go test ./internal/adapter/retriever/... -run TestFeedbackRetriever -v
```

**Test feedback flow:**
```go
func TestFeedbackLoop_BoostsEdgeWeights(t *testing.T) {
    // 1. Search with save_interaction=true → get interaction_id
    // 2. Submit positive feedback (score=1.0) with feedback_for=interaction_id
    // 3. Run same search again → assert score of same nodes improved
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| Search + `save_interaction=true` → `interaction_id` in response | ✅ |
| `FEEDBACK` strategy + `feedback_for` + score=1.0 → `{applied:true, affected_nodes:N}` | ✅ |
| Positive feedback → Neo4j edge weights × 1.1 | ✅ |
| Negative feedback → edge weights × 0.9 | ✅ |
| Weights clamped: [0.01, 10.0] | ✅ |
| `GET /api/v1/cognee/interactions` → history | ✅ |
| Feedback with text → cognee-memory MemoryFact saved | ✅ |
