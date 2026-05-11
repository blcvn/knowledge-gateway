---
id: FEAT-SEA-003
title: Search Service — Infrastructure + Wire DI
service: cognee-search
version: 1.0.0
status: Ready
priority: P0
created: 2026-05-10
updated: 2026-05-10
linked_sol: SOL-001
linked_feat: FEAT-SEA-002
---

## Mục Tiêu

Implement Layer 4 (Infrastructure) cho cognee-search — config, server, telemetry, Wire DI, Dockerfile.

## Scope

### In Scope
- Config loader: gRPC port 9013, Neo4j, Qdrant, Redis, NATS, Bifrost, OTel
- gRPC server + HTTP health (/healthz on 9093)
- OTel tracer/metrics (search latency per strategy, cache hit ratio)
- Wire DI providers + injector
- Dockerfile (multi-stage, ≤50MB)
- Redis connection pool (for query cache)

### Out of Scope
- Domain/Usecase/Adapter (FEAT-SEA-001, FEAT-SEA-002)

## Thiết Kế Kỹ Thuật

### Config

```go
type Config struct {
    Service    ServiceConfig    // name: cognee-search
    GRPC       GRPCConfig       // port: 9013
    Health     HealthConfig     // port: 9093
    Neo4j      Neo4jConfig      // graph search
    Qdrant     QdrantConfig     // vector search
    Redis      RedisConfig      // query cache
    NATS       NATSConfig       // events
    Bifrost    BifrostConfig    // LLM + reranker
    Telemetry  TelemetryConfig  // OTel
    Search     SearchConfig     // cache TTL, timeouts
}

type SearchConfig struct {
    CacheTTL          time.Duration `mapstructure:"cache_ttl" default:"5m"`
    MaxConcurrentQueries int       `mapstructure:"max_concurrent" default:"50"`
    DefaultTopK       int          `mapstructure:"default_top_k" default:"10"`
    RerankModelID     string       `mapstructure:"rerank_model" default:"bge-reranker-v2-m3"`
}
```

## Acceptance Criteria

- [ ] AC-1: Service starts on gRPC port 9013 with all retrievers registered
- [ ] AC-2: Graceful shutdown drains in-flight search requests
- [ ] AC-3: Wire generates without errors
- [ ] AC-4: Prometheus metrics: `search_latency_seconds{strategy="..."}`, `cache_hit_total`
- [ ] AC-5: Docker image ≤50MB
- [ ] AC-6: Health check returns SERVING when Neo4j + Qdrant + Redis connected

## Test Requirements

- **Smoke test**: `go run cmd/server/main.go` starts
- **Coverage**: ≥ 80%
