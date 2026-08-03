# High-Level Design — Knowledge Graph Service
## C4 Model Architecture Documentation

> **Version**: 2.0 (C4 Model)  
> **Date**: 2026-08-03  
> **Source**: [KG_Service_TDD_v1.md](../KG_Service_TDD_v1.md) + codebase  
> **Standard**: [C4 Model](https://c4model.com/) — Context → Container → Component → Code

---

## Cấu trúc tài liệu

| Level | File | Câu hỏi trả lời |
|:---:|:---|:---|
| **L1** | [c4-l1-context.md](./c4-l1-context.md) | KG Service là gì? Ai dùng? Kết nối với hệ thống nào? |
| **L2** | [c4-l2-container.md](./c4-l2-container.md) | Service gồm những process/store nào? Chạy trên technology gì? |
| **L3** | [c4-l3-component.md](./c4-l3-component.md) | Trong mỗi container có những module nào? Tương tác thế nào? |
| **L4** | [c4-l4-code.md](./c4-l4-code.md) | Các data type, interface, và flow chi tiết trong code |
| **ADR** | [adr/](./adr/) | Architecture Decision Records — các quyết định thiết kế quan trọng |

---

## Tóm tắt nhanh (Elevator Pitch)

> **KG Service** là một **Knowledge Graph Platform đa tenant, domain-agnostic** viết bằng Go.  
> Cho phép nhiều tổ chức (tenant) lưu trữ, truy vấn và tìm kiếm tri thức có cấu trúc qua REST API và MCP protocol, với đảm bảo isolation tuyệt đối giữa các tenant.  
> Mọi schema (node type, relationship, query pattern, lifecycle rule) đều là **cấu hình runtime** — không cần sửa code để thêm domain nghiệp vụ mới.

---

## Diagram Key (ký hiệu dùng trong tài liệu)

```
[ Person ]          Người dùng / Actor
[[ System ]]        Hệ thống phần mềm (software system)
{ Container }       Process, database, hoặc deployment unit
< Component >       Module / package trong một container
─────▶              Request / call (synchronous)
- - -▶              Event / message (asynchronous)
══════              Data stored / persisted
```

---

## Tài liệu liên quan

| Tài liệu | Mục đích |
|:---|:---|
| [KG_Service_TDD_v1.md](../KG_Service_TDD_v1.md) | Technical Design Document đầy đủ (source of truth) |
| [docs/api/README.md](../api/README.md) | API Reference |
| [docs/api/openapi.yaml](../api/openapi.yaml) | OpenAPI 3.0 spec |
| [docs/guides/](../guides/) | Integration, MCP, Quickstart guides |
| [docs/deployment/](../deployment/) | Environment config, Compose, K8s |
| [docs/operations/](../operations/) | Runbooks |
| [docs/business/painpoints/](../business/painpoints/) | Pain points theo actor |
| [docs/business/solutions/](../business/solutions/) | Solutions mapping |
