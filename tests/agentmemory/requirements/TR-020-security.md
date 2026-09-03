# TR-020: Security Test Requirements

**Module:** Security, Authentication, Privacy  
**Nguồn:** SRS §8, Architecture §11, URD §3.10  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-020-SEC-001 — HMAC: constant-time comparison
🔴 P0 | `[UNIT]` | **SRS §8.1**

**Given:** HMAC secret comparison  
**When:** `Authorization: Bearer` header được verified  
**Then:** Constant-time comparison được dùng (không timing-safe string compare)

**Traceability:** SRS §8.1, Architecture §11.1

---

## TR-020-SEC-002 — Bind loopback by default
🔴 P0 | `[UNIT]` | **SRS §8.3**

**Given:** `AGENTMEMORY_BIND_ADDRESS` không set  
**When:** REST server start  
**Then:** Bind đến `127.0.0.1` (loopback), không accessible từ network

**Traceability:** SRS §8.3, UR-038

---

## TR-020-SEC-003 — Production: bind 0.0.0.0 cần secret
🔴 P0 | `[INT]`

**Given:** `AGENTMEMORY_BIND_ADDRESS=0.0.0.0`, `AGENTMEMORY_SECRET` chưa set  
**When:** Server start  
**Then:** Warning hoặc error được log về security risk

**Traceability:** SRS §8.3

---

## TR-020-SEC-004 — No network calls: no API key
🔴 P0 | `[INT]` | **UR-038**

**Given:** Không có LLM API key  
**When:** 50 observations được processed  
**Then:** Zero outbound HTTP/HTTPS calls (verified via network intercept)

**Traceability:** UR-038, UR-029

---

## TR-020-SEC-005 — Privacy: data stays local
🔴 P0 | `[INT]` | **UR-038**

**Given:** LLM API key được config (`AGENTMEMORY_AUTO_COMPRESS=false`, default)  
**When:** Observations được gửi  
**Then:** Không có observation data gửi đến LLM (synthetic compression dùng)

**Traceability:** UR-038, SRS §8.2

---

## TR-020-SEC-006 — Privacy: LLM calls opt-in only
🟠 P1 | `[INT]`

**Given:** `AGENTMEMORY_AUTO_COMPRESS=true` VÀ API key set  
**When:** Observation được processed  
**Then:** LLM được gọi chỉ khi cả 2 điều kiện thỏa mãn

**Traceability:** SRS §8.2

---

## TR-020-SEC-007 — Team multi-tenant isolation
🟠 P1 | `[INT]` | **SRS §8.4**

**Given:** `TEAM_MODE=private`, 2 teams: team-A và team-B  
**When:** team-A lưu memory  
**Then:** team-B KHÔNG thể recall team-A's memories

**Traceability:** SRS §8.4

---

## TR-020-SEC-008 — TEAM_MODE=shared: cross-team visibility
🟡 P2 | `[INT]`

**Given:** `TEAM_MODE=shared`  
**When:** team-A lưu memory  
**Then:** team-B CÓ THỂ recall team-A's memories (shared mode)

**Traceability:** SRS §8.4

---

## TR-020-SEC-009 — Mesh: HMAC giữa nodes
🟠 P1 | `[INT]`

**Given:** Node A mesh sync đến Node B  
**When:** Request được gửi từ A → B  
**Then:**
- Request có `Authorization: Bearer <hmac-signed>`
- Node B verify signature
- Unauthorized sync request → 401

**Traceability:** FR-MULTI-004, Architecture §7.2

---

## TR-020-SEC-010 — Secret không exposed trong logs
🔴 P0 | `[UNIT]`

**Given:** `AGENTMEMORY_SECRET=mysecret123` set  
**When:** Bất kỳ logging operation nào  
**Then:** "mysecret123" KHÔNG xuất hiện trong log output

**Traceability:** SRS §8.1

---

## TR-020-SEC-011 — Image data: content-addressed storage
🟠 P1 | `[UNIT]`

**Given:** Observation với base64 image  
**When:** Image được lưu  
**Then:**
- Image stored tại `~/.agentmemory/images/{sha256}.ext`
- SHA256 content address được dùng (không predictable filename)
- Refcount tracking

**Traceability:** TDD §2.1 [9]

---

## TR-020-SEC-012 — Export: sensitive check
🟠 P1 | `[INT]`

**Given:** Memories có thể chứa sensitive terms  
**When:** Export hoặc Obsidian sync được chạy  
**Then:** Sensitive data patterns được flagged/redacted trước khi write

**Traceability:** UR-040
