---
id: MERGE-P3-T1
title: "pipeline-service: Tạo mới — Merge vnp-pipelines + ba-knowledge-service + ba-knowledge-worker"
phase: P3
service: pipeline-service (NEW)
priority: P2
status: Done
estimated: 8h
created: 2026-06-11
linked_sol: SOL-003
depends_on: [MERGE-P1-T1]
---

## Mục Tiêu

Tạo `pipeline-service` — service quản lý tất cả async processing và pipeline execution. Service này có **2 binaries**: gRPC server (API + routing) và worker (background queue processor).

## Services Bị Absorb

| Service | Lines | Chức Năng |
|---------|-------|-----------|
| `vnp-pipelines` | 205 | Pipeline management API |
| `ba-knowledge-service` | 2,848 | BA knowledge CRUD (stub) |
| `ba-knowledge-worker` | 1,061 | Redis queue worker (**REAL implementation**) |

**Tổng: 4,114 lines** → 1 service với 2 binaries

## Architecture

```
services/pipeline-service/
├── Dockerfile                    # Builds both binaries
├── Dockerfile.worker             # Optional: separate worker image
├── go.mod
├── cmd/
│   ├── server/
│   │   └── main.go              # gRPC server (ForwardService)
│   └── worker/
│       └── main.go              # Background queue worker
├── internal/
│   ├── domain/
│   │   ├── pipeline/
│   │   │   ├── entity.go        # Pipeline, Job, Queue, Worker, Template
│   │   │   └── errors.go
│   │   └── knowledge/
│   │       ├── entity.go        # PRD, Outline, IndexJob, KnowledgeItem
│   │       └── errors.go
│   ├── usecase/
│   │   ├── pipeline/
│   │   │   └── service.go       # Status, ListJobs, GetJob, Queues, Workers, Templates
│   │   └── knowledge/
│   │       ├── index.go         # HandleIndexPRD, HandleGenOutline
│   │       └── service.go       # KnowledgeCRUD
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── router.go        # ForwardService routes (server binary)
│   │   ├── worker/
│   │   │   ├── handlers.go      # Worker task handlers (worker binary)
│   │   │   └── registry.go      # Task type → handler registration
│   │   └── handler/
│   │       └── pipeline.go      # HTTP handler adapters
│   └── infra/
│       ├── redis/               # Queue backend (from ba-knowledge-worker)
│       ├── pg/                  # Job persistence
│       ├── nats/                # Pipeline event streaming
│       └── config/
└── migrations/
    └── 001_pipeline_init.sql
```

## Domain Entities

```go
// domain/pipeline/entity.go

type Pipeline struct {
    Engine   string      // "graphiti" | "cognee" | "memobase" | "knowledge"
    Name     string
    Status   string      // "idle" | "running" | "paused" | "error"
    JobCount PipelineJobCount
    Workers  []*Worker
    Config   map[string]any
}

type PipelineJobCount struct {
    Pending   int
    Running   int
    Completed int
    Failed    int
}

type Job struct {
    ID          string
    Engine      string
    Type        string    // "ingest" | "index" | "sync" | "cognify"
    Status      string    // "pending" | "running" | "completed" | "failed"
    Payload     map[string]any
    Result      map[string]any
    Error       string
    Priority    int
    CreatedAt   time.Time
    StartedAt   *time.Time
    CompletedAt *time.Time
}

type Queue struct {
    Name    string
    Engine  string
    Size    int
    MaxSize int
    Workers int
}

type Worker struct {
    ID       string
    Engine   string
    Status   string    // "idle" | "busy" | "offline"
    JobID    string
    LastSeen time.Time
}

type PipelineTemplate struct {
    ID          string
    Name        string
    Engine      string
    Description string
    Config      map[string]any
}
```

```go
// domain/knowledge/entity.go (từ ba-knowledge-worker)

type PRD struct {
    ID       string
    Title    string
    Content  string
    Tags     []string
    Status   string    // "draft" | "indexed" | "failed"
    TenantID string
}

type Outline struct {
    PRDID    string
    Sections []OutlineSection
    Status   string
}

type OutlineSection struct {
    Title    string
    Level    int
    Content  string
    Children []OutlineSection
}

type IndexJob struct {
    Type    string     // "index_prd" | "gen_outline"
    PRDID   string
    Status  string
    Error   string
}
```

## Usecase — Pipeline Management

```go
// usecase/pipeline/service.go
type PipelineUseCase struct {
    jobs    port.JobRepository
    workers port.WorkerRegistry
    pub     port.EventPublisher
}

func (s *PipelineUseCase) Status(ctx context.Context) ([]*Pipeline, error) {
    // Aggregate status from all registered engines
    engines := []string{"graphiti", "cognee", "memobase", "knowledge"}
    pipelines := make([]*Pipeline, 0, len(engines))
    for _, engine := range engines {
        stats, _ := s.jobs.GetStats(ctx, engine)
        workers, _ := s.workers.ListByEngine(ctx, engine)
        pipelines = append(pipelines, &Pipeline{
            Engine:  engine,
            Status:  inferStatus(stats, workers),
            JobCount: stats,
            Workers: workers,
        })
    }
    return pipelines, nil
}

func (s *PipelineUseCase) GetJob(ctx context.Context, engine, jobID string) (*Job, error) {
    return s.jobs.GetByID(ctx, jobID)
}

func (s *PipelineUseCase) ListJobs(ctx context.Context, engine string, filter JobFilter) ([]*Job, int, error) {
    return s.jobs.ListByEngine(ctx, engine, filter)
}

func (s *PipelineUseCase) Queues(ctx context.Context) ([]*Queue, error) {
    return s.workers.GetQueues(ctx)
}
```

## Worker Binary (từ ba-knowledge-worker)

```go
// cmd/worker/main.go — background queue processor

func main() {
    cfg := loadConfig()
    
    db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
    redisCfg := queue.RedisConfig{
        Addr:     cfg.RedisAddr,
        Password: cfg.RedisPassword,
        DB:       db,
    }

    consumer := queue.NewConsumer(redisCfg, cfg.WorkerConcurrency)

    // Register task handlers
    consumer.RegisterHandler(queue.TaskTypeIndexPRD,    worker.HandleIndexPRD)
    consumer.RegisterHandler(queue.TaskTypeGenOutline,  worker.HandleGenOutline)
    // New handlers for pipeline tasks
    consumer.RegisterHandler("graphiti.ingest",  worker.HandleGraphitiIngest)
    consumer.RegisterHandler("cognee.cognify",   worker.HandleCogneeCognify)
    consumer.RegisterHandler("memobase.flush",   worker.HandleMemobaseFlush)

    log.Println("Starting Pipeline Worker...")
    if err := consumer.Start(); err != nil {
        log.Fatalf("worker failed: %v", err)
    }
}

// adapter/worker/handlers.go
func HandleIndexPRD(ctx context.Context, job queue.Job) error {
    // Extract PRD from payload, index into knowledge graph
    var payload IndexPRDPayload
    if err := json.Unmarshal(job.Payload, &payload); err != nil {
        return err
    }
    // Call kg-service to index the PRD
    return kgClient.IngestDocument(ctx, payload.PRDID, payload.Content)
}

func HandleGenOutline(ctx context.Context, job queue.Job) error {
    // Generate outline from PRD using LLM
    var payload GenOutlinePayload
    json.Unmarshal(job.Payload, &payload)
    // Call LLM proxy, store result
    return outlineService.Generate(ctx, payload.PRDID)
}
```

## ForwardService Routes

```go
// adapter/grpc/router.go
func RegisterRoutes(router *forward.Router, pipe PipelineHandler) {
    router.Handle("GET", "/v1/console/pipelines/status",             pipe.Status)
    router.Handle("GET", "/v1/console/pipelines/queues",             pipe.Queues)
    router.Handle("GET", "/v1/console/pipelines/workers",            pipe.Workers)
    router.Handle("GET", "/v1/console/pipelines/templates",          pipe.Templates)
    router.Handle("GET", "/v1/console/pipelines/*",                  pipe.GetEngine)
    router.Handle("GET", "/v1/console/pipelines/*/jobs",             pipe.ListJobs)
    router.Handle("GET", "/v1/console/pipelines/*/jobs/*",           pipe.GetJob)
}
```

## Database Migration

```sql
-- migrations/001_pipeline_init.sql

CREATE TABLE IF NOT EXISTS pipeline_jobs (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engine       TEXT NOT NULL,
    type         TEXT NOT NULL,
    status       TEXT NOT NULL DEFAULT 'pending',
    payload      JSONB NOT NULL DEFAULT '{}',
    result       JSONB,
    error        TEXT,
    priority     INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);
CREATE INDEX idx_pipeline_jobs_engine ON pipeline_jobs(engine, status);
CREATE INDEX idx_pipeline_jobs_status ON pipeline_jobs(status);

CREATE TABLE IF NOT EXISTS pipeline_workers (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    engine    TEXT NOT NULL,
    status    TEXT NOT NULL DEFAULT 'idle',
    job_id    UUID,
    last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS knowledge_prds (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title     TEXT NOT NULL,
    content   TEXT NOT NULL,
    tags      TEXT[] NOT NULL DEFAULT '{}',
    status    TEXT NOT NULL DEFAULT 'draft',
    tenant_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS knowledge_outlines (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prd_id    UUID NOT NULL REFERENCES knowledge_prds(id),
    sections  JSONB NOT NULL DEFAULT '[]',
    status    TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Dockerfile (Multi-stage, Multi-binary)

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum ./
COPY pkg/ ./pkg/
COPY services/pipeline-service/ ./services/pipeline-service/
RUN go build -o /bin/pipeline-server ./services/pipeline-service/cmd/server
RUN go build -o /bin/pipeline-worker ./services/pipeline-service/cmd/worker

FROM alpine:3.19
# Default: run server
ARG BINARY=server
COPY --from=builder /bin/pipeline-${BINARY} /bin/service
CMD ["/bin/service"]
```

```yaml
# docker-compose.consolidated.yml — 2 containers from same image
pipeline-service:
  build:
    context: .
    dockerfile: services/pipeline-service/Dockerfile
    args:
      BINARY: server
  ports: ["9060:9090"]

pipeline-worker:
  build:
    context: .
    dockerfile: services/pipeline-service/Dockerfile
    args:
      BINARY: worker
  depends_on: [redis, nats]
  deploy:
    replicas: 2    # Run 2 worker instances
```

## Config Environment Variables

```bash
# Server
GRPC_PORT=9090
HEALTH_PORT=9160
DATABASE_URL=postgres://...
NATS_URL=nats://nats:4222

# Worker
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=2
WORKER_CONCURRENCY=10         # From ba-knowledge-worker
WORKER_QUEUE_POLL_MS=500

# Shared
KG_SERVICE_ADDR=kg-service:9090
```

## go.mod

```
module vnp-memory/services/pipeline-service

go 1.25.0

require (
    vnp-memory/pkg/forward     v0.0.0
    vnp-memory/pkg/telemetry   v0.0.0
    vnp-memory/pkg/tenant      v0.0.0
    google.golang.org/grpc     v1.72.1
    github.com/jackc/pgx/v5    v5.7.0
    github.com/redis/go-redis/v9 v9.x.x
    # Note: ba-knowledge-worker uses github.com/blcvn/ba-shared-libs/pkg/queue
    # Need to vendor or reimplement this queue interface
)
```

## ⚠️ Dependency Risk: ba-shared-libs

`ba-knowledge-worker` sử dụng `github.com/blcvn/ba-shared-libs/pkg/queue` — external dependency từ một repo khác. Cần giải quyết 1 trong 2 cách:
1. **Vendor**: Copy queue package vào `pkg/queue/` trong repo này
2. **Reimplement**: Tạo simple Redis queue wrapper tương đương trong `infra/redis/`

**Recommendation:** Option 2 — implement `infra/redis/consumer.go` với interface tương đương.

## Acceptance Criteria

- [ ] `GET /v1/console/pipelines/status` returns pipeline status cho tất cả engines
- [ ] `GET /v1/console/pipelines/queues` returns queue depths
- [ ] `GET /v1/console/pipelines/workers` returns worker status
- [ ] `GET /v1/console/pipelines/{engine}/jobs` lists jobs by engine
- [ ] `GET /v1/console/pipelines/{engine}/jobs/{id}` returns job details
- [ ] Worker binary starts và processes Redis queue tasks
- [ ] `HandleIndexPRD` task handler functional
- [ ] `HandleGenOutline` task handler functional
- [ ] Job status tracked in PostgreSQL
- [ ] Server binary: `/healthz` returns 200
- [ ] `go build ./services/pipeline-service/cmd/server` passes
- [ ] `go build ./services/pipeline-service/cmd/worker` passes

## Ghi Chú

- `ba-knowledge-worker` queue dependency cần được resolved trước khi implement
- 3 services gốc giữ nguyên cho đến P4 cleanup
- Worker có thể deploy multiple replicas — stateless design
