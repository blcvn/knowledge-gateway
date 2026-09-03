# 09 — Deployment Guide

> **Target**: Docker Compose (dev) + Kubernetes (production)  
> **Images**: Multi-stage distroless, <20MB per service  
> **Orchestration**: Kustomize base + environment overlays

---

## 1. Development — Docker Compose

```yaml
services:
  # ─── OpenViking Services ───
  gateway:
    build: ./services/openviking-gateway
    ports:
      - "8080:8080"   # REST
      - "8081:8081"   # gRPC
      - "8082:8082"   # MCP
    environment:
      - AUTH_MODE=dev
      - FS_SERVICE_ADDR=fs:9011
      - SEARCH_SERVICE_ADDR=search:9012
      - SESSION_SERVICE_ADDR=session:9013
      - RESOURCE_SERVICE_ADDR=resource:9014
      - CRYPTO_SERVICE_ADDR=crypto:9015
      - ADMIN_SERVICE_ADDR=admin:9030
    depends_on: [fs, search, session, resource, crypto, admin]

  fs:
    build: ./services/openviking-fs
    ports: ["9011:9011", "9091:9091"]
    volumes:
      - viking-data:/data
    environment:
      - WORKSPACE_PATH=/data
      - CRYPTO_SERVICE_ADDR=crypto:9015
      - NATS_URL=nats://nats:4222

  search:
    build: ./services/openviking-search
    ports: ["9012:9012", "9092:9092"]
    environment:
      - VECTORDB_TYPE=embedded
      - EMBEDDING_PROVIDER=openai
      - EMBEDDING_API_KEY=${OPENAI_API_KEY}
      - BIFROST_URL=http://bifrost:8080
      - NATS_URL=nats://nats:4222
      - REDIS_URL=redis://redis:6379

  session:
    build: ./services/openviking-session
    ports: ["9013:9013", "9093:9093"]
    environment:
      - FS_SERVICE_ADDR=fs:9011
      - SEARCH_SERVICE_ADDR=search:9012
      - VLM_PROVIDER=openai
      - BIFROST_URL=http://bifrost:8080
      - NATS_URL=nats://nats:4222

  resource:
    build: ./services/openviking-resource
    ports: ["9014:9014", "9094:9094"]
    environment:
      - FS_SERVICE_ADDR=fs:9011
      - SEARCH_SERVICE_ADDR=search:9012
      - BIFROST_URL=http://bifrost:8080
      - NATS_URL=nats://nats:4222
      - REDIS_URL=redis://redis:6379

  crypto:
    build: ./services/openviking-crypto
    ports: ["9015:9015", "9095:9095"]
    environment:
      - KMS_PROVIDER=local
      - KMS_LOCAL_KEY_PATH=/secrets/root.key
    volumes:
      - crypto-keys:/secrets

  admin:
    build: ./services/openviking-admin
    ports: ["9030:9030", "9099:9099"]
    environment:
      - REDIS_URL=redis://redis:6379
      - NATS_URL=nats://nats:4222

  # ─── Infrastructure ───
  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  nats:
    image: nats:2-alpine
    ports: ["4222:4222", "8222:8222"]
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - nats-data:/data

  otel-collector:
    image: otel/opentelemetry-collector:latest
    ports: ["4317:4317"]

  prometheus:
    image: prom/prometheus:latest
    ports: ["9090:9090"]

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports: ["16686:16686"]

volumes:
  viking-data:
  crypto-keys:
  nats-data:
```

---

## 2. Production — Kubernetes

### 2.1 Namespace Layout

```
Namespace: openviking-system
├── Deployments
│   ├── openviking-gateway       (replicas: 3, HPA: 3-10)
│   ├── openviking-fs            (replicas: 2, HPA: 2-6)
│   ├── openviking-search        (replicas: 3, HPA: 3-12)
│   ├── openviking-session       (replicas: 2, HPA: 2-6)
│   ├── openviking-resource      (replicas: 2, HPA: 2-8)
│   ├── openviking-crypto        (replicas: 2, HPA: 2-4)
│   └── openviking-admin         (replicas: 1)
├── StatefulSets
│   ├── redis-cluster            (replicas: 6)
│   └── nats-cluster             (replicas: 3)
├── Services (ClusterIP)
│   ├── openviking-gateway       (8080, 8081, 8082)
│   ├── openviking-fs-grpc       (9011)
│   ├── openviking-search-grpc   (9012)
│   ├── openviking-session-grpc  (9013)
│   ├── openviking-resource-grpc (9014)
│   ├── openviking-crypto-grpc   (9015)
│   └── openviking-admin-grpc    (9030)
├── Ingress
│   └── openviking-gateway (HTTPS, path-based)
├── ConfigMaps
│   └── openviking-config
├── Secrets (from Vault)
│   ├── openviking-api-keys
│   └── openviking-kms-keys
├── PersistentVolumeClaim
│   └── viking-data (ReadWriteMany for FS service)
└── ServiceMonitor (Prometheus)
    └── openviking-metrics
```

### 2.2 Resource Recommendations

| Service | CPU Request | CPU Limit | Memory Request | Memory Limit |
|---------|-----------|---------|--------------|------------|
| gateway | 200m | 1000m | 256Mi | 512Mi |
| fs | 200m | 500m | 256Mi | 1Gi |
| search | 500m | 2000m | 512Mi | 2Gi |
| session | 200m | 1000m | 256Mi | 1Gi |
| resource | 500m | 2000m | 512Mi | 2Gi |
| crypto | 100m | 500m | 128Mi | 256Mi |
| admin | 100m | 200m | 128Mi | 256Mi |

---

## 3. Docker Build (Multi-Stage)

```dockerfile
# Stage 1: Build
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" \
    -o /server ./services/<service-name>/cmd/server/

# Stage 2: Runtime
FROM gcr.io/distroless/static-debian12
COPY --from=builder /server /server
EXPOSE <port>
ENTRYPOINT ["/server"]
```

---

## 4. Configuration (`config.yaml`)

```yaml
server:
  host: "0.0.0.0"
  grpc_port: 9011
  health_port: 9091

storage:
  workspace: "/data"

embedding:
  provider: "openai"
  model: "text-embedding-3-small"
  dimension: 1536
  max_concurrent: 10

vlm:
  provider: "openai"
  model: "gpt-4o-mini"
  max_concurrent: 100

retrieval:
  hotness_alpha: 0.1
  score_propagation_alpha: 0.7
  global_search_topk: 10
  max_convergence_rounds: 3

encryption:
  enabled: true
  provider: "vault"

observability:
  tracing_enabled: true
  metrics_enabled: true
  otlp_endpoint: "otel-collector:4317"

nats:
  url: "nats://nats:4222"
  stream: "openviking"

redis:
  url: "redis://redis:6379"
  pool_size: 20
```

---

## 5. Health Check Endpoints

| Endpoint | Protocol | Purpose |
|----------|----------|---------|
| `/healthz` | HTTP | Liveness probe (always OK if process running) |
| `/readyz` | HTTP | Readiness probe (dependencies connected) |
| `/livez` | HTTP | Deep health check |
| gRPC Health | gRPC | Kubernetes gRPC health check |
| `/metrics` | HTTP | Prometheus scraping |
