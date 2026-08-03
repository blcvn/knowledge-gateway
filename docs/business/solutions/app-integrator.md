# Solutions — App Integrator

> **Actor**: App Integrator (Developer / Engineer)  
> **Pain Points nguồn**: [app-integrator.md](../painpoints/app-integrator.md)  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Phân loại giải pháp

| Ký hiệu | Ý nghĩa |
|:---:|:---|
| ✅ **Đã có** | Sản phẩm đã hỗ trợ — cần document/publicize tốt hơn |
| 🔧 **Cần bổ sung** | API skeleton có, cần hoàn thiện logic |
| 🆕 **Đề xuất mới** | Chưa có trong sản phẩm, cần phát triển |

---

## PP-AI-01 — Không có schema discovery

### ✅ Giải pháp đã có trong sản phẩm

**Domain full spec — bao gồm node type attributes**:

```bash
GET /v1/ontology/domains/{domain_id}
# → Trả về:
{
  "domain": { "id": "payment-errors", "display_name": "..." },
  "node_types": [
    {
      "name": "ErrorCode",
      "description": "...",
      "attributes": [
        { "name": "code", "type": "string", "required": true },
        { "name": "severity", "type": "enum", "values": ["critical","high","medium","low"], "required": true }
      ]
    }
  ],
  "rel_types": [...],
  "query_templates": [...],
  "status_config": {...}
}
```

**Validation tại write time** — API trả về `VALIDATION_FAILED` với thông tin field:

```bash
POST /v1/kg/write/nodes
{ "node_type": "ErrorCode", "attributes": { "severity": "critical-high" } }
# → 422 VALIDATION_FAILED: "attribute 'severity' invalid value 'critical-high'"
```

**Effective ontology** — xem tất cả visible domains:

```bash
GET /v1/tenants/{tenant_id}/ontology/effective
# → All domains + node types accessible to caller
```

**Xem thêm**: [API Reference — Ontology](../../api/README.md#ontology), [Troubleshooting — Validation Fails](../../guides/troubleshooting.md#validation-fails)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Node type JSON Schema endpoint**

```bash
GET /v1/ontology/domains/{domain_id}/node-types/{type}/schema
# → JSON Schema draft-07 format — máy đọc được, validate trước khi gọi API:
{
  "$schema": "http://json-schema.org/draft-07/schema",
  "title": "ErrorCode",
  "description": "Represents a payment error code",
  "type": "object",
  "required": ["code", "severity"],
  "properties": {
    "code": { "type": "string", "description": "Error code identifier (e.g., E001)", "example": "E001" },
    "severity": { "type": "string", "enum": ["critical","high","medium","low"] },
    "description": { "type": "string" }
  }
}
```

**2. Write request validation endpoint (dry-run)**

```bash
POST /v1/kg/write/nodes/validate
{
  "node_type": "ErrorCode",
  "domain_id": "payment-errors",
  "attributes": { "code": "E999", "severity": "critical-high" }
}
# → { "valid": false, "errors": [
#     { "field": "attributes.severity", "message": "must be one of: critical, high, medium, low", "provided": "critical-high" }
#   ] }
# Validate TRƯỚC khi gửi real request
```

---

## PP-AI-02 — Projection lag không có signal

### ✅ Giải pháp đã có trong sản phẩm

**realtime mode** — giải quyết projection lag khi đọc:

```bash
GET /v1/kg/read/nodes/{id}?mode=realtime&app_id={app_id}
# realtime: so sánh graph sync version với PostgreSQL
# Nếu graph stale → fallback về PostgreSQL (relationshipdb)
# → Luôn trả về data mới nhất dù projection chưa hoàn tất

POST /v1/kg/read/template/{domain_id}/{template_name}
{ "params": {...}, "mode": "realtime" }
```

**Write flow → 202 Accepted** — service đã signal rõ async:

```bash
POST /v1/kg/write/nodes
# → 202 Accepted (không phải 200 OK)
# Response chứa node_id để track
```

**Integrity check** — biết projection state:

```bash
GET /v1/kg/integrity/tenant/{tenant_id}
# → drift counts, missing projections
```

**Metrics endpoint** — projection lag:

```bash
GET /v1/kg/metrics
# → projection lag, worker queue depth, realtime fallback counters
```

**Xem thêm**: [Integration Workflows — Write And Read](../../guides/integration.md#4-write-data), [API Reference — Read behavior notes](../../api/README.md#knowledge-read-and-search)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Projection status per node**

```bash
GET /v1/kg/read/nodes/{id}/projection-status
# → {
#     "node_id": "...",
#     "postgres_version": 42,
#     "graph_version": 40,
#     "vector_version": 41,
#     "graph_lag": 2,  # versions behind
#     "fully_projected": false,
#     "estimated_sync_ms": 1500
#   }
# App integrator biết chính xác khi nào data available để read
```

**2. Write với wait-for-projection option**

```bash
POST /v1/kg/write/nodes?wait_for_projection=true&timeout_ms=5000
{ "node_type": "...", "attributes": {...} }
# → Block tối đa 5s cho đến khi node được project vào graph/vector
# → 200 OK khi projection hoàn tất, hoặc 202 Accepted khi timeout
```

**3. Bulk write với job tracking**

Đã có `POST /v1/kg/write/ingest/document` trả về `job_id`. Extend để hỗ trợ generic bulk node write:

```bash
POST /v1/kg/write/nodes/bulk
[
  { "node_type": "ErrorCode", "attributes": {...} },
  { "node_type": "ErrorCode", "attributes": {...} }
]
# → 202 Accepted: { "job_id": "job-abc-123", "queued_count": 2 }

GET /v1/kg/write/ingest/jobs/job-abc-123
# → { "status": "completed", "created_count": 2, "errors": [] }
```

---

## PP-AI-03 — Auth model khác convention — dễ làm sai

### ✅ Giải pháp đã có trong sản phẩm

**Identity resolution endpoint — "Who am I?"**:

```bash
GET /v1/access/resolve
# → { "tenant_id": "payment-team", "app_id": "payment-service", "visible_owners": [...] }
# Dùng ngay khi nhận API key để verify identity
```

**Identity sanitization** — middleware strip `tenant_id`/`app_id` từ request body:

> Đây là security feature: middleware strips caller-supplied `tenant_id` and `app_id` fields from JSON bodies before handler execution.

**Rõ ràng trong docs**: [Quickstart — Bootstrap Caveats](../../guides/quickstart.md#bootstrap-caveats):
> "Caller-supplied `tenant_id` and `app_id` fields in JSON bodies are ignored by middleware; identity comes from the API key."

**Xem thêm**: [Integration Workflows — Authenticate First](../../guides/integration.md#1-authenticate-first), [Troubleshooting — Authentication Fails](../../guides/troubleshooting.md#authentication-fails)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Richer `/v1/access/me` endpoint**

```bash
GET /v1/access/me  # alias friendly hơn cho /v1/access/resolve
# → {
#     "tenant_id": "payment-team",
#     "app_id": "payment-service",
#     "key_id": "key-xyz",
#     "visible_domains": ["payment", "risk"],
#     "permissions": ["read", "write"],
#     "rate_limit": { "tier": "standard", "rpm": 1000, "remaining": 856 }
#   }
```

**2. SDK encapsulation**

```python
# Thay vì developer phải biết auth header pattern:
client = KGClient(api_key="kgsk_...", base_url="http://...")

# Identity auto-resolved từ key:
me = client.whoami()
# → { tenant_id, app_id, visible_domains }

# Write không cần truyền tenant/app:
node = client.nodes.create(
    domain="payment-errors",
    node_type="ErrorCode",
    attributes={ "code": "E001", "severity": "high" }
)
```

**3. Clear onboarding error message**

Khi request thiếu Authorization header, thay vì generic 401:
```json
{
  "error": {
    "code": "INVALID_API_KEY",
    "message": "Authorization header required. KG Service uses API key bearer authentication.",
    "fix": "Add header: Authorization: Bearer <your_api_key>",
    "docs": "https://docs/guides/quickstart.md#use-a-bootstrap-api-key"
  }
}
```

---

## PP-AI-04 — Named templates là black box

### ✅ Giải pháp đã có trong sản phẩm

**List active templates**:

```bash
GET /v1/kg/read/templates?domain_id={domain_id}&limit=20
# → List of active templates với metadata
```

**Full domain spec bao gồm templates**:

```bash
GET /v1/ontology/domains/{domain_id}
# → query_templates: [{ "name": "errors-by-severity", "description": "...", "parameters": [...] }]
```

**Execute template**:

```bash
POST /v1/kg/read/template/{domain_id}/{template_name}
{ "params": { "severity": "high" }, "mode": "realtime" }
```

**Bootstrap sample templates** để tham khảo syntax và pattern:

```bash
# Sample templates trong domain sample-policy:
# - action-guide (params: topic_key)
# - topic-routing
# - reference-check
# - obligation-summary
# - schedule-trace
```

**Xem thêm**: [Integration Workflows — Read And Search](../../guides/integration.md#5-read-and-search-data)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Template detail endpoint**

```bash
GET /v1/kg/read/templates/{domain_id}/{template_name}/spec
# → {
#     "name": "errors-by-severity",
#     "description": "List error codes filtered by severity level",
#     "parameters": [
#       { "name": "severity", "type": "string", "required": true, "enum": ["critical","high","medium","low"] }
#     ],
#     "output_schema": {
#       "type": "array",
#       "items": { "$ref": "#/definitions/ErrorCode" }
#     },
#     "example_request": { "params": { "severity": "high" } },
#     "example_response": [ { "id": "...", "node_type": "ErrorCode", "attributes": {...} } ]
#   }
```

**2. Template search**

```bash
GET /v1/kg/read/templates?q=error&domain_id=payment-errors
# → Full-text search trong tên và description của templates
# → Tìm template phù hợp mà không cần biết exact name
```

---

## PP-AI-05 — Không có offline development mode

### ✅ Giải pháp đã có trong sản phẩm

**Memory adapter mode** — đã có sẵn:

```bash
# environment.md: GRAPH_ADAPTER=memory, VECTOR_ADAPTER=memory, FTS_ADAPTER=memory
# Chạy kg-service với full-memory backends — zero external dependencies:
GRAPH_ADAPTER=memory VECTOR_ADAPTER=memory FTS_ADAPTER=memory \
EMBEDDING_PROVIDER=deterministic \
./kg-service
```

**Integration smoke stack** — Docker Compose minimal:

```bash
make deploy-compose-integration
# → Chỉ cần Postgres + Redis + kg-service (memory graph/vector)
# → Không cần Neo4j, Qdrant, hay Memgraph
```

**Bootstrap seed** — sample data sẵn có để test ngay:

```bash
# Sau khi chạy make run, các seeded keys đã ready:
KG_API_KEY=kgsk_test_alpha_admin
# Domain sample-policy đã có data + templates
```

**Xem thêm**: [Testing Guide — Unit And Package Tests](../../guides/testing.md#1-unit-and-package-tests), [Deployment — Docker Compose](../../deployment/compose.md)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Official test SDK**

```python
# Hiện tại: developer tự mock hoặc dùng real service
# Đề xuất: official KGClientMock

from kg_sdk.testing import KGClientMock

def test_payment_integration():
    client = KGClientMock(seed_domain="payment-errors")
    
    node_id = client.nodes.create(
        domain="payment-errors",
        node_type="ErrorCode",
        attributes={ "code": "E001", "severity": "high" }
    )
    
    result = client.templates.execute("payment-errors", "errors-by-severity",
                                      params={"severity": "high"})
    assert len(result) == 1
    assert result[0]["attributes"]["code"] == "E001"
```

**2. Local dev Compose profile**

```bash
# Compose profile chỉ cần Postgres + Redis, không cần external graph/vector:
docker compose --profile local-dev up
# → kg-service với GRAPH_ADAPTER=memory VECTOR_ADAPTER=memory
# → RAM usage: ~300MB thay vì 4–8GB với full stack
```

---

## PP-AI-06 — Error messages không actionable

### ✅ Giải pháp đã có trong sản phẩm

**Error envelope format** — structured error responses:

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Human-readable message",
    "details": {}
  }
}
```

**Common error codes documented**:
- `BAD_REQUEST`, `INVALID_API_KEY`, `FORBIDDEN`, `NOT_FOUND`
- `VALIDATION_FAILED`, `TOO_MANY_REQUESTS`, `REQUEST_TIMEOUT`, `INTERNAL_ERROR`

**Troubleshooting guide** — common issues và fixes:  
[Troubleshooting Guide](../../guides/troubleshooting.md) covers: auth fails, malformed JSON, validation fails, forbidden access, missing resource, wrong search results.

**Xem thêm**: [API Reference — Error Envelope](../../api/README.md#error-envelope)

---

### 🆕 Giải pháp đề xuất bổ sung

**1. Enriched VALIDATION_FAILED error**

```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Node attribute 'severity' has invalid value 'critical-high'",
    "details": {
      "field": "attributes.severity",
      "provided": "critical-high",
      "allowed_values": ["critical", "high", "medium", "low"],
      "ontology_ref": "GET /v1/ontology/domains/payment-errors/node-types/ErrorCode/schema"
    },
    "fix_hint": "Use one of the allowed severity values: critical, high, medium, low"
  }
}
```

**2. Request ID trong mọi response**

```http
HTTP/1.1 422 Unprocessable Entity
X-Request-ID: req-abc-123def
Content-Type: application/json

{ "error": { "code": "VALIDATION_FAILED", ... } }
```

Integrator paste `X-Request-ID` vào support ticket → Operator lookup trong logs ngay lập tức.

**3. Error docs endpoint**

```bash
GET /v1/docs/errors/VALIDATION_FAILED
# → {
#     "code": "VALIDATION_FAILED",
#     "description": "Request body failed schema validation",
#     "http_status": 422,
#     "common_causes": [
#       "Missing required attributes",
#       "Attribute value not in allowed enum",
#       "Ontology node type does not exist"
#     ],
#     "resolution_steps": [...],
#     "related_docs": ["../guides/integration.md#model-ontology-before-writing-data"]
#   }
```

---

## Summary — App Integrator Solutions

| Pain Point | Đã có | Đề xuất mới | Priority |
|:---|:---:|:---:|:---:|
| PP-AI-01: Không có schema discovery | ✅ GET /v1/ontology/domains/{id} | 🆕 JSON Schema endpoint + write validate | 🔴 P0 |
| PP-AI-02: Projection lag không có signal | ✅ realtime mode + 202 Accepted | 🆕 Projection status API + wait_for option | 🔴 P0 |
| PP-AI-03: Auth model khác convention | ✅ /v1/access/resolve + docs | 🆕 /v1/access/me + SDK + rich error | 🔴 P0 |
| PP-AI-04: Templates là black box | ✅ GET /v1/kg/read/templates + domain spec | 🆕 Template spec + example response | 🟠 P1 |
| PP-AI-05: Không có offline mode | ✅ GRAPH_ADAPTER=memory + compose-integration | 🆕 Test SDK + local-dev Compose profile | 🟠 P1 |
| PP-AI-06: Error messages không actionable | ✅ Error envelope + troubleshooting guide | 🆕 Enriched errors + fix_hint + request ID | 🟠 P1 |
