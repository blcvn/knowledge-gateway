# Solutions — Memobase Feature Parity

**Project:** VNP Memory  
**Domain:** Memobase — User Profile-Based Long-Term Memory System  
**Path:** `specs/crs/v1/memobase/solutions/`  
**Date:** 2026-06-17  
**Status:** Draft

> Các tài liệu Solution này mô tả giải pháp kỹ thuật chi tiết để đáp ứng từng Change Request trong `specs/crs/v1/memobase/`. Mỗi solution bao gồm: chiến lược triển khai, tích hợp với kiến trúc VNP Memory hiện tại, và kế hoạch thực thi.

---

## Danh sách Solutions

| Solution File | CR Tương ứng | Mô tả | Wave |
|---|---|---|---|
| [SOL-MB-001](./SOL-MB-001-Blob-Ingestion-Buffer-Zone.md) | CR-MB-001 | Blob Ingestion & Buffer Zone Service | Wave 1 |
| [SOL-MB-002](./SOL-MB-002-Memory-Engine-Profile-YOLO.md) | CR-MB-002 | Memory Engine: Profile Extraction & YOLO Merge | Wave 2 |
| [SOL-MB-003](./SOL-MB-003-Context-Service.md) | CR-MB-003 | Context Service (Profile Read & Context Assembly) | Wave 3 |
| [SOL-MB-004](./SOL-MB-004-Event-Timeline-Semantic-Search.md) | CR-MB-004 | Event Timeline & Semantic Search | Wave 3 |
| [SOL-MB-005](./SOL-MB-005-Admin-Service.md) | CR-MB-005 | Admin Service (User, Project, Billing) | Wave 1 |
| [SOL-MB-006](./SOL-MB-006-Gateway-MCP-Server.md) | CR-MB-006 | Gateway & MCP Server Extension | Wave 4 |
| [SOL-MB-007](./SOL-MB-007-Shared-Infrastructure.md) | CR-MB-007 | Shared Infrastructure (LLM, Tokenizer, OTel) | Wave 4 |

---

## Kiến trúc Memobase trong VNP Memory

### Tích hợp với Monolith hiện tại

VNP Memory hiện tại (monolith) đã có placeholder cho 3 memobase services:
- `memobase-ingestion` (port 9041)
- `memobase-engine` (port 9042)  
- `memobase-context` (port 9043)

Giải pháp **bổ sung 2 services mới**:
- `memobase-event` (port 9044) — CR-MB-004
- `memobase-admin` (port 9045) — CR-MB-005

### Implementation Waves

```
Wave 1 (Data Layer):    SOL-MB-005 → SOL-MB-001
Wave 2 (Processing):   SOL-MB-002
Wave 3 (Read Path):    SOL-MB-003 → SOL-MB-004
Wave 4 (Access Layer): SOL-MB-007 → SOL-MB-006
```

### NATS Event Flow

```
Client → Gateway → memobase-ingestion
                        │ memobase.buffer.ready
                        ▼
               memobase-engine (LLM pipeline)
                    │             │
   memobase.profile.changed   memobase.event.created
                    │             │
               memobase-context  memobase-event
               (cache inval.)   (embed & store)
```

---

## Nguyên tắc chung cho mọi Solution

1. **Clean Architecture** — Domain → Usecase → Adapter → Infra (không có dependency ngược)
2. **gRPC Internal** — mọi service-to-service communication qua gRPC + bufconn (monolith) hoặc TCP (gateway)
3. **NATS JetStream** — async events với WorkQueue retention; không có HTTP callbacks giữa services
4. **Composite PK** — mọi entity đều dùng `(id, project_id)` để hỗ trợ multi-tenant partitioning
5. **Redis caching** — chỉ memobase-context cache profiles; các service khác không cache
6. **Tích hợp monolith** — đăng ký vào `InProcessRegistry` trong `apps/memory/internal/bootstrap/`
