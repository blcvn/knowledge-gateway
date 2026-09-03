# Supermemory Monolith Application

Supermemory Monolith là phiên bản hợp nhất của toàn bộ backend cho dự án Supermemory, chuyển đổi từ kiến trúc nhiều Microservices thành một Go binary (Monolith) duy nhất. Ứng dụng hoạt động dưới dạng Process Supervisor để quản lý 10 business services và 1 Gateway dùng chung trong một tiến trình.

## Core Stack
- **Language:** Go 1.23+
- **Architecture:** Process Supervisor, Embedded Services
- **Communication:** gRPC (localhost), REST, NATS
- **Dependencies:** PostgreSQL, Redis, NATS

## Quick Start
```bash
# 1. Install dependencies and start infra
docker-compose up -d nats postgres redis

# 2. Build the monolith app
make build

# 3. Run the application
make run
```

## Related Documentation
- [Architecture](architecture.md)
- [Changelog](changelog.md)
- [API Reference](../../gateway/README.md) (Gateway)
