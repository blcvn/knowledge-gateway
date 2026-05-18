---
id: FEAT-010
title: Context Debugger API
service: vnp-gateway
version: 1.0.0
status: Draft
priority: P1
created: 2026-05-13
updated: 2026-05-13
linked_sol: SOL-002
linked_ux: "ux_spec.md §6.6 Agent Context Debugger"
---

## Mục Tiêu

API cho Context Debugger — core differentiator của Console UI. Cho phép trace step-by-step quá trình assembly context từ 6 engines cho một AI agent request.

## Bối Cảnh Nghiệp Vụ

Context Debugger trả lời câu hỏi:
- AI agent nhận được context gì?
- Từ engine nào?
- Tại sao memory đó được chọn?
- Token allocation như thế nào?

## Scope

### In Scope
- `POST /v1/console/debugger/trace` — Simulate context assembly
- `GET /v1/console/debugger/traces/{id}` — Get saved trace
- `GET /v1/console/debugger/traces` — List recent traces

### Out of Scope
- Realtime agent interception (future scope)
- Prompt template editing

## Thiết Kế Kỹ Thuật

### API Contract

#### POST `/v1/console/debugger/trace`

**Request:**
```json
{
  "query": "Explain knowledge graph types",
  "user_id": "user_123",
  "engines": ["cognee", "graphiti", "memobase", "supermemory"],
  "model": "gpt-4",
  "max_tokens": 4096
}
```

**Response (200):**
```json
{
  "trace_id": "trace_abc123",
  "steps": [
    {
      "step": 1,
      "name": "Query Rewrite",
      "input": "Explain knowledge graph types",
      "output": "knowledge graph classification taxonomy",
      "duration_ms": 45
    },
    {
      "step": 2,
      "name": "Semantic Recall (Cognee)",
      "engine": "cognee",
      "results_count": 15,
      "top_score": 0.95,
      "tokens_used": 1200,
      "duration_ms": 180
    },
    {
      "step": 3,
      "name": "Graph Traversal (Graphiti)",
      "engine": "graphiti",
      "results_count": 8,
      "top_score": 0.88,
      "tokens_used": 800,
      "duration_ms": 120
    },
    {
      "step": 4,
      "name": "Profile Lookup (Memobase)",
      "engine": "memobase",
      "profile_sections_injected": 3,
      "tokens_used": 250,
      "duration_ms": 45
    },
    {
      "step": 5,
      "name": "Adaptive Memory Check (Supermemory)",
      "engine": "supermemory",
      "results_count": 3,
      "tokens_used": 400,
      "duration_ms": 60
    },
    {
      "step": 6,
      "name": "Salience Ranking",
      "input_count": 29,
      "output_count": 15,
      "ranking_method": "rrf",
      "duration_ms": 30
    },
    {
      "step": 7,
      "name": "Policy Filter",
      "filtered_count": 2,
      "reasons": ["expired_memory", "access_denied"],
      "duration_ms": 5
    },
    {
      "step": 8,
      "name": "Compression",
      "input_tokens": 2650,
      "output_tokens": 1800,
      "savings_pct": 32.1,
      "duration_ms": 80
    }
  ],
  "final_context": {
    "total_tokens": 1800,
    "token_budget": 4096,
    "utilization_pct": 43.9,
    "breakdown": {
      "cognee": 800,
      "graphiti": 500,
      "memobase": 200,
      "supermemory": 300
    }
  },
  "total_duration_ms": 565
}
```

### Internal Architecture
- **Handler:** `adapter/http/debugger_handler.go`
- **Usecase:** `usecase/debugger.go` — Orchestrates multi-engine recall + ranking + filtering
- **Proxy to:** `vnp-search-hub` (recall), `memobase-context` (profile), `vnp-platform` (ranking)
- Trace storage: `vnp-event` (persist trace for replay)

## Acceptance Criteria
- [ ] AC-1: Trace returns all 8+ steps with timing and token counts
- [ ] AC-2: Each step shows source engine, result count, and relevance score
- [ ] AC-3: Final context shows token breakdown per engine
- [ ] AC-4: Saved traces can be retrieved by ID
- [ ] AC-5: Trace simulates without actually calling LLM
- [ ] AC-6: Profile lookup step shows which profile sections were injected

## Test Requirements
- Unit tests: Step builder, token counting, ranking simulation
- Integration tests: Multi-engine trace with mocks
- Minimum coverage: 80%
