---
id: TASK-001
title: Foundation Scaffold (go.work, config)
service: apps-memory
status: Done
priority: P0
created: 2026-05-14
---

# TASK-001: Foundation Scaffold

## 1. Mục Tiêu
Thiết lập nền tảng dự án cho Monolithic App `apps/memory`, bao gồm Go workspace (`go.work`) và cấu trúc cấu hình tập trung (unified config).
**Tối ưu token:** Cung cấp sẵn nội dung `go.work` và cấu trúc config, agent thực thi chỉ việc copy-paste và tạo file.

## 2. Các Bước Thực Thi

1. Tạo thư mục `apps/memory` và khởi tạo Go module: `go mod init github.com/vnp-community/vnp-memory/apps/memory`
2. Tạo file `go.work` tại root của monorepo.
3. Cài đặt package struct `Config` tại `apps/memory/internal/config/config.go`.
4. Tạo file cấu hình mặc định tại `apps/memory/configs/config.yaml`.

## 3. Nội Dung Code (Dùng trực tiếp để tiết kiệm token)

### `go.work` (Root folder)
```go
go 1.25.0

use (
    ./gateway
    ./services/cognee-ingestion
    ./services/cognee-cognify
    ./services/cognee-search
    ./services/graphiti-ingestion
    ./services/graphiti-search
    ./services/graphiti-knowledge
    ./services/graphiti-store
    ./services/memobase-ingestion
    ./services/memobase-engine
    ./services/memobase-context
    ./services/ov-fs
    ./services/ov-search
    ./services/ov-session
    ./services/ov-resource
    ./services/ov-crypto
    ./services/ov-admin
    ./services/zep-user
    ./services/zep-thread
    ./services/zep-memory
    ./services/zep-graph
    ./services/zep-search
    ./services/zep-admin
    ./services/sm-document
    ./services/sm-memory
    ./services/sm-search
    ./services/sm-profile
    ./services/sm-connector
    ./services/sm-mcp
    ./services/sm-auth
    ./services/sm-analytics
    ./services/sm-project
    ./services/vnp-event
    ./services/vnp-search-hub
    ./services/vnp-admin
    ./apps/memory
    ./pkg
)
```

## 4. Acceptance Criteria
- [ ] `go.work` được tạo ở root và include đủ các services.
- [ ] `apps/memory/internal/config/config.go` định nghĩa đủ các struct (Server, Auth, Postgres, Neo4j, Redis, Qdrant, NATS, và config cho các engines).
- [ ] `apps/memory/configs/config.yaml` chứa các giá trị mặc định cho Local development.
