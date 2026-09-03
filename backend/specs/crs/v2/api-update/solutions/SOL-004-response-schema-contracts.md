# SOL-004: Response Schema Contracts — Error Normalisation & Schema Alignment

**Solution for**: [CR-004](../CR-004-response-schema-contracts.md)  
**Priority**: 🟠 Medium  
**Status**: Ready to Implement  
**Created**: 2026-06-18  
**Estimate**: 2–3 days (distributed across services)

---

## Analysis

CR-004 covers two types of issues:

1. **Error response format mismatch** — Backend wraps errors in `{ error: {...} }` but frontend expects flat `{ message, code, status }`. This is a **gateway-level fix** (one place).
2. **Domain response schema mismatches** — Field naming (`camelCase` vs `snake_case`), missing fields, wrong enum values. These require **per-service verification and fixes**.

---

## Part A: Error Response Normalisation (Gateway-Level Fix)

### Problem

Backend returns:
```json
{
  "error": {
    "code": "INVALID_ARGUMENT",
    "message": "...",
    "details": [...],
    "request_id": "..."
  }
}
```

Frontend parses with:
```typescript
interface ApiErrorResponse {
  message: string;   // ← expects flat
  code:    string;
  status:  number;
}
```

### Solution: Gateway Error Middleware

Add an **error transformer** to the gateway middleware pipeline. Since the gateway controls all HTTP responses, it can intercept `4xx`/`5xx` responses from downstream services and rewrite the error body.

#### Implementation in `gateway/internal/infra/middleware/error_normalizer.go`

```go
package middleware

import (
    "bytes"
    "encoding/json"
    "net/http"
)

// ErrorNormalizer wraps HTTP responses and transforms error payloads
// from backend format { error: {...} } to frontend-compatible format { message, code, status }.
func ErrorNormalizer(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rw := newErrorCapturingWriter(w)
        next.ServeHTTP(rw, r)
        
        // Only transform error responses (4xx, 5xx)
        if rw.status < 400 {
            rw.flush()
            return
        }
        
        // Try to parse and transform the error body
        var backendErr struct {
            Error struct {
                Code      string `json:"code"`
                Message   string `json:"message"`
                RequestID string `json:"request_id"`
            } `json:"error"`
        }
        
        if err := json.Unmarshal(rw.body.Bytes(), &backendErr); err == nil && backendErr.Error.Message != "" {
            // Transform to frontend-compatible format
            normalized := map[string]any{
                "message": backendErr.Error.Message,
                "code":    backendErr.Error.Code,
                "status":  rw.status,
            }
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(rw.status)
            json.NewEncoder(w).Encode(normalized)
            return
        }
        
        // Pass through as-is if not in expected format
        rw.flush()
    })
}
```

#### Register in Middleware Pipeline

In `gateway/internal/adapter/handler/router.go` or the server setup:

```go
// Current pipeline: Recovery → RequestID → Logger → CORS → Auth
// New pipeline:     Recovery → RequestID → Logger → CORS → Auth → ErrorNormalizer
handler = middleware.ErrorNormalizer(handler)
```

---

## Part B: Domain Schema Alignment (Per-Service Fixes)

### B1: Dashboard — `HealthStatus` Enum Values

**Service**: `vnp-dashboard`  
**Issue**: Frontend expects `"Healthy"`, `"Warning"`, `"Critical"` (capital first letter)  
**Fix**: In `services/vnp-platform/internal/domain/admin/entity.go`, update `HealthStatus` constants:

```go
// Current (guessed):
const (
    StatusHealthy  = "healthy"   // WRONG
    StatusWarning  = "warning"   // WRONG
    StatusCritical = "critical"  // WRONG
)

// Fix to:
const (
    StatusHealthy  HealthStatus = "Healthy"   // ✅
    StatusWarning  HealthStatus = "Warning"   // ✅
    StatusCritical HealthStatus = "Critical"  // ✅
)
```

### B2: Memory Explorer — `MemorySearchResult` with Facets

**Service**: `vnp-search-hub`  
**Issue**: Response must include `facets.byEngine` and `facets.byType`  
**Fix**: Ensure `vnp-search-hub` aggregates engine/type counts during search and includes them in the response:

```go
type MemorySearchResult struct {
    Results   []MemoryItem    `json:"results"`
    Total     int             `json:"total"`
    Facets    SearchFacets    `json:"facets"`    // ← Must be present
    LatencyMs int64           `json:"latencyMs"`
}

type SearchFacets struct {
    ByEngine map[string]int `json:"byEngine"`   // { "cognee": 5, "graphiti": 10 }
    ByType   map[string]int `json:"byType"`     // { "episodic": 10 }
}
```

### B3: Memory ID Format — `engine:local_id`

**Service**: `vnp-search-hub`  
**Issue**: Memory IDs must use format `"engine:local_id"` (colon-separated, e.g., `"graphiti:ep_abc123"`)  
**Fix**: Ensure `vnp-search-hub` constructs IDs consistently:

```go
func BuildMemoryID(engine, localID string) string {
    return fmt.Sprintf("%s:%s", engine, localID)
}
// Example: "graphiti:ep_abc123", "cognee:doc_xyz"
```

### B4: Memory Neighbors — Accept Query Params

**Service**: `vnp-search-hub`  
**Issue**: `GET /v1/console/memory/{id}/neighbors` must accept `strategy` and `limit` query params  
**Fix**: Ensure `vnp-search-hub` reads and honours:
- `?strategy=semantic|graph|temporal` (default: `semantic`)
- `?limit=10` (default: `10`)

### B5: Observability — `errors` Endpoint Filter

**Service**: `vnp-observability`  
**Issue**: `GET /v1/console/observability/errors` must accept `?service=xxx` filter  
**Fix**: Ensure `vnp-observability` reads and applies `service` query param as a filter.

### B6: `MetricsResponse` Shape

**Service**: `vnp-observability`  
**Issue**: Frontend expects `{ latency: MetricPoint[], error_rate: MetricPoint[], throughput: MetricPoint[] }`  
**Fix**: Ensure the response shape matches exactly. `MetricPoint` must be:
```json
{ "timestamp": "ISO 8601", "value": 0, "label": "p95" }
```

### B7: `PaginatedResponse` — Dual Naming

**Applies to**: Any paginated endpoint (sessions, profiles, etc.)  
**Issue**: Frontend uses both `page_size` and `pageSize`, both `has_more` and `hasMore`  
**Solution**: Add to the `NormalizePaginatedResponse` utility from SOL-003, which already handles both.

Alternatively, create a shared `PaginatedResponse` struct in the gateway that serialises both:

```go
type PaginatedResponse[T any] struct {
    Data     []T  `json:"data"`
    Total    int  `json:"total"`
    Page     int  `json:"page"`
    PageSize int  `json:"page_size"`  // snake_case primary
    // Aliases — additional JSON tags via custom MarshalJSON
}

func (p PaginatedResponse[T]) MarshalJSON() ([]byte, error) {
    type Alias PaginatedResponse[T]
    return json.Marshal(struct {
        Alias
        PageSizeCamel int  `json:"pageSize"`
        HasMore       bool `json:"has_more"`
        HasMoreCamel  bool `json:"hasMore"`
    }{
        Alias:         Alias(p),
        PageSizeCamel: p.PageSize,
        HasMore:       len(p.Data) < p.Total,
        HasMoreCamel:  len(p.Data) < p.Total,
    })
}
```

---

## Implementation Order

| Priority | Fix | Location | Effort |
|----------|-----|----------|--------|
| 1 | Error response normaliser middleware | `gateway/internal/infra/middleware/` | 0.5 day |
| 2 | `HealthStatus` enum capital casing | `services/vnp-platform` | 0.5 day |
| 3 | Memory ID format `engine:local_id` | `services/vnp-search-hub` (or search-service) | 0.5 day |
| 4 | `MemorySearchResult.facets` field | `services/vnp-search-hub` | 1 day |
| 5 | Memory neighbors query params | `services/vnp-search-hub` | 0.5 day |
| 6 | Observability `errors` filter | `services/obs-service` | 0.5 day |
| 7 | Metrics response shape | `services/obs-service` | 0.5 day |
| 8 | Paginated response dual naming | Gateway normaliser (reuse SOL-003) | 0 (reuse) |

---

## Files to Create/Modify

| Action | File |
|--------|------|
| **CREATE** | `gateway/internal/infra/middleware/error_normalizer.go` |
| **MODIFY** | `gateway/internal/adapter/handler/router.go` — add error normaliser to pipeline |
| **MODIFY** | `services/vnp-platform/internal/domain/admin/entity.go` — fix HealthStatus values |
| **MODIFY** | `services/vnp-platform/internal/domain/admin/entity.go` — ensure `HealthStatus` exported constants |
| **MODIFY** | `services/search-service/internal/domain/search/` — add facets to search result |
| **MODIFY** | `services/obs-service/internal/domain/observability/` — add service filter + metrics shape |

---

## Acceptance Criteria

- [ ] All `4xx`/`5xx` responses from gateway are normalised to `{ message, code, status }` flat structure
- [ ] `GET /v1/console/dashboard/health` returns `status` as `"Healthy"`, `"Warning"`, `"Critical"`
- [ ] `POST /v1/console/memory/search` returns `facets.byEngine` and `facets.byType`
- [ ] Memory IDs returned from search use `"engine:local_id"` format
- [ ] `GET /v1/console/memory/{id}/neighbors?strategy=graph&limit=5` honours params
- [ ] `GET /v1/console/observability/errors?service=cognee` filters by service
- [ ] `GET /v1/console/observability/metrics` returns `{ latency[], error_rate[], throughput[] }`
- [ ] Paginated responses include both snake_case and camelCase pagination fields
