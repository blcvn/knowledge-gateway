# TD-017: Memory Slots Test Design

**Liên kết Requirements:** [TR-017-memory-slots.md](../requirements/TR-017-memory-slots.md)  
**Source:** `references/agentmemory/src/functions/slots.ts`  
**Test file:** `tests/agentmemory/specs/memory-slots.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Memory Slots là named persistent storage positions dùng để lưu high-priority context explicitly. Tương tự "bookmarks" hay "pinned notes".

---

## 2. Test Cases

### Group A: Slot CRUD

#### TC-001 — Tạo slot với name và content
**Requirement:** TR-017-SLT-001 | **Type:** integration | 🔴 P0

**Given:** KV trống  
**When:** `mem::slot-write({name: "current-task", content: "Implementing auth middleware"})`  
**Then:**
- Slot được lưu trong `mem:slots` với key = name
- Slot có `name`, `content`, `createdAt`, `updatedAt`
- `{success: true}`

---

#### TC-002 — Slot name là unique key: overwrite khi cùng name
**Requirement:** TR-017-SLT-002 | **Type:** integration | 🔴 P0

**Given:** Slot "current-task" với content "Old content"  
**When:** `mem::slot-write({name: "current-task", content: "New content"})`  
**Then:**
- Chỉ có 1 slot với name "current-task"
- Content = "New content"
- `updatedAt > createdAt`

---

#### TC-003 — `mem::slot-read` trả về slot theo name
**Type:** integration | 🔴 P0

**Given:** Slot "goal" với content "Build auth system"  
**When:** `mem::slot-read({name: "goal"})`  
**Then:** `{success: true, slot: {name: "goal", content: "Build auth system"}}`

---

#### TC-004 — `mem::slot-read` trả về null cho slot không tồn tại
**Type:** unit | 🔴 P0

**Given:** Slot "nonexistent" không có trong KV  
**When:** `mem::slot-read({name: "nonexistent"})`  
**Then:** `{success: true, slot: null}` hoặc `{success: false, error: "not found"}`

---

#### TC-005 — `mem::slot-delete` xóa slot
**Requirement:** TR-017-SLT-003 | **Type:** integration | 🟠 P1

**Given:** Slot "current-task" tồn tại  
**When:** `mem::slot-delete({name: "current-task"})`  
**Then:**
- `{success: true, deleted: 1}`
- `mem::slot-read({name: "current-task"})` → not found

---

#### TC-006 — `mem::slot-list` trả về tất cả slots
**Type:** integration | 🟠 P1

**Given:** 3 slots: "goal", "current-task", "notes"  
**When:** `mem::slot-list`  
**Then:** Array với 3 slot entries, sorted alphabetically hoặc theo createdAt

---

### Group B: Global vs Session Slots

#### TC-007 — Global slot: `mem:slots:global` scope
**Requirement:** TR-017-SLT-004 | **Type:** integration | 🟠 P1

**Given:** Slot với `scope = "global"` được tạo  
**When:** Slot được stored  
**Then:** Lưu tại `mem:slots:global` (không phải `mem:slots`)

---

#### TC-008 — Session slot chỉ visible trong session đó
**Type:** integration | 🔴 P0

**Given:** Slot A được tạo trong session-1, Slot B trong session-2  
**When:** `mem::slot-list({sessionId: "session-1"})`  
**Then:** Chỉ Slot A xuất hiện, không có Slot B

---

### Group C: Recall Integration

#### TC-009 — Slots được include trong recall context tự động
**Requirement:** TR-017-SLT-005 | **Type:** integration | 🟠 P1

**Given:** Slot "goal" với content "Build auth middleware"  
**When:** `mem::recall({includeSlots: true})`  
**Then:** Recall result có `slots` array chứa "goal" slot content

---

#### TC-010 — Slot content ảnh hưởng token budget
**Type:** integration | 🟡 P2

**Given:** Token budget = 500, slot có 200 tokens content  
**When:** Recall với `includeSlots: true`  
**Then:** Observations được cắt để fit 300 tokens còn lại

---

### Group D: Validation

#### TC-011 — Slot name phải là valid identifier (không có special chars)
**Type:** unit | 🟠 P1

**Given:** name = `"my slot!"` (có space và !)  
**When:** `mem::slot-write` gọi  
**Then:** `{success: false, error: "invalid slot name"}` 

**Valid pattern:** alphanumeric, hyphens, underscores

---

#### TC-012 — Content không được rỗng
**Type:** unit | 🔴 P0

**Given:** content = `""`  
**When:** `mem::slot-write` gọi  
**Then:** `{success: false, error: "content is required"}`

---

#### TC-013 — Content dài quá 10KB bị truncate hoặc rejected
**Requirement:** TR-017-SLT-007 | **Type:** unit | 🟡 P2

**Given:** content = string 15KB  
**When:** `mem::slot-write` gọi  
**Then:** Error hoặc content được truncate tại giới hạn cấu hình

---

## 3. Coverage Notes

- Slot CRUD là relatively simple, high coverage expected
- Focus testing vào scope isolation (session vs global)
- Recall integration là cross-module test
