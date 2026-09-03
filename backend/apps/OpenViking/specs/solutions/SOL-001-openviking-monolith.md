---
id: SOL-001
title: OpenViking Single-Binary Monolith Implementation
package: apps/OpenViking
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_cr: PRD-OpenViking
approved_by: Architect
---

## Yêu Cầu Gốc
Thực hiện ứng dụng OpenViking bằng Golang thành một ứng dụng chỉ có 1 file chạy (single-binary monolith). Ứng dụng này sử dụng lại hoàn toàn mã nguồn từ các dịch vụ `gateway` và `services/ov-*` mà không được phép thay đổi mã nguồn gốc của chúng. Các dịch vụ giao tiếp qua gRPC hoặc NATS theo đúng thiết kế của OpenViking.

## Phân Tích Tác Động Kiến Trúc

### Services Bị Ảnh Hưởng
| Service / Package | Loại thay đổi | Mức độ ảnh hưởng |
|---|---|---|
| `apps/OpenViking` | Tạo mới App | Cao |
| `gateway` | Tích hợp (embedded) | Thấp (Không sửa code) |
| `services/ov-*` | Tích hợp (embedded) | Thấp (Không sửa code) |

### Ràng Buộc Kiến Trúc
1. **Zero Modification Constraint:** Tuyệt đối không thay đổi mã nguồn của `gateway` và `services/ov-*`.
2. **Single Binary:** Tất cả dịch vụ chạy chung trong một tiến trình (process) duy nhất.
3. **Phased Lifecycle:** Khởi động và tắt các dịch vụ theo thứ tự phụ thuộc (ví dụ: NATS/DB -> Services -> Gateway).

## Giải Pháp Đề Xuất

### Approach: Embedded Supervisor Pattern
Sử dụng pattern "Supervisor" trong Golang. Một main package tại `apps/OpenViking/cmd/openviking/main.go` sẽ:
1. Orchestrate việc khởi động tuần tự.
2. Import và chạy các dịch vụ `ov-admin`, `ov-crypto`, `ov-fs`, `ov-resource`, `ov-search`, `ov-session` như các goroutines, cấu hình chạy trên localhost gRPC port (ví dụ 9011-9030).
3. Import và chạy `gateway` để hứng request từ ngoài và trỏ đến các port gRPC của các dịch vụ đang chạy trên localhost.
4. Giao tiếp event bằng NATS (nếu cần thiết, tuỳ thuộc cấu hình env).
5. Quản lý graceful shutdown bằng `golang.org/x/sync/errgroup` và `context`.

### Trade-offs
- **Ưu điểm:** Dễ triển khai (1 binary duy nhất), footprint nhẹ hơn khi chạy multi-container, thuận tiện debug.
- **Nhược điểm:** Phải cẩn thận với port conflict và race condition khi shutdown, không scale rời rạc được.

## Kế Hoạch Triển Khai

### Thứ Tự Thực Hiện (Dependency Order)
```
Task 1: Setup Supervisor Pattern ← Phải làm trước
Task 2: Embed Domain Services ← Sau Task 1
Task 3: Embed Gateway ← Sau Task 2
Task 4: Aggregated Health API ← Sau Task 3
Task 5: Build & Deployment Artifacts ← Sau Task 4
```

### Danh Sách Tác Vụ
| ID | Tên Task | Loại Spec | Service | Phụ thuộc | Ước tính |
|---|---|---|---|---|---|
| T01 | Setup Supervisor Pattern | TASK | apps/OpenViking | - | 4h |
| T02 | Embed Domain Services | TASK | apps/OpenViking | T01 | 4h |
| T03 | Embed Gateway | TASK | apps/OpenViking | T02 | 3h |
| T04 | Implement Health Aggregation | TASK | apps/OpenViking | T03 | 2h |
| T05 | Build & Deployment Artifacts | TASK | apps/OpenViking | T04 | 2h |

## Acceptance Criteria (Solution Level)
- [x] SOL-AC-1: Tất cả tasks trong danh sách hoàn thành (T01 đến T05).
- [x] SOL-AC-2: Binary `openviking` chạy được độc lập, khởi động thành công Gateway + 6 services.
- [x] SOL-AC-3: Giao tiếp gRPC/NATS giữa các module qua loopback (localhost) không gặp lỗi, giữ nguyên Zero Modification constraint.
- [x] SOL-AC-4: Tài liệu cấp ứng dụng (README, architecture, changelog) được tạo đầy đủ theo chuẩn.
