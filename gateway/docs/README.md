---
id: DOC-S01
service: vnp-gateway
version: 1.0.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
owner: Platform Team
---

# vnp-gateway — Unified API Gateway

> **Group**: Platform | **Ports**: 8080 (REST), 8081 (gRPC), 8082 (MCP) | **Origin**: Unified

## Purpose

Single entry point for **all** VNP Memory APIs. Routes requests to 35 domain services across 6 cognitive engines (Cognee, Graphiti, Memobase, OpenViking, Zep, Supermemory) + Platform services.

### Core Responsibilities

- **Multi-Protocol Ingress**: REST (chi/v5), gRPC-Web proxy, MCP (SSE/HTTP Streamable), WebDAV, WebSocket
- **Authentication**: JWT RS256 token validation + API Key resolution → tenant isolation
- **Auto-Routing**: Content classification → automatic service targeting for `POST /v1/memory/store`
- **Rate Limiting**: Redis sliding window, per-tenant per-endpoint quotas
- **Circuit Breaking**: `sony/gobreaker` per downstream service
- **Protocol Translation**: REST ↔ gRPC bidirectional transcoding
- **Observability**: OTel traces, Prometheus metrics, structured JSON logging

## Tech Stack

- **Language**: Go 1.23+
- **HTTP Router**: chi/v5
- **gRPC**: google.golang.org/grpc + grpc-gateway
- **MCP**: Custom SSE/HTTP Streamable transport
- **WebDAV**: golang.org/x/net/webdav proxy to ov-fs
- **Auth**: go-jwt/jwt/v5 (RS256)
- **Rate Limit**: Redis 7+ sliding window
- **Circuit Breaker**: sony/gobreaker v2
- **DI**: Google Wire
- **Observability**: OTel SDK, Prometheus, slog

## Quick Start

```bash
# From monorepo root
make build-gateway
make run-gateway

# Or with Docker Compose
docker compose up vnp-gateway

# Health check
curl http://localhost:8080/healthz
# → {"status": "serving", "services": 35}
```

## API Namespaces

| Prefix | Engine | Routes |
|--------|--------|--------|
| `/v1/memory/*` | Unified | store, recall, forget, timeline |
| `/v1/cognee/*` | Cognee | datasets, cognify, search |
| `/v1/graphiti/*` | Graphiti | episodes, search, nodes, edges |
| `/v1/memobase/*` | Memobase | blobs, flush, context, profiles |
| `/v1/ov/*` | OpenViking | files, tree, grep, search, sessions |
| `/v1/zep/*` | Zep | users, sessions, memory, graph |
| `/v1/sm/*` | Supermemory | documents, memories, search, rag |
| `/v1/admin/*` | Platform | tenants, keys, health |
| `/webdav/*` | OpenViking | WebDAV file access |

## Links

- [API Reference](./api.md)
- [Architecture](./architecture.md)
- [Data Model](./data-model.md)
- [Configuration](./configuration.md)
- [Runbook](./runbook.md)
- [Changelog](./changelog.md)
- [Architecture Spec](../../specs/01-gateway.md)
- [Specs Directory](../specs/)

## Owner

- **Team**: VNP Memory — Platform
- **Contact**: TBD
