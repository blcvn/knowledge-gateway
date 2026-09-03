# SOL-INTEL-002 — Solution: Tiered Context Injection (L0/L1/L2)

| Field | Value |
|---|---|
| **Solution ID** | SOL-INTEL-002 |
| **CR** | [CR-INTEL-002](../../../../docs/crs/v4/intelligence/CR-INTEL-002-Tiered-Context-Injection.md) |
| **TDD ref** | [05-openviking-services.md](../../../tdd/architecture/05-openviking-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |

---

## 1. Giải pháp

3-tier context injection cho AI agents:
- **L0**: Raw project files (VikingFS) — codebase, docs
- **L1**: Structured summaries (Memobase) — session history, user facts
- **L2**: Knowledge graph context (Graphiti/Cognee) — entities, relationships

### 1.1 `services/ov-fs/internal/usecase/context_injection.go` [NEW]

```go
type ContextInjectionUseCase struct {
    vikingFS    port.FileSystem
    memobase    port.MemobaseClient
    graphiti    port.GraphitiClient
    tokenizer   port.Tokenizer
}

type ContextBundle struct {
    L0Files    []FileContext    // project files, codebase
    L1Summary  *SessionSummary  // recent session facts
    L2Graph    *GraphContext    // entities + relationships
    TotalTokens int
    TruncatedAt string         // which tier was truncated to fit budget
}

const DefaultTokenBudget = 8192

func (u *ContextInjectionUseCase) BuildContext(ctx context.Context, req *BuildContextRequest) (*ContextBundle, error) {
    budget := req.TokenBudget
    if budget == 0 { budget = DefaultTokenBudget }

    bundle := &ContextBundle{}

    // L2: Graph context (most compressed, inject first)
    graphCtx, _ := u.graphiti.GetEntityContext(ctx, req.TenantID, req.UserID, req.Query)
    bundle.L2Graph = graphCtx
    budget -= u.tokenizer.Count(graphCtx)

    // L1: Session summary
    if budget > 500 {
        summary, _ := u.memobase.GetSessionSummary(ctx, req.TenantID, req.UserID)
        bundle.L1Summary = summary
        budget -= u.tokenizer.Count(summary)
    }

    // L0: Project files (fill remaining budget)
    if budget > 1000 {
        files, _ := u.vikingFS.GetProjectContext(ctx, req.TenantID, req.ProjectPath, budget)
        bundle.L0Files = files
        bundle.TotalTokens = DefaultTokenBudget - budget + u.tokenizer.Count(files)
    }

    return bundle, nil
}
```

### 1.2 MCP Tool

```go
// ov_get_context MCP tool
s.Register("ov_get_context", func(ctx context.Context, params map[string]any) (any, error) {
    return h.contextUC.BuildContext(ctx, &BuildContextRequest{
        TenantID:    tenant.FromContext(ctx),
        UserID:      params["user_id"].(string),
        ProjectPath: params["project_path"].(string),
        Query:       params["query"].(string),
        TokenBudget: int(params["token_budget"].(float64)),
    })
}, mcp.Schema{"user_id": "string", "project_path": "string", "query?": "string", "token_budget?": "integer"})
```

---

## 2. File Changes

| File | Action |
|---|---|
| `services/ov-fs/internal/usecase/context_injection.go` | NEW |
| `gateway/adapter/mcp/handlers_context.go` | NEW — ov_get_context tool |
| `shared/pkg/tokenizer/` | VERIFY — tokenizer package exists |

---

## 3. Acceptance Criteria

- [ ] Context assembly completes < 300ms
- [ ] Token budget respected (no overflow)
- [ ] L0/L1/L2 tiers assembled in priority order
- [ ] `ov_get_context` MCP tool returns structured bundle
