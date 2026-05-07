---
skill_id: SKILL-014
version: 1.0.0
status: active
priority: P3
group: DevOps & Infrastructure
created_at: 2026-04-24
---

# SKILL-014 · DevOps, CI/CD & Infrastructure

## Mô tả

Thiết lập và vận hành CI/CD pipeline, containerization, orchestration, monitoring để đảm bảo sản phẩm được deploy nhanh, ổn định, và observable.

## Agents sử dụng

- `doc-consistency-agent` (CI trigger)
- `qa-pipeline-agent` (CI test stage)

## Tài liệu liên kết

- `services/*/docs/runbook.md`

---

## Năng lực cốt lõi

### 1. Docker & Container Design

```dockerfile
# Multi-stage Dockerfile cho Golang service (chuẩn)
# Stage 1: Build
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Dependencies first (cache layer)
COPY go.mod go.sum ./
RUN go mod download

# Build binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.version=${VERSION}" \
    -o /bin/server ./cmd/server

# Stage 2: Runtime (minimal image)
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/server /server

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
    CMD ["/server", "health"]

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/server"]
```

```yaml
# docker-compose.yml cho local development
version: '3.8'
services:
  knowledge-service:
    build: ./ba-knowledge-service
    environment:
      - DB_HOST=postgres
      - NEO4J_URI=bolt://neo4j:7687
    depends_on:
      postgres:
        condition: service_healthy
      neo4j:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5

  neo4j:
    image: neo4j:5-community
    healthcheck:
      test: ["CMD", "neo4j", "status"]
      interval: 10s
```

### 2. Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: knowledge-service
  labels:
    app: knowledge-service
    version: v1
spec:
  replicas: 3
  selector:
    matchLabels:
      app: knowledge-service
  template:
    spec:
      containers:
        - name: knowledge-service
          image: knowledge-gateway/knowledge-service:${VERSION}
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          readinessProbe:
            httpGet:
              path: /healthz/ready
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          livenessProbe:
            httpGet:
              path: /healthz/live
              port: 8080
            initialDelaySeconds: 15
            periodSeconds: 20
          envFrom:
            - secretRef:
                name: knowledge-service-secrets
            - configMapRef:
                name: knowledge-service-config

---
# HPA (Horizontal Pod Autoscaler)
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: knowledge-service-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: knowledge-service
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

### 3. CI/CD Pipeline (GitHub Actions)

```yaml
# .github/workflows/ci.yml
name: CI/CD Pipeline

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
          go-version: '1.22'
      - uses: golangci/golangci-lint-action@v4
        with:
          args: --timeout=5m

  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: testpass
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - name: Run tests
        run: go test ./... -race -coverprofile=coverage.out
      - name: Coverage gate (min 80%)
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | tr -d '%')
          if (( $(echo "$COVERAGE < 80" | bc) )); then
            echo "Coverage $COVERAGE% below 80% threshold"
            exit 1
          fi

  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: golang/govulncheck-action@v1
      - uses: securego/gosec@master
        with:
          args: ./...

  build:
    needs: [lint, test, security]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build & push Docker image
        run: |
          docker build -t $IMAGE_NAME:$GITHUB_SHA .
          docker push $IMAGE_NAME:$GITHUB_SHA

  deploy-staging:
    needs: build
    if: github.ref == 'refs/heads/develop'
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to staging
        run: |
          kubectl set image deployment/knowledge-service \
            knowledge-service=$IMAGE_NAME:$GITHUB_SHA \
            -n staging
          kubectl rollout status deployment/knowledge-service -n staging

  deploy-production:
    needs: build
    if: github.ref == 'refs/heads/main'
    environment:
      name: production
      url: https://api.knowledge-gateway.io
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to production
        run: |
          kubectl set image deployment/knowledge-service \
            knowledge-service=$IMAGE_NAME:$GITHUB_SHA \
            -n production
          kubectl rollout status deployment/knowledge-service -n production
```

### 4. Observability Stack

```yaml
# Prometheus scrape config
scrape_configs:
  - job_name: 'knowledge-service'
    static_configs:
      - targets: ['knowledge-service:9090']
    metrics_path: /metrics

# Grafana dashboard panels:
# - Request rate (req/s) per endpoint
# - P50/P95/P99 latency per endpoint
# - Error rate (4xx, 5xx) per endpoint
# - Pipeline stage duration histogram
# - Active jobs count
# - Neo4j query duration
# - LLM API call count and cost estimate
```

```go
// Structured logging với trace context
func LogRequest(ctx context.Context, msg string, fields ...zap.Field) {
    span := trace.SpanFromContext(ctx)
    
    baseFields := []zap.Field{
        zap.String("trace_id", span.SpanContext().TraceID().String()),
        zap.String("span_id", span.SpanContext().SpanID().String()),
    }
    
    logger.Info(msg, append(baseFields, fields...)...)
}
```

### 5. Infrastructure as Code

```hcl
# terraform/main.tf
resource "kubernetes_deployment" "knowledge_service" {
  metadata {
    name      = "knowledge-service"
    namespace = var.namespace
  }
  
  spec {
    replicas = var.replicas
    
    template {
      spec {
        container {
          name  = "knowledge-service"
          image = "${var.image_repo}:${var.image_tag}"
          
          resources {
            requests = {
              memory = "256Mi"
              cpu    = "250m"
            }
          }
        }
      }
    }
  }
}
```

### 6. Environment Strategy

```
Environments & Promotion Gates:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

develop branch → [auto] → staging
    └── Auto deploy on every push to develop
    └── Integration tests run automatically

staging → [manual gate] → production  
    └── Requires manual approval (senior engineer)
    └── Must pass smoke tests in staging first
    └── Blue-green deployment (zero downtime)
    └── Automatic rollback if health checks fail

Feature flags:
    └── New features deployed behind flags
    └── Gradual rollout: 5% → 25% → 100%
```

### 7. Database Migration

```bash
# golang-migrate — chạy trong CI/CD trước khi deploy
migrate -path ./migrations -database "${DATABASE_URL}" up

# Migration file naming convention
# {timestamp}_{description}.up.sql
# {timestamp}_{description}.down.sql
# Example: 20260424120000_add_document_status_index.up.sql
```

```sql
-- Migration best practices
-- ✅ Idempotent migrations
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_documents_status 
    ON documents(status) WHERE deleted_at IS NULL;

-- ✅ Non-blocking for large tables (CONCURRENTLY)
-- ✅ Always write .down.sql for rollback
-- ✅ Never drop columns in same migration as code deploy
```

---

## Checklist

- [ ] Docker image sử dụng distroless base image
- [ ] Multi-stage build để minimize image size
- [ ] Health check endpoints: `/healthz/live` và `/healthz/ready`
- [ ] Resource requests/limits set trên tất cả containers
- [ ] HPA đã config (min 2, max based on load test)
- [ ] CI pipeline: lint → test → security → build → deploy
- [ ] Test coverage gate ≥ 80% trong CI
- [ ] `govulncheck` chạy trong CI
- [ ] Staging promotion tự động, production promotion manual
- [ ] DB migrations chạy trước deployment
- [ ] Rollback strategy documented trong runbook
- [ ] Prometheus metrics exposed tại `/metrics`
- [ ] Structured JSON logs với trace_id
