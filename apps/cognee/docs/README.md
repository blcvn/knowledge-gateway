# Cognee App — Embedded Multi-Service Monolith

> Single-binary deployment of 3 cognee microservices + gateway via the **Embedded Service Supervisor** pattern.

## Quick Start

```bash
# Option 1: Docker Compose (recommended)
cd apps/cognee
docker compose up -d
curl http://localhost:11080/healthz   # Health check
curl http://localhost:8080/healthz    # Gateway check

# Option 2: Local binary
make run   # Starts with AUTH_DEV_MODE=true
```

## Architecture

The cognee app runs **3 gRPC services + 1 HTTP gateway** as goroutines within a single process:

```
                ┌──── SINGLE PROCESS (cognee-app) ─────────────────────┐
                │                                                      │
Client ───────→ │ Gateway REST (:8080)                                 │
                │   │ gRPC localhost                                    │
                │   ├──→ cognee-ingestion (:9011)                      │
                │   ├──→ cognee-cognify   (:9012)                      │
                │   └──→ cognee-search    (:9013)                      │
                │                                                      │
                │ NATS ←───→ async events                              │
                │ Health (:11080) ← /healthz, /readyz                  │
                └──────────────────────────────────────────────────────┘
```

**Key property: ZERO changes** to existing services (`services/cognee-*`) or gateway (`gateway/`) code.

See [architecture.md](./architecture.md) for detailed design.

## API Routes

| Method | Path | Service | Status |
|--------|------|---------|--------|
| POST | `/v1/cognee/datasets` | ingestion | Proxy stub |
| POST | `/v1/cognee/datasets/{id}/data` | ingestion | Proxy stub |
| GET | `/v1/cognee/datasets` | ingestion | Proxy stub |
| DELETE | `/v1/cognee/datasets/{id}` | ingestion | Proxy stub |
| POST | `/v1/cognee/datasets/{id}/cognify` | cognify | Proxy stub |
| GET | `/v1/cognee/cognify/{id}/status` | cognify | Proxy stub |
| POST | `/v1/cognee/search` | search | Proxy stub |
| POST | `/v1/cognee/search/rag` | search | Proxy stub |
| GET | `/v1/cognee/search/explore` | search | Proxy stub |
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
apps/cognee/
├── cmd/cognee/
│   ├── main.go          ← Entry point: config → supervisor → signal → shutdown
│   ├── services.go      ← Service start wrappers
│   ├── gateway.go       ← HTTP gateway with gRPC registry
│   └── health.go        ← Aggregated health server
├── internal/
│   ├── config/          ← Unified config + ENV mapping
│   ├── embed/           ← Generic gRPC service bootstrap
│   ├── gateway/         ← gRPC proxy registry
│   ├── handler/         ← Health handler
│   └── supervisor/      ← Goroutine lifecycle manager
├── Dockerfile           ← Multi-stage build
├── Makefile             ← Build/test/deploy targets
├── docker-compose.yml   ← Local dev with all infra
├── config.yaml          ← Config reference
└── .env.example         ← ENV var template
```
