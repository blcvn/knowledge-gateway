# Business Logic — Tenant Admin

> **Actor**: Tenant Admin  
> **Vai trò**: Quản trị viên của một tổ chức — thiết kế ontology domain, đăng ký query template, quản lý lifecycle, kiểm soát visibility  
> **Phạm vi quyền**: Chỉ trong tenant của mình

---

## Nghiệp vụ 1: Thiết kế Domain Ontology

### BL-TA-01: Tạo Domain

**Mô tả**: Tenant Admin tạo một domain để phân loại nhóm tri thức nghiệp vụ (e.g., "payment-errors", "product-catalog", "legal-policies").

**Business Rules**:
- Domain ID là string tự định nghĩa — nên mô tả rõ nội dung (e.g., `payment-errors`)
- Domain mới tạo ở trạng thái `draft` — chưa thể ghi data thật vào
- Tenant chỉ sở hữu và modify domain của chính mình
- Platform domain (owner = platform) là read-only với mọi tenant

**Vòng đời domain**:
```
draft   → Đang thiết kế ontology, không ghi production data
active  → Đang sử dụng, thay đổi cần cẩn thận (breaking change = bump version)
deprecated → Vẫn đọc được, không tạo mới data
```

**Quyết định visibility**:

| visibility | Ai thấy domain này? |
|:---|:---|
| `private` | Chỉ apps của tenant này |
| `tenant_shared` | Tất cả apps trong cùng tenant |
| `public` | Mọi tenant (thường chỉ dùng cho platform domain) |

---

### BL-TA-02: Khai báo Node Types

**Mô tả**: Mỗi loại node trong domain phải được khai báo trước — gồm tên, thuộc tính bắt buộc, thuộc tính tùy chọn, và quy tắc validate.

**Business Rules**:
- Không thể write node có `node_type` chưa được khai báo trong domain
- `required_props` phải có đủ khi write — thiếu prop → 422 VALIDATION_FAILED
- `validation_rules` là các biểu thức boolean trên props (e.g., `"end_date > start_date OR end_date IS NULL"`)
- Thêm `required_prop` vào node type đã tồn tại là **breaking change** → phải bump domain version
- Xóa `required_prop` là non-breaking

**Ví dụ khai báo**:
```
NodeType: "ErrorCode"
  required_props:
    - code: string       ← bắt buộc
    - severity: enum[critical, high, medium, low]  ← bắt buộc
  optional_props:
    - description: string
    - error_family: string
  validation_rules:
    - "severity IN ['critical','high','medium','low']"
```

---

### BL-TA-03: Khai báo Relationship Types

**Mô tả**: Khai báo loại quan hệ giữa các node — từ node type nào đến node type nào, cùng domain hay cross-domain.

**Business Rules**:
- Một relationship type có `from_node_type` và `to_node_type` cụ thể — không generic
- `same_domain = true` → cả from và to phải cùng domain
- `same_domain = false` → cross-domain relationship — cần khai báo cross-domain rules riêng
- Không thể tạo relationship với `rel_type` chưa khai báo

---

### BL-TA-04: Khai báo Cross-Domain Relationship Rules

**Mô tả**: Khi một node type bắt buộc phải có quan hệ với node type của domain khác trước khi publish.

**Business Rules**:
- `required = true` → write node mà thiếu cross-domain link → 422 VALIDATION_FAILED
- Convention: property `bridge_{rel_type_lower}_ids` trong write request chứa danh sách ID node đích
- `exception_types` → danh sách node type được miễn rule này
- Rule chỉ được evaluate tại write time, không retroactive

**Ví dụ**:
```
Rule: PaymentFlow phải có GOVERNED_BY ít nhất 1 Requirement
  from_domain: payment
  to_domain: compliance
  rel_type: GOVERNED_BY
  required: true
  → Write PaymentFlow phải gửi kèm: bridge_governed_by_ids: ["req-uuid-1"]
```

---

### BL-TA-05: Versioning Ontology

**Business Rules**:
- Mỗi thay đổi ontology tạo một `ontology_version` record
- **Breaking changes** (thêm required prop, đổi type, xóa prop):
  - Phải bump domain version
  - Nodes hiện có với `domain_version` cũ vẫn được đọc nhưng có thể không pass validate mới
- **Non-breaking changes** (thêm optional prop, thêm node type mới, thêm rel type):
  - Không cần bump version
  - Tất cả nodes vẫn valid

---

## Nghiệp vụ 2: Quản lý Query Templates

### BL-TA-06: Đăng ký Query Template

**Mô tả**: Tenant Admin định nghĩa các pattern truy vấn graph cho domain — các app sẽ gọi template theo tên, không gửi query tự tạo.

**Business Rules**:
- Template được khai báo bằng **Query Pattern DSL** (JSON) — **không phải Cypher thô**
- ACL filter được tự động inject vào mọi hop của query — domain owner không thể bỏ qua
- Template mới tạo có `status = draft` — chưa thể gọi
- **Giới hạn**: tối đa 5 hop trong một template (vượt → 422 TEMPLATE_TOO_DEEP)
- Template name phải duy nhất trong cùng domain

**Tại sao không dùng Cypher thô?**

| Tiêu chí | Cypher thô | Query Pattern DSL |
|:---|:---|:---|
| Domain owner bypass ACL? | Có thể | Không thể — compiler tự inject |
| Service biết schema cụ thể? | Có → bị gắn với domain | Không → generic |
| Thêm domain mới | Phải sửa code | Chỉ gọi API |

**Ví dụ DSL**:
```json
{
  "start": { "node_type": "ErrorCode", "match": { "severity": "$severity" } },
  "hops": [
    { "rel_type": "TRIGGERS", "to_node_type": "PaymentFlow" },
    { "rel_type": "GOVERNED_BY", "to_node_type": "Requirement", "filter_status": "valid_only" }
  ],
  "return_fields": ["ErrorCode.code", "PaymentFlow.flow_id", "Requirement.title"]
}
```

---

### BL-TA-07: Activate / Deprecate Query Template

**Business Rules**:
- Template ở `draft` → không thể gọi
- Activate (`draft → active`) → template khả dụng, tự động có route `/v1/kg/read/template/{domain}/{name}`
- Deprecate → vẫn gọi được trong giai đoạn chuyển tiếp (caller nên migrate sang template mới)
- Nên tạo template v2, activate, rồi mới deprecate v1 — không xóa ngay

---

## Nghiệp vụ 3: Cấu hình Lifecycle / Status

### BL-TA-08: Khai báo Status Field Config

**Mô tả**: Domain có thể (không bắt buộc) khai báo một field nào là "trạng thái" của node — ảnh hưởng đến filter kết quả đọc/search.

**Business Rules**:
- **Không khai báo = no-op**: đọc/search trả về tất cả nodes, không filter theo status
- Chỉ có **một** status field per domain
- `valid_status_values` → nodes có giá trị này được trả về bình thường
- `warning_status_values` → nodes được trả về kèm cờ cảnh báo
- Giá trị khác → nodes bị loại bỏ khỏi kết quả (không báo lỗi)

**Ví dụ**:
```
Domain: "payment-errors"
status_field_name: "error_status"
valid_status_values: ["active", "known"]    → trả về
warning_status_values: ["deprecated"]       → trả về + warning
# Các giá trị khác ("retired", "archived") → bị lọc bỏ
```

---

### BL-TA-09: Khai báo Authority Score

**Mô tả**: Domain có thể khai báo field để xếp hạng authority của node — dùng để rerank kết quả search.

**Business Rules**:
- Không bắt buộc — không khai báo = không rerank
- `authority_field_name` → field nào chứa loại/cấp của node
- `authority_values_map` → ánh xạ giá trị → điểm số (cao hơn = ưu tiên hơn)
- Authority rerank chỉ áp dụng cho semantic search, không ảnh hưởng graph read

---

### BL-TA-10: Khai báo Cascade Rules

**Mô tả**: Khi status của node cha thay đổi, tự động propagate xuống các node con theo relationship.

**Business Rules**:
- Chỉ khai báo nếu domain có hệ thống phân cấp node (parent → child)
- Cascade là **async** — không đồng bộ trong cùng request
- Service thực thi cascade rule generic — không biết domain đó là gì

---

## Nghiệp vụ 4: Quản lý Visibility & Sharing

### BL-TA-11: Kiểm tra Effective Ontology

**Mô tả**: Tenant Admin kiểm tra app của mình đang thấy những domain nào và từ nguồn nào.

**Luồng**:
```
GET /v1/tenants/{t}/ontology/effective
→ {
    domains: [
      { id: "platform-base", owner: "platform", permission: "read" },
      { id: "payment-errors", owner: "this-tenant", permission: "write" },
      { id: "compliance", owner: "other-tenant", permission: "search" }
    ]
  }
```

**Nguồn domain trong effective ontology**:
1. Platform domains (luôn visible)
2. Domains do tenant này sở hữu
3. Domains được share qua AccessGrant

---

## Tóm tắt Business Rules — Tenant Admin

| ID | Rule |
|:---:|:---|
| **BR-TA-01** | Tenant chỉ được tạo/sửa domain của chính mình |
| **BR-TA-02** | Template phải ở `status = active` mới gọi được |
| **BR-TA-03** | Query template dùng DSL, không nhận Cypher thô |
| **BR-TA-04** | Template bị giới hạn tối đa 5 hop |
| **BR-TA-05** | Breaking change ontology phải bump domain version |
| **BR-TA-06** | Status config là tùy chọn — không khai báo = no filter |
| **BR-TA-07** | Chỉ có một status field per domain |
| **BR-TA-08** | ACL filter luôn inject vào mọi hop của query — không thể bypass |
| **BR-TA-09** | Cross-domain rel rule chỉ enforce tại write time |
| **BR-TA-10** | Platform domain là read-only — tenant không thể sửa |
