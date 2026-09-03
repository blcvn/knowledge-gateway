# TD-016: Session Replay Test Design

**Liên kết Requirements:** [TR-016-session-replay.md](../requirements/TR-016-session-replay.md)  
**Source:** `references/agentmemory/src/functions/replay.ts`, `viewer/server.ts`  
**Test file:** `tests/agentmemory/specs/session-replay.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Test Cases

### Group A: JSONL Import

#### TC-001 — Import JSONL file với valid claude transcript
**Requirement:** TR-016-REP-001 | **Type:** integration | 🔴 P0

**Given:** File JSONL với 10 valid observation records  
**When:** `mem::import-replay({path: "transcript.jsonl"})` gọi  
**Then:**
- 10 observations được tạo trong KV
- Session được tạo với `source = "replay"`
- `{success: true, importedCount: 10}`

---

#### TC-002 — Malformed JSONL: skip invalid lines, count valid
**Requirement:** TR-016-REP-002 | **Type:** integration | 🔴 P0

**Given:** JSONL file: 8 valid lines, 2 invalid JSON lines  
**When:** Import gọi  
**Then:**
- 8 observations được tạo
- `{importedCount: 8, skippedCount: 2}` hoặc warning message

---

#### TC-003 — Import không duplicate khi gọi 2 lần (fingerprint dedup)
**Requirement:** TR-016-REP-003 | **Type:** integration | 🟠 P1

**Given:** JSONL file đã được import 1 lần  
**When:** Import gọi lần 2 với cùng file  
**Then:** `{importedCount: 0, skippedCount: 10}` — tất cả đã tồn tại (dùng fingerprintId)

---

### Group B: Viewer Server

#### TC-004 — Viewer server trả về session list qua HTTP
**Requirement:** TR-016-REP-005 | **Type:** integration | 🟠 P1

**Given:** Viewer server running, 3 sessions trong KV  
**When:** `GET /api/sessions` từ browser/test client  
**Then:**
- Status 200
- JSON array với 3 sessions

---

#### TC-005 — Viewer hiển thị observations theo timeline order
**Type:** integration | 🟠 P1

**Given:** Session với 5 observations có timestamps khác nhau  
**When:** `GET /api/sessions/:id/observations`  
**Then:** Observations được sorted theo `timestamp` ascending

---

#### TC-006 — Search trong replay: tìm obs theo keyword
**Type:** integration | 🟡 P2

**Given:** Imported session với obs có "auth" và "database" content  
**When:** `GET /api/sessions/:id/search?q=auth`  
**Then:** Chỉ obs có "auth" được trả về

---

### Group C: Replay Fidelity

#### TC-007 — Observation data được preserved đầy đủ khi import
**Requirement:** TR-016-REP-004 | **Type:** integration | 🔴 P0

**Given:** Source JSONL có obs với toolName, toolInput, toolOutput, agentId  
**When:** Import và retrieve  
**Then:** Tất cả fields được preserved, không bị truncated ngoài quy định

---

## 2. Coverage Notes

- JSONL import cần test với file paths (không phải in-memory string)
- Viewer server là separate process, dùng supertest hoặc fetch
- Fingerprint dedup dùng `fingerprintId()` từ schema.ts
