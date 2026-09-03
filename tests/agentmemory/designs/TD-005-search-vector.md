# TD-005: Vector Index Test Design

**Liên kết Requirements:** [TR-005-search-vector.md](../requirements/TR-005-search-vector.md)  
**Source:** `references/agentmemory/src/state/vector-index.ts`  
**Test file:** `tests/agentmemory/specs/vector-index.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

`VectorIndex` lưu trữ embedding vectors và thực hiện K-NN search bằng cosine similarity.

**Implementation details quan trọng:**
- Cosine similarity = dot product / (|A| × |B|), trả về 0 nếu mẫu số = 0
- Dim mismatch: `a.length !== b.length` → trả về 0 (không crash)
- Serialization: Base64 với explicit `byteOffset + byteLength` (tránh Buffer pool bug #455)
- `validateDimensions(expected)` → kiểm tra consistency trước khi restore từ disk

---

## 2. Chiến lược kiểm thử

| Khía cạnh | Kỹ thuật |
|---|---|
| Cosine similarity | Mathematical property testing (orthogonal, identical, opposite) |
| Ranking | Comparative: A more similar to Q → A ranked higher than B |
| Buffer pool regression | Test với sliced Float32Array từ shared buffer |
| Serialization | Round-trip identity test |
| Dimension validation | Injection của vectors có dimension sai |

---

## 3. Helper Test Vectors

Để test, dùng các vectors đơn giản có tính chất toán học rõ ràng:
- **Unit vector e₁ = [1,0,0,0]**: orthogonal với e₂
- **Unit vector e₂ = [0,1,0,0]**: orthogonal với e₁
- **Identical vectors**: cosine = 1.0
- **Opposite vectors** `v` và `-v`: cosine = -1.0
- **Zero vector**: cosine = 0 (edge case)

---

## 4. Test Cases

### Group A: CRUD cơ bản

#### TC-001 — Index trống: size=0, search trả về []
**Type:** unit | 🔴 P0

**Given:** `VectorIndex` mới tạo  
**When:** `size` đọc và `search(queryVec, 10)` gọi  
**Then:** `size = 0`, search trả về mảng rỗng

---

#### TC-002 — Add: size tăng sau mỗi lần add
**Type:** unit | 🔴 P0

**Given:** Index trống  
**When:** Add lần lượt 3 vectors  
**Then:** `size = 1`, `size = 2`, `size = 3` sau mỗi add

---

#### TC-003 — Remove: size giảm, doc biến mất
**Type:** unit | 🔴 P0

**Given:** 2 vectors đã được add  
**When:** `remove(obsId)` cho obs đầu tiên  
**Then:** `size = 1`, search với vector của obs 1 không còn tìm thấy nó

---

#### TC-004 — Remove ID không tồn tại: no-op, không crash
**Type:** unit | 🔴 P0

**Given:** 1 vector trong index  
**When:** `remove("unknown_id")` gọi  
**Then:** Không throw, `size` không đổi

---

### Group B: Cosine Similarity Properties

#### TC-005 — Identical vectors: score ≈ 1.0
**Requirement:** TR-005-VEC-001 | **Type:** unit | 🔴 P0

**Given:** Vector `v` được add với id "obs1"  
**When:** `search(v, 1)` với cùng vector `v` làm query  
**Then:** `results[0].score ≈ 1.0` (tolerance 1e-5)

---

#### TC-006 — Orthogonal vectors: score = 0
**Type:** unit | 🟠 P1

**Given:**
- Obs A: vector e₁ = `[1, 0, 0, 0]`
- Query: vector e₂ = `[0, 1, 0, 0]`

**When:** `search(e₂, 1)`  
**Then:** `results[0].score ≈ 0`

---

#### TC-007 — More similar vector ranked higher
**Requirement:** TR-005-VEC-004 | **Type:** unit | 🔴 P0

**Given:**
- Obs A: vector gần giống query (cosine ≈ 0.98)
- Obs B: vector xa query (cosine ≈ 0.1)

**When:** `search(query, 2)`  
**Then:** `results[0].obsId = "obs_A"`, `results[0].score > results[1].score`

---

#### TC-008 — Zero vector query: score = 0, không crash (division by zero safe)
**Type:** unit | 🔴 P0

**Given:** Obs với normal unit vector  
**When:** `search(zero_vector, 1)` với `[0,0,...,0]` làm query  
**Then:** `results[0].score = 0`, không throw

---

#### TC-009 — Dimension mismatch: score = 0, không crash
**Requirement:** TR-005-VEC-006 | **Type:** unit | 🔴 P0

**Given:** Obs với 384-dim vector đã được add  
**When:** `search(768_dim_query, 1)` với query 768 dimensions  
**Then:** `results[0].score = 0` (early return: `a.length !== b.length`)

---

### Group C: Search và Limit

#### TC-010 — Limit parameter giới hạn kết quả
**Requirement:** TR-005-VEC-002 | **Type:** unit | 🔴 P0

**Given:** 20 vectors trong index  
**When:** `search(query, 5)`, `search(query, 10)`  
**Then:** Lần lượt trả về 5 và 10 kết quả

---

#### TC-011 — Limit > total docs: trả về tất cả
**Type:** unit | 🟡 P2

**Given:** 8 vectors trong index  
**When:** `search(query, 100)`  
**Then:** 8 kết quả (không padding)

---

#### TC-012 — Results sorted by score descending
**Requirement:** TR-005-VEC-004 | **Type:** unit | 🔴 P0

**Given:** 5 vectors với similarities khác nhau  
**When:** `search(query, 5)`  
**Then:** `results[i].score >= results[i+1].score` với mọi i

---

#### TC-013 — Result trả về đúng sessionId
**Type:** unit | 🟠 P1

**Given:** Vector được add với `obsId="obs1"`, `sessionId="sess_xyz"`  
**When:** Search tìm thấy "obs1"  
**Then:** `result.sessionId = "sess_xyz"`

---

### Group D: Dimension Validation

#### TC-014 — Tất cả cùng dim: `mismatches = []`, `seenDimensions = {384}`
**Requirement:** TR-005-VEC-008 | **Type:** unit | 🟠 P1

**Given:** 3 vectors, tất cả 384 dims  
**When:** `validateDimensions(384)`  
**Then:** `mismatches.length = 0`, `seenDimensions = {384}`

---

#### TC-015 — Mixed dims: mismatch được detect
**Type:** unit | 🟠 P1

**Given:**
- Obs A: 384-dim vector (correct)
- Obs B: 768-dim vector (wrong)

**When:** `validateDimensions(384)`  
**Then:**
- `mismatches = [{obsId: "obs_B", dim: 768}]`
- `seenDimensions = {384, 768}`

---

#### TC-016 — Empty index: `mismatches = []`, `seenDimensions = {}`
**Type:** unit | 🟡 P2

**Given:** Index trống  
**When:** `validateDimensions(384)`  
**Then:** `mismatches = []`, `seenDimensions.size = 0`

---

### Group E: Serialization

#### TC-017 — Serialize/deserialize round-trip: vectors preserved
**Requirement:** TR-005-VEC-009 | **Type:** unit | 🔴 P0

**Given:** 2 vectors được add  
**When:** `serialize()` → `VectorIndex.deserialize(json)`  
**Then:**
- `restored.size = 2`
- Search với original query vectors trả về same ranking
- Scores xấp xỉ nhau (tolerance 1e-3)

---

#### TC-018 — Buffer pool slice regression: byteOffset preserved
**Requirement:** TR-005-VEC-010 | **Type:** unit | 🔴 P0

**Given:** Float32Array được tạo từ slice của shared Buffer (giả lập Buffer pool)  
**When:** Vector được serialize rồi deserialize  
**Then:**
- Restored vector có đúng 384 elements (không phải 2048 — full pool size)
- Cosine similarity của original vs restored ≈ 1.0

**Lý do:** Bug #455/#469/#584 — Buffer.from(arr.buffer) khi arr là slice của pool sẽ dùng full pool size. Fix: truyền `byteOffset + byteLength` explicitly.

---

#### TC-019 — Deserialize invalid JSON: trả về empty index
**Type:** unit | 🔴 P0

**Given:** Inputs: `"not-json"`, `"null"`, `"{}"`  
**When:** `VectorIndex.deserialize(input)` gọi  
**Then:** Trả về empty index, không throw

---

#### TC-020 — Deserialize bỏ qua malformed rows (partial corruption)
**Type:** unit | 🟠 P1

**Given:** JSON array có 3 rows, 1 row thiếu `embedding` field  
**When:** `deserialize()` gọi  
**Then:** 2 rows hợp lệ được load, row lỗi bị skip (không throw)

---

### Group F: restoreFrom

#### TC-021 — `restoreFrom` deep-copies vectors
**Type:** unit | 🟠 P1

**Given:** Source index với 2 vectors  
**When:** `dest.restoreFrom(src)`  
**Then:**
- `dest.size = 2`
- Xóa vector từ `src` sau đó không ảnh hưởng `dest` (deep copy)

---

### Group G: Clear

#### TC-022 — `clear()` emptied hoàn toàn
**Type:** unit | 🟡 P2

**Given:** Index với 5 vectors  
**When:** `clear()`  
**Then:** `size = 0`, `search` trả về `[]`

---

## 5. Coverage Notes

| Function | Critical branches |
|---|---|
| `cosineSimilarity` | Normal, zero denom (zero vec), dim mismatch |
| `search` | Empty index, limit > size, correct sorting |
| `validateDimensions` | All match, mixed, empty |
| `serialize` / `deserialize` | Round-trip, buffer pool fix, invalid JSON, malformed rows |
| `restoreFrom` | Deep copy verification |
