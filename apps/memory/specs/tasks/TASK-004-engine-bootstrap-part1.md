---
id: TASK-004
title: Engine Bootstrappers (Part 1)
service: apps-memory
status: Done
priority: P1
created: 2026-05-14
---

# TASK-004: Engine Bootstrappers (Cognee, Graphiti, Memobase)

## 1. Mục Tiêu
Tích hợp 10 service thuộc 3 engine (Cognee, Graphiti, Memobase) vào Monolithic app thông qua Dependency Injection và In-Process gRPC Bus.

## 2. Các Bước Thực Thi

Tạo các file bootstrap tại `apps/memory/internal/bootstrap/`:

1. **`cognee.go`**:
   - Import và wire `cognee-ingestion`, `cognee-cognify`, `cognee-search`.
   - Setup NATS subscribers (ví dụ: `cognee.data.ingested` → `cognify`).
   - Register vào `bus.GRPCBus`.
2. **`graphiti.go`**:
   - Wire `graphiti-ingestion`, `graphiti-search`, `graphiti-knowledge`, `graphiti-store`.
   - Tái sử dụng Neo4j connection từ shared `Infra`.
3. **`memobase.go`**:
   - Wire `memobase-ingestion`, `memobase-engine`, `memobase-context`.
   - Set up buffer flush subscriber qua NATS.

Cập nhật `main.go` để gọi 3 hàm bootstrap này trước khi start bus.

## 3. Ràng Buộc
- **KHÔNG SỬA CODE GỐC** của bất kỳ service nào. Nếu constructor yêu cầu interface, truyền trực tiếp implementation tương ứng.
- Đảm bảo prefix tên hàm rõ ràng: `bootstrap.Cognee()`, `bootstrap.Graphiti()`, `bootstrap.Memobase()`.

## 4. Acceptance Criteria
- [ ] Compile thành công sau khi add 10 services.
- [ ] Event flow của Memobase và Graphiti hoạt động bình thường qua NATS Embedded.
