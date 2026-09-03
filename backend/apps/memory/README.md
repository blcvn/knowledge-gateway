# VNP Memory Monolith

> A single Go binary consolidating 35+ domain services and the gateway into one high-performance application with in-process gRPC and embedded NATS.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    VNP Memory Monolith                          │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐       │
│  │ REST API │  │ MCP SSE  │  │ Health   │  │ gRPC Bus │       │
│  │ :8080    │  │ :8082    │  │ :8083    │  │ bufconn  │       │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘       │
│       │              │              │              │             │
│  ┌────▼──────────────▼──────────────▼──────────────▼─────┐     │
│  │                  Gateway (Router + Handlers)           │     │
│  └────────────────────────┬──────────────────────────────┘     │
│                           │ InProcessRegistry                   │
│  ┌────────────────────────▼──────────────────────────────┐     │
│  │                  Engine Services (35)                   │     │
│  │  Cognee(3) Graphiti(4) Memobase(3) OV(6) Zep(6) SM(9)│     │
│  │  Platform: vnp-admin, vnp-event, search-hub, platform │     │
│  └────────────────────────┬──────────────────────────────┘     │
│                           │                                     │
│  ┌────────────────────────▼──────────────────────────────┐     │
│  │              Embedded NATS + JetStream                 │     │
│  │              7 streams, WorkQueue retention            │     │
│  └───────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────────┘
           │           │           │           │
      PostgreSQL    Neo4j      Qdrant     Redis/MinIO
```

## Quick Start

### 1. Start Infrastructure

```bash
make infra-up
```

This starts 5 containers:
- **PostgreSQL** (pgvector) — `localhost:5432`
- **Neo4j** — `localhost:7474` / `bolt://localhost:7687`
- **Qdrant** — `localhost:6333`
- **Redis** — `localhost:6379`
- **MinIO** — `localhost:9000` (console: `localhost:9001`)

### 2. Run the Monolith

```bash
# Option A: Direct Go run
make dev

# Option B: Build and run binary
make build && make run

# Option C: Full Docker stack
make docker-up
```

### 3. Verify

```bash
# Health check (lists all 35 services)
curl http://localhost:8083/healthz | jq

# REST API
curl -X POST http://localhost:8080/v1/memory/store \
  -H "Content-Type: application/json" \
  -d '{"content":"hello world","type":"fact"}'

# MCP tools
curl -X POST http://localhost:8082/message \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## Configuration

Configuration is loaded from `configs/config.yaml` and can be overridden with environment variables:

```bash
# Pattern: VNP_MEMORY_<SECTION>_<KEY>
export VNP_MEMORY_SERVER_LOG_LEVEL=debug
export VNP_MEMORY_POSTGRES_DSN="postgres://user:pass@host:5432/db"
export VNP_MEMORY_AUTH_DEV_MODE=false
export VNP_MEMORY_NATS_MODE=external
export VNP_MEMORY_NATS_URL=nats://nats-cluster:4222
```

### Key Configuration Options

| Variable | Default | Description |
|----------|---------|-------------|
| `VNP_MEMORY_SERVER_REST_PORT` | `8080` | REST API port |
| `VNP_MEMORY_SERVER_MCP_PORT` | `8082` | MCP server port |
| `VNP_MEMORY_SERVER_HEALTH_PORT` | `8083` | Health/metrics port |
| `VNP_MEMORY_AUTH_DEV_MODE` | `true` | Skip JWT validation |
| `VNP_MEMORY_NATS_MODE` | `embedded` | `embedded` or `external` |
| `VNP_MEMORY_NATS_URL` | `` | NATS URL (when mode=external) |

## Project Structure

```
apps/memory/
├── cmd/server/main.go          # Entry point
├── configs/config.yaml         # Default configuration
├── internal/
│   ├── bootstrap/              # Service initialization
│   │   ├── infra.go            # Shared infrastructure (PG, Redis, Neo4j, Qdrant)
│   │   ├── platform.go         # vnp-admin, vnp-event, vnp-search-hub
│   │   ├── gateway.go          # Gateway handlers + router
│   │   ├── cognee.go           # Cognee engine (3 services)
│   │   ├── graphiti.go         # Graphiti engine (4 services)
│   │   ├── memobase.go         # Memobase engine (3 services)
│   │   ├── openviking.go       # OpenViking engine (6 services)
│   │   ├── zep.go              # Zep engine (6 services)
│   │   └── supermemory.go      # Supermemory engine (9 services)
│   ├── bus/                    # In-process communication
│   │   ├── grpc_bus.go         # bufconn gRPC bus
│   │   ├── nats_embedded.go    # Embedded NATS + JetStream
│   │   └── registry.go         # InProcessRegistry (implements ServiceRegistry)
│   └── config/config.go        # Unified configuration
├── tests/e2e_test.go           # E2E smoke tests
├── Dockerfile                  # Multi-stage build
├── docker-compose.yml          # Full stack (app + infra)
├── docker-compose.infra.yml    # Infrastructure only
└── Makefile                    # Build automation
```

## Make Commands

| Command | Description |
|---------|-------------|
| `make build` | Build Go binary |
| `make run` | Build and run |
| `make dev` | Start infra + run locally |
| `make docker` | Build Docker image |
| `make docker-up` | Start full stack |
| `make docker-down` | Stop full stack |
| `make infra-up` | Start infrastructure only |
| `make infra-down` | Stop infrastructure |
| `make infra-reset` | Stop infra + delete volumes |
| `make test` | Run unit tests |
| `make test-e2e` | Run E2E tests |
| `make lint` | Run linter |
| `make clean` | Remove all artifacts |

## Services (35 total)

| Engine | Services | Count |
|--------|----------|-------|
| **Cognee** | cognee-ingestion, cognee-cognify, cognee-search | 3 |
| **Graphiti** | graphiti-ingestion, graphiti-search, graphiti-knowledge, graphiti-store | 4 |
| **Memobase** | memobase-ingestion, memobase-engine, memobase-context | 3 |
| **OpenViking** | ov-fs, ov-search, ov-session, ov-resource, ov-crypto, ov-admin | 6 |
| **Zep** | zep-user, zep-thread, zep-memory, zep-graph, zep-search, zep-admin | 6 |
| **Supermemory** | sm-document, sm-memory, sm-search, sm-profile, sm-connector, sm-mcp, sm-auth, sm-analytics, sm-project | 9 |
| **Platform** | vnp-admin, vnp-event, vnp-search-hub, vnp-platform | 4 |

## Graceful Shutdown

`Ctrl+C` triggers orderly shutdown:
1. HTTP servers stop accepting new connections
2. In-flight requests drain (30s timeout)
3. NATS JetStream connections drain
4. gRPC bus gracefully stops
5. Database connection pools close
