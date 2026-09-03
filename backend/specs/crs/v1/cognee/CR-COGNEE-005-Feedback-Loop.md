# Change Request: CR-COGNEE-005 — Feedback Loop & Self-Improvement

**CR ID:** CR-COGNEE-005  
**Component:** `services/kg-service` | `gateway`  
**Priority:** Medium  
**Status:** Implemented  
**Reference:** Cognee PRD §4.5, SRS FR-SEARCH-01/FR-MEM-01, URD UR-SEARCH-03/UR-MEM-04  
**Spec:** `references/cognee/specs/services/04-cognee-search.md`, `05-cognee-memory.md`

---

## 1. Mô tả

Triển khai **Feedback Loop & Self-Improvement**: Hệ thống lưu lại tất cả interactions (query, result, user rating), và dùng feedback đó để:
1. **Reinforce** retrieval weights — kết quả tốt được boost lên trong lần tìm kiếm tiếp theo.
2. **Lưu vào MemoryFact** — feedback thành facts được persist vào Knowledge Graph.
3. **Expose SearchType `FEEDBACK`** — search query đặc biệt chỉ dùng để process feedback.

---

## 2. Vấn đề hiện tại

- `cognee-search`: `SearchRequest` không có `save_interaction` flag. Interactions không được lưu.
- `cognee-search`: Retriever Registry (`internal/adapter/retriever/registry.go`) không có entry cho strategy `FEEDBACK`.
- `cognee-memory`: `MemoryFact` entity đã có trong domain nhưng không có use case nào tạo/cập nhật nó.
- Không có cơ chế nào adjust graph edge weights dựa trên user feedback.

---

## 3. Thay đổi đề xuất

### 3.1. Service: `services/cognee-search`

**[MODIFY]** `internal/domain/entity.go`

```go
// Interaction — log mỗi search call
type Interaction struct {
    ID          uuid.UUID
    TenantID    string
    SessionID   *string
    Query       string
    Strategy    SearchStrategy
    ResultIDs   []uuid.UUID       // IDs của kết quả trả về
    ResultScores []float64
    Timestamp   time.Time
}

// FeedbackRecord — user/agent phản hồi về 1 interaction
type FeedbackRecord struct {
    ID            uuid.UUID
    InteractionID uuid.UUID
    Score         float64    // -1.0 to 1.0 (negative = bad, positive = good)
    Text          string     // optional text comment
    CreatedAt     time.Time
}
```

**[MODIFY]** `internal/usecase/search.go`

```go
type SearchRequest struct {
    // ... existing fields ...
    SaveInteraction bool      // [NEW] if true, log Interaction record
    FeedbackFor     *string   // [NEW] if set: this is a FEEDBACK call, not search
    FeedbackScore   *float64  // [NEW] -1.0 to 1.0
}

// In SearchUseCase.Execute():
// if req.SaveInteraction: save Interaction to DB
// if req.FeedbackFor != nil: route to FeedbackRetriever instead of search
```

**[NEW]** `internal/adapter/retriever/feedback.go`

```go
// FeedbackRetriever — xử lý feedback, không trả về search results
type FeedbackRetriever struct {
    interactionRepo port.InteractionRepository
    graphRepo       port.GraphRepository   // để update edge weights
    memoryClient    port.MemoryServiceClient // gRPC call → cognee-memory
}

// Execute(req SearchRequest):
// 1. Load Interaction by req.FeedbackFor (interaction ID)
// 2. Create FeedbackRecord(score, text)
// 3. If score > 0: boost edge weights của các ResultIDs trong Neo4j
//    e.g. MATCH (n)-[r]->() WHERE id(n) IN $ids SET r.weight = r.weight * 1.1
// 4. If score < 0: penalize weights
// 5. Emit event: cognee.search.feedback.applied
// 6. (Optional) gRPC call → cognee-memory RememberFact(fact about feedback)
```

**[MODIFY]** `internal/adapter/retriever/registry.go`

```go
var retrieverRegistry = map[SearchStrategy]RetrieverFactory{
    // existing ...
    StrategyFeedback: func(deps) Retriever { return &FeedbackRetriever{...} }, // [NEW]
}
```

**[NEW]** `internal/usecase/port/output.go`

```go
type InteractionRepository interface {
    Save(ctx, Interaction) error
    GetByID(ctx, uuid.UUID) (*Interaction, error)
    ListBySession(ctx, sessionID string, limit int) ([]Interaction, error)
}
```

### 3.2. Service: `services/cognee-memory`

**[MODIFY]** `internal/usecase/remember.go`

Thêm support cho `FeedbackEntry` type:
```go
// When type == "feedback":
// 1. Load referenced QA from session store
// 2. Create MemoryFact: "User rated query X as Y/10 with comment Z"
// 3. (Optional) Persist fact → cognify pipeline via PersistSession
```

**[MODIFY]** `internal/domain/entity.go`

```go
type MemoryFact struct {
    // existing fields ...
    Source      string          // "feedback" | "extraction" | "user_input"
    InteractionID *uuid.UUID    // [NEW] link back to search interaction
    FeedbackScore *float64      // [NEW]
}
```

### 3.3. Service: `services/vnp-gateway`

**[MODIFY]** `internal/adapter/http/cognee_routes.go`

Mở rộng `POST /api/v1/cognee/search`:
```json
{
  "query": "What are the main findings?",
  "strategies": ["GRAPH_COMPLETION"],
  "save_interaction": true
}
```

Response thêm trường:
```json
{
  "results": [...],
  "interaction_id": "uuid-of-this-search"
}
```

Để submit feedback:
```json
{
  "query": "helpful",
  "strategies": ["FEEDBACK"],
  "feedback_for": "uuid-of-interaction",
  "feedback_score": 0.9,
  "feedback_text": "Answer was accurate and concise"
}
```

**[NEW]** `POST /api/v1/cognee/interactions` — list interactions (history):
```
GET /api/v1/cognee/interactions?session_id=...&limit=10
```

---

## 4. NATS Events

| Subject | Direction | Payload |
|---|---|---|
| `cognee.search.feedback.applied` | Publish (from cognee-search) | `{interaction_id, score, affected_node_ids[], tenant_id}` |

---

## 5. Traceability

| Item | Ref |
|---|---|
| gRPC port | `cognee-search:9013`, `cognee-memory:9014` |
| New Strategy | `SearchStrategy = "FEEDBACK"` |
| New Retriever | `FeedbackRetriever` in `retriever/feedback.go` |
| DB changes | Postgres: `interactions` table, `feedback_records` table |
| Graph changes | Neo4j edge weight update via Cypher |
| NATS published | `cognee.search.feedback.applied` |

---

## 6. Acceptance Criteria

- [x] Search với `save_interaction: true` gửi cấu hình tới Cognee Python để lưu Interaction.
- [x] Submit feedback với `search_type: "FEEDBACK"` và `feedback_for: <interaction_id>` kích hoạt xử lý feedback trên Python.
- [x] API /v1/cognee/search hỗ trợ đầy đủ các trường mới (SaveInteraction, FeedbackFor, FeedbackScore).
- [x] Được tích hợp sẵn thông qua standard MCP tools (CR-COGNEE-007).

---

## 7. Implementation Notes

**Implemented in:** `services/kg-service` + `gateway` (MERGE-P2-T2)
Các thay đổi đã được implement dưới dạng passthrough config cho Cognee Python service.

| File | Change |
|------|--------|
| `services/kg-service/internal/domain/cognee/entity.go` | `[MODIFY]` `SearchRequest` thêm `SaveInteraction`, `FeedbackFor`, `FeedbackScore` |
| `services/kg-service/internal/adapter/cognee/client.go` | `[MODIFY]` `Search()` truyền toàn bộ `SearchRequest` object |
| `services/kg-service/internal/usecase/cognee/service.go` | `[MODIFY]` `CogneeSearchUseCase` |
| `services/kg-service/internal/adapter/grpc/router.go` | `[MODIFY]` `CogneeSearch` handler |
