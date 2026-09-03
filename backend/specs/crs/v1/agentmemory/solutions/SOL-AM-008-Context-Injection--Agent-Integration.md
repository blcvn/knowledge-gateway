# SOL-AM-008 — Solution: Context Injection & Agent Integration

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-008 |
| **CR** | CR-AM-008 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `gateway/adapter/mcp` |

---

## 1. Giải pháp

Context injection = given active session, provide relevant context to agent's next LLM call.

### `services/observe-service/internal/usecase/context.go` [NEW]

```go
// BuildAgentContext — called by MCP tool before each LLM call
func (u *ContextUseCase) BuildAgentContext(ctx context.Context, req *ContextRequest) (*AgentContext, error) {
    var wg sync.WaitGroup
    var sessionSummary *SessionSummary
    var recentObs []Observation
    var userProfile *UserProfile

    wg.Add(3)
    go func() { defer wg.Done(); sessionSummary, _ = u.summaryRepo.GetForSession(ctx, req.SessionID) }()
    go func() { defer wg.Done(); recentObs, _ = u.obsRepo.GetRecent(ctx, req.SessionID, 10) }()
    go func() { defer wg.Done(); userProfile, _ = u.profileUC.GetProfile(ctx, req.TenantID, req.UserID) }()
    wg.Wait()

    return &AgentContext{
        SessionSummary: sessionSummary,
        RecentActivity: recentObs,
        UserContext:    userProfile,
        TokenCount:     u.tokenizer.Count(sessionSummary, recentObs, userProfile),
    }, nil
}
```

### MCP integration

```go
s.Register("memory_observe", h.handleObserveHook, ...)
s.Register("memory_consolidate", h.handleConsolidate, ...)
```

## 2. File Changes

| File | Action |
|---|---|
| `services/observe-service/internal/usecase/context.go` | NEW |
| `gateway/adapter/mcp/server.go` | MODIFY — memory_observe + memory_consolidate tools |

## 3. Acceptance Criteria

- [ ] Context assembled in < 200ms
- [ ] Token budget respected (8192 default)
- [ ] MCP tool `memory_observe` submits hook to observe pipeline
- [ ] MCP tool `memory_consolidate` triggers pipeline-service

