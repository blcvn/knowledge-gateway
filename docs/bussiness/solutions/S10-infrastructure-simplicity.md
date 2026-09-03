# S10 — Zero-config Infrastructure

> **Giải quyết Pain Points:** PP-P2-01, PP-P2-02, PP-P2-04, PP-P6-01
> **Actor chính:** P2 (Platform Engineer), P1 (AI Agent Developer), P6 (Framework Integrator)
> **Features:** F01 (Monolith mode), F24 (Infra Health), F25 (Observability)

---

## Vấn đề cần giải quyết

Vận hành 35+ microservices riêng lẻ là nightmare: mỗi service có config riêng, logging riêng, health check riêng, scaling riêng. Time to deploy từ zero: 2-3 tuần.

---

## Giải pháp: Monolith-first Architecture

### Development Mode — 1 Binary, 1 Command

```bash
# Toàn bộ infrastructure
make infra-up
# Khởi động: PostgreSQL + Neo4j + Redis + Qdrant + MinIO (5 containers)

# Toàn bộ 35+ memory services trong 1 process
make dev
# Khởi động: InProcessRegistry + bufconn + embedded NATS + tất cả engines

# Verify tất cả services
curl http://localhost:8083/healthz | jq
```

**Time to first API call: < 5 phút** từ zero.

---

### InProcessRegistry — Bufconn Zero-latency

35+ services communicate qua in-process gRPC (không qua network):

```
Gateway nhận: POST /v1/memory/store
        │
        ▼
InProcessRegistry.Lookup("memobase-ingestion")
        │
        ▼
bufconn (in-memory pipe) — ZERO network hop
        │
        ▼
memobase-ingestion handler trong cùng process
        │
        ▼
Response trả về ngay (< 1ms internal latency)
```

**vs. Microservices:** Không có network latency, không cần service discovery, không cần load balancer cho internal calls.

---

### Embedded NATS JetStream

```
Development:  NATS chạy embedded trong monolith binary
Production:   VNP_MEMORY_NATS_MODE=external → External NATS cluster

Không cần deploy NATS riêng cho development.
Chuyển production chỉ cần thay 1 env var.
```

---

### Aggregated Health Check

Thay vì check 35 services riêng lẻ:

```http
GET http://localhost:8083/healthz

{
  "status": "healthy",
  "services": {
    "cognee-ingestion": {"status": "up", "latency_ms": 2},
    "graphiti-search": {"status": "up", "latency_ms": 5},
    "zep-memory": {"status": "up", "latency_ms": 3},
    "memobase-engine": {"status": "up", "latency_ms": 1},
    ...35 services...
  },
  "infrastructure": {
    "postgres": {"status": "up", "connections": 12},
    "neo4j": {"status": "up", "bolt": true},
    "redis": {"status": "up", "memory_used_mb": 45},
    "nats": {"status": "up", "streams": 8}
  }
}
```

**1 endpoint thay thế 35 endpoints.**

---

### Unified Observability (F25)

```
OpenTelemetry → Traces cross-engine
Prometheus    → Metrics: latency, throughput, errors, LLM costs
slog          → Structured JSON logs (secret redacted)

Console: /v1/console/observability/
  /metrics  → Prometheus metrics dump
  /traces   → OpenTelemetry traces
  /errors   → Error rates per engine
  /costs    → LLM cost breakdown
```

**1 dashboard thay vì 35 dashboards.**

---

### Production Path — Docker Compose

```bash
# Full stack production
make docker-up

# docker-compose.yml cấu hình:
# - Gateway container
# - Distributed engine services
# - PostgreSQL, Neo4j, Redis, Qdrant, MinIO
# - NATS external cluster
# - Prometheus + Grafana

# Graceful shutdown:
# HTTP drain → NATS drain → gRPC stop → DB close
```

---

### Port Map — Dễ nhớ

| Port | Service | Description |
|---|---|---|
| **:8080** | REST API | Primary API (50+ routes) |
| **:8082** | MCP Server | AI framework integration |
| **:8083** | Health + Metrics | Aggregated health, Prometheus |
| :5432 | PostgreSQL | Relational + pgvector |
| :7687 | Neo4j | Graph database |
| :6379 | Redis | Cache + rate limiting |
| :6333 | Qdrant | Vector DB (optional) |
| :9000 | MinIO | Object storage |

---

## So sánh: Tự xây vs VNP Memory

| Dimension | Tự xây | VNP Memory |
|---|---|---|
| Time to first memory API | 2-3 tháng | < 5 phút |
| Services cần vận hành | 35+ riêng lẻ | 1 binary (monolith) |
| Monitoring setup | Tự configure cho mỗi service | Built-in unified dashboard |
| Service discovery | Cần Consul/Etcd | Built-in InProcessRegistry |
| Message broker | Tự setup NATS/Kafka | Embedded NATS (or external) |
| Health check | 35 endpoints | 1 aggregated endpoint |
| Graceful shutdown | Tự implement | Built-in drain sequence |
| Production scaling | Custom Kubernetes configs | Docker Compose → K8s Helm |
