# TASK-006: Create Error Response Normaliser Middleware

**Solution**: [SOL-004](../solutions/SOL-004-response-schema-contracts.md)  
**CR**: CR-004  
**Priority**: 🟠 Medium  
**Estimate**: 2 hours  
**Status**: TODO

---

## Context

The gateway's `WriteError` function in `handler.go` (lines 24–46) wraps errors as:
```json
{ "error": { "code": "...", "message": "...", "details": [...] } }
```

But the frontend (`ui/src/lib/api-client.ts`) expects flat errors:
```typescript
interface ApiErrorResponse {
  message: string;
  code:    string;
  status:  number;
}
```

This is a **gateway-wide fix** — every error response must be transformed.

**Approach**: Instead of modifying every `WriteError` call across all handlers, we modify `WriteError` itself in `handler.go` to output the flat format. This is the simplest approach since `WriteError` is the single error-writing function used throughout.

---

## Exact Task

### Step 1: Update `WriteError` in `gateway/internal/adapter/handler/handler.go`

Replace the current `WriteError` function (lines 24–46) with:

```go
// WriteError writes a structured JSON error response in the format expected by the frontend:
// { "message": "...", "code": "...", "status": 400 }
func WriteError(w http.ResponseWriter, err error) {
	if gErr, ok := err.(*domain.GatewayError); ok {
		status := gErr.HTTPStatusCode()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{
			"message": gErr.Message,
			"code":    gErr.Code,
			"status":  status,
		})
		return
	}
	// Generic 500
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	json.NewEncoder(w).Encode(map[string]any{
		"message": err.Error(),
		"code":    "INTERNAL",
		"status":  500,
	})
}
```

> **Note**: This changes the format from `{ error: { code, message, details } }` to `{ message, code, status }`. The `details` field is dropped since the frontend's `ApiErrorResponse` does not include it. If needed, `details` can be added back as an optional field.

### Step 2: Update the 404 catch-all in `router.go`

The 404 handler in `router.go` (lines 329–338) also uses the old format. Update it:

```go
// BEFORE:
json.NewEncoder(w).Encode(map[string]any{
    "error": map[string]any{
        "code":    domain.ErrNotFound.Code,
        "message": "route not found: " + r.Method + " " + r.URL.Path,
    },
})

// AFTER:
json.NewEncoder(w).Encode(map[string]any{
    "message": "route not found: " + r.Method + " " + r.URL.Path,
    "code":    domain.ErrNotFound.Code,
    "status":  http.StatusNotFound,
})
```

### Step 3: Verify `console.go` requireAdmin/requireSuperAdmin

Check that `requireAdmin` and `requireSuperAdmin` in `console.go` call `WriteError` (they already do). After updating `WriteError`, these functions automatically output the correct format.

---

## Files to Modify

| File | Change |
|------|--------|
| `gateway/internal/adapter/handler/handler.go` | Replace `WriteError` function (lines 24–46) |
| `gateway/internal/adapter/handler/router.go` | Update 404 catch-all JSON encoding (lines 332–337) |

---

## Acceptance Criteria

- [ ] `WriteError` outputs `{ message, code, status }` flat structure (not nested under `error`)
- [ ] 404 catch-all returns `{ message, code, status }` format
- [ ] All existing tests in `console_test.go` still pass after the format change
- [ ] `go build ./gateway/...` passes
- [ ] A `401 Unauthenticated` response looks like: `{ "message": "unauthenticated", "code": "UNAUTHENTICATED", "status": 401 }`
