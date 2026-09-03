# TASK-INTEL-003 — Expose GET /v1/profiles/{user_id} API endpoint

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-003 |
| **Wave** | 1 |
| **Solution** | [SOL-INTEL-001](../solutions/SOL-INTEL-001-User-Profile-Assembly.md) §1.2 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-INTEL-002 |
| **Estimated** | 2h |

---

## Mục tiêu

Expose GET /v1/profiles/{user_id} API endpoint

---

## Công việc cụ thể

### `gateway/adapter/handler/profile_handler.go` [NEW]

```go
// GET /v1/profiles/{user_id}
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
    userID  := chi.URLParam(r, "user_id")
    tenantID := tenant.FromContext(r.Context())

    // Cache check (Redis, 5min TTL)
    cacheKey := fmt.Sprintf("profile:%s:%s", tenantID, userID)
    if cached := h.cache.Get(r.Context(), cacheKey); cached != nil {
        writeJSON(w, 200, cached)
        return
    }

    conn := h.registry.Get("memobase-engine")
    client := memopb.NewMemobaseEngineServiceClient(conn)
    resp, err := client.GetProfile(r.Context(), &memopb.GetProfileRequest{
        TenantId: tenantID, UserId: userID,
    })
    if err != nil {
        writeError(w, 500, "profile_error", err.Error())
        return
    }
    h.cache.Set(r.Context(), cacheKey, resp, 5*time.Minute)
    writeJSON(w, 200, resp)
}

// POST /v1/profiles/{user_id}/rebuild — force reassembly
func (h *ProfileHandler) RebuildProfile(w http.ResponseWriter, r *http.Request) { ... }
```

### Router registration

```go
r.Get("/v1/profiles/{user_id}", profileHandler.GetProfile)
r.Post("/v1/profiles/{user_id}/rebuild", profileHandler.RebuildProfile)
```

---

## Acceptance Criteria

- [ ] GET /v1/profiles/{user_id} returns profile in < 500ms (cached)
- [ ] Cache TTL: 5 minutes
- [ ] Non-admin cannot read other users' profiles
- [ ] 404 if profile doesn't exist yet

## Files

```
gateway/adapter/handler/profile_handler.go  [NEW]
gateway/adapter/handler/router.go           [MODIFY]
```

**Trạng thái:** 🔄 Partial

---

**Ghi chú audit:** /v1/console/profiles/{user_id} route exists; ProfileHandler in console.go; cross-engine aggregation (Zep+Cognee) not wired
