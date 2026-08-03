# Solutions — Tenant Admin

> **Actor**: Tenant Admin  
> **Pain Points nguồn**: [tenant-admin.md](../painpoints/tenant-admin.md)  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Phân loại giải pháp

| Ký hiệu | Ý nghĩa |
|:---:|:---|
| ✅ **Đã có** | Sản phẩm đã hỗ trợ — cần document/publicize tốt hơn |
| 🔧 **Cần bổ sung** | API skeleton có, cần hoàn thiện logic |
| 🆕 **Đề xuất mới** | Chưa có trong sản phẩm, cần phát triển |

---

## PP-TA-01 — Không có công cụ hỗ trợ thiết kế ontology domain

### ✅ Giải pháp đã có trong sản phẩm

**Ontology definition API — đầy đủ CRUD**:

```bash
# 1. Tạo domain
POST /v1/tenants/{tenant_id}/ontology/domains
{ "domain_id": "payment-errors", "display_name": "Payment Error Codes", "description": "..." }

# 2. Thêm node types
POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/node-types
{
  "name": "ErrorCode",
  "description": "Represents a payment error code",
  "attributes": [
    { "name": "code", "type": "string", "required": true },
    { "name": "severity", "type": "enum", "values": ["critical","high","medium","low"], "required": true },
    { "name": "description", "type": "string" }
  ]
}

# 3. Thêm relationship types
POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/rel-types
{
  "name": "TRIGGERS",
  "from_types": ["PaymentFlow"],
  "to_types": ["ErrorCode"],
  "description": "PaymentFlow triggers this error"
}

# 4. Xem effective ontology
GET /v1/tenants/{tenant_id}/ontology/effective
GET /v1/ontology/domains/{domain_id}  # node types, rel types, templates, status config
```

**Xem thêm**: [Integration Workflows — Model Ontology](../../guides/integration.md#3-model-ontology-before-writing-data)

**Sample domain để tham khảo pattern**:

Bootstrap seed có sẵn `sample-policy` domain với templates hoạt động được — có thể dùng làm reference:

```bash
GET /v1/ontology/domains/sample-policy
# → Full ontology structure để Tenant Admin tham khảo pattern
```

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Ontology starter templates**

```bash
GET /v1/ontology/starter-templates
# → List: "software-requirements", "product-backlog", "compliance-rules", "incident-management"

GET /v1/ontology/starter-templates/software-requirements
# → Full domain definition JSON có thể fork và customize:
{
  "domain_id": "requirements",
  "node_types": [
    { "name": "Requirement", "attributes": [...] },
    { "name": "UserStory", "attributes": [...] },
    { "name": "TestCase", "attributes": [...] }
  ],
  "rel_types": [
    { "name": "BREAKS_DOWN_TO", "from": "Requirement", "to": "UserStory" },
    ...
  ],
  "query_templates": [...]
}

# Fork template vào tenant:
POST /v1/tenants/{tenant_id}/ontology/domains/from-template
{ "template_name": "software-requirements", "domain_id": "my-requirements" }
```

**2. Ontology linter/validator**

```bash
POST /v1/ontology/validate
{
  "node_types": [...],
  "rel_types": [...]
}
# → { "valid": false, "issues": [
#     { "severity": "error", "message": "Relationship type TRIGGERS references undefined node type 'PaymentFlow'" },
#     { "severity": "warning", "message": "Node type 'ErrorCode' has no relationships defined — will be orphan" }
#   ] }
```

**3. AI-assisted ontology generation** *(LLM integration)*

```bash
POST /v1/ontology/ai-suggest
{
  "description": "Tôi muốn model payment error handling — gồm error codes, payment flows, và severity levels",
  "sample_data": "E001: QR timeout, severity: high\nE002: Insufficient balance, severity: medium"
}
# → Suggest initial ontology + explanation tại sao mỗi node type được chọn
```

---

## PP-TA-02 — Query template lifecycle phức tạp — không có preview/versioning

### ✅ Giải pháp đã có trong sản phẩm

**Template registration và activation**:

```bash
# Tạo template
POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates
{
  "name": "errors-by-severity",
  "description": "List error codes filtered by severity level",
  "query": "MATCH (e:ErrorCode {severity: $severity}) RETURN e",
  "parameters": [{ "name": "severity", "type": "string", "required": true }]
}

# Activate template (bắt buộc trước khi dùng)
PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate

# List active templates
GET /v1/kg/read/templates?domain_id={domain_id}

# Execute template
POST /v1/kg/read/template/{domain_id}/{template_name}
{ "params": { "severity": "high" } }
```

**realtime mode** để kiểm tra freshness của data khi test template:

```bash
POST /v1/kg/read/template/{domain_id}/{template_name}
{ "params": {...}, "mode": "realtime" }
# realtime so sánh graph sync version với PostgreSQL, fallback nếu stale
```

**Xem thêm**: [API Reference — Knowledge Read And Search](../../api/README.md#knowledge-read-and-search)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Template preview mode (không cần activate)**

```bash
POST /v1/ontology/templates/preview
{
  "query": "MATCH (e:ErrorCode) WHERE e.severity = $severity RETURN e LIMIT 10",
  "parameters": { "severity": "high" },
  "domain_id": "payment-errors",
  "tenant_id": "payment-team"
}
# → Execute với sample data trong domain, trả về kết quả KHÔNG activate template
# → { "results": [...], "execution_ms": 45, "node_count": 3 }
```

**2. Template versioning**

```bash
# Create new version của template đã active:
POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/versions
{ "query": "...", "changelog": "Added filter for deleted nodes" }
# → { "version": 2, "status": "draft" }

# Activate specific version (gradual rollout):
PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate
{ "version": 2 }

# Rollback:
PUT /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/activate
{ "version": 1 }
```

**3. Template usage insight**

```bash
GET /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/query-templates/{name}/stats
# → { "total_calls_30d": 15000, "avg_latency_ms": 45, "last_called": "2026-08-03T..." }
# Biết template nào được dùng nhiều → tránh breaking changes
```

---

## PP-TA-03 — Không có visibility vào effective access của apps trong tenant

### ✅ Giải pháp đã có trong sản phẩm

**Access resolve — per-app check**:

```bash
# Kiểm tra visibility của từng app:
GET /v1/access/resolve  # dùng app's API key
# → { "tenant_id": "...", "app_id": "...", "visible_owners": [...] }

# List all grants của tenant:
GET /v1/access/grants?grantor_tenant_id={tenant_id}
GET /v1/access/grants?grantee_tenant_id={tenant_id}

# Access audit:
GET /v1/access/audit?resource_owner_tenant_id={tenant_id}
```

**Effective ontology view**:

```bash
GET /v1/tenants/{tenant_id}/ontology/effective
# → Tất cả domains mà tenant đang thấy (own + granted)
```

**Xem thêm**: [Troubleshooting — Forbidden Access](../../guides/troubleshooting.md#forbidden-access), [Grant Incident Response](../../operations/grant-incident-response.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Tenant access summary — one API call**

```bash
GET /v1/tenants/{tenant_id}/access/summary
# Đề xuất response:
{
  "tenant_id": "payment-team",
  "apps": [
    {
      "app_id": "payment-service",
      "visible_domains": ["payment", "risk"],
      "granted_from": ["compliance-team"],
      "grants_given_to": []
    },
    {
      "app_id": "payment-dashboard",
      "visible_domains": ["payment"],
      "granted_from": [],
      "grants_given_to": [{ "tenant": "audit-firm", "domains": ["payment"], "expires": "2026-12-31" }]
    }
  ],
  "last_grant_change": "2026-07-15T..."
}
```

**2. Access simulation**

```bash
POST /v1/access/simulate
{
  "action": "revoke_grant",
  "grant_id": "grant-abc-123"
}
# → { "impact": [
#     { "app_id": "payment-service", "loses_access_to": ["risk"] },
#     { "app_id": "payment-dashboard", "loses_access_to": [] }
#   ] }
# Biết impact TRƯỚC khi revoke → không gây outage bất ngờ
```

---

## PP-TA-04 — Không có tooling cho lifecycle rules

### ✅ Giải pháp đã có trong sản phẩm

**Status field configuration API**:

```bash
POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/status-field-config
{
  "status_field": "status",
  "node_type": "Requirement",
  "allowed_values": ["draft", "in-review", "approved", "deprecated"],
  "terminal_values": ["deprecated"],
  "cascade_on_terminal": true,
  "authority_field": "owner"
}
```

**Xem thêm**: [Integration Workflows — Model Ontology](../../guides/integration.md#3-model-ontology-before-writing-data)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Transition rules definition**

```bash
POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/lifecycle-rules
{
  "node_type": "Requirement",
  "transitions": [
    { "from": "draft", "to": ["in-review"], "requires_role": "author" },
    { "from": "in-review", "to": ["approved", "draft"], "requires_role": "reviewer" },
    { "from": "approved", "to": ["deprecated"], "requires_role": "admin" }
  ],
  "forbidden_transitions": [
    { "from": "deprecated", "to": "*", "reason": "Terminal state — cannot be reactivated" }
  ]
}
```

**2. Lifecycle validation endpoint**

```bash
POST /v1/ontology/lifecycle/validate-transition
{
  "domain_id": "payment",
  "node_type": "Requirement",
  "current_status": "deprecated",
  "target_status": "draft"
}
# → { "valid": false, "reason": "deprecated is a terminal state" }
```

---

## PP-TA-05 — Onboard app mới phải support thủ công

### ✅ Giải pháp đã có trong sản phẩm

**Auto-generated domain documentation**:

```bash
GET /v1/ontology/domains/{domain_id}
# → Full domain spec: node types, attributes, relationship types, active templates, search profile
# App integrator có thể tự đọc để biết domain structure
```

**Search profile** giúp integrators biết search capabilities của domain:

```bash
GET /v1/ontology/domains/{domain_id}/search-profile
# → { "embedding_model": "...", "indexed_fields": [...], "hybrid_search": true }
```

**Xem thêm**: [Integration Workflows](../../guides/integration.md), [MCP Integration](../../guides/mcp.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Domain documentation endpoint — human-readable**

```bash
GET /v1/ontology/domains/{domain_id}/docs
Accept: text/markdown
# → Trả về Markdown documentation tự động generate từ ontology definition:

# Domain: Payment Errors
# Node Types:
# - ErrorCode: Represents a payment error code
#   - code (string, required): Error code identifier (e.g., E001)
#   - severity (enum: critical|high|medium|low, required)
#   ...
# Available Templates:
# - errors-by-severity: List error codes by severity level
#   Parameters: severity (string, required)
#   Example: POST /v1/kg/read/template/payment-errors/errors-by-severity
#             { "params": { "severity": "high" } }
```

**2. Sandbox environment per domain**

```bash
POST /v1/tenants/{tenant_id}/ontology/domains/{domain_id}/sandbox/enable
# → App integrator có thể write/read vào sandbox namespace mà không ảnh hưởng production data
# → Sandbox data tự động expire sau 7 ngày
```

---

## Summary — Tenant Admin Solutions

| Pain Point | Đã có | Đề xuất mới | Priority |
|:---|:---:|:---:|:---:|
| PP-TA-01: Không có hỗ trợ thiết kế ontology | ✅ Full ontology CRUD API | 🆕 Starter templates + AI suggest | 🔴 P0 |
| PP-TA-02: Template lifecycle thiếu preview | ✅ Register + activate flow | 🆕 Preview mode + versioning | 🔴 P0 |
| PP-TA-03: Không có visibility vào effective access | ✅ access/resolve + grants API | 🆕 /access/summary + simulation | 🔴 P0 |
| PP-TA-04: Lifecycle rules thiếu tooling | ✅ status-field-config API | 🆕 Transition rules + validation | 🟡 P2 |
| PP-TA-05: Onboard app mới thủ công | ✅ GET /v1/ontology/domains/{id} | 🆕 /docs endpoint + sandbox | 🟡 P2 |
