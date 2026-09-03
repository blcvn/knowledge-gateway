# Deployment Diagram

> Deployment topology của VNP Memory trong 2 mode: Development và Production.

---

## Mode 1 — Development (Monolith)

```
Developer Machine
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  $ make infra-up                          $ make dev               │
│  ┌──────────────────────────────┐         ┌──────────────────────┐ │
│  │  Docker Compose (5 containers)│         │  Single Go Binary    │ │
│  │                               │         │  backend/apps/memory │ │
│  │  postgres:5432  (pgvector)    │◄────────│                      │ │
│  │  neo4j:7687     (bolt)        │         │  :8080 REST API      │ │
│  │  redis:6379     (cache)       │         │  :8082 MCP Server    │ │
│  │  minio:9000     (S3)          │         │  :8083 Health/Metrics│ │
│  │  nats:4222      (embedded)    │         │                      │ │
│  └──────────────────────────────┘         │  InProcessRegistry   │ │
│                                            │  35+ services via    │ │
│  Time to start: < 60 seconds              │  bufconn (0 latency) │ │
│                                            └──────────────────────┘ │
│                                                                     │
│  Verify:                                                            │
│  curl http://localhost:8083/healthz | jq                            │
└─────────────────────────────────────────────────────────────────────┘
```

**Đặc điểm Development:**
- 1 binary duy nhất chứa tất cả 35+ services
- bufconn: gRPC in-memory pipe (zero network latency)
- NATS embedded (không cần external broker)
- Graceful shutdown: HTTP drain → NATS drain → gRPC stop → DB close

---

## Mode 2 — Production (Distributed)

```
Internet
    │ HTTPS
    ▼
┌──────────────────────────────────────┐
│        Load Balancer / CDN           │
│        (Nginx / Cloudflare)          │
└──────────────────┬───────────────────┘
                   │
    ┌──────────────▼──────────────────────────────────────┐
    │              API Gateway Pods (×2-3)                 │
    │              backend/gateway                         │
    │  :8080 REST  :8082 MCP  :8083 Health/Prometheus     │
    │  Auth middleware → TenantID → InProcessRegistry      │
    └──────┬───────────┬───────────┬───────────┬──────────┘
           │           │           │           │
           │ gRPC (network)        │           │
    ┌──────▼──────┐ ┌─▼─────────┐ │ ┌─────────▼──────┐
    │ Memory      │ │AgentMemory│ │ │ Platform       │
    │ Engine Pods │ │Layer Pods │ │ │ Services Pods  │
    │             │ │           │ │ │                │
    │ cognee ×2   │ │observe ×2 │ │ │ vnp-platform ×2│
    │ graphiti ×2 │ │memory ×2  │ │ │ vnp-event ×2   │
    │ memobase ×2 │ │orchestrat.│ │ │ vnp-dashboard  │
    │ zep ×2      │ │pipeline ×2│ │ └────────────────┘
    │ openviking  │ └───────────┘ │
    │ supermemory │               │
    └─────────────┘     ┌─────────▼──────────────────────┐
                        │  NATS JetStream Cluster (×3)   │
                        │  At-least-once delivery        │
                        │  Streams: memory.*, agent.*    │
                        └────────────────────────────────┘
                                    │
    ┌───────────────────────────────┼────────────────────────────────┐
    │                      Shared Infrastructure                     │
    │                                                                │
    │  PostgreSQL 17 HA         Neo4j Cluster (×3)                  │
    │  (primary + 2 replicas)   (bolt://neo4j:7687)                 │
    │  pgvector extension                                           │
    │                                                               │
    │  Redis Sentinel (×3)      MinIO Distributed (×4)             │
    │  (master + 2 sentinels)   (S3 compatible, erasure coding)    │
    │                                                               │
    │  Qdrant (optional, ×2)    LLM Proxy (Bifrost)               │
    │  (high-scale vector)      (multi-provider routing)           │
    └───────────────────────────────────────────────────────────────┘
                                    │
    ┌───────────────────────────────▼────────────────────────────────┐
    │                      Observability Stack                       │
    │  Prometheus (:8083 scrape)  Grafana (dashboards)              │
    │  OpenTelemetry Collector    Loki (logs)                        │
    └────────────────────────────────────────────────────────────────┘
```

---

## Environment Variables — Key Config

```bash
# === Core ===
VNP_MEMORY_ENV=production        # development | production
VNP_MEMORY_LOG_LEVEL=info        # debug | info | warn | error
VNP_MEMORY_SECRET_KEY=<32-byte>  # JWT signing key

# === Database ===
VNP_MEMORY_POSTGRES_DSN=postgres://user:pass@host:5432/vnpmemory
VNP_MEMORY_NEO4J_URI=bolt://neo4j:7687
VNP_MEMORY_REDIS_URL=redis://redis:6379
VNP_MEMORY_MINIO_ENDPOINT=minio:9000

# === Message Broker ===
VNP_MEMORY_NATS_MODE=external    # embedded (dev) | external (prod)
VNP_MEMORY_NATS_URL=nats://nats:4222

# === LLM ===
VNP_MEMORY_LLM_PROVIDER=openai   # openai | anthropic | google | ollama
OPENAI_API_KEY=sk-...
VNP_MEMORY_BIFROST_URL=http://bifrost:8090

# === Ports ===
VNP_MEMORY_REST_PORT=8080
VNP_MEMORY_MCP_PORT=8082
VNP_MEMORY_HEALTH_PORT=8083
```

---

## Docker Compose Services (Production)

```yaml
# deployment/dev/docker-compose.server.yaml
services:
  gateway:        # API Gateway — REST + MCP + Health
  cognee:         # Semantic Memory engine cluster
  graphiti:       # Episodic Memory engine cluster
  memobase:       # Profile Memory engine cluster
  zep:            # Conversational Memory engine cluster
  openviking:     # Procedural Memory engine cluster
  supermemory:    # Adaptive Memory engine cluster
  observe:        # AgentMemory — Observe service
  memory:         # AgentMemory — Lifecycle service
  orchestration:  # AgentMemory — Multi-agent coordination
  pipeline:       # AgentMemory — Consolidation pipeline
  platform:       # Admin, auth, events
  search-hub:     # Cross-engine search
  postgres:       # PostgreSQL 17 + pgvector
  neo4j:          # Graph database
  redis:          # Cache + rate limiting
  nats:           # Message broker (external mode)
  minio:          # Object storage
  prometheus:     # Metrics
  grafana:        # Dashboards
```

---

## Kubernetes Resources (Production)

| Resource | Replicas | CPU | Memory |
|---|---|---|---|
| gateway | 2-3 | 500m | 512Mi |
| cognee | 2 | 1000m | 1Gi |
| graphiti | 2 | 1000m | 1Gi |
| memobase | 2 | 500m | 512Mi |
| zep | 2 | 500m | 512Mi |
| openviking | 2 | 500m | 512Mi |
| supermemory | 2 | 500m | 512Mi |
| observe | 2 | 500m | 512Mi |
| pipeline | 2 | 1000m | 1Gi |
| postgres | 3 (HA) | 2000m | 4Gi |
| neo4j | 3 (cluster) | 2000m | 4Gi |
| nats | 3 (cluster) | 500m | 1Gi |
| redis | 3 (sentinel) | 200m | 512Mi |

---

*[← C4 Code](./C4-code.md) | [→ Data Flow](./data-flow.md)*
