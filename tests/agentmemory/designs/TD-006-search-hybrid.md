# TD-006: Hybrid Search (RRF) Test Design

**Liên kết Requirements:** [TR-006-search-hybrid.md](../requirements/TR-006-search-hybrid.md)  
**Source:** `references/agentmemory/src/state/hybrid-search.ts`  
**Test file:** `tests/agentmemory/specs/hybrid-search.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

`HybridSearch` kết hợp 3 luồng tìm kiếm qua Reciprocal Rank Fusion (RRF):
- **BM25** (lexical, default weight 0.4)
- **Vector** (semantic, default weight 0.6)
- **Graph** (entity-based, default weight 0.3)

**Cơ chế:** `combinedScore = Σ weight × 1/(RRF_K + rank)` với `RRF_K=60`

---

## 2. Implementation Analysis

| Behavior | Source code |
|---|---|
| `RRF_K = 60` | Line 20, hardcoded constant |
| Rank=Infinity → 0 contribution | `1 / (60 + Infinity) = 0` (JS) |
| Weight normalization | `effectiveW /= totalW` (line 201-205) |
| Vector failure → BM25-only | `try/catch` block, falls through |
| Graph failure → best-effort | `try/catch` block, ignored |
| Session diversification | max 3 results per session |
| Rerank | Opt-in via `RERANK_ENABLED=true` |

---

## 3. Chiến lược kiểm thử

| Khía cạnh | Kỹ thuật |
|---|---|
| RRF formula | Math property test (verify score từ rank) |
| Weight normalization | Algebraic verification |
| Fallback | Error injection vào embedding provider |
| Diversification | Index nhiều docs cùng session, verify capping |
| Result structure | Structural assertion |
| Performance | 1000 docs, measure p50/p95 |

---

## 4. Test Cases

### Group A: BM25-Only Mode

#### TC-001 — BM25-only search hoạt động khi không có embedding provider
**Requirement:** TR-006-HYB-003 | **Type:** integration | 🔴 P0

**Given:**
- SearchIndex có docs
- VectorIndex = null, EmbeddingProvider = null

**When:** `search("auth", 10)` gọi  
**Then:**
- Kết quả được trả về (không phải mảng rỗng)
- `results[0].bm25Score > 0`
- `results[0].vectorScore = 0`
- Không throw

---

#### TC-002 — Kết quả có đầy đủ fields của `HybridSearchResult`
**Requirement:** TR-006-HYB-010 | **Type:** integration | 🔴 P0

**Given:** BM25-only mode, 1 document khớp query  
**When:** `search("auth", 1)` gọi  
**Then:** Result có:
- `observation`: CompressedObservation object
- `bm25Score`: number
- `vectorScore`: number
- `graphScore`: number
- `combinedScore`: number > 0
- `sessionId`: string

---

### Group B: RRF Formula

#### TC-003 — RRF formula verification: k=60
**Requirement:** TR-006-HYB-001 | **Type:** unit | 🔴 P0

**Given:** Document xuất hiện ở rank 1 trong BM25 (vectorRank = Infinity, graphRank = Infinity)  
**When:** BM25-only mode với bm25Weight = 1.0 (normalized)  
**Then:**
- `combinedScore ≈ 1 / (60 + 1) ≈ 0.01639`
- Tolerance: ±0.001

---

#### TC-004 — Rank=Infinity đóng góp 0 vào combined score
**Type:** unit | 🟠 P1

**Given:** Doc chỉ xuất hiện trong BM25 (vectorRank = Infinity)  
**When:** Score tính toán  
**Then:** Vector contribution = `vectorWeight × 1/(60 + Infinity) = 0`

---

#### TC-005 — Doc xuất hiện ở cả 2 luồng được rank cao hơn doc chỉ ở 1 luồng
**Requirement:** TR-006-HYB-002 | **Type:** integration | 🟠 P1

**Given:**
- Obs A: rank 1 BM25, rank 1 vector
- Obs B: rank 1 BM25, không có trong vector

**When:** Search với cả BM25 và vector enabled  
**Then:** `score(A) > score(B)`

---

### Group C: Weight Normalization

#### TC-006 — Weights được normalize khi cả 3 streams active
**Requirement:** TR-006-HYB-007 | **Type:** unit | 🔴 P0

**Given:** Default weights: bm25=0.4, vector=0.6, graph=0.3 → total=1.3  
**When:** Weights được normalize  
**Then:**
- `bm25W_eff ≈ 0.4/1.3 ≈ 0.308`
- `vectorW_eff ≈ 0.6/1.3 ≈ 0.462`
- `graphW_eff ≈ 0.3/1.3 ≈ 0.231`
- `bm25W_eff + vectorW_eff + graphW_eff ≈ 1.0`

---

#### TC-007 — Khi vector không available, weights của bm25 và graph renormalize
**Type:** unit | 🟠 P1

**Given:** BM25 + graph chỉ (vector provider null)  
**When:** vectorWeight = 0 → total = bm25W + graphW = 0.7  
**Then:**
- Effective bm25 ≈ 0.4/0.7 ≈ 0.571
- Effective graph ≈ 0.3/0.7 ≈ 0.429
- Tổng = 1.0

---

#### TC-008 — Custom weights qua constructor
**Requirement:** TR-006-HYB-007 | **Type:** integration | 🟡 P2

**Given:** HybridSearch được tạo với `bm25Weight=1.0, vectorWeight=0.0, graphWeight=0.0`  
**When:** Search  
**Then:** `combinedScore = 1/(60+rank)` — chỉ BM25 đóng góp

---

### Group D: Fallback Behaviors

#### TC-009 — Embedding failure → graceful fallback to BM25-only
**Requirement:** TR-006-HYB-003 | **Type:** integration | 🔴 P0

**Given:** EmbeddingProvider.embed() luôn throw Error  
**When:** `search()` gọi  
**Then:**
- Không throw ra ngoài
- BM25 results vẫn được trả về
- `vectorScore = 0` cho tất cả results

---

#### TC-010 — Graph search failure → ignored, BM25+vector results vẫn trả về
**Type:** integration | 🟠 P1

**Given:** Graph retrieval gặp lỗi  
**When:** `search()` gọi  
**Then:**
- Không throw
- Results từ BM25 và/hoặc vector vẫn được trả về
- `graphScore = 0`

---

### Group E: Session Diversification

#### TC-011 — Tối đa 3 kết quả mỗi session trong top-N
**Requirement:** TR-006-HYB-011 | **Type:** integration | 🔴 P0

**Given:**
- 6 documents từ cùng `sessionId="sess_A"` (tất cả khớp query)
- 3 documents từ `sessionId="sess_B"`

**When:** `search("auth", 20)` — limit lớn  
**Then:** Đếm theo session: `sess_A ≤ 3`, `sess_B ≤ 3`

---

#### TC-012 — Khi không đủ results từ diversification, overspill được include
**Type:** integration | 🟡 P2

**Given:** Chỉ có docs từ 1 session (không thể diversify)  
**When:** Search với limit=10, chỉ có 5 unique sessions  
**Then:** Tất cả 5 sessions đều có results (không bị cắt sai)

---

### Group F: searchWithExpansion

#### TC-013 — Nhiều reformulations được search và merged
**Requirement:** TR-006-HYB-015 | **Type:** integration | 🟠 P1

**Given:**
- Expansion có `reformulations: ["authentication", "login", "signin"]`
- Mỗi term match docs khác nhau

**When:** `searchWithExpansion("auth", 10, expansion)`  
**Then:** Kết quả bao gồm docs từ nhiều reformulations, merged bằng max score

---

#### TC-014 — Duplicate obs: chỉ giữ score cao nhất
**Type:** unit | 🟠 P1

**Given:** Obs X xuất hiện ở 2 reformulations với scores khác nhau  
**When:** Merge  
**Then:** Obs X chỉ xuất hiện 1 lần với score cao hơn

---

### Group G: Performance

#### TC-015 — p50 search latency ≤ 14ms với 1000 docs (BM25-only)
**Requirement:** TR-006-HYB-016, TR-021-PRF-001 | **Type:** performance | 🔴 P0

**Given:** 1000 documents được index vào BM25  
**When:** 100 queries được chạy liên tiếp  
**Then:**
- p50 (median) latency ≤ 14ms
- p95 latency ≤ 100ms

**Setup:** Chạy trong isolated environment, không có LLM/embedding calls

---

## 5. Coverage Notes

| Function | Branches cần cover |
|---|---|
| `tripleStreamSearch` | BM25-only, BM25+vector, all 3, embedding fail, graph fail |
| Weight normalization | All 3 active, vector=0, graph=0 |
| `diversifyBySession` | Under 3 per session, over 3 per session, fallback fill |
| `enrichResults` | Obs found, Obs not found (fallback to memory lookup) |
| `searchWithExpansion` | Merge logic, dedup |
