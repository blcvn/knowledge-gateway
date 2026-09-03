# TC-017: Memory Slots — Test Cases

**Test Design tham chiếu:** [TD-017](../designs/TD-017-memory-slots.md)  
**Requirements tham chiếu:** [TR-017](../requirements/TR-017-memory-slots.md)  
**Module:** Slot CRUD, Scope Isolation, Validation  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-017-001: Slot write và read thành công

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-017-SLT-001 |

**Dữ liệu đầu vào (Write):**

| Trường | Giá trị |
|---|---|
| `name` | `current-task` |
| `content` | `Implementing auth middleware` |
| `sessionId` | `sess_slot` |

**Các bước thực hiện:**
1. Gọi `mem::slot-write({name: "current-task", content: "Implementing auth middleware", sessionId: "sess_slot"})`
2. Kiểm tra response write
3. Gọi `mem::slot-read({name: "current-task", sessionId: "sess_slot"})`
4. Kiểm tra response read

**Kết quả mong đợi (Write):**
- `{success: true}`

**Kết quả mong đợi (Read):**
- `{success: true, slot: {name: "current-task", content: "Implementing auth middleware", createdAt: "...", updatedAt: "..."}}`

---

## TC-017-002: Slot overwrite khi cùng name — content được update

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-017-SLT-002 |

**Các bước thực hiện:**

| Bước | Action | Expected |
|---|---|---|
| 1 | `slot-write({name: "goal", content: "Old content"})` | `success: true` |
| 2 | Đọc `createdAt` |  |
| 3 | `slot-write({name: "goal", content: "New content"})` | `success: true` |
| 4 | `slot-read({name: "goal"})` | content = `"New content"` |

**Kết quả mong đợi (bước 4):**
- Chỉ có 1 slot với name `"goal"` (không tạo duplicate)
- `slot.content = "New content"`
- `slot.updatedAt > slot.createdAt` (updatedAt mới hơn)

---

## TC-017-003: Read slot không tồn tại → null hoặc not found

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-003 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-017-SLT-001 |

**Dữ liệu đầu vào:** `slot-read({name: "nonexistent-slot"})`

**Kết quả mong đợi:** Một trong hai:
- `{success: true, slot: null}`
- `{success: false, error: "slot not found"}`

Không được throw exception.

---

## TC-017-004: Delete slot → slot không còn accessible

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-017-SLT-003 |

**Điều kiện tiên quyết:** Slot `"current-task"` tồn tại

**Các bước thực hiện:**
1. `slot-delete({name: "current-task"})`
2. Kiểm tra response
3. `slot-read({name: "current-task"})`

**Kết quả mong đợi (bước 1-2):** `{success: true, deleted: 1}`

**Kết quả mong đợi (bước 3):** `slot: null` hoặc `error: "not found"`

---

## TC-017-005: slot-list trả về tất cả slots của session

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-017-SLT-003 |

**Setup:** 3 slots trong session `sess_slot`: `"goal"`, `"current-task"`, `"notes"`

**Kết quả mong đợi:** `slot-list({sessionId: "sess_slot"})` → array với 3 items

---

## TC-017-006: Session slot isolation — không thấy slot của session khác

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-017-SLT-004 |

**Setup:**
- Slot A (`name = "slot-a"`) trong session-1
- Slot B (`name = "slot-b"`) trong session-2

**Các bước thực hiện:**
1. `slot-list({sessionId: "session-1"})`
2. Kiểm tra kết quả

**Kết quả mong đợi:**
- Chỉ Slot A xuất hiện (`"slot-a"`)
- Slot B (`"slot-b"`) KHÔNG xuất hiện

---

## TC-017-007: Slot name validation — invalid characters

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-007 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-017-SLT-006 |

**Dữ liệu đầu vào (test từng case):**

| name | Lý do invalid | Expected |
|---|---|---|
| `"my slot!"` | Có space và `!` | `{success: false}` |
| `"../escape"` | Path traversal | `{success: false}` |
| `""` | Rỗng | `{success: false}` |
| `"valid-name_123"` | Valid | `{success: true}` |

**Tiêu chí Pass:** Invalid names đều bị từ chối với `success: false`

---

## TC-017-008: Empty content bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-008 |
| **Loại** | Unit |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-017-SLT-007 |

**Dữ liệu đầu vào:**

| name | `"valid-slot"` |
|---|---|
| content | `""` (empty string) |

**Kết quả mong đợi:** `{success: false, error: "content is required"}`

---

## TC-017-009: Slots được bao gồm trong recall khi includeSlots=true

| Trường | Giá trị |
|---|---|
| **ID** | TC-017-009 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-017-SLT-005 |

**Setup:** Slot `"goal"` với content `"Build JWT auth system"`

**Dữ liệu đầu vào:** `mem::recall({sessionId: "sess_slot", includeSlots: true})`

**Kết quả mong đợi:**
- `response.slots` tồn tại và không rỗng
- `response.slots` chứa slot `"goal"` với content `"Build JWT auth system"`

---

## Tổng kết TC-017

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-017-001 | Slot write & read | 🔴 P0 | Integration |
| TC-017-002 | Slot overwrite | 🔴 P0 | Integration |
| TC-017-003 | Read nonexistent | 🔴 P0 | Unit |
| TC-017-004 | Delete slot | 🟠 P1 | Integration |
| TC-017-005 | List slots | 🟠 P1 | Integration |
| TC-017-006 | Session isolation | 🔴 P0 | Integration |
| TC-017-007 | Name validation | 🟠 P1 | Unit |
| TC-017-008 | Empty content rejected | 🔴 P0 | Unit |
| TC-017-009 | Recall includeSlots | 🟠 P1 | Integration |
