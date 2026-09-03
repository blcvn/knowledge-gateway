# Supermemory App Architecture

## Overview
Dự án Supermemory Monolith triển khai theo mẫu (pattern) **Embedded Service Supervisor**.
Ứng dụng thực chất không chứa business logic mới. Trách nhiệm của app này là làm "nhạc trưởng" gọi lệnh khởi tạo cho toàn bộ các services và expose các API qua một Gateway thống nhất.

## Module Structure Diagram
```mermaid
graph TD
    Client[Client Browser / SDK] -->|REST / HTTP| Gateway(VNP Gateway)
    Gateway -->|gRPC localhost| DocumentService
    Gateway -->|gRPC localhost| MemoryService
    Gateway -->|gRPC localhost| SearchService
    Gateway -->|gRPC localhost| ProfileService
    Gateway -->|gRPC localhost| ConnectorService
    Gateway -->|gRPC localhost| MCPService
    Gateway -->|gRPC localhost| AuthService
    Gateway -->|gRPC localhost| AnalyticsService
    Gateway -->|gRPC localhost| ProjectService
    Gateway -->|gRPC localhost| EngineService

    DocumentService -->|NATS| MemoryService
    DocumentService -->|NATS| SearchService

    subgraph "Supermemory App Process (Monolith)"
        Gateway
        DocumentService
        MemoryService
        SearchService
        ProfileService
        ConnectorService
        MCPService
        AuthService
        AnalyticsService
        ProjectService
        EngineService
        HealthAggregator
    end

    Postgres[(PostgreSQL)]
    Redis[(Redis)]
    NATS_Server((NATS))

    DocumentService -.-> Postgres
    MemoryService -.-> Postgres
    AuthService -.-> Postgres
    SearchService -.-> Redis
    ConnectorService -.-> NATS_Server
```

## Key Design Decisions
1. **Zero Modification to Services**: Tất cả code của services trong `services/sm-*` và `gateway/` không được sửa. Ứng dụng chỉ import code.
2. **Local gRPC Transport**: Các services giao tiếp với nhau bằng gRPC thông qua địa chỉ IP `127.0.0.1` với cổng khác nhau để tận dụng cơ chế Loopback cực nhanh mà không cần deploy sidecar.
3. **Phased Startup**: Hệ thống có Supervisor khởi chạy module theo đúng thứ tự logic để tránh lỗi race conditions.
4. **Unified Configuration**: Một tập hợp environment variable chung được map cho tất cả services thay vì từng config phân tán.
