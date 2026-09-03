# TASK-CORE-004 — RRF Cross-Engine Search Hub

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-004 |
| **Wave** | 2 |
| **Solution** | [SOL-CORE-002](../solutions/SOL-CORE-002-Cross-Engine-Recall.md) |
| **Component** | `services/vnp-search-hub/internal/usecase/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-CORE-001 |
| **Estimated** | 5h |

---

## Mục tiêu

Implement cross-engine recall với RRF fusion trong `vnp-search-hub`.

---

## Công việc cụ thể

### 1. `services/vnp-search-hub/internal/domain/search.go` [MODIFY]

```go
type RecallRequest struct {
    TenantID  string     `json:"tenant_id"`
    UserID    string     `json:"user_id"`
    Query     string     `json:"query"`
    Types     []string   `json:"types,omitempty"`  // filter by engine type
    TimeRange *TimeRange `json:"time_range,omitempty"`
    Limit     int        `json:"limit"`             // default: 10
}

type RecallResponse struct {
    Results        []MemoryUnit `json:"results"`
    TotalHits      int          `json:"total_hits"`
    EnginesQueried []string     `json:"engines_queried"`
}

type MemoryUnit struct {
    ID      string  `json:"id"`
    Content string  `json:"content"`
    Type    string  `json:"type"`
    Engine  string  `json:"engine"`
    Score   float64 `json:"score"`
}
```

### 2. `services/vnp-search-hub/internal/usecase/search.go` [MODIFY]

```go
const (
    rrfK          = 60
    defaultTimeout = 500 * time.Millisecond
)

func (u *SearchUseCase) Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error) {
    if req.Limit == 0 { req.Limit = 10 }

    ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
    defer cancel()

    engines := u.selectEngines(req.Types, req.TenantID)
    type result struct {
        units  []MemoryUnit
        engine string
    }
    ch := make(chan result, len(engines))

    for _, eng := range engines {
        go func(e EngineClient) {
            units, err := e.Search(ctx, &SearchRequest{
                TenantID:  req.TenantID, UserID: req.UserID,
                Query:     req.Query, TimeRange: req.TimeRange,
                Limit:     req.Limit * 3, // over-fetch for fusion
            })
            if err != nil { ch <- result{nil, e.Name()}; return }
            ch <- result{units, e.Name()}
        }(eng)
    }

    engineResults := map[string][]MemoryUnit{}
    for i := 0; i < len(engines); i++ {
        r := <-ch
        if r.units != nil { engineResults[r.engine] = r.units }
    }

    fused := rrfFusion(engineResults)
    if len(fused) > req.Limit { fused = fused[:req.Limit] }

    queried := make([]string, 0, len(engineResults))
    for k := range engineResults { queried = append(queried, k) }

    return &RecallResponse{
        Results: fused, TotalHits: countTotal(engineResults),
        EnginesQueried: queried,
    }, nil
}

// rrfFusion: score(d) = Σ 1/(k + rank_i(d)), k=60
func rrfFusion(engineResults map[string][]MemoryUnit) []MemoryUnit {
    scores := map[string]float64{}
    byID   := map[string]MemoryUnit{}
    for _, results := range engineResults {
        for rank, unit := range results {
            scores[unit.ID] += 1.0 / float64(rrfK + rank + 1)
            byID[unit.ID] = unit
        }
    }
    type scored struct{ id string; s float64 }
    sorted := make([]scored, 0, len(scores))
    for id, s := range scores { sorted = append(sorted, scored{id, s}) }
    sort.Slice(sorted, func(i, j int) bool { return sorted[i].s > sorted[j].s })
    out := make([]MemoryUnit, 0, len(sorted))
    for _, s := range sorted {
        u := byID[s.id]; u.Score = s.s
        out = append(out, u)
    }
    return out
}
```

### 3. Unit test

```go
func TestRRFFusion_MergesResults(t *testing.T) {
    input := map[string][]MemoryUnit{
        "graphiti": {{ID: "m1", Score: 0.9}, {ID: "m2", Score: 0.8}},
        "cognee":   {{ID: "m2", Score: 0.7}, {ID: "m3", Score: 0.6}},
    }
    fused := rrfFusion(input)
    assert.Equal(t, "m2", fused[0].ID, "m2 appears in both engines → highest RRF score")
    assert.Len(t, fused, 3)
}

func TestRecall_TimesOutGracefully(t *testing.T) {
    // slow engine → result omitted, not error
}
```

---

## Acceptance Criteria

- [ ] Parallel fan-out to all selected engines
- [ ] 500ms timeout: engines not responding are skipped (not error)
- [ ] RRF fusion: items in multiple engines rank higher
- [ ] `types` filter limits which engines are queried
- [ ] `go test ./services/vnp-search-hub/...` passes

## Files

```
services/vnp-search-hub/internal/domain/search.go   [MODIFY]
services/vnp-search-hub/internal/usecase/search.go  [MODIFY — RRF]
services/vnp-search-hub/internal/usecase/search_test.go [NEW]
```
