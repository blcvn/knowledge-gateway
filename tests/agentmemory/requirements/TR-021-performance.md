# TR-021: Performance Test Requirements

**Module:** System-wide Performance  
**Nguồn:** SRS §4.1, PRD §7, TDD §12, FR-SEARCH-005  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-021-PRF-001 — Search p50 latency ≤ 14ms
🔴 P0 | `[PERF]` | **FR-SEARCH-002**

**Given:** Index với 1000 CompressedObservations, embedding provider configured  
**When:** 100 search queries được chạy  
**Then:** p50 (median) latency ≤ 14ms

**Test Setup:**
```bash
# 100 queries: measure response time
# p50 = median, p95 = 95th percentile
```

**Traceability:** SRS §4.1, PRD §7

---

## TR-021-PRF-002 — Search p95 latency ≤ 100ms
🟠 P1 | `[PERF]`

**Given:** Same as TR-021-PRF-001  
**When:** 100 queries  
**Then:** p95 latency ≤ 100ms

**Traceability:** SRS §4.1

---

## TR-021-PRF-003 — Hook processing: non-LLM path < 500ms
🔴 P0 | `[PERF]` | **SRS §4.1**

**Given:** `AGENTMEMORY_AUTO_COMPRESS=false` (no LLM)  
**When:** Hook event được gửi đến `POST /observe`  
**Then:** Response nhận được trong < 500ms end-to-end

**Traceability:** SRS §4.1

---

## TR-021-PRF-004 — Index rebuild: 1000 observations < 30s
🟠 P1 | `[PERF]` | **SRS §4.1**

**Given:** 1000 CompressedObservations trong KV, empty BM25+Vector index  
**When:** Full index rebuild  
**Then:** Hoàn thành trong < 30 giây

**Traceability:** SRS §4.1

---

## TR-021-PRF-005 — Viewer load time < 2s
🟡 P2 | `[PERF]` | **SRS §4.1**

**Given:** Viewer server running  
**When:** Browser loads `http://localhost:3113`  
**Then:** Page fully loaded trong < 2 giây

**Traceability:** SRS §4.1

---

## TR-021-PRF-006 — KV scale: 50K observations
🟠 P1 | `[PERF]` | **SRS §4.3**

**Given:** 50,000 observations trong KV  
**When:** Bất kỳ operations nào  
**Then:**
- Reads: < 100ms
- Writes: < 100ms
- No memory leak

**Traceability:** SRS §4.3

---

## TR-021-PRF-007 — Vector search: O(n×k) complexity
🟡 P2 | `[PERF]`

**Given:** n = 50K docs, k = 20 (top-K)  
**When:** Vector search chạy  
**Then:** Hoàn thành trong < 10ms (1M comparisons trong V8)

**Traceability:** TDD §12.1

---

## TR-021-PRF-008 — LongMemEval-S: full benchmark
🔴 P0 | `[PERF]` | **FR-SEARCH-005**

**Given:** LongMemEval-S benchmark (500 questions, ICLR 2025)  
**When:** `npm run eval:longmemeval` với hybrid adapter  
**Then:**
- R@5 ≥ 95.2%
- R@10 ≥ 98.6%
- MRR ≥ 88.2%

**Traceability:** PRD §7, SRS §3.5 FR-SEARCH-005

---

## TR-021-PRF-009 — Coding Agent Life benchmark
🟠 P1 | `[PERF]`

**Given:** Coding Agent Life v1 (15 in-house sessions)  
**When:** `npm run eval:coding-agent-life`  
**Then:** 100% top-5 hit rate

**Traceability:** TDD §14.3

---

## TR-021-PRF-010 — Token cost per session ≤ 2000
🟠 P1 | `[PERF]` | **PRD §7**

**Given:** Standard session (20 tool calls), LLM compression enabled  
**When:** Session completes  
**Then:** Total LLM tokens consumed ≤ 2000

**Traceability:** PRD §7

---

## TR-021-PRF-011 — BM25 serialization: ~50KB per 1000 docs
🟡 P2 | `[PERF]`

**Given:** SearchIndex với 1000 documents  
**When:** `serialize()`  
**Then:** Serialized size ≤ 55KB

**Traceability:** Architecture §5.1

---

## TR-021-PRF-012 — Concurrent hooks: no deadlock
🔴 P0 | `[PERF]`

**Given:** 100 concurrent hook requests từ multiple sessions  
**When:** All processed  
**Then:**
- All 100 complete successfully
- No deadlocks
- Response time ≤ 1s for all

**Traceability:** TDD §10.3
