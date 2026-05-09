# Deployment Guide — Docker Compose & Kubernetes

**Version:** 2.0 | **Date:** 2026-05-09  
**Target:** Development (Docker Compose) + Production (Kubernetes)

---

## 1. Repository Structure

```
graphiti/
├── services/
│   ├── graphiti-gateway/
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   ├── api/
│   │   ├── Dockerfile
│   │   └── Makefile
│   ├── graphiti-ingestion/
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   ├── api/
│   │   ├── Dockerfile
│   │   └── Makefile
│   ├── graphiti-search/
│   │   ├── ...
│   │   └── Dockerfile
│   ├── graphiti-knowledge/
│   │   ├── ...
│   │   └── Dockerfile
│   ├── graphiti-store/
│   │   ├── ...
│   │   └── Dockerfile
│   └── graphiti-admin/
│       ├── ...
│       └── Dockerfile
├── pkg/                                # Shared packages
│   ├── proto/
│   ├── graph/
│   ├── middleware/
│   ├── resilience/
│   ├── observability/
│   └── testutil/
├── deploy/
│   ├── docker-compose/
│   │   ├── docker-compose.yml          # Full stack
│   │   ├── docker-compose.dev.yml      # Dev overrides
│   │   ├── docker-compose.neo4j.yml    # Neo4j profile
│   │   ├── docker-compose.falkordb.yml # FalkorDB profile
│   │   └── .env.example
│   ├── kubernetes/
│   │   ├── base/                       # Kustomize base
│   │   │   ├── kustomization.yaml
│   │   │   ├── namespace.yaml
│   │   │   ├── gateway/
│   │   │   ├── ingestion/
│   │   │   ├── search/
│   │   │   ├── knowledge/
│   │   │   ├── store/
│   │   │   ├── admin/
│   │   │   ├── neo4j/
│   │   │   ├── redis/
│   │   │   └── nats/
│   │   ├── overlays/
│   │   │   ├── dev/
│   │   │   ├── staging/
│   │   │   └── production/
│   │   └── charts/                     # Helm charts (optional)
│   │       └── graphiti/
│   └── ci/
│       ├── Makefile
│       └── .github/
│           └── workflows/
│               ├── ci.yml
│               ├── build.yml
│               └── deploy.yml
├── go.mod
├── go.sum
├── buf.yaml                            # Buf protobuf tooling
├── buf.gen.yaml
├── Makefile                            # Root build commands
└── README.md
```

---

## 2. Dockerfile (Standard — Multi-Stage)

```dockerfile
# services/graphiti-gateway/Dockerfile (example — same pattern for all services)

# ---- Build Stage ----
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
ARG SERVICE_NAME=graphiti-gateway
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w \
    -X main.Version=${VERSION:-dev} \
    -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
    -X main.GitCommit=${GIT_COMMIT:-unknown}" \
    -o /app/bin/server \
    ./services/${SERVICE_NAME}/cmd/server

# ---- Runtime Stage ----
FROM gcr.io/distroless/static:nonroot

COPY --from=builder /app/bin/server /server
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

USER nonroot:nonroot

EXPOSE 8080 8081 9090

ENTRYPOINT ["/server"]
```

---

## 3. Docker Compose — Development

```yaml
# deploy/docker-compose/docker-compose.yml

version: "3.9"

x-common-env: &common-env
  OTEL_ENDPOINT: otel-collector:4317
  NATS_URL: nats://nats:4222
  REDIS_URL: redis://redis:6379
  LOG_LEVEL: debug
  LOG_FORMAT: json

services:
  # ===== Application Services =====
  
  gateway:
    build:
      context: ../..
      dockerfile: services/graphiti-gateway/Dockerfile
    ports:
      - "8080:8080"   # REST
      - "8081:8081"   # gRPC
      - "8082:8082"   # MCP (SSE)
    environment:
      <<: *common-env
      HTTP_PORT: 8080
      GRPC_PORT: 8081
      MCP_PORT: 8082
      INGESTION_ADDR: ingestion:9001
      SEARCH_ADDR: search:9002
      KNOWLEDGE_ADDR: knowledge:9003
      STORE_ADDR: store:9004
      ADMIN_ADDR: admin:9005
      JWT_ISSUER: "http://localhost:8080"
      AUTH_ENABLED: "false"   # disabled for dev
    depends_on:
      ingestion:
        condition: service_healthy
      search:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "/server", "-health"]
      interval: 10s
      timeout: 5s
      retries: 3

  ingestion:
    build:
      context: ../..
      dockerfile: services/graphiti-ingestion/Dockerfile
    environment:
      <<: *common-env
      GRPC_PORT: 9001
      KNOWLEDGE_ADDR: knowledge:9003
      STORE_ADDR: store:9004
    depends_on:
      knowledge:
        condition: service_healthy
      store:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "/server", "-health"]
      interval: 10s

  search:
    build:
      context: ../..
      dockerfile: services/graphiti-search/Dockerfile
    environment:
      <<: *common-env
      GRPC_PORT: 9002
      KNOWLEDGE_ADDR: knowledge:9003
      STORE_ADDR: store:9004
    depends_on:
      knowledge:
        condition: service_healthy
      store:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "/server", "-health"]
      interval: 10s

  knowledge:
    build:
      context: ../..
      dockerfile: services/graphiti-knowledge/Dockerfile
    environment:
      <<: *common-env
      GRPC_PORT: 9003
      STORE_ADDR: store:9004
      LLM_PROVIDER: openai
      LLM_MODEL: gpt-4o
      LLM_SMALL_MODEL: gpt-4o-mini
      OPENAI_API_KEY: ${OPENAI_API_KEY}
      EMBEDDER_PROVIDER: openai
      EMBEDDER_MODEL: text-embedding-3-small
      MAX_CONCURRENT_LLM: 20
    depends_on:
      store:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "/server", "-health"]
      interval: 10s

  store:
    build:
      context: ../..
      dockerfile: services/graphiti-store/Dockerfile
    environment:
      <<: *common-env
      GRPC_PORT: 9004
      DRIVER_PROVIDER: neo4j
      NEO4J_URI: bolt://neo4j:7687
      NEO4J_USER: neo4j
      NEO4J_PASSWORD: ${NEO4J_PASSWORD:-graphiti_dev}
    depends_on:
      neo4j:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "/server", "-health"]
      interval: 10s

  admin:
    build:
      context: ../..
      dockerfile: services/graphiti-admin/Dockerfile
    environment:
      <<: *common-env
      GRPC_PORT: 9005
      INGESTION_ADDR: ingestion:9001
      SEARCH_ADDR: search:9002
      KNOWLEDGE_ADDR: knowledge:9003
      STORE_ADDR: store:9004
    depends_on:
      store:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "/server", "-health"]
      interval: 10s

  # ===== Infrastructure =====
  
  neo4j:
    image: neo4j:5.26.2-community
    ports:
      - "7474:7474"   # HTTP browser
      - "7687:7687"   # Bolt
    environment:
      NEO4J_AUTH: neo4j/${NEO4J_PASSWORD:-graphiti_dev}
      NEO4J_PLUGINS: '["apoc"]'
      NEO4J_dbms_memory_heap_max__size: 1G
      NEO4J_dbms_memory_pagecache_size: 512M
    volumes:
      - neo4j_data:/data
    healthcheck:
      test: ["CMD-SHELL", "cypher-shell -u neo4j -p ${NEO4J_PASSWORD:-graphiti_dev} 'RETURN 1'"]
      interval: 10s
      timeout: 5s
      retries: 10

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s

  nats:
    image: nats:2-alpine
    ports:
      - "4222:4222"   # Client
      - "8222:8222"   # Monitoring
    command: ["--jetstream", "--store_dir=/data"]
    volumes:
      - nats_data:/data
    healthcheck:
      test: ["CMD", "wget", "--spider", "http://localhost:8222/healthz"]
      interval: 5s

  # ===== Observability =====
  
  otel-collector:
    image: otel/opentelemetry-collector-contrib:0.95.0
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
    volumes:
      - ./otel-collector-config.yaml:/etc/otel/config.yaml
    command: ["--config=/etc/otel/config.yaml"]

  jaeger:
    image: jaegertracing/all-in-one:1.54
    ports:
      - "16686:16686" # UI
      - "14268:14268" # Collector

  prometheus:
    image: prom/prometheus:v2.50.1
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus

  grafana:
    image: grafana/grafana:10.3.3
    ports:
      - "3000:3000"
    volumes:
      - grafana_data:/var/lib/grafana
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin

volumes:
  neo4j_data:
  redis_data:
  nats_data:
  prometheus_data:
  grafana_data:
```

---

## 4. Kubernetes — Production

### 4.1 Namespace & ConfigMap

```yaml
# deploy/kubernetes/base/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: graphiti-system
  labels:
    app.kubernetes.io/part-of: graphiti
```

### 4.2 Service Deployment (Gateway Example)

```yaml
# deploy/kubernetes/base/gateway/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: graphiti-gateway
  namespace: graphiti-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: graphiti-gateway
  template:
    metadata:
      labels:
        app: graphiti-gateway
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
    spec:
      containers:
      - name: gateway
        image: ghcr.io/vnp/graphiti-gateway:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 8081
          name: grpc
        - containerPort: 9090
          name: metrics
        envFrom:
        - configMapRef:
            name: graphiti-config
        - secretRef:
            name: graphiti-secrets
        resources:
          requests:
            cpu: 250m
            memory: 256Mi
          limits:
            cpu: 1000m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /livez
            port: http
          initialDelaySeconds: 10
          periodSeconds: 15
        readinessProbe:
          httpGet:
            path: /readyz
            port: http
          initialDelaySeconds: 5
          periodSeconds: 10
        startupProbe:
          httpGet:
            path: /healthz
            port: http
          failureThreshold: 30
          periodSeconds: 3
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: kubernetes.io/hostname
        whenUnsatisfied: DoNotSchedule
        labelSelector:
          matchLabels:
            app: graphiti-gateway
```

### 4.3 HPA

```yaml
# deploy/kubernetes/base/gateway/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: graphiti-gateway
  namespace: graphiti-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: graphiti-gateway
  minReplicas: 3
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
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Pods
        value: 1
        periodSeconds: 60
```

---

## 5. CI/CD Pipeline

```yaml
# deploy/ci/.github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.23'
    - run: make lint
    - run: buf lint

  test:
    runs-on: ubuntu-latest
    services:
      neo4j:
        image: neo4j:5.26.2-community
        env:
          NEO4J_AUTH: neo4j/test
        ports: [7687:7687]
      redis:
        image: redis:7-alpine
        ports: [6379:6379]
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.23'
    - run: make test

  build:
    needs: [lint, test]
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [gateway, ingestion, search, knowledge, store, admin]
    steps:
    - uses: actions/checkout@v4
    - uses: docker/build-push-action@v5
      with:
        context: .
        file: services/graphiti-${{ matrix.service }}/Dockerfile
        push: ${{ github.ref == 'refs/heads/main' }}
        tags: ghcr.io/vnp/graphiti-${{ matrix.service }}:${{ github.sha }}
```

---

## 6. Environment Variable Summary

| Variable | Service | Default | Description |
|----------|---------|---------|-------------|
| `OPENAI_API_KEY` | knowledge | — | LLM/Embedder API key |
| `NEO4J_URI` | store | `bolt://localhost:7687` | Graph DB connection |
| `NEO4J_USER` | store | `neo4j` | Graph DB user |
| `NEO4J_PASSWORD` | store | — | Graph DB password |
| `REDIS_URL` | gateway, search | `redis://localhost:6379` | Redis connection |
| `NATS_URL` | all | `nats://localhost:4222` | NATS connection |
| `OTEL_ENDPOINT` | all | `localhost:4317` | OTel collector |
| `LOG_LEVEL` | all | `info` | Logging level |
| `LLM_PROVIDER` | knowledge | `openai` | LLM provider selection |
| `LLM_MODEL` | knowledge | `gpt-4o` | Default LLM model |
| `EMBEDDER_MODEL` | knowledge | `text-embedding-3-small` | Embedding model |
| `MAX_CONCURRENT_LLM` | knowledge | `20` | LLM concurrency limit |
| `AUTH_ENABLED` | gateway | `true` | Enable/disable auth |
| `DRIVER_PROVIDER` | store | `neo4j` | Graph DB backend |

---

## 7. Health Check Endpoints

| Service | Protocol | Endpoint | K8s Probe |
|---------|----------|----------|-----------|
| Gateway | HTTP | `GET /healthz` | Liveness |
| Gateway | HTTP | `GET /readyz` | Readiness |
| Gateway | HTTP | `GET /livez` | Startup |
| All gRPC services | gRPC | `grpc.health.v1.Health/Check` | All probes |
| All services | HTTP | `GET :9090/metrics` | Prometheus scrape |
