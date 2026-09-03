# TD-020: Security Test Design

**Liên kết Requirements:** [TR-020-security.md](../requirements/TR-020-security.md)  
**Source:** `references/agentmemory/src/functions/privacy.ts`, auth middleware  
**Test file:** `tests/agentmemory/specs/security.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Test Cases

### Group A: API Authentication

#### TC-001 — Request với valid Bearer token: accepted (200)
**Requirement:** TR-020-SEC-001 | **Type:** integration | 🔴 P0

**Given:** `AGENTMEMORY_SECRET=test-key-123`, request có `Authorization: Bearer test-key-123`  
**When:** `GET /status` request  
**Then:** Status 200

---

#### TC-002 — Request với wrong token: rejected (401)
**Type:** integration | 🔴 P0

**Given:** `AGENTMEMORY_SECRET=real-key`  
**When:** `GET /status` với `Authorization: Bearer wrong-key`  
**Then:** Status 401, body `{error: "unauthorized"}`

---

#### TC-003 — Request với no token: rejected (401) khi secret là set
**Type:** integration | 🔴 P0

**Given:** `AGENTMEMORY_SECRET=real-key`  
**When:** Request không có Authorization header  
**Then:** Status 401

---

#### TC-004 — Khi không có `AGENTMEMORY_SECRET`: tất cả requests accepted (local mode)
**Requirement:** TR-020-SEC-002 | **Type:** integration | 🔴 P0

**Given:** `AGENTMEMORY_SECRET` không được set  
**When:** Request không có Authorization header  
**Then:** Status 200 (no auth required)

---

#### TC-005 — Timing-safe comparison: không susceptible to timing oracle
**Requirement:** TR-020-SEC-003 | **Type:** unit | 🟠 P1

**Given:** Auth check function  
**When:** Compare valid token vs invalid token  
**Then:** Comparison dùng `crypto.timingSafeEqual()` hoặc equivalent — không dùng `===`

---

### Group B: Privacy Redaction (Security Aspect)

#### TC-006 — Private data không xuất hiện trong KV sau observation
**Requirement:** TR-020-SEC-004 | **Type:** integration | 🔴 P0

**Given:** Hook với toolOutput chứa `sk-ant-api03-FAKEKEY`  
**When:** Observation được stored  
**Then:** KV không chứa original secret string — chỉ có `[REDACTED_SECRET]`

---

#### TC-007 — Private data không xuất hiện trong recall response
**Type:** integration | 🔴 P0

**Given:** Observation đã được stored với redacted content  
**When:** `mem::recall` gọi  
**Then:** Recall response không chứa un-redacted secrets

---

#### TC-008 — `<private>` tags được removed hoàn toàn
**Type:** unit | 🔴 P0

**Given:** Observation text với `<private>my_api_key=sk-ant-...</private>`  
**When:** Privacy redaction chạy  
**Then:**
- Text không chứa bất kỳ nội dung bên trong `<private>` tags
- Tags bản thân cũng bị remove

---

### Group C: Data Isolation

#### TC-009 — Cross-project: session A không thể access observations của session B
**Requirement:** TR-020-SEC-005 | **Type:** integration | 🔴 P0

**Given:**
- Session A trong `project-A` với sensitive obs
- Request từ context của `project-B`

**When:** `GET /sessions/sess_A/observations` mà không có auth cho session A  
**Then:**
- Nếu không có auth: 401
- Nếu có auth nhưng wrong project: 403 hoặc 404

---

#### TC-010 — Path traversal: `sessionId` không thể dùng path-escape characters
**Requirement:** TR-020-SEC-006 | **Type:** unit | 🔴 P0

**Given:** `sessionId = "../../../etc/passwd"`  
**When:** Observation gửi với malicious sessionId  
**Then:**
- `{success: false, error: "invalid sessionId"}`
- Không tạo session với path-like key trong KV

---

#### TC-011 — SQL injection: KV keys được sanitized
**Type:** unit | 🟠 P1

**Given:** `sessionId = "'; DROP TABLE sessions;--"` (SQL injection attempt)  
**When:** Observation gửi  
**Then:** Error (invalid sessionId) — không crash, không leak

---

### Group D: Secret Management

#### TC-012 — Env vars không xuất hiện trong logs
**Requirement:** TR-020-SEC-007 | **Type:** unit | 🟠 P1

**Given:** `AGENTMEMORY_SECRET=my-secret` được set  
**When:** Server logs được captured  
**Then:** "my-secret" string không xuất hiện trong log output

---

#### TC-013 — API keys không xuất hiện trong error messages
**Type:** unit | 🟠 P1

**Given:** Error xảy ra khi call Anthropic API (có API key trong env)  
**When:** Error được caught và returned  
**Then:** Error message không chứa API key value

---

## 2. Coverage Notes

- Auth middleware cần test với Express/Fastify middleware isolation
- Timing-safe comparison: inspect source code thay vì timing measurement (flaky)
- Path traversal: verify KV key validation regex covers all bad chars
