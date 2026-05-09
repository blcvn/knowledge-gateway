# 10 — Deployment

> **Environments**: Dev (Docker Compose) · Staging · Production (Kubernetes)

---

## 1. Docker Compose (Development)

### 1.1 Infrastructure Stack

```yaml
# deploy/docker-compose/docker-compose.infra.yml
version: "3.9"

services:
  postgres:
    image: ankane/pgvector:v0.5.1
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: zep
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  neo4j:
    image: neo4j:5.22.0
    ports:
      - "7474:7474"
      - "7687:7687"
    environment:
      NEO4J_AUTH: neo4j/zepzepzep
      NEO4J_PLUGINS: '["apoc"]'
    volumes:
      - neo4j_data:/data
    healthcheck:
      test: ["CMD", "cypher-shell", "-u", "neo4j", "-p", "zepzepzep", "RETURN 1"]
      interval: 10s
      timeout: 10s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5

  nats:
    image: nats:2.10-alpine
    ports:
      - "4222:4222"
      - "8222:8222"   # monitoring
    command: ["--jetstream", "--store_dir", "/data"]
    volumes:
      - nats_data:/data
    healthcheck:
      test: ["CMD", "nats-server", "--signal", "ldm"]
      interval: 10s
      timeout: 5s
      retries: 5

  graphiti:
    image: zepai/graphiti:0.3
    ports:
      - "8003:8003"
    environment:
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      NEO4J_URI: bolt://neo4j:7687
      NEO4J_USER: neo4j
      NEO4J_PASSWORD: zepzepzep
    depends_on:
      neo4j:
        condition: service_healthy

  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    ports:
      - "4317:4317"   # gRPC OTLP
      - "4318:4318"   # HTTP OTLP
    volumes:
      - ./otel-config.yaml:/etc/otel/config.yaml

volumes:
  postgres_data:
  neo4j_data:
  redis_data:
  nats_data:
```

### 1.2 Application Stack

```yaml
# deploy/docker-compose/docker-compose.yml
version: "3.9"

services:
  zep-gateway:
    build:
      context: ../../
      dockerfile: services/zep-gateway/Dockerfile
    ports:
      - "8080:8080"   # REST
      - "8081:8081"   # gRPC
      - "8082:8082"   # MCP
      - "8083:8083"   # Health
    environment:
      - GATEWAY_AUTH_MODE=shared_secret
      - GATEWAY_AUTH_SHARED_SECRET_KEY=${ZEP_API_SECRET}
      - GATEWAY_BACKENDS_USER=zep-user:9041
      - GATEWAY_BACKENDS_THREAD=zep-thread:9042
      - GATEWAY_BACKENDS_MEMORY=zep-memory:9043
      - GATEWAY_BACKENDS_GRAPH=zep-graph:9044
      - GATEWAY_BACKENDS_SEARCH=zep-search:9045
      - GATEWAY_BACKENDS_ADMIN=zep-admin:9046
      - GATEWAY_RATE_LIMIT_REDIS_URL=redis://redis:6379/0
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      - zep-user
      - zep-thread
      - zep-memory
      - zep-graph
      - zep-search
      - zep-admin

  zep-user:
    build:
      context: ../../
      dockerfile: services/zep-user/Dockerfile
    environment:
      - USER_POSTGRES_DSN=postgres://postgres:postgres@postgres:5432/zep?sslmode=disable
      - USER_NATS_URL=nats://nats:4222
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      postgres:
        condition: service_healthy
      nats:
        condition: service_healthy

  zep-thread:
    build:
      context: ../../
      dockerfile: services/zep-thread/Dockerfile
    environment:
      - THREAD_POSTGRES_DSN=postgres://postgres:postgres@postgres:5432/zep?sslmode=disable
      - THREAD_NATS_URL=nats://nats:4222
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      postgres:
        condition: service_healthy
      nats:
        condition: service_healthy

  zep-memory:
    build:
      context: ../../
      dockerfile: services/zep-memory/Dockerfile
    environment:
      - MEMORY_POSTGRES_DSN=postgres://postgres:postgres@postgres:5432/zep?sslmode=disable
      - MEMORY_NATS_URL=nats://nats:4222
      - MEMORY_CLIENTS_THREAD=zep-thread:9042
      - MEMORY_CLIENTS_SEARCH=zep-search:9045
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      postgres:
        condition: service_healthy
      nats:
        condition: service_healthy
      zep-thread:
        condition: service_started
      zep-search:
        condition: service_started

  zep-graph:
    build:
      context: ../../
      dockerfile: services/zep-graph/Dockerfile
    environment:
      - GRAPH_GRAPHITI_SERVICE_URL=http://graphiti:8003
      - GRAPH_NEO4J_URI=bolt://neo4j:7687
      - GRAPH_NEO4J_USERNAME=neo4j
      - GRAPH_NEO4J_PASSWORD=zepzepzep
      - GRAPH_NATS_URL=nats://nats:4222
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      neo4j:
        condition: service_healthy
      nats:
        condition: service_healthy
      graphiti:
        condition: service_started

  zep-search:
    build:
      context: ../../
      dockerfile: services/zep-search/Dockerfile
    environment:
      - SEARCH_GRAPHITI_SERVICE_URL=http://graphiti:8003
      - SEARCH_REDIS_URL=redis://redis:6379/1
      - SEARCH_NATS_URL=nats://nats:4222
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      redis:
        condition: service_healthy
      nats:
        condition: service_healthy
      graphiti:
        condition: service_started

  zep-admin:
    build:
      context: ../../
      dockerfile: services/zep-admin/Dockerfile
    environment:
      - ADMIN_POSTGRES_DSN=postgres://postgres:postgres@postgres:5432/zep?sslmode=disable
      - ADMIN_NATS_URL=nats://nats:4222
      - ADMIN_BACKENDS_USER=zep-user:9041
      - ADMIN_BACKENDS_THREAD=zep-thread:9042
      - ADMIN_BACKENDS_MEMORY=zep-memory:9043
      - ADMIN_BACKENDS_GRAPH=zep-graph:9044
      - ADMIN_BACKENDS_SEARCH=zep-search:9045
      - OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      postgres:
        condition: service_healthy
      nats:
        condition: service_healthy
```

---

## 2. Dockerfile Template

```dockerfile
# Multi-stage build for all services
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build specific service
ARG SERVICE_NAME
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=$(git describe --tags)" \
    -o /bin/service \
    ./services/${SERVICE_NAME}/cmd/server/

# Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/service /bin/service

EXPOSE 9041 9141

ENTRYPOINT ["/bin/service"]
```

---

## 3. Kubernetes (Production)

### 3.1 Kustomize Structure

```
deploy/kubernetes/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── gateway/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml
│   ├── zep-user/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── zep-thread/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── zep-memory/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── zep-graph/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── zep-search/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   └── zep-admin/
│       ├── deployment.yaml
│       └── service.yaml
└── overlays/
    ├── dev/
    │   ├── kustomization.yaml
    │   └── patches/
    ├── staging/
    │   ├── kustomization.yaml
    │   └── patches/
    └── production/
        ├── kustomization.yaml
        ├── patches/
        │   ├── gateway-hpa.yaml
        │   ├── memory-hpa.yaml
        │   └── search-hpa.yaml
        └── secrets/
```

### 3.2 Service Deployment Template

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zep-memory
  labels:
    app: zep-memory
    version: v1
spec:
  replicas: 2
  selector:
    matchLabels:
      app: zep-memory
  template:
    metadata:
      labels:
        app: zep-memory
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9143"
    spec:
      containers:
        - name: zep-memory
          image: zep-platform/zep-memory:latest
          ports:
            - name: grpc
              containerPort: 9043
            - name: health
              containerPort: 9143
          envFrom:
            - configMapRef:
                name: zep-memory-config
            - secretRef:
                name: zep-memory-secrets
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
          readinessProbe:
            grpc:
              port: 9043
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            grpc:
              port: 9043
            initialDelaySeconds: 15
            periodSeconds: 20
---
apiVersion: v1
kind: Service
metadata:
  name: zep-memory
spec:
  selector:
    app: zep-memory
  ports:
    - name: grpc
      port: 9043
      targetPort: 9043
    - name: health
      port: 9143
      targetPort: 9143
```

### 3.3 HPA (Production)

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: zep-memory-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: zep-memory
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

---

## 4. Startup Dependency Order

```
Infrastructure:
  1. PostgreSQL (health check: pg_isready)
  2. Neo4j (health check: cypher-shell RETURN 1)
  3. Redis (health check: redis-cli ping)
  4. NATS JetStream (health check: signal ldm)
  5. Graphiti Service (depends: Neo4j healthy)
  6. OTel Collector

Application Services:
  7. zep-user      (depends: PostgreSQL, NATS)
  8. zep-thread    (depends: PostgreSQL, NATS)
  9. zep-admin     (depends: PostgreSQL, NATS)
  10. zep-search   (depends: Redis, NATS, Graphiti)
  11. zep-graph    (depends: Neo4j, NATS, Graphiti)
  12. zep-memory   (depends: PostgreSQL, NATS, zep-thread, zep-search)

Gateway:
  13. zep-gateway  (depends: ALL application services)
```

---

## 5. Resource Requirements (Production)

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit | Replicas |
|---------|-----------|----------|---------------|-------------|----------|
| zep-gateway | 200m | 1000m | 256Mi | 1Gi | 2-5 |
| zep-user | 100m | 500m | 128Mi | 512Mi | 2 |
| zep-thread | 100m | 500m | 128Mi | 512Mi | 2 |
| zep-memory | 200m | 1000m | 256Mi | 1Gi | 2-5 |
| zep-graph | 200m | 1000m | 256Mi | 1Gi | 2-4 |
| zep-search | 200m | 1000m | 256Mi | 1Gi | 2-5 |
| zep-admin | 50m | 250m | 64Mi | 256Mi | 1-2 |

---

## 6. Observability Stack

```
┌─────────────────────────────────────────────────────┐
│                  Observability Stack                  │
│                                                      │
│  All Services → OTel SDK → OTel Collector            │
│                    ↓               ↓        ↓        │
│              ┌─────────┐  ┌────────┐  ┌────────┐   │
│              │ Jaeger  │  │Prometheus│  │ Loki   │   │
│              │ (traces)│  │(metrics) │  │ (logs) │   │
│              └────┬────┘  └────┬─────┘  └────┬───┘   │
│                   └────────────┼─────────────┘       │
│                                ▼                      │
│                         ┌──────────┐                 │
│                         │ Grafana  │                 │
│                         │(unified) │                 │
│                         └──────────┘                 │
└─────────────────────────────────────────────────────┘
```

### Key Dashboards

| Dashboard | Metrics |
|-----------|---------|
| **API Gateway** | Request rate, latency p50/p95/p99, error rate, rate limit hits |
| **Memory Service** | PutMemory throughput, GetMemory latency, message count |
| **Graph Service** | Extraction duration, fact count, NATS consumer lag |
| **Search Service** | Search latency by reranker, cache hit rate, result count |
| **Infrastructure** | PostgreSQL connections, Neo4j heap, Redis memory, NATS streams |

---

## 7. Environment Variables Reference

| Variable | Service | Default | Description |
|----------|---------|---------|-------------|
| `ZEP_API_SECRET` | Gateway | — | Shared secret for CE-compatible auth |
| `OPENAI_API_KEY` | Graphiti | — | LLM provider API key |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | All | — | OTel collector endpoint |
| `LOG_LEVEL` | All | `info` | Log level (debug/info/warn/error) |
| `LOG_FORMAT` | All | `json` | Log format (json/console) |
