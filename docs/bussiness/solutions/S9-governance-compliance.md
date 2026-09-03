# S9 — Enterprise Governance & Compliance

> **Giải quyết Pain Points:** PP-P4-01, PP-P4-02, PP-P4-03, PP-P4-04, PP-P2-03
> **Actor chính:** P4 (Enterprise Architect), P2 (Platform Engineer)
> **Features:** F14 (Auth & Multi-tenancy), F22 (Governance Center)

---

## Vấn đề cần giải quyết

Không kiểm soát được AI đang nhớ gì, từ đâu, ai tạo. GDPR forget không thể thực thi toàn diện qua 6 engines. Không có policy enforcement. Security audit fail.

---

## Giải pháp: Enterprise Governance Stack

### 1 — Zero Cross-tenant Leakage (F14)

**Vấn đề:** 1 bug trong code → tenant A thấy data của tenant B → catastrophic.

**Giải pháp — TenantID Injection tại mọi layer:**

```
Request đến Gateway
        │
        ▼
Auth Middleware: extract TenantID từ JWT/API Key
        │
        ▼
Inject vào context: ctx.Value("tenant_id") = "tenant-abc"
        │
        ▼
MỌI service call nhận TenantID
        │
        ▼
MỌI database query:
  SELECT * FROM memories WHERE tenant_id = 'tenant-abc' AND ...
                              ^^^^^^^^^^^^^^^^^^^^^^^^^^^^
                              Không thể bỏ condition này

MỌI gRPC call có TenantID trong metadata
MỌI NATS event có TenantID trong payload
```

**Guarantee:** Không có global queries. Integration tests verify cross-tenant queries return 0 results.

**Subscription Tiers:**
| Tier | Rate Limit | Engine Access |
|---|---|---|
| `free` | Basic | Limited engines |
| `pro` | 10x higher | All engines |
| `enterprise` | Unlimited | All + custom ontology + SLA |

---

### 2 — GDPR Right to be Forgotten (F22)

**Trước:** Xóa data của 1 user = SSH vào 6 engines, manually delete, dễ bỏ sót.

**Sau — Cascading GDPR Forget:**

```http
POST /v1/console/governance/gdpr/forget/preview
{
  "user_id": "user-123",
  "tenant_id": "tenant-abc"
}
→ Dry-run: xem chính xác data nào sẽ bị xóa

Preview response:
{
  "affected": {
    "cognee_documents": 15,
    "graphiti_episodes": 43,
    "zep_sessions": 8,
    "memobase_blobs": 127,
    "memobase_profiles": 12,
    "supermemory_memories": 34,
    "openviking_files": 6
  },
  "estimated_deletion_ms": 2400
}
```

Sau khi confirm:
```http
POST /v1/console/governance/gdpr/forget
{
  "user_id": "user-123",
  "confirmation": "DELETE_ALL_DATA"
}
→ Cascading deletion across ALL 6 engines
→ Audit log: "User data deleted at 2026-09-03T11:00:00Z by admin@company.com"
```

**GDPR compliance:**
- ✅ Right to Erasure: 1 API call xóa tất cả
- ✅ Data portability: GET toàn bộ data của user
- ✅ Audit trail: evidence deletion đã hoàn thành
- ✅ 30-day deadline: automated workflow

---

### 3 — Audit Trail (F22)

Mọi memory operation đều được log:

```http
GET /v1/console/governance/audit
?actor=agent-001
&action=memory_write
&tenant=tenant-abc
&from=2026-09-01
&to=2026-09-03

Response:
[
  {
    "id": "audit-001",
    "timestamp": "2026-09-03T09:15:32Z",
    "actor": "agent-001",
    "action": "memory_write",
    "entity_type": "episodic_memory",
    "entity_id": "ep-042",
    "tenant_id": "tenant-abc",
    "engine": "graphiti",
    "result": "success"
  },
  ...
]
```

**Searchable by:** actor, action, entity, tenant, engine, time range.

**Use cases:**
- Security audit: ai đã truy cập gì?
- Incident investigation: điều gì đã xảy ra trước sự cố?
- Compliance reporting: báo cáo cho regulator

---

### 4 — OPA Policy Enforcement (F22)

Define access control rules declaratively:

```http
POST /v1/console/governance/policies
{
  "name": "no_external_agent_profile_access",
  "description": "External agents cannot read user profiles",
  "rule": {
    "resource_type": "memobase_profile",
    "action": "read",
    "condition": "agent.scope != 'internal'",
    "effect": "deny"
  }
}
```

Policies được enforce tại gateway level — agent vi phạm → 403 Forbidden + audit log.

---

### 5 — API Key Lifecycle (F14, F27)

```
Create API Key:
POST /v1/admin/tenants/{id}/keys
{
  "name": "production-agent-key",
  "expires_at": "2027-01-01T00:00:00Z",
  "rate_tier": "enterprise"
}
→ Response: {prefix: "vnp_abc12345", secret: "..."} ← Secret chỉ hiện 1 lần

Storage:
  prefix: stored plaintext (for identification)
  secret: SHA-256 hashed (never stored plaintext)

Lifecycle:
  Active → Revoked (manual)
         → Expired (auto, ExpiresAt passed)

Audit: mọi API call log prefix → biết key nào được dùng
```

---

## Checklist GDPR Compliance với VNP Memory

| Requirement | Implementation |
|---|---|
| Right to Access | `GET /v1/memobase/users/{uid}/profiles` + Memory Explorer |
| Right to Erasure | Cascading GDPR Forget (1 API call) |
| Right to Rectification | Memory update + version chain |
| Data Portability | Export all user memories |
| Privacy by Design | TenantID isolation, secret redaction, PII auto-redact |
| Records of Processing | Audit Trail (searchable) |
| Breach Notification Support | Audit log + alert system |

---

## Kết quả

| Metric | Trước | Sau |
|---|---|---|
| GDPR forget implementation | Manual, 6 engines | 1 API call, cascading |
| Cross-tenant data isolation | "Hope developer wrote it right" | TenantID injected everywhere |
| Audit trail | Không có / scattered | Searchable, centralized |
| Security audit | Fail | Pass (zero cross-tenant) |
| GDPR forget time | Hours (manual) | < 3 giây (automated) |
