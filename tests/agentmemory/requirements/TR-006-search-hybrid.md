# TR-006: Hybrid Search (RRF Fusion) Test Requirements

**Module:** HybridSearch, SmartSearch  
**Nguồn:** SRS §3.5 (FR-SEARCH-001..005), Architecture §5.3-5.4, TDD §3.3-3.4  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Mô tả

Test requirements cho hybrid search engine kết hợp BM25 + Vector + Graph qua Reciprocal Rank Fusion (RRF), bao gồm query expansion, session diversification và smart search.

**Files:** `src/state/hybrid-search.ts`, `src/functions/smart-search.ts`

---

## TR-006-RRF-001 — RRF score formula
🔴 P0 | `[UNIT]` | **FR-SEARCH-001**

**Given:**
- BM25 results: obs1@rank1, obs2@rank2
- Vector results: obs2@rank1, obs1@rank2
- Weights: BM25=0.4, Vector=0.6, Graph=0 (no graph)

**When:** RRF fusion được tính  
**Then:**
```
totalW = 0.4 + 0.6 = 1.0
score(obs1) = 0.4 × (1/(60+1)) + 0.6 × (1/(60+2))
score(obs2) = 0.4 × (1/(60+2)) + 0.6 × (1/(60+1))
→ score(obs2) > score(obs1) vì vector weight cao hơn
```

**Traceability:** Architecture §5.3, TDD §3.3

---

## TR-006-RRF-002 — RRF constant k=60
🟠 P1 | `[UNIT]`

**Given:** Document ở rank 1  
**When:** RRF score được tính  
**Then:** Component = weight × (1/(60 + 1)) = weight/61

**Traceability:** Architecture §5.3 (k=60)

---

## TR-006-RRF-003 — Weight normalization: vector stream empty
🔴 P0 | `[UNIT]` | **FR-SEARCH-001**

**Given:** Không có embedding provider  
**When:** Hybrid search chạy (BM25 + Graph only)  
**Then:**
- `effectiveVectorW = 0`
- `totalW = bm25W + graphW` (không có vectorW)
- Weights renormalized: bm25W/totalW + graphW/totalW = 1.0
- Không throw error

**Traceability:** TDD §3.3, TR-000-GRD-001

---

## TR-006-RRF-004 — Weight normalization: graph stream empty
🟠 P1 | `[UNIT]`

**Given:** `GRAPH_EXTRACTION_ENABLED=false`  
**When:** Hybrid search chạy  
**Then:**
- `effectiveGraphW = 0`
- Only BM25 + Vector weights remain, normalized
- Correct scores returned

**Traceability:** TDD §3.3

---

## TR-006-RRF-005 — Session diversification: max 3 per session
🔴 P0 | `[UNIT]` | **FR-SEARCH-001**

**Given:** 10 results tất cả từ cùng session "sess_A"  
**When:** Diversification được áp dụng với limit=10  
**Then:**
- Chỉ 3 results từ sess_A được chọn vào top results
- Remaining slots filled từ overflow pool

**Traceability:** Architecture §5.3, TDD §3.3

---

## TR-006-RRF-006 — Session diversification: multi-session
🟠 P1 | `[UNIT]`

**Given:** 9 results: sess_A (5), sess_B (3), sess_C (1)  
**When:** Diversification với limit=9  
**Then:**
- sess_A: max 3 selected
- sess_B: max 3 selected
- sess_C: 1 selected
- Total = 7 results (không đủ 9, fill từ overflow)

**Traceability:** Architecture §5.3

---

## TR-006-RRF-007 — Enrich: fetch CompressedObservation từ KV
🔴 P0 | `[INT]`

**Given:** Hybrid search trả về list obsIds  
**When:** Enrich step chạy  
**Then:**
- Mỗi result có `observation: CompressedObservation` đầy đủ
- Fallback: nếu không tìm được trong observations, thử KV.memories
- Kết quả có: `bm25Score`, `vectorScore`, `graphScore`, `combinedScore`

**Traceability:** TDD §3.3 step 7

---

## TR-006-RRF-008 — Smart search: latency ≤ 50ms
🔴 P0 | `[PERF]` | **FR-SEARCH-002**

**Given:** Index với 1000 documents, embedding provider available  
**When:** `mem::smart-search({query: "authentication"})` được gọi  
**Then:** Response nhận được trong ≤ 50ms (p50)

**Traceability:** FR-SEARCH-002, SRS §3.5

---

## TR-006-RRF-009 — Smart search: p50 target 14ms
🟠 P1 | `[PERF]` | **FR-SEARCH-002**

**Given:** Index với 1000 documents  
**When:** 100 search queries được chạy  
**Then:** p50 latency ≤ 14ms, p95 ≤ 100ms

**Traceability:** SRS §4.1, PRD §7

---

## TR-006-RRF-010 — Query expansion với LLM
🟠 P1 | `[INT]` | **FR-SEARCH-002**

**Given:** LLM provider available  
**When:** `mem::smart-search({query: "how did we fix the N+1 problem"})`  
**Then:**
- Query expansion được gọi (async)
- Reformulations: `["database query optimization", "eager loading fix", ...]`
- Temporal: dates extracted nếu có
- Entity extractions: `["N+1", "database"]`
- All queries được searched, results merged (best score per obsId)

**Traceability:** FR-SEARCH-002, TDD §3.4, Architecture §4.2

---

## TR-006-RRF-011 — Smart search: noop provider (no expansion)
🟠 P1 | `[UNIT]`

**Given:** Không có LLM provider  
**When:** `mem::smart-search({query: "auth pattern"})`  
**Then:**
- Không có query expansion
- Chỉ BM25 + Vector (nếu có embedding) được dùng
- Không throw error

**Traceability:** TDD §3.4

---

## TR-006-RRF-012 — Project scoping
🔴 P0 | `[INT]` | **FR-SEARCH-001**

**Given:** 20 observations: 10 từ project "A", 10 từ project "B"  
**When:** `mem::smart-search({query: "auth", project: "A"})`  
**Then:** Chỉ trả về results từ project "A" (10 candidates, filter post-search)

**Traceability:** UR-015, SRS §3.5

---

## TR-006-RRF-013 — Graph expansion từ vector results
🟠 P1 | `[INT]` | **FR-SEARCH-001**

**Given:** Graph extraction enabled, obs1 có entity "jose", obs2 connected qua graph  
**When:** Smart search query = "jose middleware"  
**Then:**
- Top-5 vector results dùng để expand qua graph
- obs2 (connected) xuất hiện trong graph results

**Traceability:** TDD §3.3 step 1c, Architecture §5.4

---

## TR-006-RRF-014 — LongMemEval-S benchmark
🔴 P0 | `[PERF]` | **FR-SEARCH-005**

**Given:** LongMemEval-S dataset (500 questions, ICLR 2025)  
**When:** `npm run eval:longmemeval` được chạy với hybrid search  
**Then:**
- R@5 ≥ 95.2%
- R@10 ≥ 98.6%
- MRR ≥ 88.2%

**Traceability:** SRS §3.5 FR-SEARCH-005, PRD §7

---

## TR-006-RRF-015 — BM25-only mode: outperforms grep
🟡 P2 | `[PERF]`

**Given:** LongMemEval-S với BM25-only adapter  
**When:** Eval runs  
**Then:** BM25 R@5 > grep R@5 (baseline)

**Traceability:** TDD §14.3

---

## TR-006-RRF-016 — Reranking opt-in
🟡 P2 | `[INT]`

**Given:** `RERANK_ENABLED=true`  
**When:** Smart search với 25 candidates  
**Then:**
- Cross-encoder reranking được áp dụng trên top-20
- Final results có improved relevance (tested via eval)

**Traceability:** TDD §3.3, Architecture §5.3

---

## TR-006-RRF-017 — Search result format đầy đủ
🔴 P0 | `[UNIT]` | **FR-SEARCH-003**

**Given:** Smart search trả về results  
**When:** Mỗi result được kiểm tra  
**Then:** Mỗi `HybridSearchResult` có:
```typescript
{
  observation: CompressedObservation,
  bm25Score: number,
  vectorScore: number,
  graphScore: number,
  combinedScore: number,
  sessionId: string,
  graphContext?: string
}
```

**Traceability:** SRS §3.5 FR-SEARCH-003
