# TD-004: BM25 Search Index Test Design

**Liên kết Requirements:** [TR-004-search-bm25.md](../requirements/TR-004-search-bm25.md)  
**Source:** `references/agentmemory/src/state/search-index.ts`  
**Test file:** `tests/agentmemory/specs/search-index.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

`SearchIndex` implement thuật toán BM25 với các extension:
- **Synonym expansion** — synonyms.ts (weight 0.7)
- **Prefix matching** — binary search trên sorted terms (weight 0.5 × IDF)
- **CJK segmentation** — bigram cho Chinese/Japanese/Korean
- **Porter stemming** — English word stemming

**Hyperparameters:** k1=1.2, b=0.75 (cố định trong source)

---

## 2. Chiến lược kiểm thử

| Khía cạnh | Kỹ thuật |
|---|---|
| BM25 ranking | Comparative ranking tests (A > B) |
| Synonym/prefix weight | Score comparison |
| CJK | Script-specific test cases |
| Serialization | Round-trip với state verification |
| Remove | Regression tests cho posting list cleanup |
| Performance | 1000 docs, đo thời gian index + search |

---

## 3. Test Cases

### Group A: Cơ bản

#### TC-001 — Index trống: size=0, search trả về []
**Type:** unit | 🔴 P0

**Given:** `SearchIndex` mới tạo  
**When:** `size` được đọc và `search("anything")` gọi  
**Then:** `size = 0`, search trả về mảng rỗng

---

#### TC-002 — Add document, tìm thấy bằng exact term
**Requirement:** TR-004-BM25-001 | **Type:** unit | 🔴 P0

**Given:** Index trống  
**When:** Add obs với `concepts: ["authentication"]` rồi `search("authentication")`  
**Then:** Obs được tìm thấy, `score > 0`

---

#### TC-003 — Không tìm thấy khi không có match
**Type:** unit | 🔴 P0

**Given:** Index có obs về "auth"  
**When:** `search("database")`  
**Then:** Trả về `[]`

---

#### TC-004 — Empty/whitespace query trả về []
**Requirement:** TR-004-BM25-011 | **Type:** unit | 🔴 P0

**Given:** Index có data  
**When:** `search("")` và `search("   ")`  
**Then:** Cả 2 trả về `[]`

---

### Group B: Synonym Expansion

#### TC-005 — `"auth"` tìm được docs chứa `"authentication"`
**Requirement:** TR-004-BM25-002 | **Type:** unit | 🔴 P0

**Given:** Obs có `concepts: ["authentication"]` (không có "auth")  
**When:** `search("auth")`  
**Then:** Obs xuất hiện trong kết quả (do synonym expansion)

---

#### TC-006 — Synonym score < exact score (weight 0.7 discount)
**Requirement:** TR-004-BM25-003 | **Type:** unit | 🟠 P1

**Given:**
- Obs A: chứa chính xác từ "auth"
- Obs B: chứa "authentication" (synonym của "auth")

**When:** `search("auth")`  
**Then:** `score(A) > score(B)` — exact match được ưu tiên hơn synonym

---

#### TC-007 — `"db"` tìm được docs chứa `"database"`
**Type:** unit | 🟠 P1

**Given:** Obs có `concepts: ["database", "sql"]`  
**When:** `search("db")`  
**Then:** Obs xuất hiện trong kết quả

---

### Group C: Prefix Matching

#### TC-008 — Prefix match: `"authen"` tìm `"authentication"`
**Requirement:** TR-004-BM25-004 | **Type:** unit | 🟠 P1

**Given:** Obs có `concepts: ["authentication"]`  
**When:** `search("authen")`  
**Then:** Obs được tìm thấy

---

#### TC-009 — Exact match ranked higher than prefix match
**Requirement:** TR-004-BM25-004 | **Type:** unit | 🟠 P1

**Given:**
- Obs A: chứa chính xác từ `"redis"`
- Obs B: chứa từ `"redistool"` (prefix match của "redis")

**When:** `search("redis")`  
**Then:** `score(A) >= score(B)` — exact wins over prefix

---

### Group D: Multi-term Ranking

#### TC-010 — Doc khớp nhiều terms được rank cao hơn doc khớp ít terms
**Requirement:** TR-004-BM25-007 | **Type:** unit | 🔴 P0

**Given:**
- Obs A: chứa cả `"redis"` và `"cache"`
- Obs B: chỉ chứa `"redis"`

**When:** `search("redis cache")`  
**Then:** Obs A rank cao hơn Obs B

---

#### TC-011 — Limit parameter giới hạn số kết quả
**Requirement:** TR-004-BM25-010 | **Type:** unit | 🔴 P0

**Given:** 30 documents khớp query  
**When:** `search("auth", 5)`  
**Then:** Chỉ 5 kết quả được trả về

---

#### TC-012 — Limit lớn hơn số docs có sẵn: trả về tất cả
**Type:** unit | 🟡 P2

**Given:** 8 documents trong index  
**When:** `search("auth", 100)`  
**Then:** Trả về đúng 8 kết quả (không padding)

---

### Group E: Multi-field Indexing

#### TC-013 — Tìm kiếm qua tất cả 6 fields
**Requirement:** TR-004-BM25-001 | **Type:** unit | 🟠 P1

**Given:** 6 obs riêng biệt, mỗi obs có unique term ở 1 field khác nhau:
- Obs 1: term chỉ có trong `title`
- Obs 2: term chỉ có trong `narrative`
- Obs 3: term chỉ có trong `facts`
- Obs 4: term chỉ có trong `concepts`
- Obs 5: term chỉ có trong `files`
- Obs 6: term chỉ có trong `subtitle`

**When:** Search với mỗi unique term  
**Then:** Đúng obs tương ứng được tìm thấy trong mỗi case

---

### Group F: Add/Remove

#### TC-014 — Remove: document không còn xuất hiện trong kết quả
**Requirement:** TR-004-BM25-012 | **Type:** unit | 🔴 P0

**Given:**
- Obs A: chứa "jose" và "jwt" (rank 1)
- Obs B: chứa "jwt" (rank 2)

**After:** `remove("obs_A")`  
**When:** `search("jose jwt", 1)`  
**Then:**
- 1 kết quả được trả về
- Kết quả là Obs B (không phải Obs A đã bị remove)
- `size = 1`

---

#### TC-015 — Remove: posting list trống được cleanup
**Type:** unit | 🟠 P1

**Given:** Term `"unique_concept_xyz"` chỉ xuất hiện trong Obs A  
**After:** `remove("obs_A")`  
**When:** Prefix search `"unique_"` và exact search `"unique_concept_xyz"`  
**Then:** Cả 2 trả về `[]` (posting list được cleanup, sortedTerms cache được reset)

---

#### TC-016 — Remove với ID không tồn tại: no-op, không crash
**Type:** unit | 🔴 P0

**Given:** Index có Obs A  
**When:** `remove("does_not_exist")`  
**Then:** Không throw, `size` không đổi, search vẫn hoạt động

---

#### TC-017 — Add/remove/add cycle: consistency được duy trì
**Type:** unit | 🟠 P1

**Given:** Obs được add, rồi remove, rồi add lại  
**When:** Search sau mỗi bước  
**Then:**
- Sau add lần 1: `size=1`, tìm thấy
- Sau remove: `size=0`, không tìm thấy
- Sau add lần 2: `size=1`, tìm thấy (BM25 score không bị skew)

---

### Group G: CJK Text

#### TC-018 — Tìm kiếm tiếng Nhật (kana + kanji)
**Requirement:** TR-004-BM25-006 | **Type:** unit | 🟠 P1

**Given:** Obs có title tiếng Nhật, concepts bằng Japanese  
**When:** Search với từ tiếng Nhật  
**Then:** Obs được tìm thấy

---

#### TC-019 — Tìm kiếm tiếng Trung (Han characters)
**Type:** unit | 🟠 P1

**Given:** Obs có title và concepts bằng Chinese  
**When:** Search với từ Chinese  
**Then:** Obs được tìm thấy với score > 0

---

#### TC-020 — Tìm kiếm tiếng Hàn (Hangul)
**Type:** unit | 🟠 P1

**Given:** Obs có title Korean  
**When:** Search với từ Korean  
**Then:** Obs được tìm thấy

---

#### TC-021 — Mixed CJK và ASCII được segment đúng
**Type:** unit | 🟡 P2

**Given:** Text `"hello 项目 world"` qua `segmentCjk()`  
**When:** Segment được gọi  
**Then:** Trả về `["hello", "项目", "world"]` — thứ tự preserved

---

### Group H: Porter Stemmer

#### TC-022 — Stemming: `"running"` match `"run"`, `"tests"` match `"test"`
**Requirement:** TR-004-BM25-005 | **Type:** unit | 🟠 P1

**Given:** Obs với concepts `["testing", "running"]`  
**When:** `search("run test")`  
**Then:** Obs được tìm thấy (stemmer reduces "running"→"run", "testing"→"test")

---

### Group I: Serialization

#### TC-023 — Serialize/deserialize round-trip: đầy đủ dữ liệu
**Requirement:** TR-004-BM25-013 | **Type:** unit | 🔴 P0

**Given:** Index với 2 docs  
**When:** `serialize()` → `deserialize(json)`  
**Then:**
- `restored.size = 2`
- Cả 2 docs tìm được bằng search
- Format JSON có key `v: 2` (version marker)

---

#### TC-024 — Serialize sau remove: doc đã remove không có trong serialized data
**Type:** unit | 🟠 P1

**Given:** 2 docs, remove Obs A  
**When:** Serialize rồi deserialize  
**Then:** `restored.size = 1`, search cho Obs A trả về `[]`

---

#### TC-025 — Deserialize invalid JSON: trả về empty index (không crash)
**Type:** unit | 🔴 P0

**Given:** Input: `"not-valid-json"`, `"{}"`, `"null"`  
**When:** `SearchIndex.deserialize(input)` gọi  
**Then:** Trả về empty index với size=0, không throw

---

### Group J: Clear

#### TC-026 — Clear: emptied hoàn toàn
**Type:** unit | 🟡 P2

**Given:** Index có 5 docs  
**When:** `clear()`  
**Then:** `size = 0`, search bất kỳ term nào trả về `[]`

---

### Group K: Performance

#### TC-027 — 1000 docs: index time < 1000ms
**Requirement:** TR-004-BM25-017 | **Type:** performance | 🟠 P1

**Given:** 1000 CompressedObservations với diverse concepts  
**When:** Add tất cả vào index  
**Then:** Tổng thời gian < 1000ms

---

#### TC-028 — 1000 docs: search < 10ms per query
**Requirement:** TR-004-BM25-017 | **Type:** performance | 🟠 P1

**Given:** Index với 1000 docs  
**When:** 100 queries được chạy  
**Then:** p95 < 10ms (BM25-only, không embedding)

---

## 4. Coverage Notes

| Component | Critical branches |
|---|---|
| `add()` | Normal add, re-add sau remove |
| `remove()` | Existing ID, unknown ID, empty posting list cleanup |
| `search()` | Empty query, no match, exact/prefix/synonym, limit |
| `extractTerms()` | CJK path vs non-CJK path |
| `tokenize()` | Min length filter (< 2 chars), unicode chars |
| `serialize/deserialize` | Round-trip, invalid JSON, post-remove |
