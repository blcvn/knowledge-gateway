# TR-017: Memory Slots & Working Memory Test Requirements

**Module:** Memory Slots (slots.ts)  
**Nguồn:** SRS §3.9 (FR-SLOTS-001..002), TDD §4.3  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-017-SLT-001 — Slot CRUD: create/read/update/delete
🔴 P0 | `[INT]` | **FR-SLOTS-001**

**Given:** Không có slot "preferences" tồn tại  
**When:** `mem::slot-write({label: "preferences", content: "Use Tailwind v4", scope: "global"})`  
**Then:**
- Slot được tạo trong KV: `slots/global/preferences`
- `mem::slot-read({label: "preferences", scope: "global"})` trả về content

**Traceability:** FR-SLOTS-001, TDD §4.3

---

## TR-017-SLT-002 — Slot scope: project vs global
🔴 P0 | `[UNIT]` | **FR-SLOTS-001**

**Given:** 2 slots cùng label "notes":
- `slots/project/notes` (scope=project)
- `slots/global/notes` (scope=global)

**When:** Read với `scope="project"`  
**Then:** Chỉ return content của `slots/project/notes`

**Traceability:** FR-SLOTS-001, TDD §4.3

---

## TR-017-SLT-003 — Slot: append mode
🟠 P1 | `[INT]` | **FR-SLOTS-001**

**Given:** Slot "context" với content = "Line 1"  
**When:** `mem::slot-write({label: "context", content: "Line 2", mode: "append"})`  
**Then:** content = "Line 1\nLine 2"

**Traceability:** FR-SLOTS-001, TDD §4.3

---

## TR-017-SLT-004 — Slot: replace mode
🟠 P1 | `[INT]`

**Given:** Slot với content = "Old content"  
**When:** `mode: "replace"` với "New content"  
**Then:** content = "New content" (old completely replaced)

**Traceability:** FR-SLOTS-001

---

## TR-017-SLT-005 — Slot pinned: không bị evict
🔴 P0 | `[UNIT]` | **FR-SLOTS-001**

**Given:** Slot với `pinned = true`  
**When:** Eviction policy chạy  
**Then:** Pinned slot KHÔNG bị evict bất kể score thấp

**Traceability:** FR-SLOTS-001

---

## TR-017-SLT-006 — Slot readOnly: không thể modify
🟠 P1 | `[UNIT]` | **FR-SLOTS-001**

**Given:** Slot với `readOnly = true`  
**When:** `mem::slot-write` được gọi cho slot đó  
**Then:** Error được trả về: "Slot is read-only"

**Traceability:** FR-SLOTS-001

---

## TR-017-SLT-007 — Slot size limit
🟠 P1 | `[UNIT]`

**Given:** Slot với `sizeLimit = 1000` ký tự  
**When:** Content > 1000 ký tự được written  
**Then:** Error hoặc content truncated (tùy implementation)

**Traceability:** FR-SLOTS-001, TDD §4.3

---

## TR-017-SLT-008 — Slot list: filter theo scope
🟡 P2 | `[INT]`

**Given:** 5 global slots, 3 project slots  
**When:** `mem::slot-list({scope: "global"})`  
**Then:** Chỉ 5 global slots returned

**Traceability:** FR-SLOTS-001

---

## TR-017-SLT-009 — Slot reflect: LLM generate content
🟡 P2 | `[INT]` | **FR-SLOTS-001**

**Given:** Session với rich observations, LLM available  
**When:** `mem::slot-reflect` (triggered on Stop hook)  
**Then:** LLM generates slot content từ session observations

**Traceability:** FR-SLOTS-001, TDD §4.3

---

## TR-017-SLT-010 — Working memory: evict khi vượt budget
🟠 P1 | `[UNIT]` | **FR-SLOTS-002**

**Given:** Working memory với 1900 tokens, budget = 2000  
**When:** New item (200 tokens) được added  
**Then:** Oldest/least-important item bị evicted để make room

**Traceability:** FR-SLOTS-002
