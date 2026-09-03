# OpenViking Monolith Application

Ứng dụng OpenViking thực hiện theo kiến trúc Monolith Single-Binary (ứng dụng chỉ gồm 1 file chạy duy nhất), thiết kế dựa trên quy chuẩn "Supervisor Pattern".

## Khởi nguồn (Mục đích)
OpenViking bao gồm nhiều microservices (gateway, fs, search, session, resource, crypto, admin). Việc tách rời các microservices này giúp độc lập mở rộng và bảo trì dễ dàng, nhưng lại gặp khó khăn cho các môi trường triển khai nhỏ gọn hoặc chạy cục bộ (local/edge). Ứng dụng này nhằm mục tiêu chạy toàn bộ hệ thống bằng 1 process Go duy nhất, dùng cơ chế Supervisor để quản lý vòng đời (lifecycle) của Gateway và 6 services.

Tất cả các thành phần được tải tĩnh (statically embedded) mà **KHÔNG SỬA ĐỔI** mã nguồn của chúng.

## Cấu trúc Kiến Trúc Tổng Quan (Architecture)
- **Phase 1:** Khởi chạy Infrastructure & Configuration.
- **Phase 2:** Khởi chạy Domain Services (ov-fs, ov-search...).
- **Phase 3:** Khởi chạy Gateway và điều hướng lưu lượng về các Domain Services qua localhost gRPC.
- Giao tiếp giữa Gateway và Services, hoặc giữa Service với nhau sử dụng localhost Networking hoặc In-Memory NATS (tuỳ cấu hình).

## Quick Start
Yêu cầu: Go 1.23+
```bash
# Build binary
make build

# Chạy monolith
./bin/openviking

# Hoặc khởi động bằng docker compose
docker-compose up -d
```

## Xem thêm tài liệu
- Kiến trúc chi tiết: [architecture.md](architecture.md)
- Lịch sử thay đổi: [changelog.md](changelog.md)
