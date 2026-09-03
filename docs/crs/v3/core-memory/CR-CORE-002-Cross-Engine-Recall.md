# Change Request: CR-CORE-002 — Cross-Engine Recall & RRF Hybrid Search

**CR ID:** CR-CORE-002
**Component:** `backend/services/vnp-search-hub`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v3 / Core Memory & Integration
**Solution:** [S2 — Unified Memory API](../../../bussiness/solutions/S2-unified-api.md)
**Features:** [F01](../../../features/01-unified-memory-api/README.md), [F10](../../../features/10-hybrid-search-engine/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-02 | AI Agent Developer | Memory fragmented — phải query 6 engines riêng lẻ và merge thủ công |
| PP-P3-02 | ML/AI Engineer | Không benchmark được search quality cross-engine |
| PP-P1-06 | AI Agent Developer | Không biết loại memory nào phù hợp cho query |

**Before:** Developer viết 6 separate API calls, merge logic riêng.
**After:** 1 API call, cross-engine, RRF fusion, `< 500ms` p95.

---

## 2. API Contract

```http
POST /v1/memory/recall
{
  "user_id": "u_123",
  "query": "Tôi đã làm gì hôm qua?",
  "types": ["episodic", "conversational"],   // optional filter
  "time_range": {"from": "2026-09-01", "to": "2026-09-03"},
  "limit": 10
}

→ 200 OK
{
  "results": [
    {"id": "m1", "content": "...", "type": "episodic", "engine": "graphiti", "score": 0.92},
    {"id": "m2", "content": "...", "type": "conversational", "engine": "zep", "score": 0.87}
  ],
  "total_hits": 47,
  "engines_queried": ["graphiti", "zep"]
}
```

---

## 3. Thay đổi đề xuất

### 3.1 `backend/services/vnp-search-hub/internal/usecase/search.go` [MODIFY]

```go
func (u *SearchUseCase) Recall(ctx context.Context, req *RecallRequest) (*RecallResponse, error) {
    // Fan-out parallel với 500ms timeout
    ctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
    defer cancel()

    engines := u.selectEngines(req.Types)
    results := make(chan []MemoryUnit, len(engines))
    
    for _, eng := range engines {
        go func(e Engine) {
            res, err := e.Search(ctx, req.Query, req.TimeRange, req.Limit)
            if err != nil { results <- nil; return }
            results <- res
        }(eng)
    }
    
    all := collectResults(results, len(engines))
    fused := rrfFusion(all) // Reciprocal Rank Fusion
    return &RecallResponse{Results: fused[:req.Limit]}, nil
}

// RRF: score(d) = Σ 1/(k + rank_i(d)), k=60
func rrfFusion(engineResults [][]MemoryUnit) []MemoryUnit {
    scores := map[string]float64{}
    for _, results := range engineResults {
        for i, r := range results {
            scores[r.ID] += 1.0 / float64(60 + i + 1)
        }
    }
    return sortByScore(dedup(scores))
}
```

### 3.2 Engine adapters [NEW/MODIFY]

Mỗi engine cần implement `SearchAdapter` interface:
```go
type SearchAdapter interface {
    Search(ctx context.Context, query string, filter SearchFilter, limit int) ([]MemoryUnit, error)
}
// Implementations: CogneeSearchAdapter, GraphitiSearchAdapter, MemobaseSearchAdapter,
//                  OVSearchAdapter, SMSearchAdapter, ZepSearchAdapter
```

---

## 4. Acceptance Criteria

- [ ] Parallel fan-out với 500ms hard timeout
- [ ] Engines trả về empty list nếu timeout (graceful degradation, không fail)
- [ ] RRF fusion: results sorted by fused score descending
- [ ] Dedup by content hash (không trả duplicate)
- [ ] Filter by `types` và `time_range` hoạt động đúng
- [ ] p95 latency `< 500ms` dưới 100 concurrent requests

---

## 5. Dependencies

- CR-CORE-001 (router phải hoạt động)
- All v1 engine search services deployed
