# pipeline-service — Data Models

> **Service**: `services/pipeline-service`
> **Role**: Pipeline orchestration — absorbs `vnp-pipelines`, `ba-knowledge-service`, `ba-knowledge-worker`.
> Manages async ingestion jobs, pipeline workers, and job queues for all memory engines.

---

## Pipeline

```go
type Pipeline struct {
    Engine   string           `json:"engine"` // "graphiti" | "cognee" | "memobase" | "knowledge"
    Name     string           `json:"name"`
    Status   string           `json:"status"` // "idle" | "running" | "paused" | "error"
    JobCount PipelineJobCount `json:"job_count"`
    Workers  []*Worker        `json:"workers,omitempty"`
    Config   map[string]any   `json:"config,omitempty"`
}

type PipelineJobCount struct {
    Pending   int `json:"pending"`
    Running   int `json:"running"`
    Completed int `json:"completed"`
    Failed    int `json:"failed"`
}
```

---

## Job

```go
type Job struct {
    ID          string         `json:"id"`
    Engine      string         `json:"engine"`
    Type        string         `json:"type"`   // "ingest" | "index" | "sync" | "cognify"
    Status      string         `json:"status"` // "pending" | "running" | "completed" | "failed"
    Payload     map[string]any `json:"payload,omitempty"`
    Result      map[string]any `json:"result,omitempty"`
    Error       string         `json:"error,omitempty"`
    Priority    int            `json:"priority"`
    CreatedAt   time.Time      `json:"created_at"`
    StartedAt   *time.Time     `json:"started_at,omitempty"`
    CompletedAt *time.Time     `json:"completed_at,omitempty"`
}

type JobFilter struct {
    Status string `json:"status,omitempty"`
    Type   string `json:"type,omitempty"`
    Limit  int    `json:"limit"`
    Offset int    `json:"offset"`
}
```

---

## Queue

```go
type Queue struct {
    Name    string `json:"name"`
    Engine  string `json:"engine"`
    Size    int    `json:"size"`
    MaxSize int    `json:"max_size"`
    Workers int    `json:"workers"`
}
```

---

## Worker

```go
type Worker struct {
    ID       string    `json:"id"`
    Engine   string    `json:"engine"`
    Status   string    `json:"status"` // "idle" | "busy" | "offline"
    JobID    string    `json:"job_id,omitempty"`
    LastSeen time.Time `json:"last_seen"`
}
```

---

## PipelineTemplate

```go
type PipelineTemplate struct {
    ID          string         `json:"id"`
    Name        string         `json:"name"`
    Engine      string         `json:"engine"`
    Description string         `json:"description"`
    Config      map[string]any `json:"config,omitempty"`
}
```

---

## knowledge — Knowledge Ingestion (ba-knowledge-service)

```go
// See services/pipeline-service/internal/domain/knowledge/entity.go
```

---

## Sources
- [`services/pipeline-service/internal/domain/pipeline/entity.go`](../../services/pipeline-service/internal/domain/pipeline/entity.go)
- [`services/pipeline-service/internal/domain/knowledge/entity.go`](../../services/pipeline-service/internal/domain/knowledge/entity.go)
