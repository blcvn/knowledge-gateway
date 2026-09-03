# TD-013: Governance & Audit Test Design

**Liên kết Requirements:** [TR-013-governance-audit.md](../requirements/TR-013-governance-audit.md)  
**Source:** `references/agentmemory/src/functions/governance.ts`, `audit.ts`  
**Test file:** `tests/agentmemory/specs/governance-audit.test.ts`  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## 1. Phạm vi kiểm thử

Governance bao gồm: audit trail, access control, retention policy, và GDPR-related memory deletion.

---

## 2. Test Cases

### Group A: Audit Trail

#### TC-001 — Mỗi `mem::remember` tạo audit record
**Requirement:** TR-013-GOV-001 | **Type:** integration | 🔴 P0

**Given:** `mem::remember` gọi thành công  
**When:** Memory được saved  
**Then:**
- Audit record tạo trong `mem:audit`
- Record có: `operation = "remember"`, `memoryId`, `timestamp`, `sessionId`

---

#### TC-002 — `mem::forget` tạo audit record
**Type:** integration | 🔴 P0

**Given:** Memory M tồn tại, `mem::forget({memoryId})` gọi  
**When:** Memory bị xóa  
**Then:**
- Audit record: `operation = "forget"`, `memoryId`, `timestamp`

---

#### TC-003 — `mem::recall` tạo access log entry
**Requirement:** TR-013-GOV-002 | **Type:** integration | 🟠 P1

**Given:** `mem::recall` gọi  
**When:** Recall hoàn thành  
**Then:** Entry trong `mem:access` với `operation = "recall"`, `sessionId`, `query`, `timestamp`

---

#### TC-004 — Audit records không bị xóa khi memory bị forget
**Requirement:** TR-013-GOV-003 | **Type:** integration | 🟠 P1

**Given:** Memory M, với audit record R về M  
**When:** M bị forget  
**Then:** R vẫn còn trong `mem:audit` (immutable audit trail)

---

### Group B: Retention Policy

#### TC-005 — Memory với `forgetAfter` trong quá khứ bị sweep
**Requirement:** TR-013-GOV-004 | **Type:** integration | 🟠 P1

**Given:**
- Memory A với `forgetAfter = now - 1 day` (đã hết hạn)
- Memory B với `forgetAfter = now + 7 days` (còn hạn)
- Memory C không có `forgetAfter`

**When:** Retention sweep chạy  
**Then:**
- Memory A bị xóa
- Memory B và C vẫn tồn tại

---

#### TC-006 — Retention sweep tạo audit records
**Type:** integration | 🟡 P2

**Given:** 3 expired memories  
**When:** Retention sweep chạy  
**Then:** 3 audit records với `operation = "retention-sweep"` được tạo

---

### Group C: Access Control

#### TC-007 — API key xác thực hợp lệ được accepted
**Requirement:** TR-013-GOV-006 | **Type:** integration | 🔴 P0

**Given:** `AGENTMEMORY_SECRET=test-key-123`, request có `Authorization: Bearer test-key-123`  
**When:** Request được xử lý  
**Then:** Request được accept, không trả về 401

---

#### TC-008 — API key không hợp lệ bị từ chối
**Type:** integration | 🔴 P0

**Given:** `AGENTMEMORY_SECRET=real-key`, request có `Authorization: Bearer wrong-key`  
**When:** Request được xử lý  
**Then:** HTTP 401 Unauthorized

---

#### TC-009 — Không có secret: local mode (không cần auth)
**Type:** integration | 🟠 P1

**Given:** `AGENTMEMORY_SECRET` không được set  
**When:** Request không có Authorization header  
**Then:** Request được accept (local/dev mode)

---

### Group D: GDPR / Data Subject Requests

#### TC-010 — `mem::purge-user` xóa tất cả data liên quan đến userId
**Requirement:** TR-013-GOV-008 | **Type:** integration | 🟠 P1

**Given:** Observations, memories, session data có `userId = "user-abc"`  
**When:** `mem::purge-user({userId: "user-abc"})`  
**Then:**
- Tất cả obs với userId bị xóa
- Tất cả memories với userId bị xóa
- Audit record về việc purge được tạo

---

#### TC-011 — User purge không ảnh hưởng data của users khác
**Type:** integration | 🔴 P0

**Given:** Data của user-A và user-B cùng tồn tại  
**When:** Purge user-A  
**Then:** Data của user-B không bị ảnh hưởng

---

## 3. Coverage Notes

- Audit trail tests cần verify KV state sau mỗi operation
- Access control tests cần HTTP endpoint (integration/e2e)
- GDPR purge là high-priority P0 test case vì liên quan compliance
