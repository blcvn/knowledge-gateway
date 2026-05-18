# Graphiti App — Embedded Multi-Service Monolith

> Single-binary deployment of 5 graphiti microservices + gateway via the **Embedded Service Supervisor** pattern.

## Quick Start

```bash
# Option 1: Docker Compose (recommended)
cd apps/graphiti
docker compose up -d
curl http://localhost:9090/healthz    # Health check
curl http://localhost:8080/healthz    # Gateway check

# Option 2: Local binary
make run   # Starts with AUTH_DEV_MODE=true
```

## Architecture

The graphiti app runs **5 gRPC services + 1 HTTP gateway** as goroutines within a single process:

```
                ┌──── SINGLE PROCESS (graphiti-app) ─────────────────────┐
                │                                                        │
Client ───────→ │ Gateway REST (:8080) + MCP (:8082)                     │
                │   │ gRPC localhost                                      │
                │   ├──→ graphiti-store      (:9024)  [Phase 0: Data]    │
                │   ├──→ graphiti-knowledge  (:9023)  [Phase 1: Intel]   │
                │   ├──→ graphiti-ingestion  (:9021)  [Phase 2: App]     │
                │   ├──→ graphiti-search     (:9022)  [Phase 2: App]     │
                │   └──→ graphiti-pipeline   (:9025)  [Phase 2: App]     │
                │                                                        │
                │ NATS ←───→ async events                                │
                │ Health (:9090) ← /healthz, /readyz                     │
                └────────────────────────────────────────────────────────┘
```

**Key property: ZERO changes** to existing services (`services/graphiti-*`) or gateway (`gateway/`) code.

See [architecture.md](./architecture.md) for detailed design.

## API Routes

| Method | Path | Service | Status |
|--------|------|---------|--------|
| POST | `/v1/graphiti/episodes` | ingestion | Proxy stub |
| POST | `/v1/graphiti/episodes/bulk` | ingestion | Proxy stub |
| GET | `/v1/graphiti/episodes` | ingestion | Proxy stub |
| DELETE | `/v1/graphiti/episodes/{id}` | ingestion | Proxy stub |
| POST | `/v1/graphiti/search` | search | Proxy stub |
| POST | `/v1/graphiti/search/advanced` | search | Proxy stub |
| POST | `/v1/graphiti/search/nodes` | search | Proxy stub |
| POST | `/v1/graphiti/search/communities` | search | Proxy stub |
| POST | `/v1/graphiti/triplets` | knowledge | Proxy stub |
| POST | `/v1/graphiti/communities/rebuild` | pipeline | Proxy stub |
| DELETE | `/v1/graphiti/data` | store | Proxy stub |
| GET | `/healthz` | gateway | ✅ Live |
| GET | `/readyz` | gateway | ✅ Live |

## Configuration

All settings via ENV vars. See [configuration.md](./configuration.md) and `config.yaml`.

## Development

```bash
make build       # Compile binary
make test        # Unit tests + coverage
make lint        # golangci-lint
make docker      # Build Docker image
make up          # Start full dev stack
make down        # Stop dev stack
make help        # Show all targets
```

## Project Structure

```
apps/graphiti/
├── cmd/graphiti/
│   ├── main.go          ← Entry point: config → supervisor → signal → shutdown
│   ├── services.go      ← 5 service start wrappers
│   ├── gateway.go       ← HTTP gateway with gRPC registry + Graphiti routes
│   └── health.go        ← Aggregated health server
├── internal/
│   ├── config/          ← Unified config + ENV mapping
│   ├── embed/           ← Generic gRPC service bootstrap
│   ├── gateway/         ← gRPC proxy registry
│   └── supervisor/      ← 4-phase goroutine lifecycle manager
├── Dockerfile           ← Multi-stage build
├── Makefile             ← Build/test/deploy targets
├── docker-compose.yml   ← Local dev with all infra
├── config.yaml          ← Config reference
└── .env.example         ← ENV var template
```
