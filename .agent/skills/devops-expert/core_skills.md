# Core Skills — DevOps, CI/CD & Infrastructure

## CI/CD Pipeline Design

### Pipeline Stages (Required for Every Service)
```yaml
# GitHub Actions / GitLab CI standard pipeline
stages:
  - lint          # Code style, static analysis (golangci-lint, eslint)
  - test          # Unit tests + integration tests (go test -race ./...)
  - security      # govulncheck, gosec, gitleaks, npm audit
  - build         # Docker image build (multi-stage)
  - push          # Push to container registry (tag: commit SHA + semver)
  - deploy-staging  # Auto-deploy to staging on main branch merge
  - smoke-test    # Run smoke tests against staging
  - deploy-prod   # Manual approval gate → deploy to production
```

### Docker Best Practices
```dockerfile
# Multi-stage build — keep final image minimal
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download          # Cache dependencies layer
COPY . .
RUN CGO_ENABLED=0 go build -o service ./cmd/service

FROM gcr.io/distroless/static:nonroot  # Minimal, secure base image
COPY --from=builder /app/service /service
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/service"]
```

## Kubernetes Deployment Standards
```yaml
# Every service deployment must include:
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 2                    # Minimum 2 replicas for HA
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 0          # Zero-downtime deployment
      maxSurge: 1
  template:
    spec:
      containers:
        - name: service
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"    # Always set limits to prevent OOM
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
```

## Observability Stack
| Component | Tool | Purpose |
|---|---|---|
| **Metrics** | Prometheus + Grafana | Service health dashboards, alerting |
| **Logs** | Loki + Grafana | Centralized structured log querying |
| **Traces** | Jaeger / Tempo | Distributed request tracing across services |
| **Alerts** | Alertmanager | PagerDuty/Slack notifications for SLO breaches |

### Key Metrics Every Service Must Expose
```go
// Standard metrics via Prometheus client
var (
    requestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "http_request_duration_seconds"},
        []string{"method", "path", "status"},
    )
    requestTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "http_requests_total"},
        []string{"method", "path", "status"},
    )
)
```

## Infrastructure as Code (IaC)
- **Helm Charts:** All Kubernetes resources defined as Helm charts for templating and environment-specific values.
- **Terraform:** Cloud infrastructure (VPC, managed databases, load balancers) defined in Terraform with remote state in S3/GCS.
- **GitOps:** Production changes are made exclusively through Git — no manual `kubectl apply`. ArgoCD or Flux watches the Git repository.

## Database Migration Strategy
```bash
# golang-migrate integrated into CI/CD
migrate -path ./migrations -database "${DB_URL}" up

# Always: migrations are backward compatible
# Never: remove a column in the same release that removes the code using it
# Pattern: Expand → Migrate data → Contract (3-phase migration)
```
