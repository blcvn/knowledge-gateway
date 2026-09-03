# SOL-CORE-002 — Solution: Cross-Engine Recall & RRF Hybrid Search

| Field | Value |
|---|---|
| **Solution ID** | SOL-CORE-002 |
| **CR** | [CR-CORE-002](../../../../docs/crs/v3/core-memory/CR-CORE-002-Cross-Engine-Recall.md) |
| **TDD ref** | [backend-api-specs.md](../../../tdd/backend-api-specs.md) §Recall API |
| **Status** | Open |
| **Priority** | 🔴 Critical |

**Trạng thái:** ✅ Implemented  
**Ghi chú audit:** vnp-search-hub RecallService: cross-engine recall with RRF fusion
---

## 1. Phân tích kiến trúc

`vnp-search-hub` là service tập trung xử lý cross-engine search. TDD đã mô tả:
- Parallel fan-out tới tất cả engines đã register
- BM25 + Vector + RRF fusion
- 500ms timeout toàn bộ flow

Service hiện tại cần implement:
1. Engine selector (filter by `types`)
2. RRF fusion algorithm (k=60)
3. TimeRange filter pass-through
4. Score normalization trước khi merge

---

## 2. Giải pháp

### 2.1 `services/vnp-search-hub/internal/usecase/search.go` [MODIFY]

```go
const rrfK = 60 // standard RRF constant

func (u *SearchUseCase) Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error) {
    ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
    defer cancel()

    engines := u.selectEngines(req.Types, req.TenantID)
    type engineResult struct {
        units  []MemoryUnit
        engine string
    }
    ch := make(chan engineResult, len(engines))

    for _, eng := range engines {
        go func(e EngineClient) {
            res, err := e.Search(ctx, &SearchRequest{
                TenantID:  req.TenantID,
                UserID:    req.UserID,
                Query:     req.Query,
                TimeRange: req.TimeRange,
                Limit:     req.Limit * 3, // over-fetch for RRF
            })
            if err != nil {
                ch <- engineResult{nil, e.Name()}
                return
            }
            ch <- engineResult{res.Units, e.Name()}
        }(eng)
    }

    allResults := map[string][]MemoryUnit{}
    for i := 0; i < len(engines); i++ {
        r := <-ch
        if r.units != nil {
            allResults[r.engine] = r.units
        }
    }

    fused := rrfFusion(allResults)
    if len(fused) > req.Limit {
        fused = fused[:req.Limit]
    }

    engines_queried := make([]string, 0, len(allResults))
    for k := range allResults { engines_queried = append(engines_queried, k) }

    return &RecallResponse{
        Results:        fused,
        TotalHits:      countTotal(allResults),
        EnginesQueried: engines_queried,
    }, nil
}

func rrfFusion(engineResults map[string][]MemoryUnit) []MemoryUnit {
    scores := map[string]float64{}
    byID   := map[string]MemoryUnit{}

    for _, results := range engineResults {
        for rank, unit := range results {
            scores[unit.ID] += 1.0 / float64(rrfK+rank+1)
            byID[unit.ID] = unit
        }
    }

    // Sort by fused score
    type scored struct { id string; score float64 }
    sorted := make([]scored, 0, len(scores))
    for id, s := range scores {
        sorted = append(sorted, scored{id, s})
    }
    sort.Slice(sorted, func(i, j int) bool { return sorted[i].score > sorted[j].score })

    out := make([]MemoryUnit, 0, len(sorted))
    for _, s := range sorted {
        unit := byID[s.id]
        unit.Score = s.score
        out = append(out, unit)
    }
    return out
}
```

### 2.2 Engine selector

```go
func (u *SearchUseCase) selectEngines(types []string, tenantID string) []EngineClient {
    typeSet := map[string]string{
        "episodic":       "graphiti-search",
        "semantic":       "cognee-search",
        "conversational": "zep-search",
        "profile":        "memobase-engine",
        "procedural":     "ov-search",
        "adaptive":       "sm-search",
    }
    var out []EngineClient
    for t, svc := range typeSet {
        if len(types) == 0 || contains(types, t) {
            conn := u.registry.Get(svc)
            if conn != nil {
                out = append(out, NewEngineClient(conn, t))
            }
        }
    }
    return out
}
```

---

## 3. File Changes

| File | Action |
|---|---|
| `services/vnp-search-hub/internal/usecase/search.go` | MODIFY — add RRF fusion |
| `services/vnp-search-hub/internal/domain/search.go` | MODIFY — add EnginesQueried field |
| `services/vnp-search-hub/internal/adapter/grpc/handler.go` | MODIFY — pass TimeRange |

---

## 4. Acceptance Criteria

- [ ] `POST /v1/memory/recall` p95 < 500ms với 6 engines
- [ ] Types filter hoạt động đúng (chỉ query engines tương ứng)
- [ ] TimeRange được pass xuống từng engine
- [ ] Engines không respond trong 500ms bị skip (graceful degradation)
- [ ] Response có `engines_queried` list
