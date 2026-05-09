# 07 — Admin Service (zep-admin)

> **gRPC**: 9046 | **Health**: 9146  
> **Origin**: Cross-cutting — Health Aggregation, Project/Tenant Management

---

## 1. Purpose

Shared admin service quản lý cross-cutting concerns:
- **Health Aggregation**: Aggregate health status from all 6 services
- **Project Management**: CRUD cho projects (multi-tenant isolation unit)
- **API Key Management**: Generate, validate, revoke API keys
- **Telemetry Config**: Enable/disable anonymous usage telemetry
- **Schema Management**: Trigger migration across services

---

## 2. Clean Architecture Layout

```
services/zep-admin/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── project.go             # Project entity (tenant isolation unit)
│   │   ├── api_key.go             # APIKey entity
│   │   ├── health.go              # ServiceHealth, AggregatedHealth
│   │   ├── telemetry_config.go    # TelemetryConfig
│   │   ├── event.go               # ProjectCreated, ProjectDeleted
│   │   └── errors.go
│   │
│   ├── usecase/
│   │   ├── create_project.go      # Create project + init schemas
│   │   ├── list_projects.go
│   │   ├── delete_project.go      # Delete project + cascade
│   │   ├── create_api_key.go      # Generate API key for project
│   │   ├── validate_api_key.go    # Validate + resolve project
│   │   ├── revoke_api_key.go
│   │   ├── aggregated_health.go   # Check health of all services
│   │   ├── migrate_schema.go      # Trigger schema migrations
│   │   ├── port/
│   │   │   ├── input.go
│   │   │   └── output.go          # ProjectRepository, APIKeyRepository, ServiceHealthChecker
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   │
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── project_repo.go
│   │   │       ├── api_key_repo.go
│   │   │       └── model.go
│   │   ├── client/
│   │   │   └── health_checker.go   # gRPC Health v1 client to all services
│   │   └── event/
│   │       └── publisher.go
│   │
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 3. Domain Layer

### 3.1 Project Entity

```go
package domain

type Project struct {
    UUID            string
    Name            string
    Description     string
    OrganizationID  *string
    Settings        ProjectSettings
    CreatedAt       time.Time
    UpdatedAt       time.Time
    DeletedAt       *time.Time
}

type ProjectSettings struct {
    MaxRequestSize  int64          // bytes, default 5MB
    RequestTimeout  time.Duration  // default 30s
    RateLimitRPS    int            // default 100
    TelemetryEnabled bool         // default true
    GraphitiEnabled  bool         // default true
}
```

### 3.2 API Key Entity

```go
type APIKey struct {
    UUID        string
    KeyHash     string         // SHA-256 hash of the key
    KeyPrefix   string         // first 8 chars for identification
    ProjectUUID string         // FK → Project
    Name        string
    Scopes      []string       // "read", "write", "admin"
    ExpiresAt   *time.Time
    LastUsedAt  *time.Time
    CreatedAt   time.Time
    RevokedAt   *time.Time
}

func (k *APIKey) IsValid() bool {
    if k.RevokedAt != nil {
        return false
    }
    if k.ExpiresAt != nil && time.Now().After(*k.ExpiresAt) {
        return false
    }
    return true
}
```

### 3.3 Aggregated Health

```go
type AggregatedHealth struct {
    Status     HealthStatus           // SERVING | NOT_SERVING | DEGRADED
    Services   map[string]ServiceHealth
    Timestamp  time.Time
}

type ServiceHealth struct {
    Name       string
    Status     HealthStatus
    LatencyMs  int64
    Details    map[string]string
}

type HealthStatus string

const (
    HealthServing    HealthStatus = "SERVING"
    HealthNotServing HealthStatus = "NOT_SERVING"
    HealthDegraded   HealthStatus = "DEGRADED"
)
```

---

## 4. Use Case Layer

### 4.1 AggregatedHealth Use Case

```go
func (uc *AggregatedHealthUseCase) Execute(ctx context.Context) (*dto.AggregatedHealthResponse, error) {
    services := []struct {
        Name string
        Addr string
    }{
        {"zep-user", uc.cfg.Backends.User},
        {"zep-thread", uc.cfg.Backends.Thread},
        {"zep-memory", uc.cfg.Backends.Memory},
        {"zep-graph", uc.cfg.Backends.Graph},
        {"zep-search", uc.cfg.Backends.Search},
    }
    
    // Parallel health checks with timeout
    results := make(map[string]domain.ServiceHealth)
    var wg sync.WaitGroup
    var mu sync.Mutex
    
    for _, svc := range services {
        wg.Add(1)
        go func(name, addr string) {
            defer wg.Done()
            start := time.Now()
            status, err := uc.healthChecker.Check(ctx, addr)
            health := domain.ServiceHealth{
                Name:      name,
                Status:    status,
                LatencyMs: time.Since(start).Milliseconds(),
            }
            if err != nil {
                health.Details = map[string]string{"error": err.Error()}
            }
            mu.Lock()
            results[name] = health
            mu.Unlock()
        }(svc.Name, svc.Addr)
    }
    wg.Wait()
    
    // Determine aggregate status
    overallStatus := domain.HealthServing
    for _, h := range results {
        if h.Status == domain.HealthNotServing {
            overallStatus = domain.HealthDegraded
            break
        }
    }
    
    return dto.FromAggregatedHealth(&domain.AggregatedHealth{
        Status:    overallStatus,
        Services:  results,
        Timestamp: time.Now(),
    }), nil
}
```

---

## 5. gRPC Service Definition

```protobuf
syntax = "proto3";
package zep.admin.v1;

service AdminService {
  // Health
  rpc AggregatedHealth(google.protobuf.Empty) returns (AggregatedHealthResponse);
  
  // Project management
  rpc CreateProject(CreateProjectRequest) returns (ProjectResponse);
  rpc GetProject(GetProjectRequest) returns (ProjectResponse);
  rpc ListProjects(ListProjectsRequest) returns (ProjectListResponse);
  rpc UpdateProject(UpdateProjectRequest) returns (ProjectResponse);
  rpc DeleteProject(DeleteProjectRequest) returns (google.protobuf.Empty);
  
  // API Key management
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (ValidateAPIKeyResponse);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (APIKeyListResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  
  // Schema management
  rpc MigrateSchema(MigrateSchemaRequest) returns (MigrateSchemaResponse);
}

message AggregatedHealthResponse {
  string status = 1;          // "SERVING" | "NOT_SERVING" | "DEGRADED"
  map<string, ServiceHealth> services = 2;
  google.protobuf.Timestamp timestamp = 3;
}

message ServiceHealth {
  string name = 1;
  string status = 2;
  int64 latency_ms = 3;
  map<string, string> details = 4;
}

message ProjectResponse {
  string uuid = 1;
  string name = 2;
  string description = 3;
  optional string organization_id = 4;
  ProjectSettings settings = 5;
  google.protobuf.Timestamp created_at = 6;
}

message ProjectSettings {
  int64 max_request_size = 1;
  int32 request_timeout_ms = 2;
  int32 rate_limit_rps = 3;
  bool telemetry_enabled = 4;
  bool graphiti_enabled = 5;
}

message CreateAPIKeyResponse {
  string key = 1;              // raw key (shown only once)
  string key_prefix = 2;      // for identification
  string project_uuid = 3;
  repeated string scopes = 4;
}
```

---

## 6. PostgreSQL Schema

```sql
CREATE TABLE projects (
    uuid            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL UNIQUE,
    description     TEXT,
    organization_id TEXT,
    settings        JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE TABLE api_keys (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_hash     TEXT NOT NULL UNIQUE,      -- SHA-256 of raw key
    key_prefix   TEXT NOT NULL,             -- first 8 chars
    project_uuid UUID NOT NULL REFERENCES projects(uuid),
    name         TEXT NOT NULL,
    scopes       TEXT[] DEFAULT '{"read","write"}',
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ
);

CREATE INDEX api_key_hash_idx ON api_keys(key_hash) WHERE revoked_at IS NULL;
CREATE INDEX api_key_project_idx ON api_keys(project_uuid) WHERE revoked_at IS NULL;
```

---

## 7. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `zep.admin.project.created` | `{project_uuid, name, settings}` | All services (init per-project schema/cache) |
| `zep.admin.project.deleted` | `{project_uuid}` | All services (cascade delete all project data) |

---

## 8. Configuration

```yaml
admin:
  grpc:
    port: 9046
  health:
    port: 9146
  postgres:
    dsn: "postgres://postgres:postgres@db:5432/zep?sslmode=disable"
    max_open_connections: 5
  backends:
    user: "zep-user:9041"
    thread: "zep-thread:9042"
    memory: "zep-memory:9043"
    graph: "zep-graph:9044"
    search: "zep-search:9045"
  health_check:
    timeout: 5s
    interval: 30s
  api_key:
    key_length: 32              # bytes
    default_scopes: ["read", "write"]
  nats:
    url: "nats://nats:4222"
    stream: "zep"
  telemetry:
    service_name: "zep-admin"
    otel_endpoint: "otel-collector:4317"
```
