---
id: MERGE-P3-T2
title: "obs-service: Tạo mới — Merge vnp-observability + vnp-infra + sm-engine"
phase: P3
service: obs-service (NEW)
priority: P2
status: Done
estimated: 6h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P1-T1]
---

## Mục Tiêu

Tạo `obs-service` (Observability Service) — service tập hợp metrics, traces, error tracking, cost analysis, và infrastructure topology.

## Services Bị Absorb

| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `vnp-observability` | 234 | Metrics, traces, errors, costs |
| `vnp-infra` | 203 | Service topology, databases, deployments |
| `sm-engine` | 410 | Engine metrics + status |

**Tổng: 847 lines** → 1 service

## Architecture

```
services/obs-service/
├── Dockerfile
├── go.mod
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── observability/
│   │   │   ├── entity.go        # Metric, Trace, Span, Error, CostEntry
│   │   │   └── errors.go
│   │   └── infra/
│   │       ├── entity.go        # ServiceInfo, Database, Deployment, Resource
│   │       └── errors.go
│   ├── usecase/
│   │   ├── observability/
│   │   │   └── service.go       # Metrics, ListTraces, GetTrace, Errors, Costs
│   │   └── infra/
│   │       └── service.go       # Topology, ListServices, GetService, Databases, Deployments
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── router.go        # ForwardService routes
│   │   ├── prometheus/
│   │   │   └── scraper.go       # Prometheus metrics scraping
│   │   ├── jaeger/
│   │   │   └── client.go        # Jaeger trace collection
│   │   └── docker/
│   │       └── client.go        # Docker API for container introspection
│   └── infra/
│       ├── pg/                  # Metrics + error persistence
│       └── config/
└── migrations/
    └── 001_obs_init.sql
```

## Domain Entities

```go
// domain/observability/entity.go

type MetricPoint struct {
    Name      string
    Value     float64
    Labels    map[string]string
    Timestamp time.Time
}

type MetricSummary struct {
    TotalRequests     int64
    ErrorRate         float64
    P50Latency        time.Duration
    P95Latency        time.Duration
    P99Latency        time.Duration
    RequestsPerSecond float64
    Services          []ServiceMetric
    Timestamp         time.Time
}

type ServiceMetric struct {
    Name          string
    Healthy       bool
    RequestCount  int64
    ErrorCount    int64
    AvgLatencyMs  float64
}

type Trace struct {
    TraceID   string
    Service   string
    Operation string
    Duration  time.Duration
    Status    string    // "ok" | "error"
    Spans     []*Span
    StartedAt time.Time
}

type Span struct {
    SpanID    string
    TraceID   string
    ParentID  string
    Operation string
    Service   string
    Duration  time.Duration
    Tags      map[string]string
    Error     string
}

type ErrorEntry struct {
    ID        string
    Service   string
    Type      string
    Message   string
    Stack     string
    Count     int
    FirstSeen time.Time
    LastSeen  time.Time
}

type CostEntry struct {
    Service    string
    Period     string    // "hour" | "day" | "month"
    LLMTokens  int64
    EmbedTokens int64
    StorageMB  int64
    EstCostUSD float64
    Timestamp  time.Time
}
```

```go
// domain/infra/entity.go

type ServiceInfo struct {
    Name        string
    Version     string
    Status      string    // "healthy" | "degraded" | "down"
    Uptime      time.Duration
    Replicas    int
    Port        int
    Address     string
    LastCheckAt time.Time
}

type TopologyGraph struct {
    Services  []*ServiceInfo
    Edges     []*ServiceEdge    // Service dependency edges
    UpdatedAt time.Time
}

type ServiceEdge struct {
    Source   string
    Target   string
    Protocol string   // "grpc" | "http" | "nats"
    Latency  float64  // avg latency in ms
}

type Database struct {
    Name      string
    Type      string    // "postgres" | "redis" | "neo4j" | "nats"
    Status    string
    Size      int64     // MB
    Connections int
}

type Deployment struct {
    Service   string
    Version   string
    Status    string
    Replicas  int
    CreatedAt time.Time
}

type Resource struct {
    Name   string
    Type   string    // "cpu" | "memory" | "disk" | "network"
    Used   float64
    Total  float64
    Unit   string
}
```

## Usecase Implementation

```go
// usecase/observability/service.go

type ObservabilityService struct {
    promScraper port.MetricsScraper     // Prometheus scraper
    jaegerClient port.TraceClient       // Jaeger client
    errors      port.ErrorRepository
    costs       port.CostRepository
}

func (s *ObservabilityService) Metrics(ctx context.Context) (*MetricSummary, error) {
    // Scrape Prometheus or aggregate from PostgreSQL
    metrics, err := s.promScraper.ScrapeAll(ctx)
    if err != nil {
        // Fallback: return from PostgreSQL metrics table
        return s.getMetricsFromDB(ctx)
    }
    return aggregateMetrics(metrics), nil
}

func (s *ObservabilityService) ListTraces(ctx context.Context, filter TraceFilter) ([]*Trace, int, error) {
    // Query Jaeger or PostgreSQL trace store
    return s.jaegerClient.ListTraces(ctx, filter)
}

func (s *ObservabilityService) GetTrace(ctx context.Context, traceID string) (*Trace, error) {
    return s.jaegerClient.GetTrace(ctx, traceID)
}

func (s *ObservabilityService) Errors(ctx context.Context, filter ErrorFilter) ([]*ErrorEntry, int, error) {
    return s.errors.List(ctx, filter)
}

func (s *ObservabilityService) Costs(ctx context.Context, period string) ([]*CostEntry, error) {
    return s.costs.GetByPeriod(ctx, period)
}
```

```go
// usecase/infra/service.go

type InfraService struct {
    docker   port.DockerClient
    registry port.ServiceRegistry    // Gateway's service registry interface
    db       port.DBInspector
}

func (s *InfraService) Topology(ctx context.Context) (*TopologyGraph, error) {
    // Discover all services via gateway registry
    services := s.registry.ListAll()
    graph := &TopologyGraph{Services: make([]*ServiceInfo, 0, len(services))}
    for _, svc := range services {
        healthy, _ := s.registry.HealthCheck(svc)
        info := &ServiceInfo{Name: svc, Status: statusFrom(healthy)}
        graph.Services = append(graph.Services, info)
    }
    // Build edges based on known service dependencies
    graph.Edges = buildStaticEdges()
    return graph, nil
}

func (s *InfraService) ListServices(ctx context.Context) ([]*ServiceInfo, error) {
    containers, _ := s.docker.ListContainers(ctx)
    return containerToServiceInfo(containers), nil
}

func (s *InfraService) Databases(ctx context.Context) ([]*Database, error) {
    return s.db.InspectAll(ctx)
}
```

## ForwardService Routes

```go
// adapter/grpc/router.go
func RegisterRoutes(router *forward.Router, obs ObsHandler, infra InfraHandler) {
    // Observability
    router.Handle("GET", "/v1/console/observability/metrics",      obs.Metrics)
    router.Handle("GET", "/v1/console/observability/traces",       obs.ListTraces)
    router.Handle("GET", "/v1/console/observability/traces/*",     obs.GetTrace)
    router.Handle("GET", "/v1/console/observability/errors",       obs.Errors)
    router.Handle("GET", "/v1/console/observability/costs",        obs.Costs)

    // Infrastructure
    router.Handle("GET", "/v1/console/infra/topology",             infra.Topology)
    router.Handle("GET", "/v1/console/infra/services",             infra.ListServices)
    router.Handle("GET", "/v1/console/infra/services/*",           infra.GetService)
    router.Handle("GET", "/v1/console/infra/databases",            infra.Databases)
    router.Handle("GET", "/v1/console/infra/resources",            infra.Resources)
    router.Handle("GET", "/v1/console/infra/deployments",          infra.Deployments)
}
```

## Database Migration

```sql
-- migrations/001_obs_init.sql

-- Metrics timeseries
CREATE TABLE IF NOT EXISTS obs_metrics (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL,
    value      FLOAT NOT NULL,
    labels     JSONB NOT NULL DEFAULT '{}',
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_obs_metrics_name ON obs_metrics(name, recorded_at DESC);

-- Error tracking
CREATE TABLE IF NOT EXISTS obs_errors (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service    TEXT NOT NULL,
    type       TEXT NOT NULL,
    message    TEXT NOT NULL,
    stack      TEXT,
    count      INT NOT NULL DEFAULT 1,
    first_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_obs_errors_service ON obs_errors(service, last_seen DESC);

-- Cost tracking
CREATE TABLE IF NOT EXISTS obs_costs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    service      TEXT NOT NULL,
    period       TEXT NOT NULL,    -- "2026-06-11:hour" | "2026-06-11:day"
    llm_tokens   BIGINT NOT NULL DEFAULT 0,
    embed_tokens BIGINT NOT NULL DEFAULT 0,
    storage_mb   BIGINT NOT NULL DEFAULT 0,
    est_cost_usd NUMERIC(10,4) NOT NULL DEFAULT 0,
    recorded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_obs_costs_period ON obs_costs(service, period);
```

## Config Environment Variables

```bash
GRPC_PORT=9090
HEALTH_PORT=9170
DATABASE_URL=postgres://...

# Prometheus scraping
PROMETHEUS_URL=http://prometheus:9090    # Optional
METRICS_SCRAPE_INTERVAL_SECONDS=30

# Jaeger tracing
JAEGER_URL=http://jaeger:14268          # Optional
TRACING_ENABLED=false                    # Default off (MVP)

# Docker introspection
DOCKER_SOCKET=/var/run/docker.sock      # Mount in container
DOCKER_ENABLED=true

# Service registry (to get service list)
GATEWAY_HEALTH_URL=http://vnp-gateway:11080
```

## go.mod

```
module vnp-memory/services/obs-service

go 1.25.0

require (
    vnp-memory/pkg/forward     v0.0.0
    vnp-memory/pkg/telemetry   v0.0.0
    vnp-memory/pkg/tenant      v0.0.0
    google.golang.org/grpc     v1.72.1
    github.com/jackc/pgx/v5    v5.7.0
    github.com/docker/docker   v27.x.x    # Docker SDK for container introspection
)
```

## Docker Compose — Mount Docker Socket

```yaml
obs-service:
  build: ./services/obs-service
  ports: ["9070:9090", "9170:9170"]
  volumes:
    - /var/run/docker.sock:/var/run/docker.sock:ro
  depends_on: [postgres]
```

## Acceptance Criteria

- [ ] `GET /v1/console/observability/metrics` returns MetricSummary JSON
- [ ] `GET /v1/console/observability/traces` returns trace list (empty list OK if no Jaeger)
- [ ] `GET /v1/console/observability/errors` returns error entries list
- [ ] `GET /v1/console/observability/costs` returns cost breakdown
- [ ] `GET /v1/console/infra/topology` returns service graph với 7+ nodes
- [ ] `GET /v1/console/infra/services` returns list of running containers
- [ ] `GET /v1/console/infra/databases` returns database status (postgres + redis + nats)
- [ ] `GET /v1/console/infra/resources` returns CPU/memory/disk stats
- [ ] When Prometheus unavailable → fallback to PostgreSQL metrics
- [ ] When Docker socket unavailable → returns static service list from config
- [ ] `/healthz` returns 200
- [ ] `go build ./services/obs-service/...` passes

## Ghi Chú

- **Prometheus + Jaeger optional** — MVP có thể dùng PostgreSQL aggregate queries
- **Docker socket mount** cần `volumes` config trong docker-compose
- **sm-engine** metrics → implement như engine status endpoint tổng hợp
- 3 services gốc giữ nguyên cho đến P4 cleanup
- Cost tracking: implement token counter middleware trong gateway → publish to NATS → obs-service subscribe
