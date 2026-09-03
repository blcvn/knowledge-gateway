# TASK-005: Add Pagination Normaliser to `handler.go`

**Solution**: [SOL-003](../solutions/SOL-003-session-query-params.md)  
**CR**: CR-003  
**Priority**: 🟡 High  
**Estimate**: 2 hours  
**Status**: ⏳ Pending

---

## Context

The frontend session list page calls `GET /v1/console/sessions` and expects `PaginatedResponse<Session>`:

```typescript
{
  data:      Session[];
  total:     number;
  page:      number;
  page_size: number;   // snake_case
  pageSize:  number;   // camelCase alias
  has_more:  boolean;  // snake_case
  hasMore:   boolean;  // camelCase alias
}
```

`zep-core` returns a different shape. The gateway's `SessionHandler.ListSessions` needs to normalise the response **before** writing to the HTTP response.

The current `ForwardToService` in `handler.go` (line 56) calls `registry.ForwardWithContext` which returns `[]byte`. We need a variant that normalises the JSON before writing.

---

## Exact Task

### Step 1: Add helpers to `gateway/internal/adapter/handler/handler.go`

Append the following functions **at the end of `handler.go`** (after line 159):

```go
// ─── Pagination Normalisation ───────────────────────────────────────────────

// PaginatedNormaliser transforms a downstream response into the frontend-compatible
// PaginatedResponse shape with both snake_case and camelCase pagination fields.
type PaginatedNormaliser struct {
	// DataKey is the JSON key to look for the array in the downstream response.
	// Common values: "data", "sessions", "items", "result", "results"
	DataKey string
}

// Normalise reads a raw JSON body and returns a normalised PaginatedResponse JSON.
// If the body already uses the correct shape (has "data" key), it is passed through.
func (n *PaginatedNormaliser) Normalise(rawBody []byte, statusCode int) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return rawBody, nil // pass through non-JSON
	}

	// Find the data array under known keys
	dataKeys := []string{"data", n.DataKey, "sessions", "items", "result", "results"}
	var data json.RawMessage = json.RawMessage("[]")
	for _, k := range dataKeys {
		if v, ok := raw[k]; ok {
			data = v
			break
		}
	}

	total    := extractRawInt(raw, "total", "total_count", "count")
	page     := extractRawInt(raw, "page", "current_page")
	pageSize := extractRawInt(raw, "page_size", "limit", "per_page")
	if pageSize == 0 { pageSize = 20 }
	if page == 0     { page = 1 }
	hasMore := total > (page * pageSize)

	normalised := map[string]any{
		"data":      json.RawMessage(data),
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"pageSize":  pageSize,
		"has_more":  hasMore,
		"hasMore":   hasMore,
	}
	return json.Marshal(normalised)
}

// extractRawInt tries to read an int from a map[string]json.RawMessage under multiple possible keys.
func extractRawInt(m map[string]json.RawMessage, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			var n int
			if json.Unmarshal(v, &n) == nil {
				return n
			}
		}
	}
	return 0
}

// ForwardWithNorm forwards a request to a service and normalises the response
// using the provided PaginatedNormaliser before writing to the HTTP response.
func ForwardWithNorm(registry port.ServiceRegistry, serviceName string, logger *slog.Logger, norm *PaginatedNormaliser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ReadBody(r)
		if err != nil {
			WriteError(w, domain.ErrInvalidArgument.WithMessage("failed to read request body"))
			return
		}

		target, err := registry.Resolve(serviceName)
		if err != nil {
			logger.Error("resolve service failed", "service", serviceName, "error", err)
			WriteError(w, domain.ErrCircuitOpen)
			return
		}

		fwdReq := &domain.ForwardRequest{
			Path:        r.URL.Path,
			HTTPMethod:  r.Method,
			Body:        body,
			PathParams:  extractPathParams(r),
			QueryParams: extractQueryParams(r),
		}

		resp, err := registry.ForwardWithContext(r.Context(), target, fwdReq)
		if err != nil {
			WriteError(w, err)
			return
		}

		normalised, err := norm.Normalise(resp, http.StatusOK)
		if err != nil {
			// Fall back to raw response
			normalised = resp
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(normalised)
	}
}
```

### Step 2: Update `SessionHandler.ListSessions` in `console.go`

Replace the current `ListSessions` implementation (lines ~388-394 in `console.go`):

```go
// BEFORE:
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	ForwardToService(h.registry, "zep-core", h.logger)(w, r)
}

// AFTER:
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if !requireAdmin(w, r) {
		return
	}
	// Normalise response to PaginatedResponse<Session> shape expected by frontend.
	// Supports query params: status, user_id, agent_id, search, sort, page, page_size
	norm := &PaginatedNormaliser{DataKey: "sessions"}
	ForwardWithNorm(h.registry, "zep-core", h.logger, norm)(w, r)
}
```

---

## Files to Modify

| File | Change |
|------|--------|
| `gateway/internal/adapter/handler/handler.go` | Append `PaginatedNormaliser`, `extractRawInt`, `ForwardWithNorm` |
| `gateway/internal/adapter/handler/console.go` | Update `SessionHandler.ListSessions` to use `ForwardWithNorm` |

---

## Acceptance Criteria

- [ ] `PaginatedNormaliser` struct and `Normalise` method added to `handler.go`
- [ ] `ForwardWithNorm` function added to `handler.go`
- [ ] `SessionHandler.ListSessions` uses `ForwardWithNorm` with `DataKey: "sessions"`
- [ ] Response includes both `page_size` and `pageSize` fields
- [ ] Response includes both `has_more` and `hasMore` fields
- [ ] `go build ./gateway/...` passes

---

**Audit Note:** Pagination normaliser not implemented — handler.go needs NormalizePaginatedResponse + ForwardToServiceWithNorm
