# TASK-INTEL-004 — Implement L0/L1/L2 tiered context injection for AI agents

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-004 |
| **Wave** | 1 |
| **Solution** | [SOL-INTEL-002](../solutions/SOL-INTEL-002-Tiered-Context-Injection.md) §1.1 |
| **Component** | `services/ov-fs/internal/usecase/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-INTEL-002 |
| **Estimated** | 5h |

---

## Mục tiêu

Implement L0/L1/L2 tiered context injection for AI agents

---

## Công việc cụ thể

### `services/ov-fs/internal/usecase/context_injection.go` [NEW]

```go
const DefaultTokenBudget = 8192

type BuildContextRequest struct {
    TenantID    string
    UserID      string
    ProjectPath string
    Query       string
    TokenBudget int
}

type ContextBundle struct {
    L0Files    []FileContext   `json:"l0_files"`    // project files
    L1Summary  *SessionSummary `json:"l1_summary"`  // session facts
    L2Graph    *GraphContext   `json:"l2_graph"`    // entities
    TotalTokens int            `json:"total_tokens"`
    TruncatedAt string         `json:"truncated_at,omitempty"`
}

func (u *ContextInjectionUseCase) BuildContext(ctx context.Context, req *BuildContextRequest) (*ContextBundle, error) {
    budget := req.TokenBudget
    if budget == 0 { budget = DefaultTokenBudget }

    bundle := &ContextBundle{}

    // L2: Graph (most compressed)
    graphCtx, _ := u.graphiti.GetEntityContext(ctx, req.TenantID, req.UserID, req.Query)
    bundle.L2Graph = graphCtx
    budget -= u.tokenizer.Count(graphCtx)

    // L1: Session summary
    if budget > 500 {
        summary, _ := u.memobase.GetSessionSummary(ctx, req.TenantID, req.UserID)
        bundle.L1Summary = summary
        budget -= u.tokenizer.Count(summary)
    } else {
        bundle.TruncatedAt = "L1"
    }

    // L0: Project files (fill remaining budget)
    if budget > 1000 {
        files, _ := u.vikingFS.GetProjectContext(ctx, req.TenantID, req.ProjectPath, budget)
        bundle.L0Files = files
        bundle.TotalTokens = DefaultTokenBudget - budget + u.tokenizer.Count(files)
    } else {
        bundle.TruncatedAt = "L0"
        bundle.TotalTokens = DefaultTokenBudget
    }

    return bundle, nil
}
```

### MCP tool: `ov_get_context`

```go
// gateway/adapter/mcp/handlers_context.go [NEW]
s.Register("ov_get_context", h.handleOVGetContext,
    mcp.Schema{
        "project_path": "string",
        "query?":       "string",
        "token_budget?": "integer",
    })
```

---

## Acceptance Criteria

- [ ] L0/L1/L2 assembled in priority order (L2 first)
- [ ] Token budget strictly respected
- [ ] TruncatedAt field indicates which tier was cut
- [ ] `ov_get_context` MCP tool functional
- [ ] Context assembly < 300ms

## Files

```
services/ov-fs/internal/usecase/context_injection.go   [NEW]
gateway/adapter/mcp/handlers_context.go                [NEW — ov_get_context]
gateway/adapter/mcp/server.go                          [MODIFY — register]
```
