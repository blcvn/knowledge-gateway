# Zep Monolith Application

Ứng dụng **Zep Monolith** là một single-binary go application đóng gói Gateway và 6 microservices nội bộ (`zep-admin`, `zep-user`, `zep-thread`, `zep-memory`, `zep-graph`, `zep-search`). 

## Mục Đích
Thay vì phải deploy 7 container riêng biệt cho hệ thống Zep, Monolith gom toàn bộ vào một process chạy trên OS. Các module giao tiếp với nhau qua localhost gRPC hoặc NATS JetStream, giữ nguyên kiến trúc Zero-modification để đảm bảo tính nhất quán với code base gốc.

## Business Capability
Zep Monolith cung cấp toàn bộ các khả năng của Zep Context Engine:
- **Graph Extraction:** Xử lý và trích xuất Knowledge Graph.
- **Search & Memory:** Vector search và lưu trữ semantic context.
- **User & Thread:** Quản lý hội thoại và context user.
- **Admin:** Quản trị hệ thống.

## Tech Stack
- Ngôn ngữ: Golang 1.23+
- Framework: gRPC, gRPC-Gateway, NATS JetStream, Neo4j, Redis, PostgreSQL (pgvector).
- Deployment: Docker, Docker Compose.

## Quick Start
```bash
# Clone và chuyển vào thư mục zep monolith
cd apps/zep

# Tải dependencies
go mod tidy

# Build ứng dụng
make build

# Khởi chạy cùng các DB (PostgreSQL, Redis, Neo4j, NATS)
docker-compose up -d
```

## Các Tài Liệu Liên Quan
- [Architecture](docs/architecture.md)
- [API Reference](docs/api.md)
- [Configuration](docs/configuration.md)
- [Runbook](docs/runbook.md)
- [Changelog](docs/changelog.md)
