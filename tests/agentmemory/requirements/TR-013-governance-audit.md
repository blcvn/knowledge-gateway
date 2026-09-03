# TR-013: Governance & Audit Test Requirements

**Module:** Governance, Audit, Privacy  
**Nguồn:** SRS §3.13 (FR-GOV-001..003), §8, Architecture §11, TDD §4.2, URD §3.10  
**Phiên bản:** 1.0 | **Ngày:** 2026-06-11

---

## TR-013-GOV-001 — Governance delete: cascade
🔴 P0 | `[INT]` | **FR-GOV-001**

**Given:** Memory M với obsId "obs_abc"  
**When:** `memory_governance_delete({memoryId: M.id})` được gọi  
**Then:** Cascade delete xảy ra:
- M bị xóa khỏi KV (`KV.memories`)
- M bị remove khỏi BM25 index
- M bị remove khỏi Vector index
- Related graph entries bị xóa
- Audit record được tạo

**Traceability:** FR-GOV-001, UR-016, URD §4 UR-016

---

## TR-013-GOV-002 — Governance delete: audit record
🔴 P0 | `[INT]` | **FR-GOV-001**

**Given:** `memory_governance_delete` được gọi  
**When:** Delete hoàn thành  
**Then:** AuditEntry được tạo với:
- `operation = "governance_delete"`
- `targetIds = [memoryId]`
- `timestamp` = now
- `functionId` = caller function
- `details` = reason (nếu có)

**Traceability:** FR-GOV-001, UR-019

---

## TR-013-GOV-003 — Audit trail: 40+ operation types
🟠 P1 | `[UNIT]` | **FR-GOV-002**

**Given:** Các operations xảy ra trong hệ thống  
**When:** AuditEntry được tạo  
**Then:** Hỗ trợ ít nhất 40 operation types được documented (observe, remember, forget, compress, search, consolidate, etc.)

**Traceability:** FR-GOV-002, SRS §3.13

---

## TR-013-GOV-004 — Audit log: filter theo type
🟠 P1 | `[INT]` | **FR-GOV-002**

**Given:** Mix of audit entries  
**When:** `memory_audit_log({type: "governance_delete"})` gọi  
**Then:** Chỉ trả về delete audit entries

**Traceability:** FR-GOV-002

---

## TR-013-GOV-005 — Audit log: filter theo date range
🟠 P1 | `[INT]`

**Given:** Audit entries từ nhiều ngày  
**When:** `memory_audit_log({from: "2026-06-01", to: "2026-06-10"})`  
**Then:** Chỉ entries trong date range

**Traceability:** FR-GOV-002

---

## TR-013-GOV-006 — Audit log: filter theo project
🟡 P2 | `[INT]`

**Given:** Audit entries từ nhiều projects  
**When:** Filter theo `project = "project-A"`  
**Then:** Chỉ entries liên quan đến project A

**Traceability:** FR-GOV-002, UR-019

---

## TR-013-GOV-007 — Git snapshot: enabled
🟡 P2 | `[INT]` | **FR-GOV-003**

**Given:** `SNAPSHOT_ENABLED=true`, Git repo được configure  
**When:** Snapshot interval đạt (default 3600s)  
**Then:**
- `SnapshotMeta` được tạo: `commitHash`, `stats.sessions`, `stats.observations`, `stats.memories`, `stats.graphNodes`
- Commit được tạo trong snapshot Git repo

**Traceability:** FR-GOV-003

---

## TR-013-GOV-008 — Privacy: API key redaction patterns
🔴 P0 | `[UNIT]` | **SRS §8.2**

**Given:** Raw payload JSON string  
**When:** `stripPrivateData()` xử lý  
**Then:** Các patterns sau được redact:
- `sk-ant-*` (Anthropic)
- `sk-*` (OpenAI)
- `Bearer <token>`
- `"password": "<value>"`
- Email patterns: `user@domain.com`
- Credit card: `4xxx xxxx xxxx xxxx`

**Traceability:** SRS §8.2, Architecture §11.2

---

## TR-013-GOV-009 — Privacy: redaction không ảnh hưởng structure
🔴 P0 | `[UNIT]`

**Given:** Payload với mixed sensitive + normal data  
**When:** Redaction chạy  
**Then:**
- JSON structure được preserve
- Chỉ values sensitive bị replace
- JSON vẫn parseable sau redaction

**Traceability:** Architecture §11.2

---

## TR-013-GOV-010 — mem::forget soft delete
🔴 P0 | `[INT]` | **FR-GOV-001**

**Given:** Memory M tồn tại  
**When:** `mem::forget({memoryId: M.id})`  
**Then:**
- M bị xóa (cascade như governance delete)
- Audit record tạo ra (softer operation type)
- Search không còn trả về M

**Traceability:** FR-GOV-001

---

## TR-013-GOV-011 — Obsidian export: sensitive data check
🟠 P1 | `[INT]`

**Given:** Memories có thể chứa sensitive data  
**When:** `memory_obsidian_export` được gọi  
**Then:**
- Sensitive data patterns được scan trước khi write
- Files với sensitive data được flag hoặc redacted
- Không write secrets to disk unredacted

**Traceability:** UR-040, SRS §8.2

---

## TR-013-GOV-012 — HMAC authentication: secret được set
🔴 P0 | `[INT]` | **SRS §8.1**

**Given:** `AGENTMEMORY_SECRET = "my-secret"` được set  
**When:** REST request KHÔNG có `Authorization: Bearer my-secret`  
**Then:** HTTP 401 Unauthorized

**Traceability:** SRS §8.1, UR-039, Architecture §11.1

---

## TR-013-GOV-013 — HMAC authentication: no secret (local mode)
🔴 P0 | `[INT]`

**Given:** `AGENTMEMORY_SECRET` không set  
**When:** Request đến không có auth header  
**Then:** Request được chấp nhận (local trusted mode)

**Traceability:** SRS §7.1, Architecture §11.1

---

## TR-013-GOV-014 — AuditEntry structure
🟠 P1 | `[UNIT]`

**Given:** AuditEntry được tạo  
**When:** Entry được retrieved  
**Then:** Fields đầy đủ:
```typescript
{
  id: string,
  timestamp: string,      // ISO
  operation: string,      // 40+ types
  functionId: string,
  targetIds: string[],
  details?: string,
  qualityScore?: number
}
```

**Traceability:** FR-GOV-002
