# Business Logic — Knowledge Graph Service

> **Mục đích**: Mô tả các quy tắc nghiệp vụ (business logic/rules) từ góc độ người dùng — **không phải kỹ thuật**.  
> **Nguồn gốc**: Phân tích từ [PRD](../product/prd.md), [URD](../product/urd.md), [SRS](../product/srs.md)  
> **Phiên bản**: 1.0 · 2026-08-03

---

## Mục lục

| File | Actor | Nghiệp vụ |
|:---|:---|:---|
| **README.md** (file này) | Tổng quan | Danh sách tất cả business rules, invariants chung |
| [bl-platform-admin.md](./bl-platform-admin.md) | Platform Admin | Quản lý tenant, app, key, grants |
| [bl-tenant-admin.md](./bl-tenant-admin.md) | Tenant Admin | Thiết kế ontology, query template, lifecycle |
| [bl-app-integrator.md](./bl-app-integrator.md) | App Integrator | Ghi dữ liệu, đọc, tìm kiếm |
| [bl-agent-client.md](./bl-agent-client.md) | AI Agent / MCP Client | Tìm kiếm tri thức, tool calling |
| [bl-operator-sre.md](./bl-operator-sre.md) | Operator / SRE | Deploy, validate, reconcile |

---

## Business Invariants (Bất biến nghiệp vụ toàn cục)

Các quy tắc này **không có ngoại lệ**, áp dụng mọi lúc, mọi actor:

| ID | Rule | Hệ quả khi vi phạm |
|:---:|:---|:---|
| **BI-01** | **Identity từ API key** — caller không bao giờ tự khai báo tenant/app của mình trong request | 400 hoặc ignore silently |
| **BI-02** | **Deny-by-default** — không có access nào ngầm định giữa các tenant khác nhau | Caller chỉ thấy data của chính mình + platform |
| **BI-03** | **Không raw query** — caller không bao giờ gửi Cypher hoặc raw filter trực tiếp | Không có API cho raw query |
| **BI-04** | **Ontology phải khai báo trước khi ghi** — không thể write node nếu node_type chưa có trong domain | 422 VALIDATION_FAILED |
| **BI-05** | **Platform ontology read-only với tenant** — tenant không thể sửa schema platform-owned | 403 FORBIDDEN |
| **BI-06** | **Grant phải tường minh** — không có implicit sharing, mọi cross-tenant access cần AccessGrant | 403 NO_READ_ACCESS |
| **BI-07** | **Write → async sync** — sau khi write thành công (202), data có thể chưa xuất hiện ngay ở graph/vector | Caller phải chờ projection |
| **BI-08** | **Soft delete** — node không bao giờ bị xóa vật lý, chỉ bị mark `is_deleted` | Data vẫn tồn tại trong PG |
| **BI-09** | **API key là one-time visible** — plaintext key chỉ được trả về một lần duy nhất lúc tạo/rotate | Mất key = phải rotate |
| **BI-10** | **Status filter là domain-specific** — chỉ áp dụng nếu domain đã khai báo status config | Domain không khai báo = no filter |

---

## Business Flows Tổng quan

### Flow 1: Vòng đời Tenant

```
[Tạo tenant] → [Tạo app] → [Nhận API key (once)] → [Xác thực] → [Sử dụng]
                                ↓
                          [Rotate key] ← cần khi key lộ
                          [Revoke app] ← ngừng sử dụng
                          [Suspend tenant] ← tạm khóa toàn bộ
```

### Flow 2: Vòng đời Domain

```
[Tạo domain (draft)] → [Khai báo node types] → [Khai báo rel types]
        ↓
[Đăng ký query templates] → [Activate templates] → [Cấu hình status/lifecycle]
        ↓
[domain: active] → [Sử dụng] → [Deprecate khi obsolete]
```

### Flow 3: Vòng đời Dữ liệu

```
[Write node/rel] → [Validate schema] → [Lưu PG] → [202 Accepted]
                                                          ↓ (async < 2s)
                                              [Sync → Graph DB]
                                              [Sync → Vector DB]
                                                          ↓
                                         [Read template] / [Search]
```

### Flow 4: Vòng đời Access Grant

```
[Tạo grant] → [Sync acl_visible_to] → [Grantee thấy data] (< 5s)
[Revoke grant] → [Sync ngay] → [Grantee không thấy] (immediate cache)
```

---

## Business Rules Index

| ID | Rule | Actor | File |
|:---:|:---|:---|:---|
| BR-PA-01 | Chỉ platform admin mới tạo/xóa tenant | Platform Admin | bl-platform-admin.md |
| BR-PA-02 | API key chỉ hiển thị một lần | Platform Admin | bl-platform-admin.md |
| BR-PA-03 | Cross-tenant write grant bắt buộc có `expires_at` | Platform Admin | bl-platform-admin.md |
| BR-PA-04 | Revoke app vô hiệu key ngay lập tức (< 30s) | Platform Admin | bl-platform-admin.md |
| BR-TA-01 | Tenant chỉ sửa domain của chính mình | Tenant Admin | bl-tenant-admin.md |
| BR-TA-02 | Template phải được activate mới chạy được | Tenant Admin | bl-tenant-admin.md |
| BR-TA-03 | Query template không nhận Cypher thô | Tenant Admin | bl-tenant-admin.md |
| BR-TA-04 | Status config là tùy chọn, không khai báo = no-op | Tenant Admin | bl-tenant-admin.md |
| BR-TA-05 | Breaking change ontology phải bump version | Tenant Admin | bl-tenant-admin.md |
| BR-AI-01 | Caller chỉ write vào domain trong effective ontology | App Integrator | bl-app-integrator.md |
| BR-AI-02 | Required props phải có đủ khi write | App Integrator | bl-app-integrator.md |
| BR-AI-03 | Read chỉ qua named template, không raw query | App Integrator | bl-app-integrator.md |
| BR-AI-04 | Search result được filter bởi ACL | App Integrator | bl-app-integrator.md |
| BR-AI-05 | `mode=realtime` dùng khi cần consistency tuyệt đối | App Integrator | bl-app-integrator.md |
| BR-AG-01 | MCP xác thực tại connection, không phải tool call | AI Agent | bl-agent-client.md |
| BR-AG-02 | Agent phải gọi `kg_list_templates` để biết template khả dụng | AI Agent | bl-agent-client.md |
| BR-AG-03 | Agent không nhận Cypher trong MCP tool | AI Agent | bl-agent-client.md |
| BR-OPS-01 | Profile phải khớp với deployment environment | Operator | bl-operator-sre.md |
| BR-OPS-02 | Health check là điều kiện tiên quyết trước khi sử dụng | Operator | bl-operator-sre.md |
| BR-OPS-03 | Reconciliation drift > 0.1% phải alert | Operator | bl-operator-sre.md |
