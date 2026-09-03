---
id: FEAT-PIP-008
title: Infrastructure Layer — Config, Server, Wire, Telemetry
service: graphiti-pipeline
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
---

## Mục Tiêu

Implement infrastructure layer cho graphiti-pipeline: Viper config loader, gRPC server with interceptors, Google Wire dependency injection, OTel tracing + Prometheus metrics, và Dockerfile multi-stage build.

## Scope

### In Scope
- `internal/infra/config/config.go` — Viper loader with validation + env override
- `internal/infra/server/grpc.go` — gRPC server with OTel + recovery + logging interceptors
- `internal/infra/telemetry/tracer.go` — OTel tracer provider setup
- `internal/infra/telemetry/metrics.go` — Prometheus registry + custom counters/histograms
- `internal/infra/wire/wire.go` — Wire providers + injector generation
- `cmd/server/main.go` — Entry point: bootstrap → inject → serve → graceful shutdown
- `Dockerfile` — Multi-stage build (builder + distroless runtime)

### Out of Scope
- Kubernetes manifests (ops responsibility)
- CI/CD pipeline configuration

## Thiết Kế Kỹ Thuật

### Config Structure

```go
type Config struct {
    GRPCPort    int    `mapstructure:"GRPC_PORT" validate:"required,min=1024,max=65535"`
    HealthPort  int    `mapstructure:"HEALTH_PORT"`
    LogLevel    string `mapstructure:"LOG_LEVEL"`
    
    Postgres    PostgresConfig `mapstructure:",squash"`
    Store       StoreConfig    `mapstructure:",squash"`
    LLM         LLMConfig      `mapstructure:",squash"`
    Embedder    EmbedderConfig `mapstructure:",squash"`
    NATS        NATSConfig     `mapstructure:",squash"`
    OTel        OTelConfig     `mapstructure:",squash"`
    Pipeline    PipelineConfig `mapstructure:",squash"`
}
```

### gRPC Server Setup

```go
func NewGRPCServer(cfg Config) *grpc.Server {
    server := grpc.NewServer(
        grpc.ChainUnaryInterceptor(
            otelgrpc.UnaryServerInterceptor(),        // OTel tracing
            grpc_recovery.UnaryServerInterceptor(),    // Panic recovery
            grpc_logging.UnaryServerInterceptor(),     // Structured logging
            TenantExtractorInterceptor(),              // x-tenant-id → context
        ),
    )
    return server
}
```

### Wire DI

```go
// wire.go — Provider sets
var InfraSet = wire.NewSet(
    config.LoadConfig,
    NewGRPCServer,
    telemetry.NewTracer,
    telemetry.NewMetrics,
)

var AdapterSet = wire.NewSet(
    grpc.NewIngestionHandler,
    grpc.NewKnowledgeHandler,
    llm.NewBifrostClient,
    embedder.NewBifrostEmbedder,
    postgres.NewEpisodeRepo,
    postgres.NewSagaRepo,
    neo4j.NewEntityReader,
    nats.NewPublisher,
    client.NewStoreClient,
)

var UsecaseSet = wire.NewSet(
    ingest.NewIngestEpisodeUseCase,
    ingest.NewBulkIngestUseCase,
    knowledge.NewExtractEntitiesUseCase,
    // ... remaining usecases
)
```

### main.go Bootstrap

```go
func main() {
    // 1. Load config (Viper)
    // 2. Init OTel tracer + Prometheus metrics
    // 3. Wire inject all dependencies
    // 4. Register gRPC services on :GRPC_PORT
    // 5. Start health HTTP server on :HEALTH_PORT
    // 6. Block on signal (SIGTERM, SIGINT)
    // 7. Graceful shutdown (drain → close DB → flush telemetry)
}
```

## Acceptance Criteria

- [ ] AC-1: `go run cmd/server/main.go` starts gRPC on :9021 and health on :9094
- [ ] AC-2: Config loads from environment variables with validation errors on missing required values
- [ ] AC-3: `wire gen` produces valid injector without compilation errors
- [ ] AC-4: OTel traces appear in collector on any RPC call
- [ ] AC-5: Prometheus metrics endpoint at :9094/metrics returns graphiti_pipeline_* counters
- [ ] AC-6: Graceful shutdown completes within SHUTDOWN_TIMEOUT, no leaked goroutines
- [ ] AC-7: Docker build produces image < 30MB (distroless)
- [ ] AC-8: `grpc.health.v1.Health/Check` returns SERVING after startup

## Test Requirements
- **Unit tests**: Config validation, interceptor behavior
- **Integration tests**: Server startup → health check → shutdown lifecycle
- **Minimum coverage**: 70% (infra layer has framework code)
