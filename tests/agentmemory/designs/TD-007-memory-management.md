# TD-007: Memory Management Test Design

**Liên kết Requirements:** [TR-007-memory-management.md](../requirements/TR-007-memory-management.md)  
**Source:** `references/agentmemory/src/functions/remember.ts`  
**Test file:** `tests/agentmemory/specs/memory-management.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Module quản lý bộ nhớ dài hạn: `mem::remember` và `mem::forget`.

**Điểm quan trọng từ source:**
- **Jaccard similarity threshold = 0.7** để supersede memory cũ
- **Mutex:** `withKeyedLock("mem:remember")` — global lock (không per-memory)
- **Initial strength = 7** (hardcoded)
- **Project isolation:** Không supersede memory từ project khác (khi cả 2 có project)
- Jaccard tính trên tokens (words) có độ dài ≥ 3 ký tự

---

## 2. Chiến lược kiểm thử

| Khía cạnh | Kỹ thuật |
|---|---|
| Jaccard threshold | Boundary: 0.69, 0.70, 0.71 |
| Versioning | State transition: version 1 → 2 → 3 |
| Project isolation | Cross-project isolation test |
| Validation | Equivalence partitioning |
| TTL | Time-based calculation |

---

## 3. Test Cases

### Group A: Jaccard Similarity

#### TC-001 — Identical strings: similarity = 1.0
**Requirement:** TR-007-MEM-004 | **Type:** unit | 🔴 P0

**Given:** Hai strings giống hệt nhau  
**When:** `jaccardSimilarity(a, a)` gọi  
**Then:** Trả về `1`

---

#### TC-002 — Hoàn toàn khác nhau: similarity = 0
**Type:** unit | 🔴 P0

**Given:**
- a = "auth middleware jwt authentication tokens"
- b = "database query optimization postgres connection"

**When:** `jaccardSimilarity(a, b)` gọi  
**Then:** Trả về `0` (no common words)

---

#### TC-003 — Similarity > 0.7: memory mới supersede memory cũ
**Requirement:** TR-007-MEM-004 | **Type:** integration | 🔴 P0

**Given:** Memory M1 đã tồn tại với content X (đủ từ chung để similarity > 0.7 với X')  
**When:** `mem::remember` với content X' (rất giống X)  
**Then:**
- Memory M2 được tạo với `parentId = M1.id`
- `M2.supersedes = [M1.id]`
- `M2.version = 2`
- M1 trong KV có `isLatest = false`

---

#### TC-004 — Similarity ≤ 0.7: tạo memory độc lập (không supersede)
**Type:** integration | 🔴 P0

**Given:** Memory M1 với content về "database"  
**When:** `mem::remember` với content về "auth" (không liên quan)  
**Then:**
- Memory M2 được tạo mới, `parentId = undefined`
- `M2.version = 1`
- M1 vẫn có `isLatest = true`
- KV có cả 2 memories

---

#### TC-005 — Versioning dây chuyền: M1 → M2 → M3
**Requirement:** TR-007-MEM-005 | **Type:** integration | 🟠 P1

**Given:** M1 (version 1) đang active  
**When:**
1. Remember M2 (simiarity > 0.7 với M1) → M2 supersedes M1, version=2
2. Remember M3 (similarity > 0.7 với M2) → M3 supersedes M2, version=3

**Then:**
- M1.isLatest = false
- M2.isLatest = false
- M3.isLatest = true, version=3
- Chỉ có 1 memory với isLatest=true

---

### Group B: Memory Structure

#### TC-006 — Memory mới có đầy đủ required fields
**Requirement:** TR-007-MEM-001 | **Type:** integration | 🔴 P0

**Given:** Remember được gọi với content, type, concepts, files  
**When:** Memory được tạo  
**Then:** Có đủ:
- `id`: pattern `mem_<ts>_<hex>`
- `createdAt`, `updatedAt`: ISO timestamps
- `type`: giá trị hợp lệ
- `strength = 7`
- `version = 1`
- `isLatest = true`
- `supersedes = []`
- `concepts`, `files`: arrays
- `title = content.slice(0, 80)` (80 ký tự đầu của content)

---

#### TC-007 — Tất cả 6 valid types được chấp nhận
**Requirement:** TR-007-MEM-002 | **Type:** unit | 🟠 P1

**Given:** type ∈ {`pattern`, `preference`, `architecture`, `bug`, `workflow`, `fact`}  
**When:** `mem::remember` với mỗi type  
**Then:** `memory.type = <input_type>` — giữ nguyên

**Kỹ thuật:** Parameterized test với 6 valid values

---

#### TC-008 — Unknown type bị coerced thành `"fact"`
**Requirement:** TR-007-MEM-002 | **Type:** unit | 🔴 P0

**Given:** type = `"invalid_type"` hoặc bất kỳ value không có trong valid set  
**When:** `mem::remember` gọi  
**Then:** `memory.type = "fact"` (default)

---

### Group C: Validation

#### TC-009 — Content rỗng bị từ chối
**Requirement:** TR-007-MEM-014 | **Type:** unit | 🔴 P0

**Given:** content = `""`  
**When:** `mem::remember` gọi  
**Then:** `{success: false, error: "content is required"}`

---

#### TC-010 — Content chỉ có whitespace bị từ chối
**Type:** unit | 🔴 P0

**Given:** content = `"   \t\n   "`  
**When:** `mem::remember` gọi  
**Then:** `{success: false}` — trim check

---

#### TC-011 — `files` phải là array khi được cung cấp
**Type:** unit | 🟠 P1

**Given:** files = `"not-an-array"` (string)  
**When:** `mem::remember` gọi  
**Then:** `{success: false, error: "files must be an array"}`

---

#### TC-012 — `concepts` phải là array khi được cung cấp
**Type:** unit | 🟠 P1

**Given:** concepts = `123` (number)  
**When:** `mem::remember` gọi  
**Then:** `{success: false, error: "concepts must be an array"}`

---

#### TC-013 — `sourceObservationIds` phải là array khi được cung cấp
**Type:** unit | 🟡 P2

**Given:** sourceObservationIds = `"obs_123"` (string, không phải array)  
**When:** `mem::remember` gọi  
**Then:** `{success: false, error: "sourceObservationIds must be an array"}`

---

### Group D: TTL

#### TC-014 — `ttlDays = 7` tạo `forgetAfter = now + 7 days`
**Requirement:** TR-007-MEM-008 | **Type:** unit | 🔴 P0

**Given:** `ttlDays = 7`  
**When:** Memory được tạo  
**Then:**
- `memory.forgetAfter` tồn tại
- `forgetAfter ≈ createdAt + 7 × 86400000 ms` (tolerance ±1s)

---

#### TC-015 — Không có `ttlDays`: `forgetAfter = undefined`
**Type:** unit | 🟠 P1

**Given:** `ttlDays` không được set (hoặc = 0)  
**When:** Memory được tạo  
**Then:** `memory.forgetAfter = undefined`

---

### Group E: `mem::forget`

#### TC-016 — Forget by memoryId: xóa khỏi KV và BM25 index
**Requirement:** TR-007-MEM-015 | **Type:** integration | 🔴 P0

**Given:** Memory M với id `mem_abc` đã tồn tại, đã được index trong BM25  
**When:** `mem::forget({memoryId: "mem_abc"})`  
**Then:**
- `{success: true, deleted: 1}`
- KV không còn `mem_abc` trong `mem:memories`
- BM25 search không còn tìm thấy M

---

#### TC-017 — Forget by sessionId: xóa session + tất cả observations
**Type:** integration | 🔴 P0

**Given:** Session `sess_target` với 3 observations  
**When:** `mem::forget({sessionId: "sess_target"})` (không có observationIds, không có memoryId)  
**Then:**
- Session bị xóa khỏi `mem:sessions`
- Tất cả observations bị xóa
- Summary bị xóa khỏi `mem:summaries`
- Audit record tạo ra

---

#### TC-018 — Forget by observationIds: chỉ xóa specified observations
**Type:** integration | 🟠 P1

**Given:** Session với obs1, obs2, obs3  
**When:** `mem::forget({sessionId: "sess", observationIds: ["obs1", "obs2"]})`  
**Then:**
- obs1, obs2 bị xóa
- obs3 vẫn tồn tại
- Session vẫn tồn tại

---

### Group F: Project Scope Isolation

#### TC-019 — Similarity > 0.7 nhưng khác project: KHÔNG supersede
**Requirement:** TR-007-MEM-006 | **Type:** integration | 🔴 P0

**Given:**
- M1 với `project = "project-A"`, content X
- Attempt to remember content X' (similarity > 0.7 với X) với `project = "project-B"`

**When:** `mem::remember` gọi  
**Then:**
- M2 được tạo mới (không supersede M1)
- `M2.parentId = undefined`, `M2.version = 1`
- Cả M1 và M2 đều có `isLatest = true` (mỗi project riêng)

---

#### TC-020 — Unscoped memory (không có project) có thể bị supersede bởi bất kỳ project
**Type:** integration | 🟡 P2

**Given:**
- M1 cũ không có `project` field (legacy)
- Content X' (similarity > 0.7) được remember với `project = "project-A"`

**When:** `mem::remember` gọi  
**Then:** M1 bị supersede (project guard không engage khi 1 bên không có project)

---

### Group G: AgentId

#### TC-021 — agentId từ payload được lưu
**Requirement:** TR-007-MEM-017 | **Type:** unit | 🟠 P1

**Given:** `agentId = "cursor-agent-1"` trong payload  
**When:** Memory được tạo  
**Then:** `memory.agentId = "cursor-agent-1"`

---

#### TC-022 — agentId được truncate tại 128 ký tự
**Type:** unit | 🟡 P2

**Given:** `agentId` = chuỗi dài hơn 128 ký tự  
**When:** Memory được tạo  
**Then:** `memory.agentId.length ≤ 128`

---

#### TC-023 — agentId fallback từ `AGENT_ID` env var
**Type:** unit | 🟠 P1

**Given:** `AGENT_ID=auto-cursor` env var, payload không có `agentId`  
**When:** Memory được tạo  
**Then:** `memory.agentId = "auto-cursor"`

---

### Group H: Audit Trail

#### TC-024 — `mem::forget` tạo audit record
**Requirement:** TR-007-MEM-015, TR-013-GOV-002 | **Type:** integration | 🟠 P1

**Given:** Memory tồn tại  
**When:** `mem::forget({memoryId: ...})`  
**Then:** AuditEntry được tạo trong KV `mem:audit` với `operation = "forget"`

---

## 4. Test Data

| TC | content | type | project | Expected behavior |
|---|---|---|---|---|
| TC-003 | X' (≥70% words match X) | fact | same | Supersede M1 |
| TC-004 | Hoàn toàn khác | fact | same | New memory |
| TC-019 | X' (≥70% match) | arch | project-B | No supersede |
| TC-009 | `""` | fact | - | Error |

---

## 5. Coverage Notes

| Function | Branches cần cover |
|---|---|
| `mem::remember` | Valid path, supersede path, project isolation, validation |
| `mem::forget` | By memoryId, by sessionId, by observationIds, audit |
| `jaccardSimilarity` | Equal, disjoint, partial overlap, threshold boundary |
| `withKeyedLock` | Sequential calls (tested indirectly) |
