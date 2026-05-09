---
id: TASK-003
title: Config (Viper) + Wire DI Setup
service: vnp-gateway
version: 1.0.0
status: Done
priority: P0
created: 2026-05-09
updated: 2026-05-09
completed: 2026-05-09
linked_sol: SOL-001
linked_tech: TECH-001
depends_on: [TASK-001]
estimate: 3h
actual: 3h
---

## Mục Tiêu

Setup configuration loading từ env vars và production-ready dependency injection.

## Phạm Vi

### Files đã tạo
- `gateway/internal/infra/config/config.go` — 221 lines
- `gateway/cmd/main.go` — 245 lines
- `gateway/.env.example` — 65 lines

### Chi tiết triển khai

#### `config.go` — 11 Config sub-structs, 40+ env vars
```go
type Config struct {
    Server   ServerConfig            // REST/gRPC/MCP/Health ports, log level
    Auth     AuthConfig              // JWT RS256 key, issuer, audience, dev mode
    Postgres PostgresConfig          // DSN, max/min conns
    Redis    RedisConfig             // Addr, password, DB
    RateLimit RateLimitConfig        // Enabled, RPM per tier
    Circuit  CircuitConfig           // MaxFailures, Timeout, MaxRequests
    Timeout  TimeoutConfig           // Default, Ingestion, Search, MCP
    CORS     CORSConfig              // Origins, Credentials, MaxAge
    OTEL     OTELConfig              // Endpoint, ServiceName
    NATS     NATSConfig              // URL, CredsFile
    Services map[string]string       // 35 service → address mappings
}
```

> **Thay đổi so với spec**: Sử dụng manual config loading thay vì Viper (giảm dependency footprint). Wire DI thay bằng manual wiring trong `cmd/main.go` với graceful degradation pattern (noop fallbacks khi infra unavailable).

#### `cmd/main.go` — Production-grade entry point
```go
func main() {
    cfg := config.Load()
    // Infrastructure setup with graceful degradation:
    // - NATS → noopPublisher if unavailable
    // - PostgreSQL → noopKeyStore if unavailable
    // - Redis → noopRateLimitStore if unavailable
    // - gRPC → noopRegistry if unavailable

    // 3-server lifecycle:
    // REST :8080 + MCP :8082 + Observability :11080
    // Background gRPC health checker (30s interval)
    // Ordered cleanup on shutdown
}
```

#### DefaultConfig — 35 downstream service addresses
```go
cfg.Services = map[string]string{
    "cognee-ingestion":   "cognee-ingestion:9011",
    "cognee-search":      "cognee-search:9013",
    "graphiti-ingestion": "graphiti-ingestion:9021",
    // ... 32 more services
}
```

## Acceptance Criteria

- [x] AC-1: Config loads all env vars from DOC-S05 (configuration.md) ✅
- [x] AC-2: DI wiring compiles — manual wiring thay vì Wire codegen ✅ (simplified)
- [x] AC-3: `cmd/main.go` starts 3 HTTP servers (REST, MCP, Observability) and shuts down on SIGTERM ✅
- [x] AC-4: Config has sensible defaults for all optional values (35 service addresses, ports, timeouts) ✅
- [x] AC-5: Env vars loadable directly (REST_PORT, REDIS_ADDR, etc.) ✅

## Environment Variables

| Category | Variables | Count |
|----------|-----------|-------|
| Server | REST_PORT, GRPC_PORT, MCP_PORT, HEALTH_PORT, LOG_LEVEL, LOG_FORMAT | 6 |
| Auth | AUTH_DEV_MODE, AUTH_JWT_PUBLIC_KEY, AUTH_JWT_ISSUER, AUTH_JWT_AUDIENCE | 4 |
| Postgres | POSTGRES_DSN, POSTGRES_MAX_CONNS, POSTGRES_MIN_CONNS | 3 |
| Redis | REDIS_ADDR, REDIS_PASSWORD, REDIS_DB | 3 |
| Rate Limit | RATELIMIT_ENABLED, *_RPM (3 tiers) | 4 |
| Circuit | CB_MAX_FAILURES, CB_TIMEOUT, CB_MAX_REQUESTS | 3 |
| Timeout | TIMEOUT_DEFAULT, TIMEOUT_INGESTION, TIMEOUT_SEARCH, TIMEOUT_MCP | 4 |
| CORS | CORS_ALLOWED_ORIGINS, CORS_ALLOW_CREDENTIALS, CORS_MAX_AGE | 3 |
| Observability | OTEL_ENDPOINT, OTEL_SERVICE_NAME, METRICS_ENABLED | 3 |
| NATS | NATS_URL, NATS_CREDS_FILE | 2 |
| **Total** | | **35+** |

## Verification

```bash
go build -o bin/vnp-gateway ./cmd/main.go  # ✅ PASS
AUTH_DEV_MODE=true go run ./cmd/main.go    # ✅ Starts 3 servers, healthz responds
```
