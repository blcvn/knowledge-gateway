# TR-012: Orchestration Layer Test Requirements

**Module:** Actions, Routines, Checkpoints, Sentinels, Sketches/Crystals  
**Nguồn:** SRS §3.11 (FR-ORCH-001..005), TDD §7.1-7.3  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-012-ORC-001 — Action state machine: valid transitions
🔴 P0 | `[UNIT]` | **FR-ORCH-001**

**Given:** Action ở từng state  
**When:** Transition được attempt  
**Then:**

| From → To | Valid? |
|---|---|
| `pending` → `active` | ✅ |
| `pending` → `cancelled` | ✅ |
| `pending` → `blocked` | ✅ |
| `active` → `done` | ✅ |
| `active` → `blocked` | ✅ |
| `done` → `active` | ❌ |
| `cancelled` → `active` | ❌ |

**Traceability:** FR-ORCH-001, TDD §7.1

---

## TR-012-ORC-002 — Action priority 0-100
🟠 P1 | `[UNIT]` | **FR-ORCH-001**

**Given:** Action được tạo  
**When:** Priority được set  
**Then:**
- Priority 0-100 được accept
- Priority < 0 hoặc > 100 → validation error
- Higher priority = more important (sorting)

**Traceability:** TDD §7.1

---

## TR-012-ORC-003 — Action edges: requires relationship
🟠 P1 | `[INT]` | **FR-ORCH-001**

**Given:** Action B có edge `requires` Action A  
**When:** Action B được query  
**Then:** Action B không thể start cho đến khi A là `done`

**Traceability:** FR-ORCH-001, TDD §7.1

---

## TR-012-ORC-004 — Action edges: 5 edge types
🟡 P2 | `[UNIT]`

**Given:** ActionEdge được tạo  
**When:** Type validated  
**Then:** Chỉ accept: `requires`, `unlocks`, `spawned_by`, `gated_by`, `conflicts_with`

**Traceability:** FR-ORCH-001

---

## TR-012-ORC-005 — Routine: CRUD operations
🔴 P0 | `[INT]` | **FR-ORCH-002**

**Given:** Routine "deploy-workflow" với 3 steps  
**When:** Routine được create → list → update  
**Then:**
- Create: Routine stored với steps và dependsOn[]
- List: Routine xuất hiện trong list
- Update: step status có thể update
- RoutineRun tracking: step status, pause/resume

**Traceability:** FR-ORCH-002

---

## TR-012-ORC-006 — Routine steps: dependency ordering
🟠 P1 | `[UNIT]` | **FR-ORCH-002**

**Given:** Routine với steps: [A, B(depends_on=A), C(depends_on=B)]  
**When:** Routine execution được track  
**Then:** B không thể start trước A done, C không thể start trước B done

**Traceability:** FR-ORCH-002, TDD §7.1

---

## TR-012-ORC-007 — Checkpoint: 5 types
🟠 P1 | `[UNIT]` | **FR-ORCH-003**

**Given:** Checkpoint được tạo  
**When:** Type validated  
**Then:** Hỗ trợ: `ci`, `approval`, `deploy`, `external`, `timer`

**Traceability:** FR-ORCH-003

---

## TR-012-ORC-008 — Checkpoint: status transitions
🔴 P0 | `[INT]` | **FR-ORCH-003**

**Given:** Checkpoint với `status: "pending"`  
**When:** `memory_checkpoint_resolve` gọi với result  
**Then:** Status chuyển thành `passed` hoặc `failed` (tùy result)

**Traceability:** FR-ORCH-003

---

## TR-012-ORC-009 — Sentinel: 6 types
🟡 P2 | `[UNIT]` | **FR-ORCH-004**

**Given:** Sentinel được tạo  
**When:** Type validated  
**Then:** Hỗ trợ: `webhook`, `timer`, `threshold`, `pattern`, `approval`, `custom`

**Traceability:** FR-ORCH-004

---

## TR-012-ORC-010 — Sketch: auto-expiry 24h
🔴 P0 | `[UNIT]` | **FR-ORCH-005**

**Given:** Sketch được tạo  
**When:** `expiresAt` được set  
**Then:** `expiresAt = createdAt + 24 hours`

**Traceability:** FR-ORCH-005, TDD §7.3

---

## TR-012-ORC-011 — Sketch → Crystal promotion
🔴 P0 | `[INT]` | **FR-ORCH-005**

**Given:** Sketch với 3 completed actions  
**When:** `crystallize` được gọi (LLM available)  
**Then:**
- Crystal được tạo với:
  - `narrative`: What was accomplished
  - `keyOutcomes`: Concrete results
  - `filesAffected`: Changed files
  - `lessons`: What was learned
  - `sourceActionIds`: Original action IDs
- Sketch `status = "promoted"`

**Traceability:** FR-ORCH-005, TDD §7.3

---

## TR-012-ORC-012 — Sketch discarded on expiry
🟠 P1 | `[INT]`

**Given:** Sketch không được promote trước `expiresAt`  
**When:** Expiry sweep chạy  
**Then:** `sketch.status = "discarded"`

**Traceability:** TDD §7.3

---

## TR-012-ORC-013 — Action tagging và filtering
🟡 P2 | `[INT]`

**Given:** Actions với tags ["frontend", "auth"]  
**When:** `memory_action_list({tags: ["auth"]})`  
**Then:** Chỉ trả về actions có tag "auth"

**Traceability:** FR-ORCH-001

---

## TR-012-ORC-014 — Action parent-child relationships
🟡 P2 | `[UNIT]`

**Given:** Action B được tạo với `parentId = A.id`  
**When:** Action B được query  
**Then:** Parent-child relationship được preserve

**Traceability:** FR-ORCH-001

---

## TR-012-ORC-015 — RoutineRun: pause và resume
🟡 P2 | `[INT]`

**Given:** Routine đang chạy  
**When:** Pause signal được gửi  
**Then:**
- `RoutineRun.status = "paused"`
- Steps hiện tại được preserve
- Resume tiếp tục từ current step

**Traceability:** FR-ORCH-002

---

## TR-012-ORC-016 — Sentinel escalation
🟡 P2 | `[UNIT]`

**Given:** Sentinel với timeout  
**When:** Timeout đạt mà không có response  
**Then:**
- `escalatedAt` được set
- Notification được gửi (signal hoặc alert)

**Traceability:** FR-ORCH-004

---

## TR-012-ORC-017 — Crystal structure đầy đủ
🔴 P0 | `[UNIT]`

**Given:** Crystal được tạo từ sketch  
**When:** Crystal object được lấy từ KV  
**Then:** Fields đầy đủ:
```typescript
{
  id: string,
  narrative: string,
  keyOutcomes: string[],
  filesAffected: string[],
  lessons: string[],
  sourceActionIds: string[],
  sessionId?: string,
  project?: string,
  createdAt: string
}
```

**Traceability:** TDD §7.3

---

## TR-012-ORC-018 — Action list: default sort by priority DESC
🟡 P2 | `[INT]`

**Given:** 5 actions với priorities: [50, 100, 10, 90, 5]  
**When:** `memory_action_list()` không có sort param  
**Then:** Actions trả về sorted: [100, 90, 50, 10, 5]

**Traceability:** FR-ORCH-001

---

## TR-012-ORC-019 — Checkpoint expired handling
🟠 P1 | `[UNIT]`

**Given:** Checkpoint với `expiresAt` đã qua  
**When:** Status được check  
**Then:** Status = "expired" (không còn pending)

**Traceability:** FR-ORCH-003

---

## TR-012-ORC-020 — Conflicts_with edge: prevents concurrent execution
🟡 P2 | `[INT]`

**Given:** Action A `conflicts_with` Action B  
**When:** A được mark active  
**Then:** B không thể become active đồng thời (được blocked)

**Traceability:** FR-ORCH-001
