# Architecture — Graphiti Embedded Monolith

## Overview

The Graphiti app consolidates 5 microservices and 1 HTTP gateway into a **single Go binary** using the **Embedded Service Supervisor** pattern. Services communicate via gRPC over `localhost` and NATS JetStream, identical to a distributed deployment but without network hops.

## Design Goals

1. **Zero Code Changes** — `services/graphiti-*` and `gateway/` are not modified
2. **Single Binary** — one `graphiti-app` binary for all services
3. **Production Grade** — phase-ordered startup, graceful shutdown, panic recovery
4. **Same Protocols** — gRPC + NATS, identical to microservice deployment

## Component Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    graphiti-app process                              │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │                    Supervisor                                │   │
│  │                                                              │   │
│  │  Phase 0 (Data)         ┌─────────────────┐                 │   │
│  │    graphiti-store ──────→│   Neo4j         │                 │   │
│  │    :9024                 └─────────────────┘                 │   │
│  │                                                              │   │
│  │  Phase 1 (Intelligence) ┌─────────────────┐                 │   │
│  │    graphiti-knowledge ──→│   LLM Provider  │                 │   │
│  │    :9023                 └─────────────────┘                 │   │
│  │       │ gRPC localhost                                       │   │
│  │       └──→ graphiti-store                                    │   │
│  │                                                              │   │
│  │  Phase 2 (Application)                                       │   │
│  │    graphiti-ingestion :9021 ──→ knowledge, store             │   │
│  │    graphiti-search    :9022 ──→ knowledge, store             │   │
│  │    graphiti-pipeline  :9025 ──→ ingestion, knowledge, store  │   │
│  │                                                              │   │
│  │  Phase 3 (Gateway)                                           │   │
│  │    vnp-gateway :8080 (REST) + :8082 (MCP)                   │   │
│  │       │ gRPC localhost registry                              │   │
│  │       ├──→ graphiti-ingestion                                │   │
│  │       ├──→ graphiti-search                                   │   │
│  │       ├──→ graphiti-knowledge                                │   │
│  │       ├──→ graphiti-store                                    │   │
│  │       └──→ graphiti-pipeline                                 │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌────────────────┐   ┌──────────────┐                             │
│  │ Health :9090   │   │ NATS Client  │                             │
│  │ /healthz       │   │ JetStream    │                             │
│  │ /readyz        │   │ async events │                             │
│  └────────────────┘   └──────────────┘                             │
└─────────────────────────────────────────────────────────────────────┘
         │                      │                      │
    ┌────▼────┐          ┌──────▼──────┐        ┌──────▼──────┐
    │  Neo4j  │          │    Redis    │        │    NATS     │
    │  :7687  │          │    :6379    │        │    :4222    │
    └─────────┘          └─────────────┘        └─────────────┘
```

## Startup Sequence

The Supervisor starts services in 4 phases with dependency gating:

```
Phase 0 (Data)          → graphiti-store (waits for TCP :9024)
      │ ready
Phase 1 (Intelligence)  → graphiti-knowledge (waits for TCP :9023)
      │ ready
Phase 2 (Application)   → graphiti-ingestion, graphiti-search, graphiti-pipeline
      │ ready              (parallel start, waits for all ports)
Phase 3 (Gateway)       → vnp-gateway REST + MCP
      │ ready
      ▼
ALL SERVICES READY — /readyz returns 200
```

Each phase waits for TCP port readiness before launching the next phase.

## Shutdown Sequence

Reverse phase order ensures clients disconnect before data stores:

```
Phase 3 (Gateway)       → cancel context → GracefulStop → wait
Phase 2 (Application)   → cancel context → GracefulStop → wait
Phase 1 (Intelligence)  → cancel context → GracefulStop → wait
Phase 0 (Data)          → cancel context → GracefulStop → wait
```

Default shutdown timeout: **30 seconds** (configurable via `SHUTDOWN_TIMEOUT`).

## Key Packages

| Package | Purpose |
|---------|---------|
| `internal/config` | Unified config struct, ENV loading, service ENV injection |
| `internal/supervisor` | Phase-ordered lifecycle manager with panic recovery |
| `internal/embed` | Generic gRPC service bootstrap (OTel + tenant interceptor) |
| `internal/gateway` | gRPC client registry for localhost connections |
| `cmd/graphiti/` | Entry point, service registration, gateway, health server |

## Inter-Service Communication

### gRPC (Synchronous)

All services bind to `localhost:PORT` within the same process. The gateway's gRPC registry connects to these localhost addresses via `insecure.NewCredentials()` (no TLS needed within localhost).

### NATS JetStream (Asynchronous)

Services publish/subscribe to NATS streams for:
- Ingestion events (episode added/removed)
- Pipeline triggers (community rebuild)
- Search index updates

## Monolith vs Microservices Comparison

| Aspect | Monolith (graphiti-app) | Microservices |
|--------|------------------------|---------------|
| Deployment | Single binary | 5 containers + gateway |
| Communication | gRPC localhost (µs) | gRPC network (ms) |
| Config | Single .env file | 6 separate configs |
| Health | Aggregated /readyz | Individual health checks |
| Scaling | Vertical only | Horizontal per service |
| Debugging | Single log stream | Distributed tracing required |
| Code changes | None to services | None to services |

## Design Decisions

1. **TCP Port Probing** vs gRPC Health Check — TCP probing is simpler and doesn't require gRPC client setup during startup. It catches both server startup and binding failures.

2. **Generic StartGRPCService** — Single function handles all 5 services because they follow identical bootstrap patterns. Service-specific logic is in the service packages themselves.

3. **Gateway as Phase 3** — Gateway starts last to ensure all backend gRPC services are ready before accepting client requests.

4. **Panic Recovery per Goroutine** — Each service goroutine has its own recover() to prevent one service crash from taking down the entire process.
