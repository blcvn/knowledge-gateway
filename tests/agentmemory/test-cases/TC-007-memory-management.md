# TC-007: Memory Management — Test Cases

**Test Design tham chiếu:** [TD-007](../designs/TD-007-memory-management.md)  
**Requirements tham chiếu:** [TR-007](../requirements/TR-007-memory-management.md)  
**Module:** mem::remember, mem::forget, Jaccard similarity, versioning, TTL  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## NHÓM A: JACCARD SIMILARITY

---

## TC-007-001: Identical strings — similarity = 1

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-001 |
| **Tên** | Jaccard similarity của 2 string giống nhau = 1.0 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-004 |

**Dữ liệu đầu vào:**
- a = `"auth middleware jwt authentication tokens"`
- b = `"auth middleware jwt authentication tokens"` (giống hệt a)

**Các bước thực hiện:**
1. Gọi `jaccardSimilarity(a, b)`
2. Kiểm tra return value

**Kết quả mong đợi:** `result === 1`

---

## TC-007-002: Hoàn toàn khác nhau — similarity = 0

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-002 |
| **Tên** | Jaccard similarity của 2 string không có từ chung = 0 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:**
- a = `"auth middleware jwt authentication tokens"`
- b = `"database query optimization postgres connection"`

**Kết quả mong đợi:** `result === 0` (không có từ chung có độ dài ≥ 3)

---

## TC-007-003: Partial overlap — similarity giữa 0 và 1

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-003 |
| **Tên** | Jaccard similarity của 2 string có một số từ chung |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**
- a = `"auth jwt token authentication"`
- b = `"auth jwt session management"`

**Các bước thực hiện:**
1. Tính expected: words của a = {auth, jwt, token, authentication}, b = {auth, jwt, session, management}
2. Intersection = {auth, jwt} = 2 words
3. Union = {auth, jwt, token, authentication, session, management} = 6 words
4. Expected Jaccard = 2/6 ≈ 0.333
5. Gọi `jaccardSimilarity(a, b)` và so sánh

**Kết quả mong đợi:** `|result - 0.333| < 0.01`

---

## NHÓM B: SUPERSEDE LOGIC

---

## TC-007-004: Similarity > 0.7 → memory mới supersede memory cũ

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-004 |
| **Tên** | Remember với content tương tự > 70% tạo superseded memory |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-004 |

**Setup — Memory M1:**

| Trường | Giá trị |
|---|---|
| content | `"The auth system uses JWT tokens for authentication and session management with middleware"` |
| type | `fact` |
| project | `test-project` |

**Sau khi M1 được create, gọi remember với content tương tự (M2):**

| Trường | Giá trị |
|---|---|
| content | `"The auth system uses JWT tokens for authentication and session management with express middleware"` *(thêm "express")* |
| type | `fact` |
| project | `test-project` |

**Các bước thực hiện:**
1. Gọi `mem::remember` với M1 content → lưu `m1_id`
2. Gọi `mem::remember` với M2 content
3. Đọc M2 từ KV
4. Đọc M1 từ KV

**Kết quả mong đợi:**
- M2 có `parentId = m1_id`
- M2 có `supersedes = [m1_id]`
- M2 có `version = 2`
- M2 có `isLatest = true`
- M1 có `isLatest = false`

---

## TC-007-005: Similarity ≤ 0.7 → tạo memory độc lập

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-005 |
| **Tên** | Remember với content khác biệt tạo memory mới độc lập |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |

**Memory M1 content:**
```
The auth system uses JWT tokens for authentication session management
```

**Memory M2 content (hoàn toàn khác):**
```
Database optimization with connection pooling and query caching strategies
```

**Các bước thực hiện:**
1. Create M1
2. Create M2 (hoàn toàn khác topic)
3. Đọc M2 từ KV

**Kết quả mong đợi:**
- M2 có `parentId = undefined`
- M2 có `version = 1`
- M2 có `isLatest = true`
- M1 vẫn có `isLatest = true`

---

## TC-007-006: Versioning chain M1 → M2 → M3

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-006 |
| **Tên** | Version chain được duy trì đúng khi supersede nhiều lần |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-007-MEM-005 |

**Base content:**
```
The JWT authentication system uses RS256 algorithm for token signing
```

**M2 content (similarity > 0.7):**
```
The JWT authentication system uses RS256 algorithm for token signing and verification
```

**M3 content (similarity > 0.7 với M2):**
```
The JWT authentication system uses RS256 algorithm for token signing and verification with refresh
```

**Kết quả mong đợi sau M3:**
- M1: `isLatest = false`
- M2: `isLatest = false`
- M3: `isLatest = true`, `version = 3`
- Chỉ có 1 memory với `isLatest = true`

---

## NHÓM C: MEMORY STRUCTURE

---

## TC-007-007: Memory mới có đầy đủ required fields

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-007 |
| **Tên** | Newly created memory có tất cả required fields |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-001 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| content | `"Auth uses JWT with RS256 algorithm"` |
| type | `architecture` |
| concepts | `["jwt", "rs256", "auth"]` |
| files | `["src/auth.ts"]` |
| project | `test-project` |

**Kết quả mong đợi — tất cả fields phải tồn tại và đúng:**
- `id`: pattern `mem_<ts>_<hex>`
- `createdAt`: ISO timestamp hợp lệ
- `updatedAt`: ISO timestamp hợp lệ
- `type = "architecture"`
- `strength = 7` (hardcoded initial)
- `version = 1`
- `isLatest = true`
- `supersedes = []`
- `concepts = ["jwt", "rs256", "auth"]`
- `files = ["src/auth.ts"]`
- `title = content.slice(0, 80)` (tối đa 80 ký tự đầu)

---

## TC-007-008: Tất cả 6 valid memory types được chấp nhận

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-008 |
| **Tên** | Mỗi trong 6 valid memory types được lưu đúng |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-007-MEM-002 |

**Dữ liệu đầu vào (test từng row):**

| type input | Expected memory.type |
|---|---|
| `"pattern"` | `"pattern"` |
| `"preference"` | `"preference"` |
| `"architecture"` | `"architecture"` |
| `"bug"` | `"bug"` |
| `"workflow"` | `"workflow"` |
| `"fact"` | `"fact"` |

**Tiêu chí Pass:** Tất cả 6 types được lưu chính xác.

---

## TC-007-009: Unknown type bị coerced thành `"fact"`

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-009 |
| **Tên** | Type không hợp lệ được default sang fact |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-002 |

**Dữ liệu đầu vào:**
- `type = "invalid_type_xyz"`

**Kết quả mong đợi:** `memory.type = "fact"`

---

## NHÓM D: VALIDATION

---

## TC-007-010: Content rỗng bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-010 |
| **Tên** | mem::remember từ chối khi content rỗng |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-014 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `content` | `""` (empty string) |
| `type` | `fact` |

**Kết quả mong đợi:**
- `{success: false, error: "..."}` với error message đề cập đến content

---

## TC-007-011: Content chỉ whitespace bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-011 |
| **Tên** | mem::remember từ chối khi content chỉ có khoảng trắng |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |

**Dữ liệu đầu vào:** `content = "   \t\n   "`

**Kết quả mong đợi:** `{success: false}`

---

## NHÓM E: TTL

---

## TC-007-012: `ttlDays = 7` tạo `forgetAfter` đúng

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-012 |
| **Tên** | Memory với ttlDays=7 có forgetAfter = createdAt + 7 days |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-008 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `content` | `"Test TTL memory"` |
| `ttlDays` | `7` |

**Tính toán expected:**
- `createdAt` = T (thời điểm create)
- `forgetAfter` = T + 7 × 86400000 ms

**Kết quả mong đợi:**
- `memory.forgetAfter` tồn tại
- `|forgetAfter - (createdAt + 7 × 86400000)| < 1000` ms (tolerance 1 giây)

---

## TC-007-013: Không có `ttlDays` → `forgetAfter = undefined`

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-013 |
| **Tên** | Memory không có TTL có forgetAfter = undefined |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `content` | `"Permanent memory"` |
| `ttlDays` | *(không set)* |

**Kết quả mong đợi:** `memory.forgetAfter = undefined`

---

## NHÓM F: FORGET

---

## TC-007-014: `mem::forget` by memoryId xóa memory

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-014 |
| **Tên** | mem::forget xóa memory khỏi KV và BM25 index |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-015 |

**Điều kiện tiên quyết:**
- Memory `mem_target` tồn tại trong KV
- Memory đã được indexed trong BM25

**Các bước thực hiện:**
1. Verify `mem_target` tồn tại (search tìm được)
2. Gọi `mem::forget({memoryId: "mem_target"})`
3. Kiểm tra response
4. Kiểm tra KV
5. Search lại

**Kết quả mong đợi:**
- Response: `{success: true, deleted: 1}`
- KV không có `mem_target` trong `mem:memories`
- BM25 search không còn tìm thấy `mem_target`

---

## NHÓM G: PROJECT ISOLATION

---

## TC-007-015: Similarity > 0.7 nhưng khác project → KHÔNG supersede

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-015 |
| **Tên** | Cross-project memories không supersede nhau dù nội dung tương tự |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-007-MEM-006 |

**Memory M1:**

| Trường | Giá trị |
|---|---|
| content | `"JWT authentication system uses RS256 for token signing management"` |
| project | `project-A` |

**Memory M2 (similarity > 0.7 với M1):**

| Trường | Giá trị |
|---|---|
| content | `"JWT authentication system uses RS256 for token signing and management"` |
| project | `project-B` *(khác project!)* |

**Các bước thực hiện:**
1. Create M1 với project-A
2. Create M2 với project-B
3. Đọc M2 từ KV
4. Đọc M1 từ KV

**Kết quả mong đợi:**
- M2 có `parentId = undefined` (không supersede M1)
- M2 có `version = 1`
- M1 vẫn có `isLatest = true`
- M2 có `isLatest = true`
- Cả 2 memories đều active

---

## NHÓM H: AGENTID

---

## TC-007-016: agentId từ payload được lưu vào memory

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-016 |
| **Tên** | agentId được gán đúng vào memory khi có trong payload |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-007-MEM-017 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| content | `"Test memory"` |
| agentId | `cursor-agent-1` |

**Kết quả mong đợi:** `memory.agentId = "cursor-agent-1"`

---

## TC-007-017: agentId từ env var khi payload không có

| Trường | Giá trị |
|---|---|
| **ID** | TC-007-017 |
| **Tên** | agentId lấy từ AGENT_ID env var khi không có trong payload |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |

**Điều kiện tiên quyết:** `AGENT_ID=auto-cursor` trong env  

**Dữ liệu đầu vào:**
- Payload không có `agentId`

**Kết quả mong đợi:** `memory.agentId = "auto-cursor"`

---

## Tổng kết Module TC-007

| TC ID | Tên ngắn | Priority |
|---|---|---|
| TC-007-001 | Jaccard identical = 1 | 🔴 P0 |
| TC-007-002 | Jaccard disjoint = 0 | 🔴 P0 |
| TC-007-003 | Jaccard partial overlap | 🟠 P1 |
| TC-007-004 | Supersede sim > 0.7 | 🔴 P0 |
| TC-007-005 | No supersede sim ≤ 0.7 | 🔴 P0 |
| TC-007-006 | Version chain M1→M2→M3 | 🟠 P1 |
| TC-007-007 | Memory required fields | 🔴 P0 |
| TC-007-008 | 6 valid types | 🟠 P1 |
| TC-007-009 | Unknown type → fact | 🔴 P0 |
| TC-007-010 | Empty content rejected | 🔴 P0 |
| TC-007-011 | Whitespace content rejected | 🔴 P0 |
| TC-007-012 | TTL forgetAfter correct | 🔴 P0 |
| TC-007-013 | No TTL → undefined | 🟠 P1 |
| TC-007-014 | Forget by memoryId | 🔴 P0 |
| TC-007-015 | Project isolation no supersede | 🔴 P0 |
| TC-007-016 | agentId from payload | 🟠 P1 |
| TC-007-017 | agentId from env | 🟠 P1 |
