# Solution: SOL-005 — Feedback Loop & Self-Improvement

**CR ID:** CR-COGNEE-005  
**Solution ID:** SOL-005  
**Priority:** Medium (Wave 3)  
**Architecture:** EXTEND `services/cognee-search/` + `services/cognee-memory/` + PostgreSQL 2 tables

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- `services/cognee-search/internal/adapter/retriever/registry.go` — registry map `SearchStrategy → RetrieverFactory`.
- `cognee-search` expose gRPC port 9013, monolith via bufconn.
- `cognee-memory` — lưu `MemoryFact` entities (đã tồn tại).
- **Neo4j** — graph store với edges có `weight` property — có thể UPDATE weight.
- **PostgreSQL** — metadata storage, sẵn sàng cho `interactions` + `feedback_records` tables.
- NATS event bus embedded — thêm `cognee.search.feedback.applied` subject.

---

## 2. Giải pháp chi tiết

### 2.1. [NEW] Domain Entities — `services/cognee-search/internal/domain/entity.go`

```go
// Thêm vào services/cognee-search/internal/domain/entity.go

// Interaction — log mỗi search call (khi save_interaction: true)
type Interaction struct {
    ID           uuid.UUID
    TenantID     string
    SessionID    *string
    DatasetID    *uuid.UUID
    Query        string
    Strategy     SearchStrategy
    ResultIDs    []string       // IDs của kết quả trả về (node IDs hoặc chunk IDs)
    ResultScores []float64
    NodeSets     []string       // NodeSets used in search
    Timestamp    time.Time
}

// FeedbackRecord — phản hồi về 1 interaction
type FeedbackRecord struct {
    ID            uuid.UUID
    InteractionID uuid.UUID
    TenantID      string
    Score         float64     // -1.0 to 1.0 (negative = bad, positive = good)
    Text          string      // optional text comment
    AffectedNodes []string    // node IDs whose weights were adjusted
    CreatedAt     time.Time
}
```

### 2.2. [MODIFY] Search Use Case

```go
// services/cognee-search/internal/usecase/search.go

type SearchRequest struct {
    Query           string
    Strategies      []SearchStrategy
    DatasetID       *uuid.UUID
    TenantID        string
    NodeSets        []string    // from CR-002
    TopK            int
    // [NEW] Feedback flags
    SaveInteraction bool        // if true: log Interaction record
    SessionID       *string     // optional session grouping
    FeedbackFor     *string     // if set: this is a FEEDBACK call (interaction UUID)
    FeedbackScore   *float64    // -1.0 to 1.0
    FeedbackText    string
}

type SearchResponse struct {
    Results       []SearchResult
    // [NEW]
    InteractionID *string   // returned when SaveInteraction=true
}

func (uc *SearchUseCase) Execute(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
    // Handle FEEDBACK strategy
    if req.FeedbackFor != nil {
        return uc.handleFeedback(ctx, req)
    }

    // Normal search
    results, err := uc.searchWithStrategies(ctx, req)
    if err != nil { return nil, err }

    resp := &SearchResponse{Results: results}

    // [NEW] Save interaction if requested
    if req.SaveInteraction {
        interactionID := uuid.New()
        interaction := domain.Interaction{
            ID:           interactionID,
            TenantID:     req.TenantID,
            SessionID:    req.SessionID,
            DatasetID:    req.DatasetID,
            Query:        req.Query,
            Strategy:     req.Strategies[0],
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

// handleFeedback processes FEEDBACK strategy (no search results returned)
func (uc *SearchUseCase) handleFeedback(ctx context.Context, req SearchRequest) (*SearchResponse, error) {
    interactionID, err := uuid.Parse(*req.FeedbackFor)
    if err != nil { return nil, fmt.Errorf("invalid feedback_for UUID: %w", err) }

    // Load original interaction
    interaction, err := uc.interactionRepo.GetByID(ctx, interactionID)
    if err != nil { return nil, fmt.Errorf("interaction not found: %w", err) }

    // Apply edge weight reinforcement
    var affectedNodes []string
    if req.FeedbackScore != nil && len(interaction.ResultIDs) > 0 {
        affectedNodes, err = uc.feedbackRetriever.ApplyWeightAdjustment(ctx, ApplyWeightRequest{
            NodeIDs:  interaction.ResultIDs,
            TenantID: req.TenantID,
            Score:    *req.FeedbackScore,
        })
        if err != nil {
            // Non-fatal: log but continue
            _ = err
        }
    }

    // Save feedback record
    feedback := domain.FeedbackRecord{
        ID:            uuid.New(),
        InteractionID: interactionID,
        TenantID:      req.TenantID,
        Score:         req.FeedbackScore != nil ? *req.FeedbackScore : 0,
        Text:          req.FeedbackText,
        AffectedNodes: affectedNodes,
        CreatedAt:     time.Now(),
    }
    uc.feedbackRepo.Save(ctx, feedback)

    // Publish NATS event
    uc.eventPub.Publish(ctx, "cognee.search.feedback.applied", map[string]any{
        "interaction_id":  interactionID.String(),
        "score":           feedback.Score,
        "affected_nodes":  affectedNodes,
        "tenant_id":       req.TenantID,
    })

    // Optionally: save to cognee-memory as a MemoryFact
    if req.FeedbackText != "" {
        go uc.memoryClient.RememberFact(context.Background(), MemoryFactRequest{
            TenantID:      req.TenantID,
            Fact:          fmt.Sprintf("User rated query '%s' as %.1f: %s", interaction.Query, feedback.Score, req.FeedbackText),
            Source:        "feedback",
            InteractionID: &interactionID,
            FeedbackScore: req.FeedbackScore,
        })
    }

    return &SearchResponse{
        Results: nil,
        Metadata: map[string]any{
            "applied":        true,
            "affected_nodes": len(affectedNodes),
        },
    }, nil
}
```

### 2.3. [NEW] FeedbackRetriever — Neo4j Edge Weight Adjustment

```go
// services/cognee-search/internal/adapter/retriever/feedback.go

package retriever

import (
    "context"
    "fmt"
)

const (
    BoostFactor  = 1.1    // positive feedback: multiply edge weights by 1.1
    PenaltyFactor = 0.9   // negative feedback: multiply edge weights by 0.9
    MinWeight    = 0.01   // prevent weights from reaching 0
    MaxWeight    = 10.0   // cap weights to prevent runaway reinforcement
)

type ApplyWeightRequest struct {
    NodeIDs  []string
    TenantID string
    Score    float64   // -1.0 to 1.0
}

type FeedbackRetriever struct {
    graphRepo port.GraphRepository
}

func NewFeedbackRetriever(graphRepo port.GraphRepository) *FeedbackRetriever {
    return &FeedbackRetriever{graphRepo}
}

// ApplyWeightAdjustment updates Neo4j edge weights for nodes in the result set
// Positive feedback → boost; Negative feedback → penalize
func (r *FeedbackRetriever) ApplyWeightAdjustment(ctx context.Context, req ApplyWeightRequest) ([]string, error) {
    if len(req.NodeIDs) == 0 { return nil, nil }

    factor := BoostFactor
    if req.Score < 0 { factor = PenaltyFactor }

    // Cypher: UPDATE edge weights for outgoing edges from these nodes
    // Clamp to [MinWeight, MaxWeight]
    cypher := `
        MATCH (n)-[r]->(m)
        WHERE n.id IN $node_ids AND n.tenant_id = $tenant_id
        SET r.weight = CASE
            WHEN (coalesce(r.weight, 1.0) * $factor) < $min_weight THEN $min_weight
            WHEN (coalesce(r.weight, 1.0) * $factor) > $max_weight THEN $max_weight
            ELSE (coalesce(r.weight, 1.0) * $factor)
        END
        RETURN n.id as node_id
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

### 2.4. [MODIFY] Retriever Registry — Add FEEDBACK Strategy

```go
// services/cognee-search/internal/adapter/retriever/registry.go

type SearchStrategy string
const (
    StrategySimilarity      SearchStrategy = "SIMILARITY"
    StrategyGraphCompletion SearchStrategy = "GRAPH_COMPLETION"
    StrategyGraphSummary    SearchStrategy = "GRAPH_SUMMARY"
    StrategyKeyword         SearchStrategy = "KEYWORD"
    StrategyChunks          SearchStrategy = "CHUNKS"
    StrategyTemporal        SearchStrategy = "TEMPORAL"
    StrategyMultiHop        SearchStrategy = "MULTI_HOP"
    StrategyHybrid          SearchStrategy = "HYBRID"
    StrategyFeelingLucky    SearchStrategy = "FEELING_LUCKY"
    StrategyFeedback        SearchStrategy = "FEEDBACK"   // [NEW]
)

var retrieverRegistry = map[SearchStrategy]RetrieverFactory{
    StrategySimilarity:      func(deps) Retriever { return &VectorRetriever{...} },
    StrategyGraphCompletion: func(deps) Retriever { return &GraphRetriever{...} },
    StrategyKeyword:         func(deps) Retriever { return &KeywordRetriever{...} },
    StrategyChunks:          func(deps) Retriever { return &ChunkRetriever{...} },
    StrategyTemporal:        func(deps) Retriever { return &TemporalRetriever{...} },
    StrategyMultiHop:        func(deps) Retriever { return &MultiHopRetriever{...} },
    StrategyHybrid:          func(deps) Retriever { return &HybridRetriever{...} },
    StrategyFeelingLucky:    func(deps) Retriever { return &FeelingLuckyRetriever{...} },
    StrategyFeedback:        func(deps) Retriever { return &FeedbackRetriever{...} },  // [NEW]
}
```

### 2.5. [MODIFY] `services/cognee-memory` — MemoryFact + FeedbackEntry

```go
// services/cognee-memory/internal/usecase/remember.go

type RememberRequest struct {
    TenantID      string
    Type          string   // "qa" | "feedback" | "fact" | "summary"
    Content       string
    Source        string   // "feedback" | "extraction" | "user_input"
    InteractionID *uuid.UUID
    FeedbackScore *float64
}

func (uc *RememberUseCase) Execute(ctx context.Context, req RememberRequest) error {
    fact := domain.MemoryFact{
        ID:            uuid.New(),
        TenantID:      req.TenantID,
        Content:       req.Content,
        Source:        req.Source,
        InteractionID: req.InteractionID,
        FeedbackScore: req.FeedbackScore,
        CreatedAt:     time.Now(),
    }
    return uc.factRepo.Save(ctx, fact)
}
```

```go
// services/cognee-memory/internal/domain/entity.go

type MemoryFact struct {
    ID            uuid.UUID
    TenantID      string
    Content       string
    Source        string          // "feedback" | "extraction" | "user_input"
    InteractionID *uuid.UUID      // [NEW]
    FeedbackScore *float64        // [NEW]
    CreatedAt     time.Time
}
```

### 2.6. [NEW] PostgreSQL Schema

```sql
-- Migration: 0022_cognee_interactions.up.sql

CREATE TABLE cognee_interactions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     TEXT NOT NULL,
    session_id    TEXT,
    dataset_id    UUID,
    query         TEXT NOT NULL,
    strategy      TEXT NOT NULL,
    result_ids    TEXT[] DEFAULT '{}',
    result_scores FLOAT8[] DEFAULT '{}',
    node_sets     TEXT[] DEFAULT '{}',
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE cognee_feedback_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id  UUID NOT NULL REFERENCES cognee_interactions(id),
    tenant_id       TEXT NOT NULL,
    score           FLOAT8 NOT NULL,
    text            TEXT,
    affected_nodes  TEXT[] DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cognee_interactions_tenant ON cognee_interactions(tenant_id, timestamp DESC);
CREATE INDEX idx_cognee_interactions_session ON cognee_interactions(session_id, timestamp DESC) WHERE session_id IS NOT NULL;
CREATE INDEX idx_cognee_feedback_interaction ON cognee_feedback_records(interaction_id);
```

### 2.7. [MODIFY] Gateway Routes

```go
// gateway/internal/adapter/handler/router.go

// Extended POST /api/v1/cognee/search — handles both search and FEEDBACK strategy
// Response: {results, interaction_id} (interaction_id present only if save_interaction=true)

// [NEW] Interactions history
r.Get("/api/v1/cognee/interactions", h.ForwardTo("cognee-search", "SearchService/ListInteractions"))
// Query params: ?session_id=...&limit=10&offset=0
```

### 2.8. [MODIFY] Search Response in Gateway Handler

```go
// gateway/internal/adapter/handler/cognee_handler.go

func (h *CogneeHandler) Search(w http.ResponseWriter, r *http.Request) {
    var body struct {
        Query           string   `json:"query"`
        Strategies      []string `json:"strategies"`
        DatasetName     string   `json:"dataset_name"`
        NodeSets        []string `json:"node_sets"`
        TopK            int      `json:"top_k"`
        SaveInteraction bool     `json:"save_interaction"`    // [NEW]
        SessionID       string   `json:"session_id"`          // [NEW]
        FeedbackFor     string   `json:"feedback_for"`        // [NEW]
        FeedbackScore   *float64 `json:"feedback_score"`      // [NEW]
        FeedbackText    string   `json:"feedback_text"`       // [NEW]
    }
    json.NewDecoder(r.Body).Decode(&body)

    // Forward to cognee-search gRPC with new fields
    // Response includes interaction_id if save_interaction=true
}
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/cognee-search/internal/adapter/retriever/feedback.go` | FeedbackRetriever, ApplyWeightAdjustment |
| `services/cognee-search/internal/adapter/repository/postgres/interaction_repo.go` | Interaction CRUD |
| `services/cognee-search/internal/adapter/repository/postgres/feedback_repo.go` | FeedbackRecord CRUD |
| `db/migrations/0022_cognee_interactions.up.sql` | interactions + feedback_records tables |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `services/cognee-search/internal/domain/entity.go` | + Interaction, FeedbackRecord types |
| `services/cognee-search/internal/usecase/search.go` | + save_interaction, handleFeedback() |
| `services/cognee-search/internal/usecase/port/output.go` | + InteractionRepository, FeedbackRepository |
| `services/cognee-search/internal/adapter/retriever/registry.go` | + StrategyFeedback |
| `services/cognee-search/internal/adapter/grpc/handler.go` | + ListInteractions() RPC |
| `api/proto/cognee/search/v1/search.proto` | + save_interaction, feedback_for, feedback_score, interaction_id |
| `services/cognee-memory/internal/domain/entity.go` | + InteractionID, FeedbackScore on MemoryFact |
| `services/cognee-memory/internal/usecase/remember.go` | + feedback type handling |
| `gateway/internal/adapter/handler/cognee_handler.go` | + feedback fields in request/response |
| `gateway/internal/adapter/handler/router.go` | + GET /api/v1/cognee/interactions |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-COGNEE-005 | Covered by |
|--------------------|-----------|
| Search + save_interaction → Interaction saved, response có interaction_id | handleSaveInteraction() |
| FEEDBACK strategy + feedback_for → {applied:true, affected_nodes:N} | handleFeedback() |
| Positive feedback → Neo4j edge weights tăng | FeedbackRetriever × BoostFactor=1.1 |
| Negative feedback → edge weights giảm | FeedbackRetriever × PenaltyFactor=0.9 |
| Cùng query sau feedback → order results thay đổi | Weight-based ranking in GRAPH_COMPLETION |
| GET /api/v1/cognee/interactions → history | ListInteractions() + interaction_repo |
