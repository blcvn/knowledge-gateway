# TASK-BE-008 — Console Profiles Handler (Memobase)

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-008 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-006 CR-006](../solutions/SOL-006-Adaptive-to-Org-Solutions.md) |
| **Priority** | 🔴 P0 |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/console_profiles_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

```go
package handler

type ConsoleProfilesHandler struct {
    memoCtx    MemobaseContextClient  // gRPC → memobase-context
    memoEngine MemobaseEngineClient   // gRPC → memobase-engine
    eventSvc   VNPEventClient         // gRPC → vnp-event
}

// GET /v1/console/profiles
func (h *ConsoleProfilesHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    profiles, _ := h.memoEngine.ListUsers(r.Context(), tenantID)
    // Response: [{user_id, created_at, updated_at, profiles:[]}]
    httputil.JSON(w, 200, profiles)
}

// GET /v1/console/profiles/{user_id}
func (h *ConsoleProfilesHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
    userID   := r.PathValue("user_id")
    tenantID := authctx.TenantID(r.Context())
    profile, err := h.memoEngine.GetUserProfiles(r.Context(), userID, tenantID)
    if err != nil { httputil.Error(w, "Not found", "NOT_FOUND", 404); return }
    // Trả về {user_id, profiles:[{topic, sub_topic, content}]}
    httputil.JSON(w, 200, profile)
}

// GET /v1/console/profiles/{user_id}/buffers
func (h *ConsoleProfilesHandler) GetBuffers(w http.ResponseWriter, r *http.Request) {
    userID   := r.PathValue("user_id")
    tenantID := authctx.TenantID(r.Context())
    buffer, _ := h.memoEngine.GetBuffer(r.Context(), userID, tenantID)
    httputil.JSON(w, 200, buffer)
}

// GET /v1/console/profiles/{user_id}/context
func (h *ConsoleProfilesHandler) GetContext(w http.ResponseWriter, r *http.Request) {
    userID   := r.PathValue("user_id")
    tenantID := authctx.TenantID(r.Context())
    start    := time.Now()
    ctx, _   := h.memoCtx.AssembleContext(r.Context(), &memobase.ContextRequest{
        UserID: userID, TenantID: tenantID,
    })
    httputil.JSON(w, 200, map[string]any{
        "user_id":                userID,
        "context_string":         ctx.ContextString,
        "token_count":            ctx.Tokens,
        "profile_section_tokens": ctx.ProfileTokens,
        "event_section_tokens":   ctx.EventTokens,
        "latency_ms":             time.Since(start).Milliseconds(),
    })
}

// GET /v1/console/profiles/{user_id}/events
func (h *ConsoleProfilesHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
    userID   := r.PathValue("user_id")
    tenantID := authctx.TenantID(r.Context())
    events, _ := h.eventSvc.GetTimeline(r.Context(), tenantID, userID)
    httputil.JSON(w, 200, events)
}

// GET /v1/console/profiles/config
func (h *ConsoleProfilesHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
    config, _ := h.memoEngine.GetConfig(r.Context(), authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, config)
}

// PUT /v1/console/profiles/config
func (h *ConsoleProfilesHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
    var cfg map[string]any
    json.NewDecoder(r.Body).Decode(&cfg)
    updated, _ := h.memoEngine.UpdateConfig(r.Context(), authctx.TenantID(r.Context()), cfg)
    httputil.JSON(w, 200, updated)
}
```

### Routes

```go
mux.HandleFunc("GET /v1/console/profiles",                    authMiddleware(prof.ListProfiles))
mux.HandleFunc("GET /v1/console/profiles/config",             authMiddleware(prof.GetConfig))
mux.HandleFunc("PUT /v1/console/profiles/config",             authMiddleware(prof.UpdateConfig))
mux.HandleFunc("GET /v1/console/profiles/{user_id}",          authMiddleware(prof.GetProfile))
mux.HandleFunc("GET /v1/console/profiles/{user_id}/buffers",  authMiddleware(prof.GetBuffers))
mux.HandleFunc("GET /v1/console/profiles/{user_id}/context",  authMiddleware(prof.GetContext))
mux.HandleFunc("GET /v1/console/profiles/{user_id}/events",   authMiddleware(prof.GetEvents))
```
