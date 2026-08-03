# Pain Points — App Integrator

> **Actor**: App Integrator  
> **Phạm vi**: Developer / Engineer — sử dụng kg-service API để write nodes/relationships, execute templates, search knowledge  
> **Loại**: Developer Experience (DX) pain points  
> **Phiên bản**: 1.0.0 | Ngày tạo: 2026-08-03

---

## Tổng quan

App Integrator là người **trực tiếp gọi API** của kg-service từ code — viết nodes, đọc knowledge, chạy templates, và consume search. Đây là user group lớn nhất và có daily interaction cao nhất với service.

Nếu DX của App Integrator kém → adoption thấp → service không được dùng → toàn bộ investment vào knowledge graph bị lãng phí.

---

## Pain Points chi tiết

### PP-AI-01 — Phải tự biết schema của node trước khi write — không có schema discovery

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi lần write node thuộc loại mới  

**Mô tả**:  
Khi App Integrator muốn write một node type mới (vd: `Requirement`), họ cần biết:
- Attribute nào là mandatory?
- Attribute nào là optional?
- Attribute nào có type constraint (enum, regex, numeric range)?
- Relationship types nào có thể create từ node này?

Hiện tại, không có endpoint nào trả về schema theo cách developer-friendly:
```
Developer: "Tôi muốn write node type ErrorCode. Cần những fields gì?"
→ Phải đọc ontology YAML trong repo
→ Hoặc hỏi Tenant Admin
→ Hoặc trial-and-error gọi API → đọc error message
→ Error message: "attribute 'severity' is required" → add severity
→ Gọi lại → error: "severity must be one of: [critical, high, medium, low]"
→ Mất 30-60 phút chỉ để biết schema
```

**Hệ quả kinh doanh**:
- Developer productivity giảm mạnh trong ngày đầu integrate
- Frustration → team abandon integration, use alternative (Confluence, spreadsheet)
- API adoption phụ thuộc hoàn toàn vào documentation quality (thường không đủ)

**Giải pháp cần có**:
- `GET /v1/tenants/{t}/ontology/domains/{d}/node-types/{type}/schema` — trả về JSON Schema đầy đủ với description, examples, constraints
- OpenAPI spec với schema inline
- SDK với typed models: `kg.Node.Requirement{Title: "...", Status: RequirementStatus.Draft}`

---

### PP-AI-02 — Projection lag không có signal — không biết khi nào data sẵn sàng để read/search

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi lần write rồi read-back  

**Mô tả**:  
KG Service architecture: Write → PostgreSQL → Project → Graph/Vector. Giữa write và read có **projection lag**. Nhưng App Integrator không biết:
- Lag hiện tại là bao nhiêu? (vài ms? vài giây? vài phút?)
- Data của mình đã được project chưa?
- Nếu read ngay sau write mà không có data → lỗi code hay lag?

```python
# App Integrator code:
resp = kg_client.write_node(node)
node_id = resp["id"]

# Đọc lại ngay sau đó:
result = kg_client.get_node(node_id)  # → Trả về gì?
# Case 1: Trả về node vừa write → "OK, works fine"
# Case 2: 404 Not Found → "Bug! Vừa write xong sao không thấy?"
# Case 3: Stale data → "Ủa, version cũ sao?"
# Không có cách distinguish: lag vs. bug
```

**Hệ quả kinh doanh**:
- Integrators viết code không correct (assume immediate consistency)
- Race condition bugs trong production khó debug
- Support tickets tăng: "write rồi read không thấy" → investigation mất nhiều giờ

**Giải pháp cần có**:
- Write response trả về `projection_eta_ms` hoặc `projection_version`
- `GET /v1/kg/nodes/{id}/projection-status` — "projected: true/false, lag: 234ms"
- Webhooks/SSE: notify khi node đã project xong
- Wait mode: `POST /v1/kg/write/nodes?wait_for_projection=true&timeout_ms=5000`

---

### PP-AI-03 — Caller identity phải derive từ API key nhưng không có SDK support — dễ làm sai

**Mức độ**: 🔴 Critical  
**Tần suất**: Mỗi lần integrate lần đầu  

**Mô tả**:  
URD yêu cầu "confirm that caller identity is derived from the key, not request payloads". Nhưng trong thực tế:

```
Sai lầm thường gặp (từ developer quen với REST APIs khác):

// Wrong: Truyền tenant_id trong request body
POST /v1/kg/write/nodes
{
  "tenant_id": "payment-team",    // ← Sai! Ignored hoặc security issue
  "app_id": "payment-service",    // ← Sai!
  "node_type": "ErrorCode"
}

// Đúng: tenant/app derive từ Authorization header
POST /v1/kg/write/nodes
Authorization: Bearer <api-key-của-payment-service>
{
  "node_type": "ErrorCode"         // ← Không cần tenant_id
}
```

Developer không hiểu pattern này sẽ:
- Gửi request thiếu header → 401
- Gửi request có body fields không cần → confused khi body field bị ignore
- Không biết request của mình thuộc tenant/app nào → debugging khó

**Hệ quả kinh doanh**:
- Onboarding developer mới vào kg-service mất nhiều giờ chỉ vì auth model khác convention
- Security bug nếu developer cố override tenant từ body và API không validate đúng

**Giải pháp cần có**:
- `GET /v1/access/me` — "Who am I?" endpoint: trả về tenant, app, permissions của current API key
- SDK encapsulate auth pattern: `KGClient(api_key=...).write_node(...)` — developer không cần biết header
- Clear error message: khi thiếu header → "Missing Authorization header. KG Service uses API key bearer auth, not request body identity."

---

### PP-AI-04 — Named templates là black box — không biết template trả về gì trước khi gọi

**Mức độ**: 🟠 High  
**Tần suất**: Mỗi lần cần dùng template lần đầu  

**Mô tả**:  
SRS không cho phép raw graph queries — integrators phải dùng named templates. Nhưng với templates hiện tại:
- Không có documentation cho template (mô tả template làm gì, trả về gì)
- Không biết parameters của template (cần truyền gì?)
- Không biết output schema (trả về list? object? nested?)
- Không có example response

```
GET /v1/tenants/{t}/ontology/domains/{d}/templates → Trả về list template names
→ "requirements-by-status", "impact-analysis", "traceability-matrix"
→ Tên thì biết, nhưng:
   - "requirements-by-status" cần param gì? "status=approved"? "statusIn=[approved,in-review]"?
   - Trả về list nodes? Hay list node IDs? Hay full node objects?
   - Có pagination không?
   - Rate limit riêng không?
→ Phải hỏi Tenant Admin → đợi response → chậm development
```

**Hệ quả kinh doanh**:
- Developer productivity giảm: phải ask-wait-implement cycle thay vì self-service
- Template underused vì integrators không biết templates nào có và làm gì
- Wrong usage: gọi template sai params → empty response → assume "không có data"

**Giải pháp cần có**:
- Template self-documentation: `GET /v1/ontology/templates/{id}` trả về name, description, parameters, output_schema, example_request, example_response
- Template playground: gọi template với sample data trong browser/CLI

---

### PP-AI-05 — Không có offline development mode — mọi test đều cần real backend stack

**Mức độ**: 🟠 High  
**Tần suất**: Khi develop và test integration code  

**Mô tả**:  
Để develop và test integration code, App Integrator cần một kg-service running với:
- PostgreSQL
- Redis
- Graph backend (Neo4j / MemGraph / FalkorDB)
- Vector backend

Setup local environment này tốn:
- 2-4 giờ lần đầu (theo URD friction list: "Hidden environment variables that are not documented")
- RAM: 4-8GB chỉ cho dependencies
- Nhiều version conflicts giữa Docker images

Không có:
- Mock server / test server mode
- In-memory mode đủ để test write/read flow
- Recording/replay mode (record API calls → replay in tests)

**Hệ quả kinh doanh**:
- CI/CD chậm vì phải spin up full stack
- Developer không thể develop offline (khi không có internet hoặc company VPN)
- Unit testing không thể mock kg-service một cách standard

**Giải pháp cần có**:
- `kg-service --profile=memory` — in-memory mode, tất cả backends là in-memory, zero external dependencies
- Test SDK: `KGClientMock()` với same interface nhưng in-memory implementation
- Docker Compose lightweight profile: chỉ PostgreSQL + kg-service (skip graph/vector backends), chấp nhận limited search functionality

---

### PP-AI-06 — Error messages không actionable — không biết phải làm gì khi request fail

**Mức độ**: 🟠 High  
**Tần suất**: Thường xuyên trong integration phase  

**Mô tả**:  
Khi request fail, error messages hiện tại không đủ context:

```json
// Hiện tại:
{
  "error": "validation_failed",
  "message": "invalid node attributes"
}

// Cần thiết:
{
  "error": "validation_failed",
  "message": "Node attribute 'severity' has invalid value 'critical-high'",
  "details": {
    "field": "attributes.severity",
    "provided": "critical-high",
    "allowed_values": ["critical", "high", "medium", "low"],
    "ontology_ref": "payment/ErrorCode#severity"
  },
  "fix_hint": "Use one of the allowed severity values. See ontology definition at GET /v1/ontology/node-types/ErrorCode"
}
```

**Hệ quả kinh doanh**:
- Debug time tăng: phải correlate error với ontology definition thủ công
- Slack/email interruptions: developer hỏi Tenant Admin/Platform Admin về error meaning

**Giải pháp cần có**:
- Structured error response với `field`, `provided`, `expected`, `fix_hint`
- Error code catalog: `GET /v1/docs/errors/{code}` — full explanation + common causes + fix steps
- Request ID trong mọi response để correlate với logs

---

## Ma trận Pain Points — App Integrator

| ID | Pain Point | Mức độ | Impact | Giải pháp cần có |
|:---|:---|:---:|:---|:---|
| PP-AI-01 | Không có schema discovery | 🔴 | Slow onboarding, trial-and-error | Node type schema endpoint |
| PP-AI-02 | Projection lag không có signal | 🔴 | Race condition bugs, wrong assumptions | Projection status API, wait mode |
| PP-AI-03 | Auth model khác convention, dễ làm sai | 🔴 | Security issues, confusion | /v1/access/me endpoint + SDK |
| PP-AI-04 | Templates là black box | 🟠 | Underused, misused | Template self-documentation |
| PP-AI-05 | Không có offline/mock mode | 🟠 | Slow dev, CI/CD overhead | Memory profile + test SDK |
| PP-AI-06 | Error messages không actionable | 🟠 | Debug time cao, support overhead | Structured errors + fix hints |

---

## Tại sao App Integrator phải dùng kg-service

1. **Không cần viết raw graph queries**: Chỉ cần gọi named templates → Service handle graph traversal complexity
2. **Identity-aware by default**: Không cần implement ACL trong application code — service enforce tenant isolation automatically
3. **Dual retrieval**: Cùng một write request → automatically available qua cả graph traversal lẫn semantic search
4. **Audit trail built-in**: Mọi write đều có source-of-truth record trong PostgreSQL — không cần maintain audit log riêng
5. **Multi-backend portability**: Swap graph backend (Neo4j → FalkorDB) mà không cần thay application code

> **Kết luận**: App Integrator là người convert investment vào KG Service thành business value — mỗi pain point của họ là một rào cản adoption. Giải quyết PP-AI-01 đến PP-AI-06 sẽ giảm onboarding từ tuần xuống còn giờ.
