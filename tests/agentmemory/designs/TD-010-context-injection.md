# TD-010: Context Injection Test Design

**Liên kết Requirements:** [TR-010-context-injection.md](../requirements/TR-010-context-injection.md)  
**Source:** `references/agentmemory/src/functions/context.ts`  
**Test file:** `tests/agentmemory/specs/context-injection.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

`mem::recall` là function chính để truy xuất context: kết hợp session summary + relevant observations + memories + graph results.

---

## 2. Test Cases

### Group A: Basic Recall

#### TC-001 — Recall trả về kết quả khi có observations liên quan
**Requirement:** TR-010-CTX-001 | **Type:** integration | 🔴 P0

**Given:** 5 observations trong session, trong đó 3 liên quan đến "auth"  
**When:** `mem::recall({query: "auth", sessionId: "sess_test", maxObs: 5})`  
**Then:**
- Kết quả không rỗng
- Mỗi result có `observation`, `combinedScore`
- Results sorted by score descending

---

#### TC-002 — Recall với query rỗng: trả về observations theo recent
**Requirement:** TR-010-CTX-003 | **Type:** integration | 🟠 P1

**Given:** 10 observations trong session  
**When:** `mem::recall({query: "", sessionId: "sess_test", maxObs: 5})`  
**Then:** 5 observations gần đây nhất được trả về (không search, chỉ list)

---

#### TC-003 — `maxObs` giới hạn số lượng observations trong result
**Requirement:** TR-010-CTX-002 | **Type:** integration | 🔴 P0

**Given:** 20 observations liên quan đến query  
**When:** `mem::recall({query: "auth", maxObs: 5})`  
**Then:** `result.observations.length ≤ 5`

---

#### TC-004 — `maxMemories` giới hạn memories trong result
**Type:** integration | 🟠 P1

**Given:** 10 memories liên quan  
**When:** `mem::recall({query: "auth", maxMemories: 3})`  
**Then:** `result.memories.length ≤ 3`

---

### Group B: Context Assembly

#### TC-005 — Result bao gồm session summary nếu có
**Requirement:** TR-010-CTX-004 | **Type:** integration | 🟠 P1

**Given:** Session đã có summary trong `mem:summaries`  
**When:** `mem::recall({sessionId: "sess_test"})`  
**Then:** `result.sessionSummary` có nội dung từ KV

---

#### TC-006 — Result bao gồm relevant memories
**Type:** integration | 🟠 P1

**Given:** Memories về "auth" trong KV, query = "auth"  
**When:** `mem::recall({query: "auth"})`  
**Then:** `result.memories` có memories liên quan đến "auth"

---

#### TC-007 — Chỉ trả về `isLatest = true` memories
**Type:** integration | 🟠 P1

**Given:** M1 (isLatest=false, superseded) và M2 (isLatest=true, superseder)  
**When:** `mem::recall({query: ...})`  
**Then:** M1 không xuất hiện, M2 xuất hiện

---

### Group C: Token Budget

#### TC-008 — Token budget được tính và giới hạn
**Requirement:** TR-010-CTX-005 | **Type:** integration | 🟠 P1

**Given:** `TOKEN_BUDGET = 2000`, nhiều observations và memories có tổng tokens > 2000  
**When:** `mem::recall({query: "auth"})`  
**Then:** Tổng tokens trong result ≤ 2000

---

#### TC-009 — High-importance items được ưu tiên trong token budget
**Type:** integration | 🟠 P1

**Given:** Mix of high-importance (strength=9) và low-importance (strength=3) memories  
**When:** Token budget bị tight  
**Then:** High-importance memories được giữ lại, low-importance bị drop

---

### Group D: Cross-session Recall

#### TC-010 — `includeOtherSessions=true` include obs từ sessions khác
**Requirement:** TR-010-CTX-007 | **Type:** integration | 🟡 P2

**Given:** 2 sessions, cả 2 có obs về "auth"  
**When:** `mem::recall({query: "auth", includeOtherSessions: true})`  
**Then:** Results từ cả 2 sessions xuất hiện

---

#### TC-011 — Mặc định: chỉ recall trong cùng sessionId
**Type:** integration | 🔴 P0

**Given:** 2 sessions, cả 2 có obs về "auth"  
**When:** `mem::recall({query: "auth", sessionId: "sess_A"})` (không có includeOtherSessions)  
**Then:** Chỉ obs từ `sess_A` xuất hiện

---

### Group E: Smart-Search Tracking

#### TC-012 — Search được record vào `mem:recent-searches`
**Requirement:** TR-010-CTX-008 | **Type:** integration | 🟡 P2

**Given:** `mem::recall` được gọi với sessionId  
**When:** Recall hoàn thành  
**Then:** `mem:recent-searches[sessionId]` được cập nhật với metadata của search này

---

## 3. Coverage Notes

| Function | Branches cần cover |
|---|---|
| `mem::recall` | Empty query path, search path, token budget cut-off |
| Context assembly | With/without summary, with/without memories, with/without graph |
| Cross-session | Default off, include=true |
