# TC-023: Export & Import — Test Cases

**Test Design tham chiếu:** [TD-023](../designs/TD-023-export-import.md)  
**Requirements tham chiếu:** [TR-023](../requirements/TR-023-export-import.md)  
**Module:** Export (JSON/JSONL), Import, Migration, Idempotency  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-023-001: Export toàn bộ KV state sang JSON hợp lệ

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-023-EXP-001 |

**Điều kiện tiên quyết:**

| Loại | Số lượng |
|---|---|
| Sessions | 3 |
| Observations | 30 (10/session) |
| Memories | 10 |
| Graph nodes | 5 |

**Các bước thực hiện:**
1. Gọi `mem::export({format: "json"})` hoặc `GET /export`
2. Parse JSON response/file
3. Kiểm tra structure và counts

**Kết quả mong đợi:**
- JSON hợp lệ (parseable bằng `JSON.parse`)
- `json.version` tồn tại (string semver)
- `json.exportedAt` là ISO timestamp
- `json.data.sessions.length = 3`
- `json.data.observations.length = 30`
- `json.data.memories.length = 10`
- `json.data.graph.nodes.length = 5`

---

## TC-023-002: Import từ valid export JSON → restore đầy đủ

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-023-EXP-002 |

**Điều kiện tiên quyết:**
- KV trống (clean state)
- File export JSON từ TC-023-001

**Các bước thực hiện:**
1. Gọi `mem::import({source: "export.json"})`
2. Kiểm tra response
3. Verify KV contents

**Kết quả mong đợi:**
- `{success: true, importedSessions: 3, importedObservations: 30, importedMemories: 10}`
- KV có đúng 3 sessions, 30 obs, 10 memories sau import

---

## TC-023-003: Import idempotent — gọi 2 lần không tạo duplicates

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-003 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-023-EXP-003 |

**Điều kiện tiên quyết:** Import lần 1 đã thành công (TC-023-002)

**Các bước thực hiện:**
1. Gọi `mem::import({source: "export.json"})` lần 2 (cùng file)
2. Kiểm tra response
3. Đếm records trong KV

**Kết quả mong đợi:**
- `{importedCount: 0, skippedCount: 43}` (3+30+10 = 43 records, tất cả đã tồn tại)
- KV count không tăng (vẫn là 3 sessions, 30 obs, 10 memories)

---

## TC-023-004: Import corrupt JSON → fail gracefully, không partial import

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-023-EXP-004 |

**Dữ liệu đầu vào:** Malformed JSON: `'{"sessions": [{"id": "se'` (truncated/corrupted)

**Điều kiện tiên quyết:** KV trống

**Các bước thực hiện:**
1. Gọi `mem::import({source: "corrupt.json"})`
2. Kiểm tra response
3. Đếm records trong KV

**Kết quả mong đợi:**
- `{success: false, error: "invalid JSON"}` hoặc tương đương
- KV vẫn trống (không có partial import)

---

## TC-023-005: Export không chứa expired memories

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-023-EXP-005 |

**Setup:**

| Memory | forgetAfter | Expected trong export |
|---|---|---|
| Memory A | `[hôm qua]` (expired) | ❌ Không có |
| Memory B | `[tuần sau]` (active) | ✅ Có |
| Memory C | *(không set)* | ✅ Có |

**Kết quả mong đợi:**
- Export JSON chứa Memory B và C
- Export JSON KHÔNG chứa Memory A

---

## TC-023-006: Import không overwrite data mới hơn

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-023-EXP-006 |

**Setup:**

| | Memory M1 trong KV (current) | Memory M1 trong import file |
|---|---|---|
| `updatedAt` | `2026-06-10T14:00:00Z` *(mới hơn)* | `2026-06-09T14:00:00Z` *(cũ hơn)* |
| `content` | `"New content"` | `"Old content"` |

**Các bước thực hiện:**
1. Import file chứa M1 cũ hơn
2. Đọc M1 từ KV

**Kết quả mong đợi:**
- M1 trong KV vẫn giữ `content = "New content"` và `updatedAt = 2026-06-10T14:00:00Z`
- Import M1 bị skip (not overwritten)

---

## TC-023-007: Migration backfill missing `isLatest` field

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-023-EXP-007 |

**Setup:** Seed KV với 5 memories ở format cũ (thiếu `isLatest` field, không có `supersedes` field)

**Các bước thực hiện:**
1. Seed 5 old-format memories vào KV
2. Gọi `mem::migrate`
3. Đọc tất cả 5 memories từ KV

**Kết quả mong đợi:**
- Tất cả 5 memories có `isLatest = true` (được backfill)
- `supersedes = []` (backfill với empty array)
- `version = 1` (backfill)
- Không có data loss (content, type, createdAt vẫn giữ nguyên)

---

## TC-023-008: Migration idempotent — chạy 2 lần an toàn

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-008 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-023-EXP-007 |

**Điều kiện tiên quyết:** Migration lần 1 đã chạy (TC-023-007)

**Các bước thực hiện:**
1. Gọi `mem::migrate` lần 2
2. Đọc tất cả memories

**Kết quả mong đợi:**
- Không có lỗi
- Data không bị corrupt
- Không có duplicate fields (e.g., `isLatest` không bị duplicate thành array)

---

## TC-023-009: Export/Import round-trip — tất cả fields preserved

| Trường | Giá trị |
|---|---|
| **ID** | TC-023-009 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-023-EXP-008 |

**Setup:** Memory M với đầy đủ fields:
- `content`, `type`, `concepts`, `files`, `version`, `isLatest`, `parentId`, `supersedes`, `strength`, `agentId`, `project`, `createdAt`, `updatedAt`

**Các bước thực hiện:**
1. Export → `export.json`
2. Clean KV
3. Import từ `export.json`
4. Đọc M từ KV
5. So sánh mỗi field với original

**Kết quả mong đợi:** Tất cả fields có giá trị giống hệt original (round-trip fidelity)

---

## Tổng kết TC-023

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-023-001 | Export → valid JSON | 🔴 P0 | Integration |
| TC-023-002 | Import → restore | 🔴 P0 | Integration |
| TC-023-003 | Import idempotent | 🟠 P1 | Integration |
| TC-023-004 | Corrupt JSON → fail graceful | 🔴 P0 | Integration |
| TC-023-005 | Export no expired | 🟠 P1 | Integration |
| TC-023-006 | Import no overwrite newer | 🟠 P1 | Integration |
| TC-023-007 | Migration backfill | 🟠 P1 | Integration |
| TC-023-008 | Migration idempotent | 🟠 P1 | Integration |
| TC-023-009 | Round-trip fidelity | 🔴 P0 | Integration |
