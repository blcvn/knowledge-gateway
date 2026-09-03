---
id: TASK-001
title: "Go Module + Unified Configuration"
app: apps/graphiti
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu

Khởi tạo Go module cho `apps/graphiti` và implement unified config system cho tất cả embedded services + gateway.

## Scope

### In Scope
- `go.mod` + `go.sum` (same Go module as monorepo root hoặc replace directives)
- `internal/config/config.go` — Unified config struct
- `config.yaml` — Default config file
- `.env.example` — Environment variable reference

### Out of Scope
- Supervisor logic (TASK-002)
- Service embedding (TASK-003)
- Gateway embedding (TASK-004)

## Thiết Kế Kỹ Thuật

### Go Module Setup

```
apps/graphiti/go.mod
  module github.com/vnp-community/vnp-memory/apps/graphiti
  go 1.23

  require (
    github.com/vnp-community/vnp-memory/gateway => ../../gateway
    // Services are NOT imported (internal/ restriction)
    // Instead, we replicate startup logic
  )
```

> **Quan trọng:** Chỉ gateway module được import trực tiếp (vì gateway/internal được phép import trong same module parent). Các services KHÔNG thể import `internal/` packages do Go module boundary.

### Config Structure

```go
// internal/config/config.go
package config

import (
    "fmt"
    "os"
    "strconv"
)

type Config struct {
    // ── Graph Database ──
    Neo4jURI    string
    Neo4jUser   string
    Neo4jPass   string

    // ── Cache ──
    RedisAddr     string
    RedisPassword string
    RedisDB       int

    // ── Messaging ──
    NATSURL     string
    NATSCreds   string

    // ── LLM/AI ──
    LLMProvider    string
    LLMModel       string
    LLMSmallModel  string
    LLMAPIKey      string
    LLMBaseURL     string
    EmbedderProvider string
    EmbedderModel    string
    EmbedderAPIKey   string
    RerankerProvider string
    RerankerModel    string

    // ── Service Ports ──
    IngestionGRPCPort int
    SearchGRPCPort    int
    KnowledgeGRPCPort int
    StoreGRPCPort     int
    PipelineGRPCPort  int

    // ── Gateway ──
    GatewayRESTPort int
    GatewayMCPPort  int
    HealthPort      int

    // ── Auth ──
    AuthDevMode    bool
    AuthJWTKey     string
    AuthJWTIssuer  string
    AuthJWTAudience string

    // ── Postgres (Gateway key store) ──
    PostgresDSN string

    // ── Observability ──
    OTelEndpoint string
    LogLevel     string

    // ── Timeouts ──
    ShutdownTimeout int // seconds
}

func Load() *Config {
    return &Config{
        // Graph
        Neo4jURI:  env("NEO4J_URI", "bolt://localhost:7687"),
        Neo4jUser: env("NEO4J_USERNAME", "neo4j"),
        Neo4jPass: env("NEO4J_PASSWORD", ""),

        // Cache
        RedisAddr:     env("REDIS_ADDR", "localhost:6379"),
        RedisPassword: env("REDIS_PASSWORD", ""),
        RedisDB:       envInt("REDIS_DB", 0),

        // Messaging
        NATSURL:   env("NATS_URL", "nats://localhost:4222"),
        NATSCreds: env("NATS_CREDS_FILE", ""),

        // LLM
        LLMProvider:    env("LLM_PROVIDER", "openai"),
        LLMModel:       env("LLM_MODEL", "gpt-4o"),
        LLMSmallModel:  env("LLM_SMALL_MODEL", "gpt-4o-mini"),
        LLMAPIKey:      env("LLM_API_KEY", ""),
        LLMBaseURL:     env("LLM_BASE_URL", ""),
        EmbedderProvider: env("EMBEDDER_PROVIDER", "openai"),
        EmbedderModel:    env("EMBEDDER_MODEL", "text-embedding-3-small"),
        EmbedderAPIKey:   env("EMBEDDER_API_KEY", ""),
        RerankerProvider: env("RERANKER_PROVIDER", "openai"),
        RerankerModel:    env("RERANKER_MODEL", "gpt-4o-mini"),

        // Ports
        IngestionGRPCPort: envInt("INGESTION_GRPC_PORT", 9021),
        SearchGRPCPort:    envInt("SEARCH_GRPC_PORT", 9022),
        KnowledgeGRPCPort: envInt("KNOWLEDGE_GRPC_PORT", 9023),
        StoreGRPCPort:     envInt("STORE_GRPC_PORT", 9024),
        PipelineGRPCPort:  envInt("PIPELINE_GRPC_PORT", 9025),

        // Gateway
        GatewayRESTPort: envInt("GATEWAY_REST_PORT", 8080),
        GatewayMCPPort:  envInt("GATEWAY_MCP_PORT", 8082),
        HealthPort:      envInt("HEALTH_PORT", 9090),

        // Auth
        AuthDevMode:     envBool("AUTH_DEV_MODE", true),
        AuthJWTKey:      env("AUTH_JWT_PUBLIC_KEY", ""),
        AuthJWTIssuer:   env("AUTH_JWT_ISSUER", "vnp-memory"),
        AuthJWTAudience: env("AUTH_JWT_AUDIENCE", "vnp-api"),

        // Postgres
        PostgresDSN: env("POSTGRES_DSN", ""),

        // Observability
        OTelEndpoint: env("OTEL_ENDPOINT", ""),
        LogLevel:     env("LOG_LEVEL", "info"),

        // Timeouts
        ShutdownTimeout: envInt("SHUTDOWN_TIMEOUT", 30),
    }
}

// SetServiceEnvVars exports config as ENV vars cho các embedded services
// Services đọc config qua os.Getenv() — pattern đã có sẵn
func (c *Config) SetServiceEnvVars() {
    os.Setenv("NEO4J_URI", c.Neo4jURI)
    os.Setenv("NEO4J_USERNAME", c.Neo4jUser)
    os.Setenv("NEO4J_PASSWORD", c.Neo4jPass)
    os.Setenv("REDIS_ADDR", c.RedisAddr)
    os.Setenv("NATS_URL", c.NATSURL)
    os.Setenv("LLM_PROVIDER", c.LLMProvider)
    os.Setenv("LLM_MODEL", c.LLMModel)
    os.Setenv("LLM_SMALL_MODEL", c.LLMSmallModel)
    os.Setenv("LLM_API_KEY", c.LLMAPIKey)
    os.Setenv("EMBEDDER_PROVIDER", c.EmbedderProvider)
    os.Setenv("EMBEDDER_MODEL", c.EmbedderModel)
    os.Setenv("OTEL_ENDPOINT", c.OTelEndpoint)
    os.Setenv("LOG_LEVEL", c.LogLevel)
    // ... more as needed by services
}

// GatewayServicesMap returns service→localhost:PORT cho gateway GRPCRegistry
func (c *Config) GatewayServicesMap() map[string]string {
    return map[string]string{
        "graphiti-ingestion": fmt.Sprintf("localhost:%d", c.IngestionGRPCPort),
        "graphiti-search":    fmt.Sprintf("localhost:%d", c.SearchGRPCPort),
        "graphiti-knowledge": fmt.Sprintf("localhost:%d", c.KnowledgeGRPCPort),
        "graphiti-store":     fmt.Sprintf("localhost:%d", c.StoreGRPCPort),
        "graphiti-pipeline":  fmt.Sprintf("localhost:%d", c.PipelineGRPCPort),
    }
}

func env(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

func envInt(key string, fallback int) int {
    if v := os.Getenv(key); v != "" {
        if n, err := strconv.Atoi(v); err == nil {
            return n
        }
    }
    return fallback
}

func envBool(key string, fallback bool) bool {
    if v := os.Getenv(key); v != "" {
        if b, err := strconv.ParseBool(v); err == nil {
            return b
        }
    }
    return fallback
}
```

### Config File (config.yaml)

```yaml
# Graphiti Monolith — Unified Configuration
# All values can be overridden via ENV vars

# Graph Database
neo4j:
  uri: "bolt://neo4j:7687"
  username: "neo4j"
  password: ""

# LLM Provider
llm:
  provider: "openai"
  model: "gpt-4o"
  small_model: "gpt-4o-mini"
  api_key: ""

# Embedder
embedder:
  provider: "openai"
  model: "text-embedding-3-small"

# Infrastructure
redis:
  addr: "redis:6379"
nats:
  url: "nats://nats:4222"

# Service Ports (gRPC)
ports:
  ingestion: 9021
  search: 9022
  knowledge: 9023
  store: 9024
  pipeline: 9025
  gateway_rest: 8080
  gateway_mcp: 8082
  health: 9090

# Gateway
auth:
  dev_mode: true

# Observability
log_level: "info"
```

### .env.example

```env
# ── Graph Database ──
NEO4J_URI=bolt://localhost:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your-password

# ── LLM / AI ──
LLM_PROVIDER=openai
LLM_MODEL=gpt-4o
LLM_SMALL_MODEL=gpt-4o-mini
LLM_API_KEY=sk-...
EMBEDDER_MODEL=text-embedding-3-small

# ── Infrastructure ──
REDIS_ADDR=localhost:6379
NATS_URL=nats://localhost:4222

# ── Ports ──
GATEWAY_REST_PORT=8080
GATEWAY_MCP_PORT=8082
HEALTH_PORT=9090

# ── Auth ──
AUTH_DEV_MODE=true

# ── Observability ──
LOG_LEVEL=info
OTEL_ENDPOINT=
```

## Acceptance Criteria

- [x] AC-1: `go.mod` valid, `go mod tidy` thành công
- [x] AC-2: `config.Load()` trả về Config với tất cả default values
- [x] AC-3: `config.SetServiceEnvVars()` đặt đúng ENV vars cho embedded services
- [x] AC-4: `config.GatewayServicesMap()` trả về đúng map service→localhost:PORT
- [x] AC-5: `config.yaml` chứa tất cả configurable values
- [x] AC-6: `.env.example` chứa tất cả ENV vars reference
- [x] AC-7: Config override: ENV vars override default values

## Definition of Done

- [x] `go mod tidy` pass
- [x] Unit tests cho config loading + ENV override (5 test cases pass)
- [x] Không có lint errors ✅
