# TASK-CORE-008 — Temporal Reasoning Search (Graphiti)

| Field | Value |
|---|---|
| **Task ID** | TASK-CORE-008 |
| **Wave** | 3 |
| **Solution** | [SOL-CORE-005](../solutions/SOL-CORE-005-Temporal-Reasoning.md) |
| **Component** | `services/graphiti-search/internal/usecase/` |
| **Priority** | 🟡 High |
| **Depends On** | TASK-CORE-004 |
| **Estimated** | 4h |

---

## Mục tiêu

Implement temporal episode query với time range filter và relative time resolution.

---

## Công việc cụ thể

### 1. `services/graphiti-search/internal/usecase/temporal.go` [NEW]

```go
type TemporalUseCase struct {
    graphRepo port.GraphRepository
    llm       port.LLMClient
}

const resolveTimePrompt = `Extract time range from the query. 
Return JSON only: {"from": "ISO8601", "to": "ISO8601"}
Use current time as reference: %s

Query: "%s"
Example: "yesterday" → {"from": "2026-09-02T00:00:00Z", "to": "2026-09-02T23:59:59Z"}`

func (u *TemporalUseCase) SearchTemporal(ctx context.Context, req *TemporalSearchRequest) ([]*Episode, error) {
    // Resolve relative time if needed
    if req.TimeRange == nil && req.Query != "" {
        now := time.Now().UTC().Format(time.RFC3339)
        prompt := fmt.Sprintf(resolveTimePrompt, now, req.Query)
        resp, err := u.llm.Complete(ctx, &port.CompletionRequest{
            Prompt: prompt, MaxTokens: 50, Temperature: 0.0, Task: "temporal_resolve",
        })
        if err == nil {
            var tr TimeRange
            if json.Unmarshal([]byte(resp.Content), &tr) == nil {
                req.TimeRange = &tr
            }
        }
    }

    cypher := `
        MATCH (e:Episode)
        WHERE e.tenant_id = $tenant_id AND e.user_id = $user_id`
    params := map[string]any{
        "tenant_id": req.TenantID, "user_id": req.UserID, "limit": req.Limit,
    }

    if req.TimeRange != nil {
        cypher += ` AND e.created_at >= $from AND e.created_at <= $to`
        params["from"] = req.TimeRange.From
        params["to"] = req.TimeRange.To
    }
    cypher += ` RETURN e ORDER BY e.created_at DESC LIMIT $limit`

    return u.graphRepo.QueryEpisodes(ctx, cypher, params)
}
```

### 2. `gateway/adapter/handler/memory_handler.go` [MODIFY] — add Timeline endpoint

```go
// GET /v1/memory/timeline?from=2026-09-01&to=2026-09-03&query=...&limit=20
func (h *MemoryHandler) Timeline(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    userID := r.URL.Query().Get("user_id")
    query  := r.URL.Query().Get("query")
    from   := r.URL.Query().Get("from")
    to     := r.URL.Query().Get("to")
    limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
    if limit == 0 { limit = 20 }

    conn := h.registry.Get("graphiti-search")
    client := graphitipb.NewGraphitiSearchServiceClient(conn)
    resp, err := client.SearchTemporal(r.Context(), &graphitipb.TemporalSearchRequest{
        TenantId: tenantID, UserId: userID, Query: query,
        From: from, To: to, Limit: int32(limit),
    })
    if err != nil { writeError(w, 500, "search_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}
```

---

## Acceptance Criteria

- [ ] `GET /v1/memory/timeline?from=2026-09-01&to=2026-09-03` returns episodes in range
- [ ] `GET /v1/memory/timeline?query=yesterday` resolves time via LLM
- [ ] Results ordered by created_at DESC
- [ ] Tenant isolation enforced in Cypher query

## Files

```
services/graphiti-search/internal/usecase/temporal.go  [NEW]
gateway/adapter/handler/memory_handler.go              [MODIFY — Timeline endpoint]
```
