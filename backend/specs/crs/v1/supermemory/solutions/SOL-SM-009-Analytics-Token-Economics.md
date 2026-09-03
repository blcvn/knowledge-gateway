# SOL-SM-009 — Solution: Analytics & Token Economics

| Field | Value |
|---|---|
| **Solution ID** | SOL-SM-009 |
| **CR** | CR-SM-009 |
| **TDD ref** | [07-supermemory-services.md](../../../tdd/architecture/07-supermemory-services.md) |
| **Status** | Open |
| **Priority** | 🟠 Medium |
| **Component** | `services/vnp-platform` |

---

## 1. Giải pháp

Track token usage per tenant/space → billing dashboard.

```go
// Analytics endpoint
// GET /v1/admin/sm/analytics?tenant_id=&from=&to=
type SMAnalytics struct {
    TokensIngested   int64   // total tokens stored
    TokensQueried    int64   // tokens used in search
    LLMCostUSD       float64 // estimated cost
    MemoryCount      int64   // total memories
    ActiveSpaces     int     // spaces with activity
}
```

Use telemetry metrics: `vnp_sm_tokens_ingested`, `vnp_llm_cost_usd`.

## 2. Acceptance Criteria

- [ ] Per-tenant analytics dashboard
- [ ] LLM cost tracked separately from storage
- [ ] Daily/weekly/monthly aggregation

