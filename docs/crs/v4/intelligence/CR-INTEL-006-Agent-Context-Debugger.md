# Change Request: CR-INTEL-006 — Agent Context Debugger

**CR ID:** CR-INTEL-006
**Component:** `backend/services/vnp-observability`, `backend/gateway`
**Priority:** 🟠 Medium
**Status:** Open
**Version:** v4 / Intelligence Layer
**Solution:** [S7 — Agent Observability](../../../bussiness/solutions/S7-agent-observability.md)
**Features:** [F25](../../../features/25-context-debugger/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P1-07 | AI Agent Developer | Không biết context nào đang được inject vào LLM — black box |
| PP-P3-03 | ML/AI Engineer | Không trace được retrieval quality — không optimize được |

**Before:** LLM nhận context từ nhiều engines — không thể trace.
**After:** Mỗi API call có `X-Trace-ID` → trace UI shows full context breakdown.

---

## 2. Context Trace Output

```json
{
  "trace_id": "trace_abc123",
  "request": {"query": "auth middleware", "user_id": "u_123"},
  "context_breakdown": [
    {
      "engine": "openviking",
      "tier": "L1",
      "files": ["gateway/auth.go", "shared/tenant/middleware.go"],
      "tokens": 340,
      "retrieval_ms": 45
    },
    {
      "engine": "graphiti",
      "facts": ["Auth uses JWT RS256 (valid since 2026-01-01)"],
      "tokens": 18,
      "retrieval_ms": 120
    },
    {
      "engine": "memobase",
      "profile": "Thích Clean Architecture, Go developer",
      "tokens": 12,
      "retrieval_ms": 8
    }
  ],
  "total_tokens": 370,
  "total_retrieval_ms": 173,
  "llm_prompt_preview": "System: [auth.go summary...]
User profile: [Bình, Go dev...]

Query: auth middleware"
}
```

---

## 3. API Contract

```http
# Enable trace on a call
POST /v1/memory/recall
X-Debug-Trace: true
→ Adds "trace_id" to response

# Get trace detail
GET /v1/traces/{trace_id}
→ Full context breakdown JSON (above)

# List recent traces
GET /v1/traces?user_id=u_123&limit=20
```

---

## 4. Acceptance Criteria

- [ ] `X-Debug-Trace: true` header bật trace mode
- [ ] Trace ghi lại: engine, tier, tokens, retrieval_ms per source
- [ ] `llm_prompt_preview` shows assembled context string
- [ ] Traces searchable by user_id, time range
- [ ] Traces auto-expire sau 7 ngày (không tốn storage)
