# TD-015: REST API Test Design

**Liên kết Requirements:** [TR-015-rest-api.md](../requirements/TR-015-rest-api.md)  
**Source:** `references/agentmemory/src/triggers/api.ts`  
**Test file:** `tests/agentmemory/specs/rest-api.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

REST API expose agentmemory qua HTTP, thường được dùng bởi Viewer UI và external integrations.

---

## 2. Chiến lược kiểm thử

| Layer | Phương pháp |
|---|---|
| Unit | Handler logic với MockKV |
| Integration | HTTP supertest qua express/fastify middleware |
| E2E | HTTP requests tới running server |

**Quy ước:**
- Test chính dùng **supertest** với express app được inject MockKV
- Verify HTTP status codes, response body schema, error formats

---

## 3. Test Cases

### Group A: Session Endpoints

#### TC-001 — `GET /sessions` trả về danh sách sessions
**Requirement:** TR-015-API-001 | **Type:** integration | 🔴 P0

**Given:** 3 sessions trong KV  
**When:** `GET /sessions`  
**Then:**
- Status 200
- Body: `{sessions: [{id, project, status, observationCount}...]}` — mảng 3 items

---

#### TC-002 — `GET /sessions/:id/observations` trả về observations của session
**Requirement:** TR-015-API-002 | **Type:** integration | 🔴 P0

**Given:** Session `sess_abc` với 5 observations  
**When:** `GET /sessions/sess_abc/observations`  
**Then:**
- Status 200
- Body có `observations` array với 5 items
- Mỗi obs có `id`, `timestamp`, `type`, `title`

---

#### TC-003 — `GET /sessions/:id` với ID không tồn tại → 404
**Type:** integration | 🔴 P0

**Given:** Session `sess_ghost` không tồn tại trong KV  
**When:** `GET /sessions/sess_ghost`  
**Then:** Status 404, body có `error` field

---

### Group B: Search Endpoints

#### TC-004 — `GET /search?q=auth` trả về search results
**Requirement:** TR-015-API-003 | **Type:** integration | 🔴 P0

**Given:** 5 observations có "auth" trong content  
**When:** `GET /search?q=auth`  
**Then:**
- Status 200
- Body: `{results: [{observation: {...}, bm25Score, combinedScore}...]}`
- Sorted by `combinedScore` descending

---

#### TC-005 — `GET /search` với `q` rỗng → recent observations
**Type:** integration | 🟠 P1

**Given:** 10 observations  
**When:** `GET /search?q=&limit=5`  
**Then:**
- Status 200
- 5 most recent observations returned

---

#### TC-006 — `GET /search?limit=5` giới hạn kết quả
**Type:** integration | 🟠 P1

**Given:** 20 matching observations  
**When:** `GET /search?q=auth&limit=5`  
**Then:** `results.length = 5`

---

### Group C: Memory Endpoints

#### TC-007 — `GET /memories` trả về all isLatest=true memories
**Requirement:** TR-015-API-004 | **Type:** integration | 🔴 P0

**Given:** M1 (isLatest=false), M2 (isLatest=true), M3 (isLatest=true)  
**When:** `GET /memories`  
**Then:** Body có M2 và M3, không có M1

---

#### TC-008 — `POST /memories` tạo memory mới
**Type:** integration | 🔴 P0

**Given:** Request body: `{content: "Auth uses JWT", type: "architecture"}`  
**When:** `POST /memories`  
**Then:**
- Status 201
- Body chứa memory với `id`, `version: 1`, `isLatest: true`

---

#### TC-009 — `DELETE /memories/:id` xóa memory
**Type:** integration | 🟠 P1

**Given:** Memory `mem_abc` tồn tại  
**When:** `DELETE /memories/mem_abc`  
**Then:**
- Status 200, body `{success: true, deleted: 1}`
- `GET /memories/mem_abc` → 404

---

### Group D: Graph Endpoints

#### TC-010 — `GET /graph/query?entity=jose` trả về related nodes
**Requirement:** TR-015-API-005 | **Type:** integration | 🟠 P1

**Given:** Graph node "jose" với edges đến 3 other nodes  
**When:** `GET /graph/query?entity=jose`  
**Then:**
- Status 200
- Body: `{nodes: [...], edges: [...]}`
- "jose" node included trong nodes

---

#### TC-011 — `GET /graph/stats` trả về aggregate type counts
**Type:** integration | 🟡 P2

**Given:** Graph với 10 file nodes, 5 function nodes  
**When:** `GET /graph/stats`  
**Then:**
- Status 200
- Body: `{totalNodes: 15, typeCounts: {file: 10, function: 5}}`

---

### Group E: Error Handling

#### TC-012 — Invalid JSON body → 400 Bad Request
**Type:** integration | 🔴 P0

**Given:** Request với malformed JSON body  
**When:** `POST /memories` với body = `"not-json"`  
**Then:** Status 400, `{error: "invalid JSON"}`

---

#### TC-013 — Missing required field → 422 Unprocessable Entity
**Type:** integration | 🔴 P0

**Given:** `POST /memories` với body `{type: "fact"}` (thiếu `content`)  
**When:** Request được xử lý  
**Then:** Status 422, `{error: "content is required"}`

---

#### TC-014 — 500 Internal Server Error có safe error message
**Requirement:** TR-015-API-011 | **Type:** integration | 🟠 P1

**Given:** KV operation throw unexpected error  
**When:** Endpoint được gọi  
**Then:**
- Status 500
- Body `{error: "internal server error"}` — không expose stack trace

---

### Group F: Health và Status

#### TC-015 — `GET /health` trả về 200 khi healthy
**Requirement:** TR-015-API-008 | **Type:** integration | 🔴 P0

**Given:** Server running, KV accessible  
**When:** `GET /health`  
**Then:** Status 200, `{status: "ok"}`

---

#### TC-016 — `GET /status` trả về metrics tổng quan
**Type:** integration | 🟠 P1

**Given:** Server có data  
**When:** `GET /status`  
**Then:**
- Status 200
- Body có `totalSessions`, `totalObservations`, `totalMemories`, `uptime`

---

## 4. Coverage Notes

- Error response format phải consistent: `{error: string, [details]: any}`
- Auth header check (nếu `AGENTMEMORY_SECRET` set) cần test ở endpoint level
- Pagination (offset/limit) cần boundary tests
