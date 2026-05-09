# 09 — Deployment & Infrastructure

---

## 1. Docker Compose (Development)

```yaml
version: "3.9"

services:
  # ─── Infrastructure ─────────────────────
  postgres:
    image: pgvector/pgvector:pg17
    ports: ["15432:5432"]
    environment:
      POSTGRES_DB: memobase
      POSTGRES_USER: memobase
      POSTGRES_PASSWORD: "${DB_PASSWORD}"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U memobase"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7.4-alpine
    ports: ["16379:6379"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s

  nats:
    image: nats:2.10-alpine
    ports: ["4222:4222", "8222:8222"]
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - natsdata:/data

  # ─── Application Services ───────────────
  memobase-gateway:
    build: ./services/memobase-gateway
    ports:
      - "${GATEWAY_HTTP_PORT:-8080}:8080"
      - "${GATEWAY_MCP_PORT:-8082}:8082"
      - "8083:8083"      # health/metrics
    environment:
      ROOT_ACCESS_TOKEN: "${ROOT_ACCESS_TOKEN}"
      BACKEND_INGESTION: "memobase-ingestion:9041"
      BACKEND_ENGINE: "memobase-engine:9042"
      BACKEND_CONTEXT: "memobase-context:9043"
      BACKEND_EVENT: "memobase-event:9044"
      BACKEND_ADMIN: "memobase-admin:9045"
      REDIS_URL: "redis://redis:6379/0"
    depends_on:
      redis: { condition: service_healthy }

  memobase-ingestion:
    build: ./services/memobase-ingestion
    ports: ["9041:9041", "9091:9091"]
    environment:
      DATABASE_URL: "postgres://memobase:${DB_PASSWORD}@postgres:5432/memobase?sslmode=disable"
      NATS_URL: "nats://nats:4222"
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_started }

  memobase-engine:
    build: ./services/memobase-engine
    ports: ["9042:9042", "9092:9092"]
    environment:
      DATABASE_URL: "postgres://memobase:${DB_PASSWORD}@postgres:5432/memobase?sslmode=disable"
      NATS_URL: "nats://nats:4222"
      LLM_API_KEY: "${LLM_API_KEY}"
      LLM_BASE_URL: "${LLM_BASE_URL:-https://api.openai.com/v1}"
      LLM_MODEL: "${LLM_MODEL:-gpt-4o-mini}"
      EMBEDDING_API_KEY: "${EMBEDDING_API_KEY}"
      EMBEDDING_MODEL: "${EMBEDDING_MODEL:-text-embedding-3-small}"
      EMBEDDING_DIM: "${EMBEDDING_DIM:-1536}"
    depends_on:
      postgres: { condition: service_healthy }
      nats: { condition: service_started }
    deploy:
      resources:
        limits: { memory: 1G }

  memobase-context:
    build: ./services/memobase-context
    ports: ["9043:9043", "9093:9093"]
    environment:
      DATABASE_URL: "postgres://memobase:${DB_PASSWORD}@postgres:5432/memobase?sslmode=disable"
      REDIS_URL: "redis://redis:6379/1"
      NATS_URL: "nats://nats:4222"
      BACKEND_EVENT: "memobase-event:9044"
    depends_on:
      postgres: { condition: service_healthy }
      redis: { condition: service_healthy }

  memobase-event:
    build: ./services/memobase-event
    ports: ["9044:9044", "9094:9094"]
    environment:
      DATABASE_URL: "postgres://memobase:${DB_PASSWORD}@postgres:5432/memobase?sslmode=disable"
      EMBEDDING_API_KEY: "${EMBEDDING_API_KEY}"
      EMBEDDING_MODEL: "${EMBEDDING_MODEL:-text-embedding-3-small}"
      EMBEDDING_DIM: "${EMBEDDING_DIM:-1536}"
    depends_on:
      postgres: { condition: service_healthy }

  memobase-admin:
    build: ./services/memobase-admin
    ports: ["9045:9045", "9095:9095"]
    environment:
      DATABASE_URL: "postgres://memobase:${DB_PASSWORD}@postgres:5432/memobase?sslmode=disable"
      ROOT_ACCESS_TOKEN: "${ROOT_ACCESS_TOKEN}"
      NATS_URL: "nats://nats:4222"
    depends_on:
      postgres: { condition: service_healthy }

volumes:
  pgdata:
  natsdata:
```

---

## 2. Kubernetes (Production)

### 2.1 Resource Planning

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit | Replicas |
|---------|-----------|----------|---------------|-------------|----------|
| memobase-gateway | 200m | 500m | 128Mi | 256Mi | 2-4 |
| memobase-ingestion | 200m | 500m | 128Mi | 256Mi | 2-3 |
| memobase-engine | 500m | 2000m | 256Mi | 1Gi | 2-5 |
| memobase-context | 200m | 500m | 128Mi | 256Mi | 2-4 |
| memobase-event | 300m | 1000m | 128Mi | 512Mi | 2-3 |
| memobase-admin | 100m | 250m | 64Mi | 128Mi | 2 |

### 2.2 HPA Configuration

```yaml
# memobase-engine (LLM-intensive, primary scale target)
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: memobase-engine-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: memobase-engine
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Pods
      pods:
        metric:
          name: memobase_llm_latency_ms
        target:
          type: AverageValue
          averageValue: "3000"   # Scale when avg LLM latency > 3s
```

### 2.3 Kustomize Structure

```
deploy/kubernetes/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── gateway/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml
│   ├── ingestion/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── engine/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── hpa.yaml
│   ├── context/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── event/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── admin/
│       ├── deployment.yaml
│       └── service.yaml
├── overlays/
│   ├── dev/
│   │   └── kustomization.yaml
│   ├── staging/
│   │   └── kustomization.yaml
│   └── production/
│       ├── kustomization.yaml
│       ├── hpa-patches.yaml
│       └── resource-patches.yaml
```

---

## 3. Health & Readiness

Mỗi service expose 3 endpoints trên health port:

| Endpoint | Purpose | Check |
|----------|---------|-------|
| `/healthz` | Liveness | Process alive |
| `/readyz` | Readiness | DB connected, NATS connected |
| `/livez` | Deep health | All dependencies OK |

```go
// pkg/observability/health.go
type HealthChecker struct {
    checks []Check
}

type Check struct {
    Name string
    Fn   func(ctx context.Context) error
}

// Register per service:
// - PostgreSQL: pgx.Ping()
// - Redis: redis.Ping()
// - NATS: nats.Status() == CONNECTED
// - Downstream gRPC: grpc_health_v1.Check()
```

---

## 4. Observability Stack

```
┌──────────────────────────────────────────────────┐
│                Application Services               │
│  (OTel SDK: traces + metrics + logs)              │
└───────┬──────────────┬───────────────┬────────────┘
        │              │               │
        ▼              ▼               ▼
┌───────────┐  ┌──────────────┐ ┌────────────┐
│ OTel      │  │ Prometheus   │ │ Loki       │
│ Collector │  │ (scrape      │ │ (log       │
│ (traces)  │  │  :909x/      │ │  aggregation│
│           │  │  metrics)    │ │  )          │
└─────┬─────┘  └──────┬───────┘ └──────┬─────┘
      │               │                │
      ▼               ▼                ▼
┌──────────┐  ┌──────────────┐  ┌────────────┐
│  Jaeger  │  │  Grafana     │  │  Grafana   │
│  (UI)    │  │  (dashboards)│  │  (logs)    │
└──────────┘  └──────────────┘  └────────────┘
```

---

## 5. Migration Strategy (Python → Go)

### Phase 1: Foundation (Week 1-2)
- Setup monorepo structure, go.mod, Makefile
- Implement `pkg/` shared packages
- Define all Protobuf schemas, generate Go code
- Setup Docker Compose with infrastructure

### Phase 2: Admin + Ingestion (Week 3-4)
- Implement memobase-admin (User, Project CRUD)
- Implement memobase-ingestion (Blob insert, Buffer zone)
- Implement gateway (REST → gRPC translation)
- Integration tests with existing Python clients

### Phase 3: Engine (Week 5-6)
- Implement memobase-engine (LLM pipeline)
- Port prompt templates (EN/ZH)
- YOLO merge algorithm in Go
- NATS event-driven pipeline

### Phase 4: Context + Event (Week 7-8)
- Implement memobase-context (Profile read, Context assembly)
- Implement memobase-event (Vector search, Gist search)
- Redis caching integration
- MCP server in gateway

### Phase 5: Production Hardening (Week 9-10)
- Circuit breakers, rate limiting, bulkhead
- Kubernetes manifests, HPA
- Load testing, performance tuning
- Client SDK compatibility verification
- Documentation update
