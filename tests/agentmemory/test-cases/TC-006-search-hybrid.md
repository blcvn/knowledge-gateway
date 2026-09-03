# TC-006: Hybrid Search (RRF) — Test Cases

**Test Design tham chiếu:** [TD-006](../designs/TD-006-search-hybrid.md)  
**Requirements tham chiếu:** [TR-006](../requirements/TR-006-search-hybrid.md)  
**Module:** HybridSearch (RRF_K=60, weight normalization, fallback)  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-006-001: BM25-only search hoạt động khi không có embedding provider

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-001 |
| **Tên** | HybridSearch trả về BM25 results khi vectorIndex = null |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-006-HYB-003 |

**Điều kiện tiên quyết:**
- SearchIndex có 5 documents về "auth"
- VectorIndex = null
- EmbeddingProvider = null

**Dữ liệu đầu vào:**
- `query = "auth"`, `limit = 5`

**Các bước thực hiện:**
1. Khởi tạo HybridSearch với SearchIndex, VectorIndex=null, EmbeddingProvider=null
2. Gọi `search("auth", 5)`
3. Kiểm tra results

**Kết quả mong đợi:**
- `results.length >= 1` (không phải empty)
- `results[0].bm25Score > 0`
- `results[0].vectorScore = 0`
- Không có exception

---

## TC-006-002: Kết quả có đầy đủ fields `HybridSearchResult`

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-002 |
| **Tên** | Mỗi search result có đầy đủ các fields bắt buộc |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-006-HYB-010 |

**Điều kiện tiên quyết:** BM25-only mode, 1 document khớp query

**Các bước thực hiện:**
1. `search("auth", 1)`
2. Inspect `results[0]`

**Kết quả mong đợi — tất cả fields phải tồn tại:**
- `results[0].observation` là CompressedObservation object (có id, title, ...)
- `results[0].bm25Score` là number
- `results[0].vectorScore` là number (0 nếu no vector)
- `results[0].graphScore` là number
- `results[0].combinedScore` là number > 0
- `results[0].sessionId` là string

---

## TC-006-003: RRF formula verification: score ≈ 1/(60+1) cho rank-1 doc

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-003 |
| **Tên** | combinedScore của rank-1 BM25 doc xấp xỉ RRF formula |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-006-HYB-001 |

**Điều kiện:**
- BM25-only mode (vectorWeight=0, graphWeight=0, bm25Weight=1.0)
- Document ở rank 1 trong BM25 (chỉ có 1 document trong index)

**Tính toán expected:**
- `RRF_K = 60`
- Rank của doc = 1
- `combinedScore = 1.0 × 1/(60 + 1) = 1/61 ≈ 0.01639`

**Kết quả mong đợi:**
- `|results[0].combinedScore - 0.01639| < 0.001`

---

## TC-006-004: Doc xuất hiện ở cả 2 luồng được rank cao hơn doc chỉ ở 1 luồng

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-004 |
| **Tên** | Multi-stream presence boost document ranking |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-006-HYB-002 |

**Điều kiện tiên quyết:** BM25 + Vector cả hai enabled

**Dữ liệu đầu vào:**

| obsId | BM25 rank | Vector rank |
|---|---|---|
| `obs_dual` | 1 | 1 |
| `obs_bm25_only` | 1 | *(không xuất hiện)* |

**Các bước thực hiện:**
1. Setup index với 2 obs
2. Search với matching query
3. So sánh `combinedScore`

**Kết quả mong đợi:**
- `score(obs_dual) > score(obs_bm25_only)`

---

## TC-006-005: Weight normalization — tổng effective weights = 1.0

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-005 |
| **Tên** | Weights được normalize để tổng = 1.0 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-006-HYB-007 |

**Dữ liệu đầu vào (default weights):**
- `bm25Weight = 0.4`
- `vectorWeight = 0.6`
- `graphWeight = 0.3`
- Total = 1.3

**Expected normalized weights:**
- `bm25W_eff = 0.4 / 1.3 ≈ 0.308`
- `vectorW_eff = 0.6 / 1.3 ≈ 0.462`
- `graphW_eff = 0.3 / 1.3 ≈ 0.231`

**Verification:** Tạo doc ở rank-1 tất cả streams, tính `combinedScore` và verify từng weight đóng góp đúng.

**Kết quả mong đợi:**
- `bm25W_eff + vectorW_eff + graphW_eff ≈ 1.0` (tolerance 0.001)

---

## TC-006-006: Embedding failure → graceful fallback to BM25-only

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-006 |
| **Tên** | Khi embedding throw error, BM25 results vẫn được trả về |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-006-HYB-003 |

**Điều kiện tiên quyết:**
- EmbeddingProvider được inject nhưng `embed()` luôn throw Error("API down")
- SearchIndex có data

**Các bước thực hiện:**
1. `search("auth", 5)` — embedding sẽ fail
2. Kiểm tra response

**Kết quả mong đợi:**
- Không throw ra ngoài caller
- BM25 results được trả về (length >= 1)
- `results[i].vectorScore = 0` cho tất cả results

---

## TC-006-007: Session diversification — tối đa 3 results/session

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-007 |
| **Tên** | Kết quả được diversify: không quá 3 từ cùng session |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-006-HYB-011 |

**Dữ liệu đầu vào:**
- 6 observations từ `session-A` (tất cả khớp "auth")
- 3 observations từ `session-B` (tất cả khớp "auth")

**Các bước thực hiện:**
1. Index tất cả 9 observations
2. `search("auth", 20)` (limit lớn để không bị cut)
3. Đếm số kết quả theo sessionId

**Kết quả mong đợi:**
- Số results từ `session-A` ≤ 3
- Số results từ `session-B` ≤ 3

---

## TC-006-008: searchWithExpansion — nhiều reformulations được merge

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-008 |
| **Tên** | searchWithExpansion tổng hợp results từ nhiều query reformulations |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-006-HYB-015 |

**Dữ liệu đầu vào:**
- obs_A: chỉ khớp term "authentication"
- obs_B: chỉ khớp term "login"
- obs_C: chỉ khớp term "signin"

**Query expansion input:**
- Original query: `"auth"`
- Reformulations: `["authentication", "login", "signin"]`

**Các bước thực hiện:**
1. Index obs_A, obs_B, obs_C
2. Gọi `searchWithExpansion("auth", 10, {reformulations: ["authentication", "login", "signin"]})`
3. Kiểm tra results

**Kết quả mong đợi:**
- Cả 3 observations xuất hiện trong results

---

## TC-006-009: Duplicate trong expansion merge → chỉ giữ score cao nhất

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-009 |
| **Tên** | Obs xuất hiện trong nhiều reformulations chỉ được include 1 lần |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Điều kiện:** obs_X khớp với cả reformulation 1 (score=0.8) và reformulation 2 (score=0.5)

**Kết quả mong đợi:**
- obs_X chỉ xuất hiện 1 lần trong results
- Score của obs_X = 0.8 (giữ score cao nhất)

---

## TC-006-010: Performance — BM25-only p50 ≤ 14ms (1000 docs)

| Trường | Giá trị |
|---|---|
| **ID** | TC-006-010 |
| **Tên** | HybridSearch BM25-only mode đáp ứng SLA latency |
| **Loại** | Performance |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-006-HYB-016, TR-021-PRF-001 |

**Setup:**
- 1000 documents indexed vào BM25
- EmbeddingProvider = null (BM25-only)
- 100 queries với 10 diverse terms

**Các bước thực hiện:**
1. Warm up: 5 queries (không đo)
2. Chạy 100 queries, ghi latency mỗi query
3. Tính p50 và p95

**Kết quả mong đợi:**
- `p50 ≤ 14ms`
- `p95 ≤ 100ms`

---

## Tổng kết Module TC-006

| TC ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-006-001 | BM25-only mode | 🔴 P0 | Integration |
| TC-006-002 | Result fields complete | 🔴 P0 | Integration |
| TC-006-003 | RRF formula verify | 🔴 P0 | Unit |
| TC-006-004 | Multi-stream boost | 🟠 P1 | Integration |
| TC-006-005 | Weight normalization | 🔴 P0 | Unit |
| TC-006-006 | Embedding fail fallback | 🔴 P0 | Integration |
| TC-006-007 | Session diversification | 🔴 P0 | Integration |
| TC-006-008 | Expansion merge | 🟠 P1 | Integration |
| TC-006-009 | Expansion dedup | 🟠 P1 | Unit |
| TC-006-010 | Performance SLA | 🔴 P0 | Performance |
