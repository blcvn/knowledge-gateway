# TD-021: Performance Test Design

**Liên kết Requirements:** [TR-021-performance.md](../requirements/TR-021-performance.md)  
**Test file:** `tests/agentmemory/specs/performance.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Performance tests đo latency và throughput của các operations quan trọng nhất. Chạy **riêng biệt** (không mix với unit tests).

---

## 2. SLAs cần đáp ứng

| Operation | p50 SLA | p95 SLA | Source |
|---|---|---|---|
| BM25 search (1000 docs) | ≤ 14ms | ≤ 100ms | FR-SEARCH-002 |
| Observe pipeline (no LLM) | ≤ 50ms | ≤ 200ms | FR-OBS-007 |
| Context recall (1000 obs) | ≤ 50ms | ≤ 200ms | FR-RECALL-003 |
| Memory remember | ≤ 100ms | ≤ 500ms | FR-MEM-005 |
| Graph query | ≤ 100ms | ≤ 500ms | FR-GRAPH-004 |
| BM25 index: 1000 docs | Total < 1000ms | - | FR-PERF-001 |
| Vector search (1000 docs) | ≤ 20ms | ≤ 100ms | FR-SEARCH-003 |

---

## 3. Test Cases

### Group A: Search Performance

#### TC-001 — BM25 search p50 ≤ 14ms (1000 docs)
**Requirement:** TR-021-PRF-001 | **Type:** performance | 🔴 P0

**Setup:**
- Index: 1000 documents với diverse concepts
- Queries: 100 queries, 10 unique terms x 10 iterations

**Execution:** Chạy queries, ghi từng latency  
**Assert:**
- p50 ≤ 14ms
- p95 ≤ 100ms
- No errors

---

#### TC-002 — Vector search p50 ≤ 20ms (1000 vectors)
**Requirement:** TR-021-PRF-002 | **Type:** performance | 🔴 P0

**Setup:** 1000 random unit vectors (384 dim) trong VectorIndex  
**Execution:** 100 queries với random query vectors  
**Assert:** p50 ≤ 20ms

---

#### TC-003 — BM25 index throughput: 1000 docs < 1000ms
**Requirement:** TR-021-PRF-003 | **Type:** performance | 🟠 P1

**Setup:** 1000 documents prepared  
**Execution:** `add()` 1000 lần, đo total time  
**Assert:** Total < 1000ms

---

### Group B: Pipeline Performance

#### TC-004 — Observe pipeline p50 ≤ 50ms (no embedding)
**Requirement:** TR-021-PRF-004 | **Type:** performance | 🔴 P0

**Setup:**
- MockKV (in-memory)
- `EMBEDDING_PROVIDER=none`
- `AGENTMEMORY_AUTO_COMPRESS=false`

**Execution:** 50 observe calls, đo từng latency  
**Assert:** p50 ≤ 50ms, p95 ≤ 200ms

---

#### TC-005 — Recall p50 ≤ 50ms (1000 observations, BM25-only)
**Requirement:** TR-021-PRF-005 | **Type:** performance | 🔴 P0

**Setup:** 1000 observations indexed, MockKV  
**Execution:** 50 recall queries  
**Assert:** p50 ≤ 50ms

---

### Group C: Memory Operations

#### TC-006 — `mem::remember` p50 ≤ 100ms (100 existing memories)
**Requirement:** TR-021-PRF-006 | **Type:** performance | 🟠 P1

**Setup:** 100 memories đã tồn tại trong KV (để trigger Jaccard check)  
**Execution:** 20 remember calls  
**Assert:** p50 ≤ 100ms (bao gồm Jaccard similarity scan)

---

#### TC-007 — Jaccard similarity với 500 existing memories < 500ms total
**Type:** performance | 🟠 P1

**Setup:** 500 memories trong KV  
**Execution:** 1 remember call (triggers scan của tất cả memories)  
**Assert:** Operation hoàn thành < 500ms

---

### Group D: Scale Tests

#### TC-008 — Memory usage ổn định qua 1000 observe calls
**Requirement:** TR-021-PRF-007 | **Type:** performance | 🟠 P1

**Execution:**
1. Ghi heap memory trước
2. Run 1000 observe calls
3. Ghi heap memory sau
4. Force GC và ghi lại

**Assert:** Heap growth < 50MB (không có memory leak)

---

#### TC-009 — Search performance không degraded sau 5000 docs
**Type:** performance | 🟡 P2

**Setup:** Index 5000 documents  
**Execution:** 100 search queries  
**Assert:** p50 không tăng quá 2x so với 1000-doc baseline

---

### Group E: Serialization Performance

#### TC-010 — SearchIndex serialize 1000 docs < 500ms
**Type:** performance | 🟡 P2

**Setup:** SearchIndex với 1000 docs  
**Execution:** Serialize → Deserialize cycle 5 lần  
**Assert:** Mỗi cycle < 500ms

---

## 4. Performance Test Protocol

### Isolation
- Chạy performance tests trên dedicated test environment
- Không chạy cùng với unit tests
- Đóng background processes trước khi chạy

### Measurement
- Dùng `performance.now()` (high-resolution timer, không phải `Date.now()`)
- Warm-up: 5 iterations trước khi bắt đầu đo
- Sample size tối thiểu: 50 iterations để tính percentile

### Reporting
```
[PERF] BM25 search (1000 docs):
  p50: 8.2ms  ✓ (SLA: 14ms)
  p95: 42.1ms ✓ (SLA: 100ms)
  max: 67.8ms
```

### Flakiness
- Tests có thể flaky trên heavily loaded CI → chạy lại 3 lần
- Nếu p95 fail nhưng p50 pass: warning, không hard fail
