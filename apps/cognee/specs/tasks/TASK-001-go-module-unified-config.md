---
id: TASK-001
title: "Go Module + Unified Config"
app: apps/cognee
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: []
estimated: 1.5h
---

## Mục Tiêu

Setup Go module cho cognee app với đúng dependencies, và tạo unified config struct quản lý tất cả ENV vars cho embedded services + gateway.

## Scope

### In Scope
- `go.mod` — module path, dependencies (grpc, nats, telemetry packages từ monorepo)
- `internal/config/config.go` — Unified config struct, ENV loading, validation
- `config.yaml` — Example config cho local dev
- Cấu trúc thư mục skeleton

### Out of Scope
- Service startup logic (TASK-003)
- Gateway embedding (TASK-004)

## Thiết Kế Kỹ Thuật

### go.mod

```go
module vnp-memory/apps/cognee

go 1.23.0

require (
    google.golang.org/grpc v1.65.0
    google.golang.org/protobuf v1.34.2
    github.com/nats-io/nats.go v1.36.0
    vnp-memory/pkg/telemetry v0.0.0
    vnp-memory/pkg/tenant v0.0.0
)

replace (
    vnp-memory/pkg/telemetry => ../../pkg/telemetry
    vnp-memory/pkg/tenant => ../../pkg/tenant
)
```

> [!NOTE]
> Module KHÔNG import `services/*/internal/` (Go rejects cross-module internal imports).
> Thay vào đó, app replicate init logic từ services.

### Config

```go
// internal/config/config.go
package config

type Config struct {
    // App identity
    AppName     string `env:"APP_NAME" default:"cognee-app"`
    Environment string `env:"APP_ENV" default:"development"`
    LogLevel    string `env:"LOG_LEVEL" default:"info"`

    // Shared infrastructure URLs
    DatabaseURL   string `env:"DATABASE_URL" required:"true"`
    NATSURL       string `env:"NATS_URL" required:"true"`
    Neo4jURI      string `env:"NEO4J_URI" required:"true"`
    Neo4jUsername  string `env:"NEO4J_USERNAME" default:"neo4j"`
    Neo4jPassword  string `env:"NEO4J_PASSWORD"`
    QdrantURL     string `env:"QDRANT_URL" required:"true"`
    RedisAddr     string `env:"REDIS_ADDR"`
    MinIOEndpoint string `env:"MINIO_ENDPOINT"`
    MinIOAccessKey string `env:"MINIO_ACCESS_KEY"`
    MinIOSecretKey string `env:"MINIO_SECRET_KEY"`
    MinIOBucket   string `env:"MINIO_BUCKET" default:"cognee-ingestion"`

    // LLM
    LLMGatewayURL string `env:"LLM_GATEWAY_URL"`

    // Service ports (embedded gRPC servers)
    IngestionPort int `env:"INGESTION_GRPC_PORT" default:"9011"`
    CognifyPort   int `env:"COGNIFY_GRPC_PORT" default:"9012"`
    SearchPort    int `env:"SEARCH_GRPC_PORT" default:"9013"`

    // Gateway ports
    GatewayRESTPort int `env:"GATEWAY_REST_PORT" default:"8080"`
    GatewayMCPPort  int `env:"GATEWAY_MCP_PORT" default:"8082"`
    HealthPort      int `env:"HEALTH_PORT" default:"9090"`

    // Auth
    AuthDevMode    bool   `env:"AUTH_DEV_MODE" default:"true"`
    JWTPublicKey   string `env:"AUTH_JWT_PUBLIC_KEY"`

    // OTel
    OTelEndpoint string `env:"OTEL_ENDPOINT"`
}

func Load() (*Config, error) // ENV-first loading (os.Getenv + defaults)
func (c *Config) Validate() error // Check required fields

// SetServiceEnvVars injects config as ENV vars
// so embedded services read via their own os.Getenv() patterns
func (c *Config) SetServiceEnvVars() {
    os.Setenv("DATABASE_URL", c.DatabaseURL)
    os.Setenv("NATS_URL", c.NATSURL)
    os.Setenv("NEO4J_URI", c.Neo4jURI)
    os.Setenv("QDRANT_URL", c.QdrantURL)
    os.Setenv("MINIO_ENDPOINT", c.MinIOEndpoint)
    os.Setenv("MINIO_ACCESS_KEY", c.MinIOAccessKey)
    os.Setenv("MINIO_SECRET_KEY", c.MinIOSecretKey)
    // Port overrides per service
    os.Setenv("GRPC_PORT", strconv.Itoa(c.IngestionPort)) // each service reads own port
    os.Setenv("LOG_LEVEL", c.LogLevel)
    // ...
}

// GatewayServicesMap builds the services→localhost:PORT map
// matching gateway/internal/infra/config.defaultServiceAddresses()
func (c *Config) GatewayServicesMap() map[string]string {
    return map[string]string{
        "cognee-ingestion": fmt.Sprintf("localhost:%d", c.IngestionPort),
        "cognee-cognify":   fmt.Sprintf("localhost:%d", c.CognifyPort),
        "cognee-search":    fmt.Sprintf("localhost:%d", c.SearchPort),
    }
}
```

## Acceptance Criteria

- [x] AC-1: `go mod tidy` thành công
- [x] AC-2: Config loads from ENV vars with sensible defaults
- [x] AC-3: Validation catches missing required fields (DATABASE_URL, NATS_URL, etc.)
- [x] AC-4: `SetServiceEnvVars()` sets all vars needed by embedded services
- [x] AC-5: `GatewayServicesMap()` returns correct localhost:PORT map
- [x] AC-6: `config.yaml` example file present

## Definition of Done

- [x] go.mod + go.sum valid
- [x] Config struct + Load() + Validate() + SetServiceEnvVars()
- [x] Directory skeleton created
