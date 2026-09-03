# SOL-INTEL-006 — Solution: Agent Context Debugger

| Field | Value |
|---|---|
| **Solution ID** | SOL-INTEL-006 |
| **CR** | [CR-INTEL-006](../../../../docs/crs/v4/intelligence/CR-INTEL-006-Agent-Context-Debugger.md) |
| **TDD ref** | [08-platform-services.md](../../../tdd/architecture/08-platform-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |

---

## 1. Giải pháp

Agent Context Debugger = diagnostic API cho agent developers để inspect what the AI "knows" at any point.

### 1.1 `gateway/adapter/handler/debug_handler.go` [NEW]

```go
type DebugHandler struct {
    recall     RecallUseCase
    profile    ProfileUseCase
    sessions   SessionUseCase
    obsRepo    ObservationRepo
}

// GET /v1/debug/context/{user_id}
func (h *DebugHandler) GetAgentContext(w http.ResponseWriter, r *http.Request) {
    userID := chi.URLParam(r, "user_id")
    tenant := tenant.FromContext(r.Context())

    var wg sync.WaitGroup
    type DebugContext struct {
        ActiveSession    *Session       `json:"active_session,omitempty"`
        RecentMemories   []MemoryUnit   `json:"recent_memories"`
        UserProfile      *UserProfile   `json:"user_profile"`
        ObservationCount int            `json:"observation_count"`
        LastHookType     string         `json:"last_hook_type,omitempty"`
        TokenEstimate    int            `json:"token_estimate"`
    }

    ctx := &DebugContext{}
    
    wg.Add(4)
    go func() { defer wg.Done()
        ctx.ActiveSession, _ = h.sessions.GetActiveSession(r.Context(), tenant, userID)
    }()
    go func() { defer wg.Done()
        res, _ := h.recall.Recall(r.Context(), &RecallRequest{
            TenantID: tenant, UserID: userID, Query: "*", Limit: 5,
        })
        if res != nil { ctx.RecentMemories = res.Results }
    }()
    go func() { defer wg.Done()
        ctx.UserProfile, _ = h.profile.GetProfile(r.Context(), tenant, userID)
    }()
    go func() { defer wg.Done()
        if ctx.ActiveSession != nil {
            ctx.ObservationCount = ctx.ActiveSession.ObservationCount
        }
    }()
    wg.Wait()

    ctx.TokenEstimate = estimateTokens(ctx)
    writeJSON(w, 200, ctx)
}
```

---

## 2. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/debug_handler.go` | NEW |
| `gateway/adapter/handler/router.go` | MODIFY — register /v1/debug/* |

---

## 3. Acceptance Criteria

- [ ] Returns full context summary in < 500ms
- [ ] Debug endpoint dev-only (gated by ENABLE_DEBUG_API=true)
- [ ] Token estimate calculated for LLM context window planning
- [ ] Accessible via MCP: `admin_get_agent_context` tool
