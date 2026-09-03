# SOL-CORE-005 — Solution: Temporal Reasoning Pipeline

| Field | Value |
|---|---|
| **Solution ID** | SOL-CORE-005 |
| **CR** | [CR-CORE-005](../../../../docs/crs/v3/core-memory/CR-CORE-005-Temporal-Reasoning.md) |
| **TDD ref** | [03-graphiti-services.md](../../../tdd/architecture/03-graphiti-services.md) |
| **Status** | Open |
| **Priority** | 🟡 High |

**Trạng thái:** 🔄 Partial  
**Ghi chú audit:** Temporal filter in search spec; full temporal reasoning in search_orchestrator partial
---

## 1. Giải pháp

Graphiti episodic graph đã có temporal edges. Cần thêm:
1. Relative time resolution ("hôm qua", "tuần trước")
2. Timeline query endpoint
3. Temporal graph traversal

### 1.1 `services/graphiti-search/internal/usecase/temporal.go` [NEW]

```go
type TemporalSearchUseCase struct {
    graphRepo port.GraphRepository
    llm       port.LLMClient
}

// ResolveTemporalQuery — "what did I do last week?" → time range + cypher
func (u *TemporalSearchUseCase) ResolveTemporalQuery(ctx context.Context, query string) (*TimeRange, error) {
    // Use LLM to extract temporal references
    prompt := fmt.Sprintf(`Extract time range from: "%s". Return JSON: {"from": "ISO8601", "to": "ISO8601"}`, query)
    resp, _ := u.llm.Complete(ctx, &port.CompletionRequest{Prompt: prompt, MaxTokens: 50})
    var tr TimeRange
    json.Unmarshal([]byte(resp.Content), &tr)
    return &tr, nil
}

// SearchTemporal — graph traversal with time constraints
func (u *TemporalSearchUseCase) SearchTemporal(ctx context.Context, req *TemporalSearchRequest) ([]*Episode, error) {
    cypher := `
        MATCH (e:Episode)
        WHERE e.tenant_id = $tenant_id AND e.user_id = $user_id
          AND e.created_at >= $from AND e.created_at <= $to
        OPTIONAL MATCH (e)-[:INVOLVES]->(en:Entity)
        RETURN e, collect(en) as entities
        ORDER BY e.created_at DESC
        LIMIT $limit`
    
    return u.graphRepo.QueryEpisodes(ctx, cypher, map[string]any{
        "tenant_id": req.TenantID, "user_id": req.UserID,
        "from": req.TimeRange.From, "to": req.TimeRange.To,
        "limit": req.Limit,
    })
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/graphiti-search/internal/usecase/temporal.go` | NEW |
| `gateway/adapter/handler/memory_handler.go` | MODIFY — add timeline endpoint |
| `services/graphiti-ingestion/internal/usecase/ingest.go` | MODIFY — add timestamp to episodes |

---

## 3. Acceptance Criteria

- [ ] `GET /v1/memory/timeline?from=...&to=...` returns episodes in time range
- [ ] Relative time queries ("yesterday", "last week") resolved by LLM
- [ ] Temporal graph traversal in < 300ms
