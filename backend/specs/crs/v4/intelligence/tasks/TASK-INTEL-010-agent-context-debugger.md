# TASK-INTEL-010 — Agent Context Debugger: GET /v1/debug/context/{user_id}

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-010 |
| **Wave** | 3 |
| **Solution** | [SOL-INTEL-006](../solutions/SOL-INTEL-006-Agent-Context-Debugger.md) §1.1 |
| **Component** | `gateway/adapter/handler/` |
| **Priority** | 🟠 Medium |
| **Depends On** | TASK-INTEL-003 |
| **Estimated** | 3h |

---

## Mục tiêu

Agent Context Debugger: GET /v1/debug/context/{user_id}

---

## Công việc cụ thể

### `gateway/adapter/handler/debug_handler.go` [NEW]

```go
type DebugHandler struct {
    sessions SessionUseCase
    recall   RecallUseCase
    profile  ProfileUseCase
}

type DebugContext struct {
    ActiveSession    *Session      `json:"active_session,omitempty"`
    RecentMemories   []MemoryUnit  `json:"recent_memories"`
    UserProfile      *UserProfile  `json:"user_profile,omitempty"`
    ObservationCount int           `json:"observation_count"`
    LastHookType     string        `json:"last_hook_type,omitempty"`
    TokenEstimate    int           `json:"token_estimate"`
}

// GET /v1/debug/context/{user_id}
// Only available when ENABLE_DEBUG_API=true
func (h *DebugHandler) GetAgentContext(w http.ResponseWriter, r *http.Request) {
    if os.Getenv("ENABLE_DEBUG_API") != "true" {
        writeError(w, 404, "not_found", "debug API disabled")
        return
    }

    userID   := chi.URLParam(r, "user_id")
    tenantID := tenant.FromContext(r.Context())

    var wg sync.WaitGroup
    ctx := &DebugContext{}
    var mu sync.Mutex

    wg.Add(3)
    go func() { defer wg.Done()
        session, _ := h.sessions.GetActiveSession(r.Context(), tenantID, userID)
        mu.Lock()
        ctx.ActiveSession = session
        if session != nil { ctx.ObservationCount = session.ObservationCount }
        mu.Unlock()
    }()
    go func() { defer wg.Done()
        res, _ := h.recall.Recall(r.Context(), &RecallRequest{
            TenantID: tenantID, UserID: userID, Query: "*", Limit: 5,
        })
        mu.Lock()
        if res != nil { ctx.RecentMemories = res.Results }
        mu.Unlock()
    }()
    go func() { defer wg.Done()
        profile, _ := h.profile.GetProfile(r.Context(), tenantID, userID)
        mu.Lock(); ctx.UserProfile = profile; mu.Unlock()
    }()
    wg.Wait()

    ctx.TokenEstimate = estimateContextTokens(ctx)
    writeJSON(w, 200, ctx)
}
```

### Router (gated)

```go
if os.Getenv("ENABLE_DEBUG_API") == "true" {
    r.Get("/v1/debug/context/{user_id}", debugHandler.GetAgentContext)
}
```

---

## Acceptance Criteria

- [ ] Returns assembled debug context < 500ms
- [ ] ENABLE_DEBUG_API=false → 404
- [ ] Token estimate calculated from recent memories + profile
- [ ] Parallel fetch: sessions + recall + profile
- [ ] Integration with MCP: admin_get_agent_context tool

## Files

```
gateway/adapter/handler/debug_handler.go  [NEW]
gateway/adapter/handler/router.go         [MODIFY — gated debug route]
```

**Trạng thái:** ✅ Implemented

---

**Ghi chú audit:** DebuggerHandler.GetAgentContext() added to console.go; GET /v1/debug/context/{user_id} route registered in router.go
