# TC-005: Vector Index — Test Cases

**Test Design tham chiếu:** [TD-005](../designs/TD-005-search-vector.md)  
**Requirements tham chiếu:** [TR-005](../requirements/TR-005-search-vector.md)  
**Module:** VectorIndex (cosine similarity, K-NN search, serialization)  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## Quy ước Vector Test

Các test dùng các vectors toán học đơn giản với tính chất rõ ràng:

| Vector | Ký hiệu | Giá trị (4-dim) |
|---|---|---|
| Unit vector 1 | e₁ | `[1, 0, 0, 0]` |
| Unit vector 2 | e₂ | `[0, 1, 0, 0]` |
| Negative e₁ | -e₁ | `[-1, 0, 0, 0]` |
| Zero vector | **0** | `[0, 0, 0, 0]` |
| Similar to e₁ | v_similar | `[0.99, 0.1, 0.05, 0.02]` (normalized) |

---

## TC-005-001: Index trống — size=0, search trả về []

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-001 |
| **Tên** | VectorIndex mới có size=0 và search rỗng |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Các bước thực hiện:**
1. Tạo VectorIndex mới
2. Đọc `index.size`
3. Gọi `index.search(e₁, 10)`

**Kết quả mong đợi:**
- `size === 0`
- search trả về `[]`

---

## TC-005-002: Identical vectors: cosine similarity ≈ 1.0

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-002 |
| **Tên** | Cùng một vector với chính nó có similarity = 1.0 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-005-VEC-001 |

**Dữ liệu đầu vào:**
- Vector v = `[0.5774, 0.5774, 0.5774, 0]` (normalized, độ dài ≈ 1.0)
- Add với obsId = `"obs_same"`, sessionId = `"sess_1"`

**Các bước thực hiện:**
1. Add vector v vào index
2. `search(v, 1)` với cùng vector v làm query
3. Kiểm tra `results[0].score`

**Kết quả mong đợi:**
- `|results[0].score - 1.0| < 1e-5` (tolerance nhỏ cho floating point)
- `results[0].obsId = "obs_same"`

---

## TC-005-003: Orthogonal vectors: cosine similarity = 0

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-003 |
| **Tên** | Hai vectors vuông góc có cosine similarity = 0 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- Obs A với vector e₁ = `[1, 0, 0, 0]`
- Query vector: e₂ = `[0, 1, 0, 0]`

**Các bước thực hiện:**
1. Add obs A (vector e₁)
2. `search(e₂, 1)` (orthogonal query)
3. Kiểm tra score

**Kết quả mong đợi:** `|results[0].score - 0.0| < 1e-5`

---

## TC-005-004: More similar vector ranked higher

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-004 |
| **Tên** | Vector gần query hơn được rank cao hơn |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-005-VEC-004 |

**Dữ liệu đầu vào:**

| obsId | Vector | Mô tả |
|---|---|---|
| `obs_near` | `[0.99, 0.1, 0.05, 0.05]` (normalized) | Gần query [1,0,0,0] |
| `obs_far` | `[0.1, 0.99, 0.05, 0.05]` (normalized) | Xa query [1,0,0,0] |

- Query: `[1, 0, 0, 0]`

**Các bước thực hiện:**
1. Add `obs_near` và `obs_far`
2. `search([1, 0, 0, 0], 2)`
3. So sánh rank và scores

**Kết quả mong đợi:**
- `results[0].obsId = "obs_near"`
- `results[0].score > results[1].score`

---

## TC-005-005: Zero vector query: score = 0, không crash

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-005 |
| **Tên** | Query với zero vector không crash (division by zero safe) |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**
- Index có obs với unit vector
- Query: `[0, 0, 0, 0]` (zero vector)

**Kết quả mong đợi:**
- `search([0,0,0,0], 1)` không throw exception
- `results[0].score = 0` (safe division)

---

## TC-005-006: Dimension mismatch: score = 0, không crash

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-006 |
| **Tên** | Query với sai số chiều trả về score=0 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-005-VEC-006 |

**Dữ liệu đầu vào:**
- Index có obs với 4-dim vector
- Query: 8-dim vector `[1, 0, 0, 0, 0, 0, 0, 0]`

**Kết quả mong đợi:**
- Không throw exception
- `results[0].score = 0` (dim mismatch → early return 0)

---

## TC-005-007: Limit parameter giới hạn kết quả

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-007 |
| **Tên** | search(query, limit) trả về đúng số lượng kết quả |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-005-VEC-002 |

**Dữ liệu đầu vào:**
- 20 vectors (tất cả unique)
- Query gần với tất cả

**Kết quả mong đợi:**
- `search(query, 5).length = 5`
- `search(query, 10).length = 10`

---

## TC-005-008: Results sorted by score descending

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-008 |
| **Tên** | Search results được sắp xếp theo score từ cao đến thấp |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-005-VEC-004 |

**Dữ liệu đầu vào:**
- 5 vectors với các độ tương đồng khác nhau với query

**Các bước thực hiện:**
1. Search với `limit = 5`
2. Kiểm tra thứ tự scores

**Kết quả mong đợi:** Với mỗi `i`: `results[i].score >= results[i+1].score`

---

## TC-005-009: Result trả về đúng sessionId

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-009 |
| **Tên** | Search result chứa đúng sessionId của observation |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- Vector được add với `obsId = "obs_1"`, `sessionId = "sess_xyz_789"`

**Kết quả mong đợi:**
- Search tìm được `obs_1`
- `result.sessionId = "sess_xyz_789"`

---

## TC-005-010: Remove: size giảm, doc biến mất

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-010 |
| **Tên** | Sau remove, vector không còn trong search results |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**
- 2 vectors: `obs_remove` và `obs_keep`

**Các bước thực hiện:**
1. Add cả 2 vectors, verify size = 2
2. `index.remove("obs_remove")`
3. Kiểm tra `index.size`
4. Search với vector của `obs_remove`

**Kết quả mong đợi:**
- `index.size = 1`
- `obs_remove` không xuất hiện trong search results

---

## TC-005-011: validateDimensions — tất cả cùng dim → mismatches = []

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-011 |
| **Tên** | validateDimensions báo cáo không có mismatch khi tất cả cùng dim |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-005-VEC-008 |

**Dữ liệu đầu vào:**
- 3 vectors, tất cả 4-dim (cùng dimension)
- Expected dim = 4

**Kết quả mong đợi:**
- `result.mismatches.length = 0`
- `result.seenDimensions = {4}`

---

## TC-005-012: validateDimensions — mixed dims: mismatch được detect

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-012 |
| **Tên** | validateDimensions phát hiện vector có dimension sai |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- `obs_correct`: 4-dim vector
- `obs_wrong`: 8-dim vector

**Kết quả mong đợi:**
- `result.mismatches = [{obsId: "obs_wrong", dim: 8}]`
- `result.seenDimensions` chứa cả `4` và `8`

---

## TC-005-013: Buffer pool slice regression (byteOffset preservation)

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-013 |
| **Tên** | Serialization bảo toàn byteOffset khi vector là slice của shared Buffer |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-005-VEC-010 |

**Mô tả lỗi (bug #455):**
Khi `Float32Array` được tạo từ slice của Buffer pool (shared ArrayBuffer), method `Buffer.from(arr.buffer)` sẽ serialize toàn bộ pool thay vì chỉ phần của arr. Sau khi deserialize, vector có độ dài sai.

**Dữ liệu đầu vào:**
- SharedBuffer = 32 floats
- Slice: bytes 64-79 (4 floats = vector `[1, 2, 3, 4]`)
- `byteOffset = 64`, `byteLength = 16`

**Các bước thực hiện:**
1. Tạo Float32Array từ slice của shared buffer
2. Add vào index
3. Serialize và deserialize
4. Kiểm tra độ dài của restored vector
5. Kiểm tra cosine similarity của original vs restored

**Kết quả mong đợi:**
- Restored vector có đúng 4 elements (không phải 32)
- Cosine similarity của original và restored ≈ 1.0

---

## TC-005-014: Serialize/deserialize round-trip

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-014 |
| **Tên** | Serialized VectorIndex khôi phục đầy đủ state |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-005-VEC-009 |

**Dữ liệu đầu vào:**
- 2 vectors với obsIds `"obs_A"` và `"obs_B"`

**Các bước thực hiện:**
1. Add 2 vectors
2. `const json = index.serialize()`
3. `const restored = VectorIndex.deserialize(json)`
4. Kiểm tra `restored.size = 2`
5. `restored.search(queryA, 1)` → tìm obs_A
6. `restored.search(queryB, 1)` → tìm obs_B

**Tiêu chí Pass:** Cả 2 docs tìm được từ restored index.

---

## TC-005-015: Deserialize invalid JSON → empty index, không crash

| Trường | Giá trị |
|---|---|
| **ID** | TC-005-015 |
| **Tên** | Deserialize với input không hợp lệ trả về empty index |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào (3 cases):**
- `"not-json"`, `"null"`, `"{}"`

**Kết quả mong đợi:**
- Không throw exception
- Trả về empty VectorIndex với `size = 0`

---

## Tổng kết Module TC-005

| TC ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-005-001 | Empty index | 🔴 P0 | Unit |
| TC-005-002 | Identical vectors sim=1 | 🔴 P0 | Unit |
| TC-005-003 | Orthogonal sim=0 | 🟠 P1 | Unit |
| TC-005-004 | Similar ranked higher | 🔴 P0 | Unit |
| TC-005-005 | Zero vector safe | 🔴 P0 | Unit |
| TC-005-006 | Dim mismatch safe | 🔴 P0 | Unit |
| TC-005-007 | Limit parameter | 🔴 P0 | Unit |
| TC-005-008 | Results sorted | 🔴 P0 | Unit |
| TC-005-009 | sessionId in result | 🟠 P1 | Unit |
| TC-005-010 | Remove doc | 🔴 P0 | Unit |
| TC-005-011 | validateDimensions OK | 🟠 P1 | Unit |
| TC-005-012 | validateDimensions mismatch | 🟠 P1 | Unit |
| TC-005-013 | Buffer pool regression | 🔴 P0 | Unit |
| TC-005-014 | Serialize/deserialize | 🔴 P0 | Unit |
| TC-005-015 | Deserialize invalid | 🔴 P0 | Unit |
