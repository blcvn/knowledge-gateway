---
id: FEAT-COG-003
title: Cognify Service — Infrastructure + Wire DI
service: cognee-cognify
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_feat: FEAT-COG-002
---

## Mục Tiêu

Implement Layer 4 (Infrastructure) cho cognee-cognify — config, server, telemetry, Wire DI, Dockerfile.

## Scope

### In Scope
- Config loader (Viper): gRPC port, Neo4j, Qdrant, PostgreSQL, NATS, Bifrost, OTel
- gRPC server + graceful shutdown
- OTel tracer/metrics/logger
- Wire DI providers + injector
- Health check (gRPC Health v1 + HTTP /healthz)
- Dockerfile (multi-stage, ≤50MB)
- Bulkhead semaphore for LLM call concurrency control

### Out of Scope
- Domain/Usecase/Adapter (FEAT-COG-001, FEAT-COG-002)

## Thiết Kế Kỹ Thuật

### Config

```go
type Config struct {
    Service    ServiceConfig    // name, version
    GRPC       GRPCConfig       // port: 9012
    Health     HealthConfig     // port: 9092
    Postgres   PostgresConfig   // job state DB
    Neo4j      Neo4jConfig      // graph DB
    Qdrant     QdrantConfig     // vector DB
    NATS       NATSConfig       // event bus
    Bifrost    BifrostConfig    // LLM gateway
    Telemetry  TelemetryConfig  // OTel
    Pipeline   PipelineConfig   // concurrency, timeouts
}

type PipelineConfig struct {
    MaxConcurrentLLMCalls int           `mapstructure:"max_concurrent_llm" default:"5"`
    StageTimeout          time.Duration `mapstructure:"stage_timeout" default:"5m"`
    ChunkSize             int           `mapstructure:"chunk_size" default:"512"`
    ChunkOverlap          int           `mapstructure:"chunk_overlap" default:"50"`
}
```

## Acceptance Criteria

- [ ] AC-1: Service starts on configured gRPC port with all adapters wired
- [ ] AC-2: Graceful shutdown completes in-progress pipeline stages
- [ ] AC-3: Wire generates without errors
- [ ] AC-4: Bulkhead limits concurrent LLM calls to configured max
- [ ] AC-5: Docker image builds ≤50MB
- [ ] AC-6: Health check returns SERVING when all adapters connected

## Test Requirements

- **Smoke test**: `go run cmd/server/main.go` starts
- **Coverage**: ≥ 80%
