# TR-016: Session Replay Test Requirements

**Module:** Session Replay (replay.ts, viewer/server.ts)  
**Nguồn:** SRS §3.8 (FR-REPLAY-001..002), URD §3.5  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-016-RPL-001 — Session được recorded đầy đủ
🔴 P0 | `[INT]` | **FR-REPLAY-001**

**Given:** Session đang active  
**When:** Agent gửi prompts và tool calls  
**Then:** Các events được record:
- User prompts
- Tool calls với input
- Tool results/output
- Agent responses
- Mỗi event có `timestamp` và sequence order

**Traceability:** FR-REPLAY-001, UR-022

---

## TR-016-RPL-002 — Import JSONL transcript từ Claude Code
🟠 P1 | `[INT]` | **FR-REPLAY-001**

**Given:** JSONL file từ `~/.claude/projects/`  
**When:** `memory_replay_import_jsonl({path: "..."})` được gọi  
**Then:**
- Observations được import từ transcript
- Session được tạo/updated trong KV
- Search hoạt động cho imported data

**Traceability:** FR-REPLAY-001, UR-008

---

## TR-016-RPL-003 — Viewer tại :3113
🔴 P0 | `[E2E]` | **FR-REPLAY-002**

**Given:** Viewer server running  
**When:** Browser navigate đến `http://localhost:3113`  
**Then:** HTTP 200, viewer UI loaded

**Traceability:** FR-REPLAY-002, UR-021

---

## TR-016-RPL-004 — Real-time observation stream
🔴 P0 | `[E2E]` | **FR-REPLAY-002**

**Given:** Viewer UI open  
**When:** New observation được gửi  
**Then:** Observation xuất hiện trong live stream trong <500ms

**Traceability:** UR-021, Architecture §3.2

---

## TR-016-RPL-005 — Replay load: đúng session
🟠 P1 | `[INT]`

**Given:** Session "sess_abc" với 10 events  
**When:** `memory_replay_load({sessionId: "sess_abc"})`  
**Then:**
- 10 events trả về với đầy đủ timestamp và payload
- Events sorted by sequence

**Traceability:** FR-REPLAY-002

---

## TR-016-RPL-006 — Session timeline: ordered events
🟠 P1 | `[UNIT]`

**Given:** Events với timestamps T1 < T2 < T3  
**When:** Replay data được loaded  
**Then:** Events sorted ascending by timestamp

**Traceability:** FR-REPLAY-002

---

## TR-016-RPL-007 — Session replay list: available sessions
🟡 P2 | `[INT]`

**When:** `memory_replay_sessions` được gọi  
**Then:** List các sessions có data để replay

**Traceability:** FR-REPLAY-002

---

## TR-016-RPL-008 — Viewer: session explorer
🟡 P2 | `[E2E]`

**Given:** Viewer UI  
**When:** User click vào session  
**Then:** Session detail view với observations list

**Traceability:** UR-022, FR-REPLAY-002

---

## TR-016-RPL-009 — Replay import: sensitive data redacted
🔴 P0 | `[INT]`

**Given:** JSONL transcript chứa API keys  
**When:** Import được chạy  
**Then:** Sensitive data redacted trước khi store (như TR-002-OBS-006)

**Traceability:** SRS §8.2

---

## TR-016-RPL-010 — Memory explorer trong viewer
🟡 P2 | `[E2E]`

**Given:** Viewer UI  
**When:** User navigate to memories section  
**Then:** List memories với search/filter capability

**Traceability:** UR-021
