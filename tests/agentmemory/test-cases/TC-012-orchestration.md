# TC-012: Orchestration — Test Cases

**Test Design tham chiếu:** [TD-012](../designs/TD-012-orchestration.md)  
**Requirements tham chiếu:** [TR-012](../requirements/TR-012-orchestration.md)  
**Module:** Actions, State Transitions, Routines, Sketches  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TC-012-001: Action được tạo với status "pending"

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-001 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-012-ORC-001 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `title` | `Refactor auth module` |
| `description` | `Improve JWT validation logic` |
| `steps` | `["Analyze current code", "Write tests", "Implement changes"]` |
| `sessionId` | `sess_orch` |

**Các bước thực hiện:**
1. Gọi `action-create({title, description, steps, sessionId})`
2. Kiểm tra response
3. Đọc action từ KV `mem:actions`

**Kết quả mong đợi:**
- `response.success = true`
- `action.status = "pending"`
- `action.steps.length = 3`
- `action.createdAt` là ISO timestamp hợp lệ
- `action.id` là string unique

---

## TC-012-002: State transition pending → in-progress → completed

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-002 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-012-ORC-002 |

**Điều kiện tiên quyết:** Action `action_1` ở `status = "pending"`

**Các bước thực hiện:**

| Bước | Action | Expected status |
|---|---|---|
| 1 | `action-update({id: "action_1", status: "in-progress"})` | `in-progress` |
| 2 | Đọc action từ KV | `status = "in-progress"` |
| 3 | `action-update({id: "action_1", status: "completed"})` | `completed` |
| 4 | Đọc action từ KV | `status = "completed"` |

**Kết quả mong đợi:** Mỗi transition được accept, `updatedAt` tăng sau mỗi update

---

## TC-012-003: Invalid state transition bị từ chối

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-003 |
| **Loại** | Unit |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-012-ORC-003 |

**Setup:** Action ở `status = "completed"`

**Transitions bị từ chối (test từng case):**

| Từ | Sang | Expected |
|---|---|---|
| `completed` | `pending` | ❌ Rejected |
| `completed` | `in-progress` | ❌ Rejected |

**Kết quả mong đợi:**
- `{success: false, error: "...invalid transition..."}` (hoặc tương đương)
- Status của action không thay đổi

---

## TC-012-004: Action update: đánh dấu step hoàn thành

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-004 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-012-ORC-004 |

**Setup:** Action có 3 steps chưa done

**Dữ liệu đầu vào:**
- `action-update({id: "action_1", completedStep: 0})` — đánh dấu step[0] done

**Kết quả mong đợi:**
- `action.steps[0].done = true`
- `action.steps[1].done = false`
- `action.steps[2].done = false`

---

## TC-012-005: Sketch write → read

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-005 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-012-ORC-007 |

**Dữ liệu đầu vào:** `sketch-write({sessionId: "sess_orch", content: "Initial note"})`

**Các bước thực hiện:**
1. Write `"Initial note"`
2. `sketch-read({sessionId: "sess_orch"})`
3. Kiểm tra content

**Kết quả mong đợi:** `content = "Initial note"`

---

## TC-012-006: Sketch append → content được nối

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-006 |
| **Loại** | Integration |
| **Ưu tiên** | 🔴 P0 |
| **Requirement** | TR-012-ORC-008 |

**Các bước thực hiện:**

| Bước | Action | Expected content |
|---|---|---|
| 1 | `sketch-write("Initial note")` | `"Initial note"` |
| 2 | `sketch-append("Additional note")` | `"Initial note\nAdditional note"` |

**Kết quả mong đợi:** Content sau append = `"Initial note\nAdditional note"`

---

## TC-012-007: Sketch clear → content rỗng

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-007 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-012-ORC-009 |

**Điều kiện tiên quyết:** Sketch có content `"Some notes"`

**Các bước thực hiện:**
1. `sketch-clear({sessionId: "sess_orch"})`
2. `sketch-read({sessionId: "sess_orch"})`

**Kết quả mong đợi:** `content = ""`

---

## TC-012-008: Routine được tạo với schedule và enabled=true

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-008 |
| **Loại** | Integration |
| **Ưu tiên** | 🟠 P1 |
| **Requirement** | TR-012-ORC-010 |

**Dữ liệu đầu vào:**

| Trường | Giá trị |
|---|---|
| `name` | `daily-digest` |
| `schedule` | `0 9 * * *` (mỗi ngày 9AM) |
| `action` | `{ type: "consolidate" }` |

**Kết quả mong đợi:**
- `routine.schedule = "0 9 * * *"`
- `routine.enabled = true`
- `routine.name = "daily-digest"`
- Routine lưu trong KV `mem:routines`

---

## TC-012-009: Routine disable → skip execution

| Trường | Giá trị |
|---|---|
| **ID** | TC-012-009 |
| **Loại** | Integration |
| **Ưu tiên** | 🟡 P2 |

**Các bước thực hiện:**
1. Create routine với `enabled = true`
2. Update `{id, enabled: false}`
3. Trigger schedule manually
4. Kiểm tra execution count

**Kết quả mong đợi:** Routine bị skip, không execute khi `enabled = false`

---

## Tổng kết TC-012

| ID | Tên ngắn | Priority | Loại |
|---|---|---|---|
| TC-012-001 | Action tạo với pending | 🔴 P0 | Integration |
| TC-012-002 | Pending → in-progress → completed | 🔴 P0 | Integration |
| TC-012-003 | Invalid transition rejected | 🟠 P1 | Unit |
| TC-012-004 | Step completion marking | 🟠 P1 | Integration |
| TC-012-005 | Sketch write/read | 🔴 P0 | Integration |
| TC-012-006 | Sketch append | 🔴 P0 | Integration |
| TC-012-007 | Sketch clear | 🟠 P1 | Integration |
| TC-012-008 | Routine creation | 🟠 P1 | Integration |
| TC-012-009 | Routine disable | 🟡 P2 | Integration |
