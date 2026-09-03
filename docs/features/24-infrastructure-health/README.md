# Feature 24 — Infrastructure Health

> **Loại:** Operations | **Priority:** High | **Status:** Implemented (UI)

## Mô tả

Infrastructure Health hiển thị topology và health của toàn bộ VNP Memory infrastructure — 35 services, các databases, và external dependencies. DevOps Engineer có thể xem service topology, database status, và deployment metadata.

---

## Business Logic

### Service Topology

Hiển thị graph-like topology:
- 35 services với dependencies
- Health indicator per service (healthy/degraded/down)
- Port information
- Connection lines giữa services

### Service Detail

Khi click vào service:
- Service name, version, port
- Health: `healthy` | `degraded` | `unhealthy`
- Uptime
- Recent errors
- Metrics: CPU/memory/goroutines (nếu expose)

### Database Status

Per-database health:
- **PostgreSQL**: Connection pool status, active queries, storage usage
- **Neo4j**: Connection status, node/edge count
- **Redis**: Memory usage, hit rate
- **Qdrant**: Vector count per collection
- **MinIO**: Storage usage, bucket count
- **NATS**: Message throughput, consumer lag

### Resource Overview

High-level resource consumption:
- Total storage across engines
- Memory usage
- Network traffic

---

## Dataflow

```
Console UI (Infrastructure Health)
        │
        ├── GET /v1/console/infra/topology
        │         └── Service dependency graph + health status
        │
        ├── GET /v1/console/infra/services
        │         └── All 35 services với health + metrics
        │
        ├── GET /v1/console/infra/services/{name}
        │         └── Specific service detail
        │
        ├── GET /v1/console/infra/databases
        │         └── Database health + metrics
        │
        ├── GET /v1/console/infra/resources
        │         └── Resource usage (storage, memory, network)
        │
        └── GET /v1/console/infra/deployments
                  └── Deployment metadata, versions

Data sources:
        ├── GET /healthz (port :8083) → aggregate 35-service health
        ├── Prometheus metrics → CPU/memory/latency
        └── Direct DB queries → connection pool status
```

## API Endpoints

| Method | Path | Mô tả |
|--------|------|-------|
| `GET` | `/v1/console/infra/topology` | Service topology graph |
| `GET` | `/v1/console/infra/services` | All services |
| `GET` | `/v1/console/infra/services/{name}` | Service detail |
| `GET` | `/v1/console/infra/databases` | Database status |
| `GET` | `/v1/console/infra/resources` | Resource usage |
| `GET` | `/v1/console/infra/deployments` | Deployments |

---

## Business Value

### Pain Points được giải quyết

- **PP-P2-01 (35+ services phức tạp)**
- **PP-P2-02 (Monitoring fragmented)**

### Actors hưởng lợi

P2 Platform Engineer

### Giải pháp tham chiếu

- [S10 — Zero-config Infrastructure](../../bussiness/solutions/S10-infrastructure-simplicity.md)

### ROI / Kết quả đo được

> 1 aggregated /healthz endpoint | Service topology map | Database health stats

---

*Xem thêm: [Pain Points](../../bussiness/painpoints/README.md) | [Solutions](../../bussiness/solutions/README.md)*
