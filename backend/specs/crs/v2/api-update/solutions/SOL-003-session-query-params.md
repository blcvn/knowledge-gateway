# SOL-003: Session API — Query Parameters & Response Schema Fix

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** Pagination normaliser (NormalizePaginatedResponse/ForwardToServiceWithNorm) not yet added to handler.go; TASK-005 not implemented  

**Solution for**: [CR-003](../CR-003-session-query-params.md)  
**Priority**: 🟡 High  
**Status**: Ready to Implement  
**Created**: 2026-06-18  
**Estimate**: 1 day

---

## Analysis

### Current State

`GET /v1/console/sessions` is already registered in `router.go` and handled by `SessionHandler.ListSessions` in `console.go`. The handler forwards directly to `zep-core` via `ForwardToService`.

The problem is a **schema contract mismatch** between what `zep-core` currently returns and what the frontend expects:

| Frontend Expects (`PaginatedResponse<Session>`) | Likely What `zep-core` Returns |
|-----|-----|
| `data: Session[]` | May use `sessions`, `items`, or `result` key |
| `total: number` | May be `count` or `total_count` |
| `page: number` | May be absent |
| `page_size: number` (snake_case) | May be `limit` |
| `pageSize: number` (camelCase alias) | Likely absent |
| `has_more: boolean` | May be absent |
| `hasMore: boolean` (camelCase alias) | Likely absent |

Additionally, the frontend sends **7 query parameters** that `zep-core` may not support: `status`, `user_id`, `agent_id`, `search`, `sort`, `page`, `page_size`.

### Architecture Decision

Instead of modifying `zep-core` (which would require changes to a downstream service), implement a **response normaliser** middleware in the `SessionHandler` at the gateway level. This keeps the fix isolated and doesn't require changes to `zep-core`.

For query parameters, we pass them through as-is (since `ForwardToService` already forwards all query params). The normaliser only transforms the response.

---

## Implementation Plan

### Step 1: Add Response Normaliser to `handler.go`

Add a utility function that intercepts the response from `zep-core`, transforms it to `PaginatedResponse<Session>` shape, then writes it to the HTTP response:

```go
// gateway/internal/adapter/handler/handler.go (or a new normalize.go)

// NormalizePaginatedResponse reads a response from a downstream service,
// attempts to normalize it into PaginatedResponse shape, and writes to w.
// If the downstream already uses the correct shape, it's passed through.
func NormalizePaginatedResponse(dataKey string) func(http.ResponseWriter, *http.Response) {
    return func(w http.ResponseWriter, resp *http.Response) {
        defer resp.Body.Close()
        var raw map[string]json.RawMessage
        if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
            // Pass through on error
            for k, v := range resp.Header {
                w.Header()[k] = v
            }
            w.WriteHeader(resp.StatusCode)
            return
        }

        // Try to find data array under known keys
        possibleDataKeys := []string{"data", dataKey, "sessions", "items", "result", "results"}
        var data json.RawMessage
        for _, k := range possibleDataKeys {
            if v, ok := raw[k]; ok {
                data = v
                break
            }
        }

        // Extract pagination fields with fallbacks
        total     := extractInt(raw, "total", "total_count", "count")
        page      := extractInt(raw, "page", "current_page")
        pageSize  := extractInt(raw, "page_size", "limit", "per_page")
        hasMore   := extractBool(raw, "has_more", "hasMore")
        if pageSize == 0 { pageSize = 20 }
        if page == 0     { page = 1 }

        normalized := map[string]any{
            "data":      data,
            "total":     total,
            "page":      page,
            "page_size": pageSize,  // snake_case
            "pageSize":  pageSize,  // camelCase alias (frontend compat)
            "has_more":  hasMore,   // snake_case
            "hasMore":   hasMore,   // camelCase alias (frontend compat)
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(resp.StatusCode)
        json.NewEncoder(w).Encode(normalized)
    }
}
```

### Step 2: Update `SessionHandler.ListSessions`

Modify `console.go` to use the normaliser only for the list endpoint:

```go
// ListSessions handles GET /v1/console/sessions — list sessions (paginated).
// Query params forwarded to zep-core: status, user_id, agent_id, search, sort, page, page_size
// Response is normalised to PaginatedResponse<Session> shape.
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
    if !requireAdmin(w, r) {
        return
    }
    // Use normalising forwarder instead of plain ForwardToService
    ForwardToServiceWithNorm(h.registry, "zep-core", h.logger,
        NormalizePaginatedResponse("sessions"))(w, r)
}
```

### Step 3: Add `ForwardToServiceWithNorm` to `handler.go`

```go
// ForwardToServiceWithNorm is like ForwardToService but transforms the response
// before writing it to the HTTP response writer.
func ForwardToServiceWithNorm(
    registry port.ServiceRegistry,
    service  string,
    logger   *slog.Logger,
    norm     func(http.ResponseWriter, *http.Response),
) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Forward request, capture response
        resp, err := doForward(registry, service, r)
        if err != nil {
            logger.Error("forward failed", "service", service, "error", err)
            http.Error(w, "service unavailable", http.StatusServiceUnavailable)
            return
        }
        norm(w, resp)
    }
}
```

### Step 4: Verify Sub-Route Response Shapes

Check the following endpoints return the correct shapes (add integration tests):

#### `GET /v1/console/sessions/{id}` → `Conversation`

```json
{
  "session_id": "string",
  "messages": [
    {
      "id":       "string",
      "role":     "user | assistant | system | tool",
      "content":  "string",
      "timestamp": "ISO 8601",
      "memory_sources": ["string"]
    }
  ]
}
```

If `zep-core` returns a different shape (e.g., `messages` nested under `memory`), add a sub-route normaliser.

#### `GET /v1/console/sessions/{id}/diff` → `SessionDiff`

```json
{
  "session_id": "string",
  "added":   [{ "engine": "string", "memory_id": "string", "content": "string" }],
  "updated": [{ "engine": "string", "memory_id": "string", "field": "string", "before": {}, "after": {} }],
  "deleted": [{ "engine": "string", "memory_id": "string" }]
}
```

This route forwards to `vnp-event`, not `zep-core`. Verify `vnp-event` returns this shape.

#### `GET /v1/console/sessions/{id}/working-memory` → `WorkingMemory`

```json
{
  "session_id": "string",
  "summary":   "string",
  "entities":  ["string"]
}
```

This route forwards to `ov-session`. Verify shape.

---

## Query Parameter Forwarding Validation

The `ForwardToService` function should already pass all query parameters through. Confirm with a test that these parameters are correctly forwarded:

```
GET /v1/console/sessions?status=active&user_id=abc&agent_id=xyz&search=keyword&sort=created_at&page=2&page_size=10
```

Expected `zep-core` call includes all params in the forwarded request URL.

---

## Files to Create/Modify

| Action | File |
|--------|------|
| **MODIFY** | `gateway/internal/adapter/handler/handler.go` — add `NormalizePaginatedResponse`, `ForwardToServiceWithNorm` |
| **MODIFY** | `gateway/internal/adapter/handler/console.go` — use `ForwardToServiceWithNorm` for `ListSessions` |
| **ADD TESTS** | `gateway/internal/adapter/handler/console_test.go` — test pagination normalisation |

---

## Acceptance Criteria

- [ ] `GET /v1/console/sessions` returns `PaginatedResponse<Session>` with both camelCase and snake_case pagination fields
- [ ] `GET /v1/console/sessions?status=active` filters correctly
- [ ] `GET /v1/console/sessions?user_id=xxx&page=2&page_size=10` paginates correctly
- [ ] `GET /v1/console/sessions?search=keyword` full-text filters
- [ ] Response includes both `page_size` (snake) and `pageSize` (camel)
- [ ] Response includes both `has_more` (snake) and `hasMore` (camel)
- [ ] `GET /v1/console/sessions/{id}` returns `Conversation` with `messages[]`
- [ ] `GET /v1/console/sessions/{id}/diff` returns `SessionDiff` with `added`, `updated`, `deleted` arrays
- [ ] `GET /v1/console/sessions/{id}/working-memory` returns `WorkingMemory` with `summary` and `entities[]`
