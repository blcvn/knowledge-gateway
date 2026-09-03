# TC-021: Performance — Test Cases

**Test Design tham chiếu:** [TD-021](../designs/TD-021-performance.md)  
**Requirements tham chiếu:** [TR-021](../requirements/TR-021-performance.md)  
**Module:** Latency SLA (BM25, Observe, Recall), Memory Usage, Throughput  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Quy ước đo lường

- **p50**: Giá trị trung vị (50th percentile) của phân phối latency
- **p95**: 95th percentile
- **Warm-up**: Các request đầu tiên (thường 5) không tính vào kết quả đo — để tránh JIT/cache cold start
- **Environment**: Isolated (không có background CPU-heavy processes)
- **Pass/Fail**: Tất cả scenarios phải đạt đồng thời, không chỉ average

---

## TC-021-001: BM25 search p50 ≤ 14ms (1000 docs)

| Trường | Giá trị |
|---|---|
| **ID** | TC-021-001 |
| **Loại** | Performance |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-021-PRF-001 |

**Setup:**
- BM25 index: 1000 documents (mỗi doc có 10-20 keywords, diverse vocabulary)
- EmbeddingProvider = none (BM25-only, không có vector overhead)
- 100 queries với 10 search terms × 10 iterations

**Các bước thực hiện:**
1. Index 1000 documents
2. Warm-up: 5 queries (không đo)
3. Run 100 queries, ghi latency từng query
4. Tính p50 = percentile(latencies, 50), p95 = percentile(latencies, 95)

**Kết quả mong đợi:**
- `p50 ≤ 14ms`
- `p95 ≤ 100ms`
- Error count = 0

---

## TC-021-002: Observe pipeline p50 ≤ 50ms (no LLM/embedding)

| Trường | Giá trị |
|---|---|
| **ID** | TC-021-002 |
| **Loại** | Performance |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-021-PRF-002 |

**Setup:**
- MockKV (in-memory, không có disk I/O)
- `EMBEDDING_PROVIDER = none`
- `AGENTMEMORY_AUTO_COMPRESS = false`
- 50 observe calls, warm-up 5 trước

**Nội dung observe:** toolName = `"edit_file"`, đủ fields để đi qua pipeline đầy đủ (extractConcepts, extractFiles, generateNarrative synthetic)

**Kết quả mong đợi:**
- `p50 ≤ 50ms`
- `p95 ≤ 200ms`

---

## TC-021-003: Recall p50 ≤ 50ms (1000 observations, BM25-only)

| Trường | Giá trị |
|---|---|
| **ID** | TC-021-003 |
| **Loại** | Performance |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-021-PRF-003 |

**Setup:**
- Session `perf_sess` với 1000 observations
- BM25 index đã được warm up
- EmbeddingProvider = none
- 50 recall queries (diverse terms), warm-up 5 trước

**Kết quả mong đợi:**
- `p50 ≤ 50ms`
- `p95 ≤ 200ms`

---

## TC-021-004: Memory usage ổn định sau 1000 observe calls

| Trường | Giá trị |
|---|---|
| **ID** | TC-021-004 |
| **Loại** | Performance |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-021-PRF-005 |

**Setup:** MockKV, `EMBEDDING_PROVIDER = none`, `AUTO_COMPRESS = false`

**Các bước thực hiện:**
1. Gọi GC, ghi `heapBefore = process.memoryUsage().heapUsed`
2. Run 1000 observe calls
3. Gọi GC lần 2
4. Ghi `heapAfter = process.memoryUsage().heapUsed`
5. Tính `growth = heapAfter - heapBefore`

**Kết quả mong đợi:** `growth < 50MB` (không có memory leak nghiêm trọng)

---

## TC-021-005: BM25 bulk index throughput < 1000ms (1000 docs)

| Trường | Giá trị |
|---|---|
| **ID** | TC-021-005 |
| **Loại** | Performance |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-021-PRF-004 |

**Setup:** 1000 documents (average 100 tokens mỗi doc)

**Các bước thực hiện:**
1. `start = Date.now()`
2. Add tất cả 1000 documents vào BM25 index
3. `elapsed = Date.now() - start`

**Kết quả mong đợi:** `elapsed < 1000ms`

---

## TC-021-006: Hybrid search (BM25 + vector) p50 ≤ 50ms (500 docs)

| Trường | Giá trị |
|---|---|
| **ID** | TC-021-006 |
| **Loại** | Performance |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-021-PRF-006 |

**Điều kiện tiên quyết:** Requires embedding provider (mock với pre-computed vectors)

**Setup:**
- 500 documents, mỗi doc có cả BM25 và vector
- Mock embedder (instant, returns pre-computed vectors)
- 30 queries, warm-up 5 trước

**Kết quả mong đợi:**
- `p50 ≤ 50ms`
- `p95 ≤ 200ms`

---

## TC-021-007: Concurrent requests không degrade latency > 3×

| Trường | Giá trị |
|---|---|
| **ID** | TC-021-007 |
| **Loại** | Performance |
| **Ưu tiên** | 🟡 P2 |
| **Requirement** | TR-021-PRF-007 |

**Setup:** 10 concurrent recall requests với cùng sessionId

**Các bước thực hiện:**
1. Đo latency của 1 request (baseline)
2. Dispatch 10 concurrent requests, đo latency từng request
3. Tính median latency của concurrent batch

**Kết quả mong đợi:** `median_concurrent ≤ 3 × baseline`

---

## Tổng kết TC-021

| ID | Scenario | SLA | Priority |
|---|---|---|---|
| TC-021-001 | BM25 search 1000 docs | p50 ≤ 14ms | 🔴 P0 |
| TC-021-002 | Observe pipeline no LLM | p50 ≤ 50ms | 🔴 P0 |
| TC-021-003 | Recall 1000 obs BM25-only | p50 ≤ 50ms | 🔴 P0 |
| TC-021-004 | Memory no leak 1000 obs | growth < 50MB | 🟠 P1 |
| TC-021-005 | BM25 bulk index 1000 | < 1000ms total | 🟠 P1 |
| TC-021-006 | Hybrid search 500 docs | p50 ≤ 50ms | 🟠 P1 |
| TC-021-007 | Concurrent no degrade | ≤ 3× baseline | 🟡 P2 |
