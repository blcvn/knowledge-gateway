# ADR-008 — Memory Consolidation 4-Tier Pipeline (Sleep Model)

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-04 |
| **Deciders** | ML + Platform Team |
| **Feature** | F12 (Memory Consolidation Pipeline) |

---

## Context

Agent observe hooks tích lũy rất nhanh (100-1000 hooks/session). Nếu không consolidate, storage sẽ bùng nổ và retrieval sẽ chậm. Câu hỏi: **Consolidate như thế nào để preserve knowledge mà không mất signal?**

Neuroscience insight từ [`sleep.md`](../research/sleep.md): Não consolidate memory qua nhiều giai đoạn ngủ (NREM → REM → insight extraction) — không phải 1 giai đoạn.

---

## Decision

**4-tier pipeline chạy offline (sau khi session complete), mirror sleep consolidation stages.**

```
Raw Hooks (input)
     │
     ▼ TIER 1 — LLM Compression (mirrors NREM Stage 1-2)
     │  Group hooks by 5-minute window
     │  LLM: compress batch → 70-90% size reduction
     │  Output: compressed_blobs
     │  Circuit breaker: stop if LLM fails 3× consecutive
     │
     ▼ TIER 2 — Session Summary (mirrors NREM Stage 3-4)
     │  LLM: "What happened in this session?"
     │  Extract: attempted / succeeded / failed / decisions / entities
     │  Output: session_summary record
     │
     ▼ TIER 3 — Procedure Extraction (mirrors REM)
     │  Multi-session analysis (N sessions threshold)
     │  LLM: Extract generic, reusable procedures
     │  Output: procedural memory → OpenViking L1
     │
     ▼ TIER 4 — Cross-session Insights (mirrors multi-night integration)
        Weekly / N-session batch
        LLM: Cross-agent patterns, shared insights
        Output: adaptive memory → Supermemory
```

**Trigger:** NATS event `agent.session.complete` → pipeline-service consumer.

---

## Consequences

**Positive:**
- **Storage reduction: 70-90%** (145 hooks → 12 summaries → 1 session_summary)
- Session recall 20× faster (query session_summary vs 145 raw hooks)
- Neuroscience-aligned: higher tiers = more durable memory (mirror long-term potentiation)
- Procedure sharing across agents (Tier 4)

**Negative:**
- LLM cost per consolidation (3+ calls depending on tier)
- Latency: Tier 1-2 chạy ngay sau session; Tier 3-4 chạy theo schedule → delay
- Information loss: compressed summaries may miss edge cases

**Mitigations:**
- Circuit breaker: `ConsolidationOptions.CircuitBreaker = true`
- Raw hooks preserved configurable duration (default 30 days) before cleanup
- Token budget control per tier

---

## Alternatives Considered

### A1 — 1-stage "summarize everything" LLM call
- **Rejected:** Context window limit (too many hooks); expensive; no tiered durability (all-or-nothing)

### A2 — No LLM consolidation (only deduplication)
- **Rejected:** Dedup không giảm được storage nhiều; semantic compression needed for quality

### A3 — Consolidate during session (real-time)
- **Rejected:** Adds latency to every hook capture; better to do offline (like sleep)
