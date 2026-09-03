# Change Request: CR-ENT-003 — Governance Center (GDPR + OPA + Audit)

**CR ID:** CR-ENT-003
**Component:** `backend/services/vnp-admin`, `backend/services/vnp-observability`
**Priority:** 🔴 Critical
**Status:** Open
**Version:** v5 / Enterprise & Operations
**Solution:** [S9 — Enterprise Governance](../../../bussiness/solutions/S9-governance-compliance.md)
**Features:** [F14](../../../features/14-auth-multitenancy/README.md), [F22](../../../features/22-governance-center/README.md)

---

## 1. Pain Points được giải quyết

| ID | Actor | Vấn đề |
|---|---|---|
| PP-P4-01 | Enterprise Architect | Không biết AI nhớ gì về user — no visibility |
| PP-P4-02 | Enterprise Architect | GDPR forget manual, incomplete — bỏ sót engines |
| PP-P4-03 | Enterprise Architect | Không có audit trail cho AI memory operations |

**Compliance requirement:** GDPR Art. 17 (Right to erasure) — delete trong 72h.
**Before:** Manual process, bỏ sót, không documented.
**After:** 1 click → cascading delete + immutable audit trail.

---

## 2. Governance Features

```
1. Memory Visibility Dashboard
   - Xem tất cả memories của 1 user
   - Filter by engine, type, time range
   - Search trong content

2. GDPR Forget
   - 1 API call → cascading delete 6 engines
   - Immutable audit record (xem CR-CORE-003)
   - Completion certificate (PDF export)

3. OPA Policy Enforcement
   - Policy: "Không store PII trong semantic memory"
   - Policy: "Profile memory chỉ trong EU region"
   - Violation → reject store + alert

4. Audit Trail
   - Immutable log: ai đọc/ghi memory gì, khi nào
   - Searchable: filter by user, operation, time
   - Export: CSV / JSON cho compliance review
```

---

## 3. API Contract

```http
# List all memories for user (admin)
GET /v1/console/governance/memories?user_id=u_123&tenant_id=t_456

# GDPR forget (xem CR-CORE-003)
POST /v1/admin/forget

# Get audit trail
GET /v1/console/governance/audit?user_id=u_123&from=2026-01-01&to=2026-09-03
→ {
    "events": [
      {"timestamp": "...", "operation": "store", "memory_id": "m_1", "actor": "agent_a"},
      {"timestamp": "...", "operation": "recall", "query": "project deadline", "actor": "agent_b"}
    ]
  }

# Create OPA policy
POST /v1/console/governance/policies
{
  "name": "no-pii-semantic",
  "rule": "deny if memory.type == 'semantic' and contains_pii(memory.content)"
}
```

---

## 4. Acceptance Criteria

- [ ] Memory visibility: admin có thể xem mọi memory của bất kỳ user nào trong tenant
- [ ] GDPR forget: hoàn tất `< 3s`, immutable audit record
- [ ] Audit trail: mọi store/recall/forget operation được log (immutable)
- [ ] OPA integration: policy violation → reject + log
- [ ] Compliance export: PDF/CSV audit report
- [ ] Role-based access: chỉ admin role access governance APIs
