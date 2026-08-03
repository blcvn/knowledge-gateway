# Solutions — Platform Admin

> **Actor**: Platform Admin  
> **Pain Points nguồn**: [platform-admin.md](../painpoints/platform-admin.md)  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Tổng quan phương pháp

Mỗi giải pháp được phân loại rõ:

| Ký hiệu | Ý nghĩa |
|:---:|:---|
| ✅ **Đã có** | Sản phẩm đã hỗ trợ — cần document/publicize tốt hơn |
| 🔧 **Cần bổ sung** | API endpoint/feature đã có skeleton, cần hoàn thiện |
| 🆕 **Đề xuất mới** | Chưa có trong sản phẩm, cần phát triển |

---

## PP-PA-01 — Không có tenant/app management portal — mọi thao tác qua raw API calls

### ✅ Giải pháp đã có trong sản phẩm

**REST API đầy đủ cho tenant/app lifecycle**:

kg-service đã có toàn bộ REST API để quản lý tenant và app:

```bash
# Tạo tenant
POST /v1/tenants
{ "slug": "payment-team", "name": "Payment Team", "tier": "standard" }

# Tạo app dưới tenant
POST /v1/tenants/{tenant_id}/apps
{ "name": "payment-service" }
# → Response chứa api_key (chỉ hiển thị lần này)

# Xem danh sách apps
GET /v1/tenants/{tenant_id}/apps?limit=20

# Xem tenant record
GET /v1/tenants/{tenant_id}

# Verify identity sau khi tạo
GET /v1/access/resolve  # dùng api_key vừa nhận
```

**Access resolve — verification step built-in**:

```bash
curl -H "Authorization: Bearer ${NEW_APP_API_KEY}" \
  "${KG_BASE_URL}/v1/access/resolve"
# → { "tenant_id": "...", "app_id": "...", "visible_owners": [...] }
```

**Audit trail qua access audit API**:

```bash
GET /v1/access/audit?resource_owner_tenant_id={tenant_id}&limit=50
```

**Xem thêm**: [Integration Workflows — Onboard A Tenant](../../guides/integration.md#2-onboard-a-tenant-and-app), [Tenant And App Setup](../../guides/tenant-app-setup.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Admin CLI — `kg-admin`**

Wrapper CLI tạo ra từ REST API, giúp admin thao tác nhanh mà không cần nhớ API format:

```bash
# Thay vì 4 curl calls:
kg-admin tenant create --slug payment-team --name "Payment Team"
kg-admin app provision --tenant payment-team --name payment-service
kg-admin access verify --app payment-service
kg-admin grants list --tenant payment-team

# Bulk provisioning từ file YAML:
kg-admin provision --from-file tenants.yaml
```

Format file `tenants.yaml`:
```yaml
tenants:
  - slug: payment-team
    name: Payment Team
    apps:
      - name: payment-service
      - name: payment-dashboard
    grants:
      - from: payment-team
        to: compliance-team
        domains: [payment, risk]
```

**2. Secure key delivery channel**

Hiện tại api_key chỉ xuất hiện một lần trong response → cần mechanism an toàn hơn:
- Ghi key vào encrypted vault (Vault/AWS Secrets Manager) tự động sau `app create`
- Hoặc: webhook delivery khi app provisioned

---

## PP-PA-02 — Key rotation và revocation không có automated rollout

### ✅ Giải pháp đã có trong sản phẩm

**Key rotation endpoint**:

```bash
POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key
# → { "api_key": "<new_plaintext_key>", "rotated_at": "2026-08-03T..." }
```

**App revocation (delete)**:

```bash
DELETE /v1/tenants/{tenant_id}/apps/{app_id}
# → { "id": "...", "status": "revoked", "revoked_at": "..." }
```

**Lưu ý quan trọng từ docs**: 
> "The app create and rotate-key flows are the only times plaintext API keys are returned."

**Xem thêm**: [API Reference — Tenant Management](../../api/README.md#access-and-tenant-management), [API Key Revocation Response](../../operations/api-key-revocation-response.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Dual-key grace period rotation**

Thêm parameter `grace_period_hours` vào rotate-key để tránh downtime:

```bash
POST /v1/tenants/{tenant_id}/apps/{app_id}/rotate-key
{ "grace_period_hours": 24 }
# → { "new_api_key": "...", "old_key_expires_at": "2026-08-04T..." }
# Cả hai key valid trong 24h để consumer có thời gian update
```

**2. Key expiry policy**

```bash
PUT /v1/tenants/{tenant_id}/apps/{app_id}/key-policy
{ "expiry_days": 90, "auto_rotate": true, "notify_days_before": 7 }
```

**3. Inactivity alert**

Background job check key inactivity → alert khi key không được dùng trong N ngày:
```
GET /v1/admin/keys/inactive?days=30
→ [{ "app_id": "...", "last_used": null, "created": "2026-01-01" }]
```

---

## PP-PA-03 — Cross-tenant access grants không có policy template

### ✅ Giải pháp đã có trong sản phẩm

**Grant CRUD đầy đủ**:

```bash
# Tạo grant
POST /v1/access/grants
{
  "grantor_tenant_id": "payment-team",
  "grantee_tenant_id": "compliance-team",
  "domains": ["payment", "risk"],
  "permissions": ["read"]
}

# Xem grants hiện có
GET /v1/access/grants?grantor_tenant_id=payment-team

# Revoke grant
DELETE /v1/access/grants/{grant_id}

# Review access outcomes
GET /v1/access/audit?resource_owner_tenant_id=payment-team
```

**Verify sau khi cấu hình grant**:

```bash
# Dùng grantee app key để verify visibility
curl -H "Authorization: Bearer ${COMPLIANCE_APP_KEY}" \
  "${KG_BASE_URL}/v1/access/resolve"
# → visible_owners nên bao gồm payment-team
```

**Xem thêm**: [Integration Workflows — Share Access](../../guides/integration.md#6-share-access-across-tenants-or-apps), [Grant Incident Response](../../operations/grant-incident-response.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Grant templates**

```bash
POST /v1/access/grant-templates
{
  "name": "compliance-read-all-risk",
  "description": "Compliance team đọc được mọi risk-related domains",
  "grantee_tenant_pattern": "compliance-*",
  "domain_patterns": ["risk", "audit", "compliance"],
  "permissions": ["read"]
}

# Apply template
POST /v1/access/grants/from-template
{ "template_name": "compliance-read-all-risk", "grantee_tenant_id": "compliance-team" }
```

**2. Grant expiry**

```bash
POST /v1/access/grants
{
  "grantee_tenant_id": "audit-firm",
  "domains": ["payment"],
  "expires_at": "2026-12-31T23:59:59Z"  # ← tạm thời cho audit
}
```

**3. Grant review flow**

```bash
GET /v1/access/grants/review-candidates?older_than_days=180
# → Grants chưa được verify trong 6 tháng → cần Platform Admin review
```

---

## PP-PA-04 — Service health không đủ granular

### ✅ Giải pháp đã có trong sản phẩm

**Health endpoint** (`/healthz` — public):

```bash
GET /healthz
# Response hiện tại:
{
  "service": "ok",
  "postgres": "ok",
  "redis": "ok"
}
```

**Metrics endpoint** (projection và worker):

```bash
GET /v1/kg/metrics
# → worker và projection lag metrics, realtime fallback counters, graph scope conflict counters
```

**Integrity checks**:

```bash
GET /v1/kg/integrity/tenant/{tenant_id}       # tenant-level drift
GET /v1/kg/integrity/missing-bridges?tenant_id=X  # bridge gaps
GET /v1/kg/integrity/orphans?tenant_id=X          # orphaned data
```

**Xem thêm**: [Reconciliation Incident Handling](../../operations/reconciliation-incident-handling.md), [Replica Recovery](../../operations/replica-recovery.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Detailed health với per-subsystem status**

```bash
GET /v1/health/detailed
# Đề xuất response:
{
  "service": "degraded",
  "subsystems": {
    "postgres": { "status": "ok", "latency_ms": 2 },
    "redis": { "status": "ok", "latency_ms": 1 },
    "graph_backend": { "status": "degraded", "latency_ms": 850, "adapter": "memgraph" },
    "vector_backend": { "status": "ok", "latency_ms": 45, "adapter": "qdrant" },
    "projection_worker": { "status": "ok", "queue_depth": 12, "lag_ms": 340 }
  },
  "capabilities": {
    "write": "available",
    "graph_read": "degraded",
    "vector_search": "available"
  }
}
```

**2. Prometheus metrics export**

```bash
GET /v1/kg/metrics/prometheus
# → text/plain Prometheus exposition format
# Cho phép scrape vào Grafana/Prometheus stack
```

---

## PP-PA-05 — Không có usage visibility

### ✅ Giải pháp đã có trong sản phẩm

**Metrics endpoint** (hiện tại tập trung vào projection):

```bash
GET /v1/kg/metrics
# → projection lag, worker metrics, graph scope conflict counters
```

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Usage metrics per tenant**

```bash
GET /v1/admin/usage?tenant_id={t}&period=30d
# Đề xuất response:
{
  "tenant_id": "payment-team",
  "period": { "from": "2026-07-04", "to": "2026-08-03" },
  "nodes": { "created": 1250, "updated": 340, "deleted": 12, "total_active": 8923 },
  "relationships": { "created": 4500, "total_active": 31200 },
  "api_calls": { "reads": 45000, "writes": 1600, "searches": 23000 },
  "top_templates": [
    { "name": "action-guide", "calls": 15000 },
    { "name": "requirements-by-status", "calls": 8200 }
  ],
  "storage_estimate_mb": 245
}
```

**2. Inactivity detection**

```bash
GET /v1/admin/apps/inactive?days=30
# → Apps không có API call trong 30 ngày → candidate để cleanup
```

---

## Summary — Platform Admin Solutions

| Pain Point | Đã có | Đề xuất mới | Priority |
|:---|:---:|:---:|:---:|
| PP-PA-01: Không có admin portal | ✅ Full REST API | 🆕 kg-admin CLI + bulk provisioning | 🔴 P0 |
| PP-PA-02: Key rotation không automated | ✅ rotate-key endpoint | 🆕 Grace period + expiry policy | 🔴 P0 |
| PP-PA-03: Cross-tenant grants thủ công | ✅ Full grants CRUD | 🆕 Grant templates + expiry | 🟠 P1 |
| PP-PA-04: Health không granular | ✅ /healthz + /v1/kg/metrics | 🆕 /v1/health/detailed + Prometheus | 🟠 P1 |
| PP-PA-05: Không có usage visibility | 🔧 Basic metrics | 🆕 Usage API per tenant | 🟡 P2 |
