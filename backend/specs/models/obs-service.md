# obs-service — Data Models

> **Service**: `services/obs-service`
> **Role**: Observability & infrastructure topology service — absorbs `vnp-observability`, `vnp-infra`.
> Exposes metrics, distributed traces, error aggregation, cost tracking, and service topology.

---

## observability — Metrics & Traces

### MetricPoint

```go
type MetricPoint struct {
    Name      string            `json:"name"`
    Value     float64           `json:"value"`
    Labels    map[string]string `json:"labels,omitempty"`
    Timestamp time.Time         `json:"timestamp"`
}
```

### MetricSummary

```go
type MetricSummary struct {
    TotalRequests     int64           `json:"total_requests"`
    ErrorRate         float64         `json:"error_rate"`
    P50LatencyMs      float64         `json:"p50_latency_ms"`
    P95LatencyMs      float64         `json:"p95_latency_ms"`
    P99LatencyMs      float64         `json:"p99_latency_ms"`
    RequestsPerSecond float64         `json:"requests_per_second"`
    Services          []ServiceMetric `json:"services"`
    Timestamp         time.Time       `json:"timestamp"`
}

type ServiceMetric struct {
    Name         string  `json:"name"`
    Healthy      bool    `json:"healthy"`
    RequestCount int64   `json:"request_count"`
    ErrorCount   int64   `json:"error_count"`
    AvgLatencyMs float64 `json:"avg_latency_ms"`
}
```

### Trace & Span

```go
type Trace struct {
    TraceID   string        `json:"trace_id"`
    Service   string        `json:"service"`
    Operation string        `json:"operation"`
    Duration  time.Duration `json:"duration_ms"`
    Status    string        `json:"status"` // "ok" | "error"
    Spans     []*Span       `json:"spans,omitempty"`
    StartedAt time.Time     `json:"started_at"`
}

type Span struct {
    SpanID    string            `json:"span_id"`
    TraceID   string            `json:"trace_id"`
    ParentID  string            `json:"parent_id,omitempty"`
    Operation string            `json:"operation"`
    Service   string            `json:"service"`
    Duration  time.Duration     `json:"duration_ms"`
    Tags      map[string]string `json:"tags,omitempty"`
    Error     string            `json:"error,omitempty"`
}
```

### ErrorEntry

```go
type ErrorEntry struct {
    ID        string    `json:"id"`
    Service   string    `json:"service"`
    Type      string    `json:"type"`
    Message   string    `json:"message"`
    Stack     string    `json:"stack,omitempty"`
    Count     int       `json:"count"`
    FirstSeen time.Time `json:"first_seen"`
    LastSeen  time.Time `json:"last_seen"`
}
```

### CostEntry

```go
type CostEntry struct {
    Service     string    `json:"service"`
    Period      string    `json:"period"` // "2026-06-11:hour" | "2026-06-11:day"
    LLMTokens   int64     `json:"llm_tokens"`
    EmbedTokens int64     `json:"embed_tokens"`
    StorageMB   int64     `json:"storage_mb"`
    EstCostUSD  float64   `json:"est_cost_usd"`
    Timestamp   time.Time `json:"timestamp"`
}
```

### Filters

```go
type TraceFilter struct {
    Service string `json:"service,omitempty"`
    Status  string `json:"status,omitempty"`
    Limit   int    `json:"limit"`
    Offset  int    `json:"offset"`
}

type ErrorFilter struct {
    Service string `json:"service,omitempty"`
    Limit   int    `json:"limit"`
}
```

---

## infra — Infrastructure Topology

### ServiceInfo

```go
type ServiceInfo struct {
    Name        string        `json:"name"`
    Version     string        `json:"version,omitempty"`
    Status      string        `json:"status"` // "healthy" | "degraded" | "down"
    Uptime      time.Duration `json:"uptime_seconds"`
    Replicas    int           `json:"replicas"`
    Port        int           `json:"port,omitempty"`
    Address     string        `json:"address,omitempty"`
    LastCheckAt time.Time     `json:"last_check_at"`
}
```

### Topology Graph

```go
type TopologyGraph struct {
    Services  []*ServiceInfo `json:"services"`
    Edges     []*ServiceEdge `json:"edges"`
    UpdatedAt time.Time      `json:"updated_at"`
}

type ServiceEdge struct {
    Source    string  `json:"source"`
    Target    string  `json:"target"`
    Protocol  string  `json:"protocol"` // "grpc" | "http" | "nats"
    LatencyMs float64 `json:"latency_ms,omitempty"`
}
```

### Database

```go
type Database struct {
    Name        string `json:"name"`
    Type        string `json:"type"` // "postgres" | "redis" | "neo4j" | "nats"
    Status      string `json:"status"`
    SizeMB      int64  `json:"size_mb"`
    Connections int    `json:"connections"`
}
```

### Deployment & Resource

```go
type Deployment struct {
    Service   string    `json:"service"`
    Version   string    `json:"version"`
    Status    string    `json:"status"`
    Replicas  int       `json:"replicas"`
    CreatedAt time.Time `json:"created_at"`
}

type Resource struct {
    Name  string  `json:"name"`
    Type  string  `json:"type"` // "cpu" | "memory" | "disk" | "network"
    Used  float64 `json:"used"`
    Total float64 `json:"total"`
    Unit  string  `json:"unit"`
}
```

---

## Known Services (Topology Registry)

```
gateway, vnp-platform, storage-service, kg-service,
memory-service, search-service, pipeline-service, obs-service
```

---

## Sources
- [`services/obs-service/internal/domain/observability/entity.go`](../../services/obs-service/internal/domain/observability/entity.go)
- [`services/obs-service/internal/domain/infra/entity.go`](../../services/obs-service/internal/domain/infra/entity.go)
