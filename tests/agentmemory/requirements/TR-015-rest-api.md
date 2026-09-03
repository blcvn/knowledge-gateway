# TR-015: REST API Test Requirements

**Module:** REST API (triggers/api.ts)  
**Nguồn:** SRS §7.1, Architecture §3.2, §8.1, URD §3.9  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-015-API-001 — 128 endpoints registered
🟠 P1 | `[UNIT]` | **SRS §7.1**

**Given:** REST API server start  
**When:** Route list được inspect  
**Then:** Đúng 128 endpoints registered tại base path `/agentmemory/`

**Traceability:** SRS §7.1, Architecture §3.2

---

## TR-015-API-002 — POST /observe: nhận hook events
🔴 P0 | `[INT]`

**Given:** Server running  
**When:** `POST /agentmemory/observe` với valid hook payload  
**Then:**
- HTTP 200
- `{observationId: "obs_xxx"}` trong response body
- Observation được lưu vào KV

**Traceability:** SRS §7.1

---

## TR-015-API-003 — POST /smart-search: hybrid search
🔴 P0 | `[INT]`

**Given:** Data trong memory  
**When:** `POST /agentmemory/smart-search` với body `{query: "auth", limit: 10}`  
**Then:**
- HTTP 200
- `HybridSearchResult[]` trong response
- Latency ≤ 50ms

**Traceability:** SRS §7.1

---

## TR-015-API-004 — POST /remember: lưu memory
🔴 P0 | `[INT]`

**When:** `POST /agentmemory/remember` với valid memory payload  
**Then:**
- HTTP 200
- `{memoryId: "mem_xxx"}` trong response
- Memory được lưu và index

**Traceability:** SRS §7.1

---

## TR-015-API-005 — GET /sessions: list sessions
🟠 P1 | `[INT]`

**When:** `GET /agentmemory/sessions`  
**Then:**
- HTTP 200
- JSON array của Session objects

**Traceability:** SRS §7.1

---

## TR-015-API-006 — GET /sessions/{id}: session details
🟠 P1 | `[INT]`

**Given:** Session với `id = "sess_abc"` tồn tại  
**When:** `GET /agentmemory/sessions/sess_abc`  
**Then:** HTTP 200 với full Session object

**Traceability:** SRS §7.1

---

## TR-015-API-007 — GET /sessions/{id}: 404 cho unknown
🟠 P1 | `[INT]`

**Given:** Session "sess_unknown" không tồn tại  
**When:** `GET /agentmemory/sessions/sess_unknown`  
**Then:** HTTP 404

**Traceability:** SRS §7.1

---

## TR-015-API-008 — GET /memories: list memories
🟠 P1 | `[INT]`

**When:** `GET /agentmemory/memories`  
**Then:** HTTP 200 với Memory[] (chỉ `isLatest=true`)

**Traceability:** SRS §7.1

---

## TR-015-API-009 — DELETE /memories/{id}: governance delete
🔴 P0 | `[INT]`

**Given:** Memory `mem_abc` tồn tại  
**When:** `DELETE /agentmemory/memories/mem_abc`  
**Then:**
- HTTP 200 hoặc 204
- Memory deleted với cascade
- Audit record tạo ra

**Traceability:** SRS §7.1, FR-GOV-001

---

## TR-015-API-010 — GET /health: health status
🔴 P0 | `[INT]`

**When:** `GET /agentmemory/health`  
**Then:**
- HTTP 200
- Body: `{status: "ok" | "degraded" | "critical", ...details}`

**Traceability:** UR-032, SRS §7.1

---

## TR-015-API-011 — GET /graph: knowledge graph query
🟡 P2 | `[INT]`

**When:** `GET /agentmemory/graph?limit=20`  
**Then:** HTTP 200 với graph nodes/edges

**Traceability:** SRS §7.1

---

## TR-015-API-012 — GET /export: full data export
🟡 P2 | `[INT]`

**When:** `GET /agentmemory/export`  
**Then:**
- HTTP 200
- ExportData với sessions, observations, memories, summaries
- Version field present

**Traceability:** SRS §7.1

---

## TR-015-API-013 — POST /import: import data
🟡 P2 | `[INT]`

**Given:** Valid ExportData JSON  
**When:** `POST /agentmemory/import`  
**Then:**
- HTTP 200
- Data được restore vào KV

**Traceability:** SRS §7.1

---

## TR-015-API-014 — Authentication: secret required
🔴 P0 | `[INT]`

**Given:** `AGENTMEMORY_SECRET=mysecret`  
**When:** Request mà không có `Authorization: Bearer mysecret`  
**Then:** HTTP 401 với error message

**Traceability:** SRS §8.1, UR-039

---

## TR-015-API-015 — Authentication: secret match
🔴 P0 | `[INT]`

**Given:** `AGENTMEMORY_SECRET=mysecret`  
**When:** Request có `Authorization: Bearer mysecret`  
**Then:** HTTP 200 (authenticated)

**Traceability:** SRS §8.1

---

## TR-015-API-016 — Default port 3111
🟠 P1 | `[UNIT]`

**Given:** Không có `III_REST_PORT` env var  
**When:** REST API start  
**Then:** Bind đến port 3111

**Traceability:** SRS §5.3, SRS §9.3

---

## TR-015-API-017 — Port allocation: streams = REST+1, viewer = REST+2
🟠 P1 | `[UNIT]`

**Given:** `III_REST_PORT=4000`  
**When:** Server start  
**Then:**
- REST API: 4000
- Streams: 4001
- Viewer: 4002

**Traceability:** SRS §5.3

---

## TR-015-API-018 — JSON request/response
🟠 P1 | `[UNIT]`

**When:** Bất kỳ REST endpoint nào  
**Then:**
- Request: `Content-Type: application/json`
- Response: `Content-Type: application/json`

**Traceability:** SRS §7.1

---

## TR-015-API-019 — Endpoint maps 1:1 với iii function
🟠 P1 | `[UNIT]`

**Given:** REST endpoint được gọi  
**When:** Request được processed  
**Then:** Mỗi endpoint trigger đúng 1 iii function

**Traceability:** Architecture §3.2
