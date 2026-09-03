# TC-004: BM25 Search Index — Test Cases

**Test Design tham chiếu:** [TD-004](../designs/TD-004-search-bm25.md)  
**Requirements tham chiếu:** [TR-004](../requirements/TR-004-search-bm25.md)  
**Module:** SearchIndex (BM25 với synonym, prefix, CJK, stemmer)  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## NHÓM A: CƠ BẢN

---

## TC-004-001: Index trống — size=0, search trả về []

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-001 |
| **Tên** | SearchIndex mới tạo có size=0 và search rỗng |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Điều kiện tiên quyết:** SearchIndex mới (chưa add document nào)

**Các bước thực hiện:**
1. Tạo SearchIndex mới
2. Đọc `index.size`
3. Gọi `index.search("anything", 10)`

**Kết quả mong đợi:**
- `index.size === 0`
- `index.search(...)` trả về `[]`

---

## TC-004-002: Add document → tìm thấy bằng exact term

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-002 |
| **Tên** | Document được add và tìm thấy qua exact keyword |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-004-BM25-001 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `obsId` | `obs_auth_001` |
| `sessionId` | `sess_test` |
| `concepts` | `["authentication", "jwt", "token"]` |
| `title` | `"Auth middleware implementation"` |

**Các bước thực hiện:**
1. Add observation vào SearchIndex
2. Gọi `index.search("authentication", 5)`
3. Kiểm tra results

**Kết quả mong đợi:**
- `results.length >= 1`
- `results[0].obsId = "obs_auth_001"`
- `results[0].score > 0`

---

## TC-004-003: Không tìm thấy khi không có match

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-003 |
| **Tên** | Search trả về [] khi không có document khớp |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**
- Index có obs với concepts `["authentication", "jwt"]`
- Query: `"database"`

**Kết quả mong đợi:** `results = []`

---

## TC-004-004: Empty/whitespace query trả về []

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-004 |
| **Tên** | Empty hoặc whitespace-only query trả về mảng rỗng |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-004-BM25-011 |

**Dữ liệu đầu vào (test 2 cases):**

| Query | Expected |
|---|---|
| `""` (empty) | `[]` |
| `"   "` (spaces) | `[]` |
| `"\t\n"` (whitespace) | `[]` |

**Tiêu chí Pass:** Tất cả 3 queries trả về `[]`

---

## NHÓM B: SYNONYM EXPANSION

---

## TC-004-005: `"auth"` tìm được docs chứa `"authentication"`

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-005 |
| **Tên** | Synonym expansion: auth → authentication |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-004-BM25-002 |

**Điều kiện tiên quyết:** Index có obs với `concepts: ["authentication"]` (không có "auth")

**Các bước thực hiện:**
1. Add obs_A với `concepts: ["authentication"]`
2. Gọi `search("auth", 5)` (từ "auth" không có trong index)
3. Kiểm tra results

**Kết quả mong đợi:** obs_A xuất hiện trong results (do synonym expansion `auth` → `authentication`)

---

## TC-004-006: Synonym score thấp hơn exact match score

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-006 |
| **Tên** | Exact match được rank cao hơn synonym match |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-004-BM25-003 |

**Dữ liệu đầu vào:**

| obsId | concepts | Loại match với query "auth" |
|---|---|---|
| `obs_exact` | `["auth"]` | Exact match |
| `obs_synonym` | `["authentication"]` | Synonym match |

**Các bước thực hiện:**
1. Add cả 2 obs vào index
2. Gọi `search("auth", 5)`
3. So sánh scores

**Kết quả mong đợi:**
- Cả 2 obs đều xuất hiện trong results
- `score(obs_exact) > score(obs_synonym)`

---

## TC-004-007: `"db"` tìm được docs chứa `"database"`

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-007 |
| **Tên** | Synonym expansion: db → database |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- Index có obs với `concepts: ["database", "sql"]`

**Kết quả mong đợi:** `search("db", 5)` tìm được obs đó

---

## NHÓM C: PREFIX MATCHING

---

## TC-004-008: Prefix match: `"authen"` tìm `"authentication"`

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-008 |
| **Tên** | Prefix matching: authen tìm authentication |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-004-BM25-004 |

**Dữ liệu đầu vào:**
- Index có obs với `concepts: ["authentication"]`

**Kết quả mong đợi:** `search("authen", 5)` tìm được obs

---

## TC-004-009: Exact match ranked higher than prefix match

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-009 |
| **Tên** | Exact match có score >= prefix match |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| obsId | concepts | Loại match với query "redis" |
|---|---|---|
| `obs_exact` | `["redis"]` | Exact match |
| `obs_prefix` | `["redistool"]` | Prefix match |

**Kết quả mong đợi:** `score(obs_exact) >= score(obs_prefix)`

---

## NHÓM D: MULTI-TERM RANKING

---

## TC-004-010: Doc khớp nhiều terms được rank cao hơn

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-010 |
| **Tên** | Document khớp cả hai query terms được rank cao hơn doc khớp một |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-004-BM25-007 |

**Dữ liệu đầu vào:**

| obsId | concepts | Terms matched với query "redis cache" |
|---|---|---|
| `obs_both` | `["redis", "cache", "memory"]` | 2 terms (redis, cache) |
| `obs_one` | `["redis", "store"]` | 1 term (redis) |

**Các bước thực hiện:**
1. Add cả 2 obs
2. Gọi `search("redis cache", 5)`
3. Kiểm tra rank

**Kết quả mong đợi:**
- `results[0].obsId = "obs_both"` (rank 1)
- `score(obs_both) > score(obs_one)`

---

## TC-004-011: Limit parameter giới hạn số kết quả

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-011 |
| **Tên** | search(query, limit) trả về đúng số lượng kết quả |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-004-BM25-010 |

**Dữ liệu đầu vào:**
- 30 documents đều có concept `"authentication"`
- Query: `"authentication"`, limit: `5`

**Kết quả mong đợi:** `results.length = 5` (không phải 30)

---

## TC-004-012: Limit > total docs: trả về tất cả

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-012 |
| **Tên** | Khi limit lớn hơn số docs, trả về tất cả docs |
| **Loại** | Unit |
| **Ưu tiên** | 🟡 P2 |

**Dữ liệu đầu vào:**
- 8 documents trong index
- `search("auth", 100)` (limit = 100)

**Kết quả mong đợi:** `results.length = 8` (không padding)

---

## NHÓM E: MULTI-FIELD INDEXING

---

## TC-004-013: Tìm kiếm qua tất cả 6 fields

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-013 |
| **Tên** | Search tìm được document qua bất kỳ trong 6 indexed fields |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-004-BM25-001 |

**Dữ liệu đầu vào và setup (6 observations, mỗi cái có unique term ở 1 field):**

| obsId | Field có unique term | Term |
|---|---|---|
| `obs_title` | `title` | `titleuniqterm001` |
| `obs_narrative` | `narrative` | `narrativeuniqterm002` |
| `obs_facts` | `facts` | `factsuniqterm003` |
| `obs_concepts` | `concepts` | `conceptsuniqterm004` |
| `obs_files` | `files` | `filesuniqterm005.ts` |
| `obs_subtitle` | `subtitle` | `subtitleuniqterm006` |

**Các bước thực hiện (6 searches):**
1. `search("titleuniqterm001", 1)` → phải tìm thấy `obs_title`
2. `search("narrativeuniqterm002", 1)` → phải tìm thấy `obs_narrative`
3. `search("factsuniqterm003", 1)` → phải tìm thấy `obs_facts`
4. `search("conceptsuniqterm004", 1)` → phải tìm thấy `obs_concepts`
5. `search("filesuniqterm005", 1)` → phải tìm thấy `obs_files`
6. `search("subtitleuniqterm006", 1)` → phải tìm thấy `obs_subtitle`

**Tiêu chí Pass:** Tất cả 6 searches đều tìm thấy đúng obs.

---

## NHÓM F: ADD/REMOVE

---

## TC-004-014: Remove: document không còn xuất hiện trong kết quả

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-014 |
| **Tên** | Sau khi remove, document bị loại khỏi search results |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-004-BM25-012 |

**Dữ liệu đầu vào:**

| obsId | concepts | Rank trước remove |
|---|---|---|
| `obs_remove_me` | `["jose", "jwt"]` | Rank 1 |
| `obs_keep` | `["jwt"]` | Rank 2 |

**Các bước thực hiện:**
1. Add cả 2 obs, verify `obs_remove_me` là rank 1 với `search("jose jwt")`
2. Gọi `index.remove("obs_remove_me")`
3. Gọi `search("jose jwt", 1)`
4. Kiểm tra `index.size`

**Kết quả mong đợi:**
- Sau remove: `results[0].obsId = "obs_keep"` (không phải `obs_remove_me`)
- `index.size = 1`

---

## TC-004-015: Remove: posting list trống được cleanup

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-015 |
| **Tên** | Sau remove, term chỉ có ở 1 doc thì posting list bị cleanup |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- `obs_unique` có concept `"unique_concept_xyz"` (chỉ doc duy nhất có term này)

**Các bước thực hiện:**
1. Add `obs_unique`
2. Verify `search("unique_concept_xyz", 1)` tìm được
3. `index.remove("obs_unique")`
4. `search("unique_concept_xyz", 1)` 
5. `search("unique_", 1)` (prefix search)

**Kết quả mong đợi:**
- Cả 2 searches ở bước 4, 5 đều trả về `[]`
- Posting list cho `"unique_concept_xyz"` đã bị cleanup

---

## TC-004-016: Remove ID không tồn tại: no-op, không crash

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-016 |
| **Tên** | remove() với ID không tồn tại không gây lỗi |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Các bước thực hiện:**
1. Index có `obs_A`
2. `index.remove("does_not_exist_id")`
3. Kiểm tra index vẫn hoạt động bình thường

**Kết quả mong đợi:**
- Không throw exception
- `index.size` không thay đổi (= 1)

---

## NHÓM G: CJK TEXT

---

## TC-004-017: Tìm kiếm tiếng Nhật

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-017 |
| **Tên** | SearchIndex xử lý và tìm kiếm tiếng Nhật |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-004-BM25-006 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `obsId` | `obs_jp` |
| `title` | `"認証システム"` (Auth system in Japanese) |
| `concepts` | `["認証", "トークン"]` |

**Các bước thực hiện:**
1. Add `obs_jp`
2. `search("認証", 5)` (search "auth" in Japanese)
3. Kiểm tra results

**Kết quả mong đợi:** `obs_jp` xuất hiện trong results, `score > 0`

---

## TC-004-018: Tìm kiếm tiếng Trung

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-018 |
| **Tên** | SearchIndex xử lý và tìm kiếm tiếng Trung |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `obsId` | `obs_cn` |
| `concepts` | `["身份验证", "数据库"]` |

**Kết quả mong đợi:** `search("身份验证", 5)` tìm được `obs_cn`

---

## NHÓM H: PORTER STEMMER

---

## TC-004-019: Stemming — "running" match "run"

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-019 |
| **Tên** | Porter stemmer cho phép tìm kiếm theo root form |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-004-BM25-005 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `obsId` | `obs_stem` |
| `concepts` | `["testing", "running", "processes"]` |

**Các bước thực hiện:**
1. Add `obs_stem`
2. `search("run test", 5)` (stemmed forms)
3. Kiểm tra results

**Kết quả mong đợi:** `obs_stem` xuất hiện trong results

---

## NHÓM I: SERIALIZATION

---

## TC-004-020: Serialize/deserialize round-trip

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-020 |
| **Tên** | Serialized và deserialized SearchIndex có state giống nhau |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-004-BM25-013 |

**Dữ liệu đầu vào:**
- 2 documents: `obs_A` (concepts: `["auth"]`), `obs_B` (concepts: `["database"]`)

**Các bước thực hiện:**
1. Add `obs_A` và `obs_B`
2. `const json = index.serialize()`
3. Kiểm tra `json` có key `"v"` = `2` (version marker)
4. `const restored = SearchIndex.deserialize(json)`
5. Kiểm tra `restored.size = 2`
6. `restored.search("auth", 1)` → phải tìm được `obs_A`
7. `restored.search("database", 1)` → phải tìm được `obs_B`

**Tiêu chí Pass:** Cả 2 docs tìm được từ restored index.

---

## TC-004-021: Deserialize invalid JSON: trả về empty index (không crash)

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-021 |
| **Tên** | Deserialize với input không hợp lệ trả về empty index |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào (test 3 cases):**

| Input | Type |
|---|---|
| `"not-valid-json"` | malformed JSON |
| `"{}"` | empty object JSON |
| `"null"` | null JSON |

**Kết quả mong đợi:**
- `SearchIndex.deserialize(input)` không throw exception
- Trả về empty index với `size = 0`

---

## NHÓM K: PERFORMANCE

---

## TC-004-022: 1000 docs index time < 1000ms

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-022 |
| **Tên** | Index throughput: 1000 documents trong vòng 1000ms |
| **Loại** | Performance |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-004-BM25-017 |

**Dữ liệu đầu vào:**
- 1000 CompressedObservations với concepts đa dạng (10-20 concepts mỗi obs)

**Điều kiện môi trường:**
- Isolated process, không background workloads
- Warm-up: 5 obs trước khi bắt đầu đo

**Các bước thực hiện:**
1. Chuẩn bị 1000 docs
2. Bắt đầu timer
3. Add tất cả 1000 docs vào index
4. Dừng timer
5. Kiểm tra total time

**Kết quả mong đợi:** `totalTime < 1000ms`

---

## TC-004-023: 1000 docs search latency < 10ms per query (p95)

| Trường | Giá trị |
|---|---|
| **ID** | TC-004-023 |
| **Tên** | BM25 search latency SLA: p95 < 10ms |
| **Loại** | Performance |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- Index: 1000 docs (từ TC-004-022)
- 100 queries với 10 diverse terms

**Các bước thực hiện:**
1. Chạy 100 queries, ghi từng latency
2. Tính p50 và p95

**Kết quả mong đợi:**
- `p50 ≤ 10ms`
- `p95 ≤ 10ms`

---

## Tổng kết Module TC-004

| TC ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-004-001 | Empty index | 🔴 P0 | Unit |
| TC-004-002 | Exact match | 🔴 P0 | Unit |
| TC-004-003 | No match | 🔴 P0 | Unit |
| TC-004-004 | Empty/whitespace query | 🔴 P0 | Unit |
| TC-004-005 | Synonym auth→authentication | 🔴 P0 | Unit |
| TC-004-006 | Exact > synonym score | 🟠 P1 | Unit |
| TC-004-007 | Synonym db→database | 🟠 P1 | Unit |
| TC-004-008 | Prefix match | 🟠 P1 | Unit |
| TC-004-009 | Exact > prefix score | 🟠 P1 | Unit |
| TC-004-010 | Multi-term ranking | 🔴 P0 | Unit |
| TC-004-011 | Limit parameter | 🔴 P0 | Unit |
| TC-004-012 | Limit > docs | 🟡 P2 | Unit |
| TC-004-013 | 6 fields indexed | 🟠 P1 | Unit |
| TC-004-014 | Remove doc | 🔴 P0 | Unit |
| TC-004-015 | Posting list cleanup | 🟠 P1 | Unit |
| TC-004-016 | Remove unknown ID | 🔴 P0 | Unit |
| TC-004-017 | Japanese search | 🟠 P1 | Unit |
| TC-004-018 | Chinese search | 🟠 P1 | Unit |
| TC-004-019 | Porter stemming | 🟠 P1 | Unit |
| TC-004-020 | Serialize/deserialize | 🔴 P0 | Unit |
| TC-004-021 | Deserialize invalid | 🔴 P0 | Unit |
| TC-004-022 | Index 1000 docs perf | 🟠 P1 | Perf |
| TC-004-023 | Search latency p95 | 🟠 P1 | Perf |
