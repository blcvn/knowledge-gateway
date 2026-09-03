# ADR-007 — Memobase YOLO Engine: 3 Fixed LLM Calls

| Field | Value |
|---|---|
| **Status** | ✅ Accepted |
| **Date** | 2026-03 |
| **Deciders** | ML Team |
| **Feature** | F05 (Profile Memory — Memobase) |

---

## Context

Memobase cần extract và duy trì **user profiles** từ conversations. Vấn đề: LLM calls rất tốn kém. Bao nhiêu calls là đủ, bao nhiêu là quá nhiều?

Naive approach: 1 LLM call per blob insertion → với 100 blobs/day × $0.01/call = $1/user/day = không sustainable.

---

## Decision

**YOLO Engine (You Only LLM Once per stage): 3 fixed LLM calls per flush, bất kể số lượng blobs.**

```
Flush trigger (auto @ 20 blobs hoặc manual):

Call 1 — EXTRACT:
  Input: ALL blobs trong buffer (batch)
  Task:  Extract profile candidates {key, value, category, confidence}
  Output: List of candidates
  Cost:  1 LLM call, O(blobs) tokens

Call 2 — MERGE:
  Input: New candidates + Existing profiles
  Task:  Merge, resolve conflicts, update scores
  Output: Updated profile set
  Cost:  1 LLM call, O(profiles) tokens

Call 3 — EVENTS:
  Input: Blob content + profile changes
  Task:  Generate GistText (human-readable summary)
  Output: Event list for timeline
  Cost:  1 LLM call, O(events) tokens

Total: 3 LLM calls, regardless of buffer size (20 or 200 blobs)
```

**Context assembly (separate, không phải flush):**
```
GET /v1/memobase/users/{uid}/context
  → SQL query pre-computed profiles (no LLM)
  → Return prompt-ready string
  → Target: < 100ms
```

---

## Consequences

**Positive:**
- **Predictable LLM cost:** 3 calls per flush, regardless of volume
- **Fast context retrieval:** < 100ms (SQL only, no LLM at read time)
- Cost với 100 blobs/day: 3 calls/flush ÷ 20 blobs/flush × 1 flush = 3 calls/20 blobs/day
- Cheaper than naive: 3 vs 100 calls (33× cheaper)

**Negative:**
- Batch extraction có thể miss nuances so với per-blob analysis
- Fixed threshold (20 blobs) không adaptive theo user behavior
- 3-call latency during flush (user waits during manual flush)

**Mitigations:**
- `FlushThreshold` là configurable (default 20, adjustable per tenant)
- Manual flush available cho immediate profile update
- Async flush = user không bị block

---

## Alternatives Considered

### A1 — 1 LLM call per blob
- **Rejected:** O(n) cost scaling; expensive at scale; latency per insertion

### A2 — No LLM extraction (keyword rules)
- **Rejected:** Too rigid; cannot capture nuanced profile attributes; poor extraction quality

### A3 — Async background LLM với batching
- **Partially implemented:** Flush IS async; YOLO is the batching strategy
