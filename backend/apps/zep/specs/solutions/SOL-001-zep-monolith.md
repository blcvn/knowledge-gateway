---
id: SOL-001
title: Scaffolding Zep Golang Monolith
service: zep
version: 1.0.0
status: Approved
priority: P0
created: 2026-05-12
updated: 2026-05-12
approved_by: Architect
---

## Yêu Cầu Gốc
Hiện thực hoá ứng dụng Zep dùng Golang thành một single-binary (monolith) application. Sử dụng code có sẵn từ `gateway` và `services` (không được thay đổi source code gốc). Các module giao tiếp nội bộ qua gRPC hoặc NATS JetStream. Yêu cầu tạo bộ docs và specs hoàn chỉnh.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng
| Service | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| apps/zep | Tạo ứng dụng mới (monolith) | Cao |
| gateway | Import as library | Không (Zero-modification) |
| services/zep-* | Import as library | Không (Zero-modification) |

### Breaking Changes
- [ ] API response format thay đổi? Không
- [ ] Database schema migration cần thiết? Không
- [ ] Consumer downstream cần cập nhật? Không

### Ràng Buộc Kiến Trúc
- **Zero-modification constraint**: KHÔNG được sửa đổi mã nguồn hiện tại của `gateway` và các service `zep-*` trong thư mục `services/`.
- **Single-binary**: Tất cả các thành phần phải được đóng gói vào một tệp thực thi duy nhất bằng Golang.
- **Inter-service communication**: Các component giao tiếp nội bộ qua gRPC loopback (localhost) hoặc NATS JetStream.
- **Phải tuân thủ chuẩn**: Quy trình phân rã task và viết docs/specs phải theo đúng `workflow-guide.md` và `specs-catalog.md`.

## Giải Pháp Đề Xuất

### Approach
Sử dụng pattern Application Supervisor để gom nhóm, khởi tạo, và quản lý lifecycle (phased startup, graceful shutdown) của toàn bộ các services và gateway. Thay vì chạy mỗi service như một container riêng biệt, ta sẽ import các hàm khởi tạo (bootstrap/wire) của từng service vào ứng dụng monolith. 

### Trade-offs
- **Ưu điểm:** Đơn giản hóa quá trình deploy (chỉ cần quản lý 1 binary thay vì n container), dễ dàng test tích hợp, tiết kiệm chi phí network do gọi qua localhost loopback.
- **Nhược điểm / Rủi ro:** Cần quản lý cấu hình (config) tập trung cẩn thận để tránh xung đột môi trường. Logic graceful shutdown phức tạp hơn.

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)
```text
Task 1: Tạo Go module và cấu hình tập trung (Unified Config)
Task 2: Hiện thực hoá Supervisor pattern (Graceful Shutdown)
Task 3: Embed các Zep services (User, Thread, Memory, Graph, Search, Admin)
Task 4: Embed Gateway
Task 5: Tạo các deployment artifacts (Dockerfile, Makefile, Docker Compose)
```

### Danh Sách Tác Vụ
| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính | Status |
|---|---|---|---|---|---|---|
| T01 | Setup Go Module & Unified Config | TASK | zep | - | 2h | ✅ Done |
| T02 | Implement Supervisor Lifecycle | TASK | zep | T01 | 3h | ✅ Done |
| T03 | Embed Zep Services | TASK | zep | T02 | 4h | ✅ Done |
| T04 | Embed Gateway | TASK | zep | T03 | 2h | ✅ Done |
| T05 | Create Deployment Artifacts | TASK | zep | T04 | 2h | ✅ Done |

## Acceptance Criteria (Solution Level)
- [x] SOL-AC-1: Zep monolith build thành công thành một file binary duy nhất.
- [x] SOL-AC-2: Supervisor quản lý khởi động và tắt an toàn (graceful shutdown) cho tất cả services và gateway.
- [x] SOL-AC-3: Các module giao tiếp thành công qua gRPC/NATS trong cùng process/localhost.
- [x] SOL-AC-4: KHÔNG có bất kỳ file source code nào trong thư mục `gateway` và `services/zep-*` bị chỉnh sửa.
- [x] SOL-AC-5: Hoàn tất toàn bộ tài liệu specs trong `apps/zep/specs`.
