# Change Request: CR-INTEL-002 — Tiered Context Injection (L0/L1/L2)

**CR ID:** CR-INTEL-002
**Component:** `backend/services/ov-fs`, `backend/services/ov-search`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v4 / Intelligence Layer
**Solution:** [S6 — Context Efficiency](../../../bussiness/solutions/S6-context-efficiency.md)
**Features:** [F06](../../../features/06-procedural-memory-openviking/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-06 | AI Agent Developer | Context token cost $0.50/call — không scale |
| PP-P5-04 | IDE Plugin User | AI phải re-read toàn bộ codebase mỗi session |
| PP-P3-03 | ML/AI Engineer | Không biết context nào đang được inject — không trace |

**Cost Impact:**
- Before: Full file content inject → $0.50/call × 1000 calls/day = **$500/day**
- After: L0 headlines inject → $0.02/call × 1000 calls/day = **$20/day (−96%)**

---

## 2. Tiered Context Architecture

```
L0 — Headlines (< 200 tokens/file)
  File name + purpose + key exports
  Example: "auth.go — JWT middleware, exports: AuthMiddleware, ExtractClaims"

L1 — Summaries (< 2000 tokens/file)
  L0 + function signatures + business logic description
  Example: auth.go full docstring + func signatures

L2 — Full Content (full tokens)
  Complete file content
  Only loaded on explicit demand
```

---

## 3. API Contract

```http
# Get tiered context for query
POST /v1/ov/context
{
  "query": "auth middleware JWT",
  "scope": "project",
  "tier": "auto",          // auto | L0 | L1 | L2
  "token_budget": 4096
}
→ {
    "results": [
      {"path": "gateway/auth.go", "tier": "L1", "content": "...", "tokens": 340},
      {"path": "shared/tenant/middleware.go", "tier": "L0", "content": "...", "tokens": 45}
    ],
    "total_tokens": 385,
    "budget_used": "9.4%"
  }

# Semantic grep (MCP: ov_grep)
POST /v1/ov/grep
{ "pattern": "JWT", "scope": "project" }
→ { "matches": [{"path": "...", "line": 42, "content": "..."}] }
```

---

## 4. Tier Selection Algorithm

```go
// backend/services/ov-search/internal/usecase/context_assembly.go [NEW]
func (u *ContextAssembler) Assemble(ctx context.Context, query string, budget int) *ContextResult {
    // Step 1: Retrieve relevant files by semantic similarity
    files := u.fsRepo.SearchRelevant(ctx, query, limit=20)
    
    // Step 2: Start with L0 for all files
    result := &ContextResult{}
    tokensUsed := 0
    
    for _, f := range files {
        l0 := u.getL0(ctx, f)
        
        if tokensUsed + l0.Tokens + budget*0.3 < budget {
            // Upgrade to L1 if most relevant
            l1 := u.getL1(ctx, f)
            result.Add(f, "L1", l1)
            tokensUsed += l1.Tokens
        } else {
            // L0 only
            result.Add(f, "L0", l0)
            tokensUsed += l0.Tokens
        }
        
        if tokensUsed >= budget { break }
    }
    
    return result
}
```

---

## 5. Acceptance Criteria

- [ ] 3 tiers: L0 (headlines), L1 (summaries), L2 (full)
- [ ] Token budget enforced — không vượt budget
- [ ] `tier=auto` chọn đúng tier dựa trên budget và relevance
- [ ] Semantic grep hoạt động (ov_grep MCP tool)
- [ ] Context cost giảm ≥ 80% so với full injection baseline
