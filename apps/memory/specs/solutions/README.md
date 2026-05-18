# VNP Memory — Monolithic App Solutions

## Solution Cluster Overview

Bộ giải pháp xây dựng **single Go binary** hợp nhất 35 domain services + gateway, tái sử dụng 100% code từ `gateway/` và `services/` mà **không sửa đổi**.

## Solutions

| ID | Document | Description | Priority |
|----|----------|-------------|----------|
| SOL-001 | [Monolithic Architecture](SOL-001-monolithic-architecture.md) | Kiến trúc tổng thể, import strategy, app structure | P0 |
| SOL-002 | [In-Process Communication](SOL-002-inprocess-communication.md) | gRPC bufconn + NATS embedded cho inter-module | P0 |
| SOL-003 | [Service Bootstrap](SOL-003-service-bootstrap.md) | Wire 35 services, Go workspace, bootstrap pattern | P0 |
| SOL-004 | [Config & Deployment](SOL-004-config-deployment.md) | Unified config, Dockerfile, docker-compose | P1 |
| SOL-005 | [Implementation Roadmap](SOL-005-implementation-roadmap.md) | 32 tasks, 6-week phased plan, dependency graph | P0 |

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Inter-service sync | gRPC bufconn (in-process) | ~0.05ms latency vs ~2ms TCP |
| Inter-service async | NATS embedded (DontListen) | Zero network, JetStream durable |
| Code reuse | Direct Go import | No duplication, same packages |
| Module management | Go workspace (go.work) | Local module resolution |
| Config | Single YAML + ENV override | One place to configure all |

## Constraints

- ❌ KHÔNG sửa đổi code trong `gateway/`, `services/`, `pkg/`
- ✅ Import và tái sử dụng trực tiếp internal packages
- ✅ Giữ nguyên gRPC API contracts, NATS subjects, DB schemas
