---
id: TASK-001
title: "Go Module + Unified Config"
app: apps/memobase
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
---

## Mục Tiêu

Thiết lập Go module cho `apps/memobase` và tạo unified config struct quản lý tất cả thông số cấu hình cho 4 memobase services + gateway trong 1 file duy nhất.

## Scope

### In Scope (AI phải implement)
- `apps/memobase/go.mod` — Go module definition (replace directives cho local packages)
- `apps/memobase/internal/config/config.go` — Unified config struct + loader
- `apps/memobase/config.yaml` — Default YAML config file
- `apps/memobase/.env.example` — Environment variable template

### Out of Scope
- Service startup logic (TASK-003)
- Gateway startup logic (TASK-004)
- Supervisor implementation (TASK-002)

## Thiết Kế Kỹ Thuật

### Config Structure (từ reference specs)

```go
type Config struct {
    // Server
    Server ServerConfig `yaml:"server"`

    // Service Ports
    Services ServicesConfig `yaml:"services"`

    // Shared Infrastructure
    Database DatabaseConfig `yaml:"database"`
    Redis    RedisConfig    `yaml:"redis"`
    NATS     NATSConfig     `yaml:"nats"`

    // LLM / Embedding
    LLM       LLMConfig       `yaml:"llm"`
    Embedding EmbeddingConfig `yaml:"embedding"`

    // Auth
    Auth AuthConfig `yaml:"auth"`

    // Gateway-specific
    Postgres PostgresConfig `yaml:"postgres"`
}

type ServerConfig struct {
    RESTPort        int           `yaml:"rest_port" env:"REST_PORT" default:"8080"`
    MCPPort         int           `yaml:"mcp_port" env:"MCP_PORT" default:"8082"`
    HealthPort      int           `yaml:"health_port" env:"HEALTH_PORT" default:"9090"`
    LogLevel        string        `yaml:"log_level" env:"LOG_LEVEL" default:"info"`
    ShutdownTimeout time.Duration `yaml:"shutdown_timeout" default:"30s"`
}

type ServicesConfig struct {
    IngestionPort int `yaml:"ingestion_port" env:"INGESTION_GRPC_PORT" default:"9041"`
    EnginePort    int `yaml:"engine_port" env:"ENGINE_GRPC_PORT" default:"9042"`
    ContextPort   int `yaml:"context_port" env:"CONTEXT_GRPC_PORT" default:"9043"`
    PipelinePort  int `yaml:"pipeline_port" env:"PIPELINE_GRPC_PORT" default:"9044"`
}
```

### Config Loading Order
1. YAML file (`config.yaml`)
2. ENV overrides
3. Defaults

### SetServiceEnvVars()
Mỗi embedded service đọc config qua `os.Getenv()`.
Config struct phải export tất cả settings thành ENV vars trước khi start services.

### Validate()
- Tất cả ports phải duy nhất
- Database URL required
- NATS URL required (memobase uses NATS for buffer pipeline)

### go.mod Pattern
```
module github.com/vnp-community/vnp-memory/apps/memobase

go 1.23

require (
    github.com/vnp-community/vnp-memory/gateway v0.0.0
    github.com/vnp-community/vnp-memory/services/memobase-ingestion v0.0.0
    github.com/vnp-community/vnp-memory/services/memobase-engine v0.0.0
    github.com/vnp-community/vnp-memory/services/memobase-context v0.0.0
    github.com/vnp-community/vnp-memory/services/memobase-pipeline v0.0.0
)

replace (
    github.com/vnp-community/vnp-memory/gateway => ../../gateway
    github.com/vnp-community/vnp-memory/services/memobase-ingestion => ../../services/memobase-ingestion
    github.com/vnp-community/vnp-memory/services/memobase-engine => ../../services/memobase-engine
    github.com/vnp-community/vnp-memory/services/memobase-context => ../../services/memobase-context
    github.com/vnp-community/vnp-memory/services/memobase-pipeline => ../../services/memobase-pipeline
)
```

## Acceptance Criteria

- [x] AC-1: `go.mod` created with correct module path and replace directives
- [x] AC-2: `config.go` loads YAML + ENV overrides with proper defaults
- [x] AC-3: `Validate()` rejects duplicate ports and missing required fields
- [x] AC-4: `SetServiceEnvVars()` exports all config as ENV vars
- [x] AC-5: `config.yaml` has sensible defaults matching reference specs
- [x] AC-6: `.env.example` documents all configurable variables
- [x] AC-7: Unit tests for config loading and validation (≥ 80% coverage)

## Test Requirements
- Unit tests: config loading, validation logic, ENV override
- Minimum coverage: 80%

## Definition of Done
- [x] Code implement đủ Acceptance Criteria
- [x] Unit tests pass, coverage ≥ 80%
- [x] `go vet` pass
- [x] Không có lint errors
