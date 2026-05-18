# Memobase Monolith App

> **Single-binary deployment** of the Memobase memory platform — 4 microservices + gateway in one process.

## Overview

`memobase-app` hợp nhất 4 Memobase microservices và VNP Gateway thành **1 binary duy nhất**, sử dụng **supervisor pattern** đã proven từ `apps/cognee` và `apps/graphiti`.

### Embedded Services

| Service | gRPC Port | Responsibility |
|---------|----------|----------------|
| `memobase-ingestion` | 9041 | Blob insert, Buffer Zone, Flush trigger |
| `memobase-engine` | 9042 | Profile extraction, YOLO merge, Event summary |
| `memobase-context` | 9043 | Context assembly, Profile query |
| `memobase-pipeline` | 9044 | Pipeline orchestration |
| `vnp-gateway` | 8080 | REST API, Auth, MCP Server |

### Tech Stack

- **Language**: Go 1.23+
- **Internal RPC**: gRPC (localhost loopback)
- **Async**: NATS JetStream
- **Database**: PostgreSQL + pgvector
- **Cache**: Redis 7+
- **LLM**: Bifrost / OpenAI-compatible

## Quick Start

```bash
# 1. Clone & navigate
cd apps/memobase

# 2. Start infrastructure
make compose-up

# 3. Build
make build

# 4. Run
make run

# 5. Test
curl http://localhost:8080/api/v1/healthcheck
```

## Project Structure

```
apps/memobase/
├── cmd/memobase/
│   ├── main.go         # Entry point
│   ├── services.go     # Service start functions
│   ├── gateway.go      # Gateway start function
│   └── health.go       # Health aggregation server
├── internal/
│   ├── config/         # Unified configuration
│   └── supervisor/     # Service lifecycle manager
├── docs/               # Documentation
├── specs/              # Specifications
│   ├── solutions/      # Solution specs
│   └── tasks/          # Task specs
├── config.yaml         # Default config
├── .env.example        # ENV template
├── Dockerfile          # Multi-stage build
├── Makefile            # Build targets
└── docker-compose.yml  # Local dev
```

## Links

- [Architecture](architecture.md)
- [Configuration](configuration.md)
- [Runbook](runbook.md)
- [API Reference](api.md)
- [Changelog](changelog.md)

## Design Principle

**ZERO code changes** to existing services and gateway. This app only adds orchestration code (~500 lines) to embed existing services in a single process using gRPC localhost and NATS.
