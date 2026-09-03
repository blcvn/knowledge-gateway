# VNP Gateway

Unified API gateway for the VNP Memory ecosystem. Routes memory operations across 6 cognitive engines (Cognee, Graphiti, Memobase, OpenViking, Zep, Supermemory) via REST, gRPC, MCP, and WebDAV protocols.

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    VNP Gateway :8080                       │
│                                                            │
│  ┌─ REST ──┐  ┌─ gRPC ──┐  ┌─ MCP ───┐  ┌─ WebDAV ─┐    │
│  │  :8080  │  │  :8081  │  │  :8082  │  │  :8080   │    │
│  └────┬────┘  └────┬────┘  └────┬────┘  └────┬─────┘    │
│       │            │            │             │          │
│  ┌────┴────────────┴────────────┴─────────────┴──┐       │
│  │           Middleware Pipeline                  │       │
│  │  Recovery → RequestID → Logger → CORS → Auth  │       │
│  │  → RateLimit → Metrics → Timeout             │       │
│  └───────────────────┬──────────────────────────┘       │
│                      │                                    │
│  ┌───────────────────┴──────────────────────────┐       │
│  │            Usecase Layer                      │       │
│  │   RouteUC   │  AuthUC   │  RateLimitUC       │       │
│  └──────┬──────┴──────┬────┴───────┬────────────┘       │
│         │             │            │                      │
│  ┌──────┴─────────────┴────────────┴──────────────┐     │
│  │       Infrastructure Layer                      │     │
│  │  GRPCRegistry → CircuitBreaker → 35 Services   │     │
│  │  Redis (RateLimit) │ Postgres (Tenants/Keys)   │     │
│  │  NATS (Events)     │ Prometheus (Metrics)      │     │
│  └────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────┘
```

## Quick Start

```bash
# Local development (with Docker)
docker compose up -d

# Or run directly
AUTH_DEV_MODE=true go run ./cmd/main.go

# Run tests
make test

# Build binary
make build
```

## API Namespaces

| Prefix | Engine | Routes | Description |
|--------|--------|--------|-------------|
| `/v1/memory/*` | Auto-route | 4 | Unified memory: store, recall, forget, timeline |
| `/v1/cognee/*` | Cognee | 4 | Datasets, cognify, semantic search |
| `/v1/graphiti/*` | Graphiti | 4 | Episodes, graph search, nodes, edges |
| `/v1/memobase/*` | Memobase | 5 | Blobs, flush, context, profiles, events |
| `/v1/ov/*` | OpenViking | 10 | Files, tree, grep, sessions, resources |
| `/v1/zep/*` | Zep | 9 | Users, memory, graph search, facts, ontology |
| `/v1/sm/*` | Supermemory | 9 | Documents, memories, search, RAG, profiles |
| `/v1/admin/*` | Admin | 4 | Tenants, API keys, health, metrics |
| `/webdav/*` | OpenViking | — | WebDAV file access |
| `/mcp/*` | MCP Server | 16 tools | AI agent tool integration |

## MCP Tools (16)

| Tool | Description |
|------|-------------|
| `memory_store` | Store with auto-classification |
| `memory_recall` | Cross-engine semantic recall |
| `memory_search` | Knowledge graph search |
| `memory_timeline` | Temporal event query |
| `memory_profile` | User profile from context |
| `memory_forget` | Cascading delete |
| `graph_query` | Knowledge graph with filters |
| `ov_read_file` | Read from context DB |
| `ov_write_file` | Write to context DB |
| `ov_search` | Hierarchical semantic search |
| `ov_list_dir` | List directory contents |
| `ov_grep` | Regex file search |
| `ov_tree` | Directory tree |
| `ov_session_commit` | Commit editing session |
| `ov_ingest` | Ingest resource |
| `ov_delete` | Delete file/resource |

## Configuration

All configuration via environment variables. See `.env.example` for full list.

| Variable | Default | Description |
|----------|---------|-------------|
| `REST_PORT` | 8080 | HTTP REST API port |
| `GRPC_PORT` | 8081 | gRPC port |
| `MCP_PORT` | 8082 | MCP server port |
| `HEALTH_PORT` | 11080 | Health/metrics port |
| `AUTH_DEV_MODE` | false | Skip auth for development |
| `REDIS_ADDR` | redis:6379 | Redis for rate limiting |
| `POSTGRES_DSN` | — | PostgreSQL for tenants/keys |
| `NATS_URL` | nats://nats:4222 | NATS for domain events |

## Project Structure

```
gateway/
├── cmd/main.go                          # Entry point
├── internal/
│   ├── domain/                          # Layer 1 — Zero external deps
│   │   ├── entity.go                    # AuthContext, RouteTarget, StoreRequest
│   │   ├── errors.go                    # 7 sentinel errors (gRPC/HTTP mapped)
│   │   └── event.go                     # 5 NATS domain events
│   ├── usecase/                         # Layer 2 — Business logic
│   │   ├── port/input.go               # 4 input interfaces
│   │   ├── port/output.go              # 5 output interfaces
│   │   ├── route.go                     # Content classification + routing
│   │   ├── auth.go                      # JWT RS256 + API Key auth
│   │   └── ratelimit.go                 # Per-tenant rate limiting
│   ├── adapter/                         # Layer 3 — Protocol adapters
│   │   ├── handler/                     # 50+ REST route handlers
│   │   ├── client/                      # gRPC registry + circuit breaker
│   │   ├── mcp/                         # MCP server (16 tools)
│   │   └── webdav/                      # WebDAV reverse proxy
│   └── infra/                           # Layer 4 — Infrastructure
│       ├── config/                      # Viper config (40+ env vars)
│       ├── middleware/                  # HTTP middleware + Prometheus
│       ├── persistence/                 # Redis rate limiter
│       └── server/                      # HTTP + observability servers
├── tests/integration/                   # Integration test suite
├── docs/                                # 7 operational documents
├── specs/                               # 23 specification artifacts
├── Dockerfile                           # Multi-stage build
├── docker-compose.yml                   # Local dev stack
├── Makefile                             # Build automation
└── .env.example                         # Configuration template
```

## Development

```bash
make help              # Show all targets
make build             # Build binary
make test              # Run all tests
make test-domain       # Domain unit tests only
make test-integration  # Integration tests only
make test-cover        # Tests with coverage report
make lint              # Run go vet
make check-domain      # Verify domain has zero external deps
make run               # Run locally (dev mode)
make docker            # Build Docker image
make info              # Show project stats
```

## License

VNP Community — Internal Use
