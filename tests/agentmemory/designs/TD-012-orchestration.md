# TD-012: Orchestration Test Design

**Liên kết Requirements:** [TR-012-orchestration.md](../requirements/TR-012-orchestration.md)  
**Source:** `references/agentmemory/src/functions/actions.ts`, `routines.ts`, `sketches.ts`  
**Test file:** `tests/agentmemory/specs/orchestration.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Orchestration bao gồm:
- **Actions:** Structured steps với status tracking
- **Routines:** Scheduled/recurring workflows
- **Sketches:** Working memory / scratchpad

---

## 2. Test Cases

### Group A: Actions

#### TC-001 — Action được tạo với status `pending`
**Requirement:** TR-012-ORC-001 | **Type:** integration | 🔴 P0

**Given:** `mem::action-create` được gọi với title, steps  
**When:** Action created  
**Then:**
- `action.status = "pending"`
- `action.steps[]` có đủ bước
- `action.createdAt` hợp lệ
- Action được ghi vào `mem:actions`

---

#### TC-002 — State transition: pending → in-progress → completed
**Requirement:** TR-012-ORC-002 | **Type:** integration | 🔴 P0

**Given:** Action ở trạng thái pending  
**When:**
1. `mem::action-update({id, status: "in-progress"})` → action.status = "in-progress"
2. `mem::action-update({id, status: "completed"})` → action.status = "completed"

**Then:** Mỗi state transition hợp lệ được accept

---

#### TC-003 — Invalid state transition bị từ chối
**Type:** unit | 🟠 P1

**Given:** Action ở `completed`  
**When:** Transition sang `pending` (backward)  
**Then:** `{success: false, error: "invalid transition"}`

---

#### TC-004 — Action edges: parent-child relationship
**Requirement:** TR-012-ORC-003 | **Type:** integration | 🟠 P1

**Given:** Action Parent với sub-actions Child1, Child2  
**When:** Children được linked  
**Then:**
- `mem:action-edges` có entries: Parent→Child1, Parent→Child2
- `children[]` trên Parent listing

---

### Group B: Routines

#### TC-005 — Routine được tạo với schedule
**Requirement:** TR-012-ORC-005 | **Type:** integration | 🟠 P1

**Given:** `mem::routine-create` với `schedule: "0 */6 * * *"` (mỗi 6 giờ)  
**When:** Routine created  
**Then:**
- Routine lưu trong `mem:routines`
- `routine.schedule = "0 */6 * * *"`
- `routine.enabled = true`

---

#### TC-006 — Routine run được record
**Type:** integration | 🟠 P1

**Given:** Routine được executed  
**When:** Execution hoàn thành  
**Then:**
- Run record được tạo trong `mem:routine-runs`
- Record có `routineId`, `startedAt`, `completedAt`, `status`

---

#### TC-007 — Disable routine: skip execution
**Type:** integration | 🟡 P2

**Given:** Routine enabled=true  
**When:** `mem::routine-update({id, enabled: false})`  
**Then:** `routine.enabled = false`, routine không được trigger nữa

---

### Group C: Sketches (Working Memory)

#### TC-008 — Sketch được tạo và retrieve
**Requirement:** TR-012-ORC-009 | **Type:** integration | 🔴 P0

**Given:** `mem::sketch-write({content: "Working on auth refactor..."})` gọi  
**When:** Sketch được saved  
**Then:** Sketch retrieve được từ `mem:sketches` với cùng content

---

#### TC-009 — Sketch append: nội dung mới được thêm vào cuối
**Type:** integration | 🟠 P1

**Given:** Sketch với content "Initial note"  
**When:** `mem::sketch-append({content: "Additional note"})` gọi  
**Then:** Sketch content = "Initial note\nAdditional note"

---

#### TC-010 — Sketch clear: xóa toàn bộ content
**Type:** integration | 🟠 P1

**Given:** Sketch với content  
**When:** `mem::sketch-clear` gọi  
**Then:** Sketch content = `""`

---

#### TC-011 — Sketch có TTL theo session
**Requirement:** TR-012-ORC-010 | **Type:** integration | 🟡 P2

**Given:** Sketch được tạo gắn với `sessionId`  
**When:** Session kết thúc và cleanup chạy  
**Then:** Sketch cho session đó bị xóa

---

## 3. Coverage Notes

- Actions: focus vào state machine transitions
- Routines: mock scheduler, không cần real cron
- Sketches: simple CRUD với append/prepend semantics
