# Solution: SOL-ZEP-008 — Admin Service & Multi-Tenant Project Management

**CR ID:** CR-ZEP-008  
**Solution ID:** SOL-ZEP-008  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `services/zep-admin/` (port gRPC 9046) với 4 responsibilities: Health Aggregation, Project/Tenant Management, API Key Management (format `vnp_` + base62), và Telemetry Control. Service này fanout gRPC health checks tới tất cả 6 downstream services và publish NATS cascade events khi project bị tạo/xóa.

---

## 2. Phân tích Kiến trúc Hiện tại

### Điểm bắt đầu

| Thành phần hiện có | Vị trí | Trạng thái |
|--------------------|--------|------------|
| `admin-service` domain | `services/vnp-platform/internal/domain/admin/` | Có: Tenant, User, APIKey — khác format với Zep |
| `GET /healthz` | `gateway/` | Có: single-service health, chưa aggregate |
| `project_uuid` fields | Tất cả domains | Có: đã dùng nhưng chưa có Project CRUD |
| NATS events | Infrastructure | Có: `zep.admin.*` events defined |

### Gap phân tích

- Thiếu aggregated health check (fanout tới 6 services)
- Không có Project CRUD API (chỉ có Tenant trong vnp-platform)
- API key format `vnp_` + base62 chưa implement
- Không có NATS cascade events cho project lifecycle
- Thiếu Telemetry toggle per-project

---

## 3. Thiết kế Giải pháp

### 3.1. Cấu trúc Service

```
services/zep-admin/
├── internal/
│   ├── domain/
│   │   ├── project.go     # Project entity
│   │   ├── apikey.go      # APIKey với vnp_ prefix format
│   │   └── repository.go  # ProjectRepository, APIKeyRepository
│   ├── usecase/
│   │   ├── health/
│   │   │   └── aggregate_health.go    # Fan-out gRPC health checks
│   │   ├── project/
│   │   │   ├── create_project.go
│   │   │   ├── get_project.go
│   │   │   ├── list_projects.go
│   │   │   └── delete_project.go      # Cascade via NATS
│   │   ├── apikey/
│   │   │   ├── create_api_key.go      # vnp_ + base62
│   │   │   ├── list_api_keys.go
│   │   │   └── revoke_api_key.go      # Immediate invalidation
│   │   └── telemetry/
│   │       └── update_telemetry.go
│   ├── adapter/
│   │   ├── grpc/
│   │   │   └── admin_server.go
│   │   └── subscriber/
│   │       └── cascade_handler.go     # Handle cascade delete
│   └── infra/
│       ├── postgres/
│       │   ├── project_repo.go
│       │   └── apikey_repo.go
│       └── health/
│           └── health_checker.go      # Fan-out gRPC health checks
```

### 3.2. Domain Model

```go
// services/zep-admin/internal/domain/project.go

package domain

import "time"

type Project struct {
    UUID      string
    Name      string
    APIKeys   []APIKey
    Settings  ProjectSettings
    CreatedAt time.Time
    DeletedAt *time.Time  // soft delete
}

type ProjectSettings struct {
    MaxRequestSizeMB int   `json:"max_request_size_mb"` // default: 5
    TimeoutSeconds   int   `json:"timeout_seconds"`      // default: 30
    TelemetryEnabled bool  `json:"telemetry_enabled"`    // anonymous usage tracking
}

// services/zep-admin/internal/domain/apikey.go

type APIKey struct {
    UUID        string
    ProjectUUID string
    Name        string        // human-friendly label (e.g. "Production Key")
    Hash        string        // SHA-256 of plaintext key (stored, never plaintext)
    Prefix      string        // first 8 chars for identification (e.g. "vnp_a7Kj")
    CreatedAt   time.Time
    LastUsedAt  *time.Time
    RevokedAt   *time.Time    // nil = active; non-nil = revoked
}

// GenerateAPIKey creates a new API key in format: "vnp_" + base62(32 random bytes)
// Returns (plaintext, sha256hash)
// Plaintext is shown ONCE and never stored
func GenerateAPIKey() (plaintext, hash, prefix string) {
    raw := make([]byte, 32)
    rand.Read(raw)

    // Base62 encode (A-Za-z0-9, 43 chars for 32 bytes)
    encoded := base62Encode(raw)
    plaintext = "vnp_" + encoded

    h := sha256.Sum256([]byte(plaintext))
    hash = fmt.Sprintf("%x", h)
    prefix = plaintext[:8]  // "vnp_" + 4 chars = 8 chars for identification
    return
}

func base62Encode(data []byte) string {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
    n := new(big.Int).SetBytes(data)
    base := big.NewInt(62)
    var result []byte
    for n.Sign() > 0 {
        mod := new(big.Int)
        n.DivMod(n, base, mod)
        result = append(result, chars[mod.Int64()])
    }
    // Pad + reverse
    for len(result) < 43 { result = append(result, chars[0]) }
    for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
        result[i], result[j] = result[j], result[i]
    }
    return string(result)
}
```

### 3.3. Health Aggregation (Fan-out)

```go
// services/zep-admin/internal/infra/health/health_checker.go

package health

import (
    "context"
    "sync"
    "time"
    "google.golang.org/grpc/health/grpc_health_v1"
)

type ServiceHealth struct {
    Status  string  // "up" | "down" | "degraded"
    Latency int64   // response time in ms
    Error   string  // error message if down
}

type AggregatedHealth struct {
    Status   string                     // "healthy" | "degraded" | "unhealthy"
    Services map[string]ServiceHealth
    Uptime   time.Duration
    CheckedAt time.Time
}

// HealthChecker fans out gRPC health checks to all downstream services
type HealthChecker struct {
    services map[string]grpc_health_v1.HealthClient
    startTime time.Time
}

func NewHealthChecker(userConn, threadConn, memoryConn, graphConn, searchConn, adminConn *grpc.ClientConn) *HealthChecker {
    return &HealthChecker{
        services: map[string]grpc_health_v1.HealthClient{
            "zep-user":   grpc_health_v1.NewHealthClient(userConn),
            "zep-thread": grpc_health_v1.NewHealthClient(threadConn),
            "zep-memory": grpc_health_v1.NewHealthClient(memoryConn),
            "zep-graph":  grpc_health_v1.NewHealthClient(graphConn),
            "zep-search": grpc_health_v1.NewHealthClient(searchConn),
        },
        startTime: time.Now(),
    }
}

// CheckAll fans out health checks concurrently to all services
func (hc *HealthChecker) CheckAll(ctx context.Context) *AggregatedHealth {
    timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    type result struct {
        name   string
        health ServiceHealth
    }

    results := make(chan result, len(hc.services))
    var wg sync.WaitGroup

    for name, client := range hc.services {
        wg.Add(1)
        go func(name string, client grpc_health_v1.HealthClient) {
            defer wg.Done()
            start := time.Now()

            resp, err := client.Check(timeout, &grpc_health_v1.HealthCheckRequest{})
            latency := time.Since(start).Milliseconds()

            if err != nil {
                results <- result{name, ServiceHealth{Status: "down", Latency: latency, Error: err.Error()}}
                return
            }

            status := "up"
            if resp.Status != grpc_health_v1.HealthCheckResponse_SERVING {
                status = "degraded"
            }
            results <- result{name, ServiceHealth{Status: status, Latency: latency}}
        }(name, client)
    }

    go func() {
        wg.Wait()
        close(results)
    }()

    services := make(map[string]ServiceHealth)
    downCount := 0
    for r := range results {
        services[r.name] = r.health
        if r.health.Status == "down" { downCount++ }
    }

    // Overall status: healthy / degraded / unhealthy
    overallStatus := "healthy"
    if downCount > 0 && downCount < len(hc.services) { overallStatus = "degraded" }
    if downCount == len(hc.services) { overallStatus = "unhealthy" }

    return &AggregatedHealth{
        Status:    overallStatus,
        Services:  services,
        Uptime:    time.Since(hc.startTime),
        CheckedAt: time.Now(),
    }
}
```

### 3.4. Project Lifecycle (NATS Cascade)

```go
// services/zep-admin/internal/usecase/project/create_project.go

func (uc *CreateProjectUseCase) Execute(ctx context.Context, req CreateProjectRequest) (*Project, error) {
    project := &Project{
        UUID: newUUID(),
        Name: req.Name,
        Settings: ProjectSettings{
            MaxRequestSizeMB: 5,
            TimeoutSeconds:   30,
            TelemetryEnabled: true,
        },
    }

    if err := uc.repo.Create(ctx, project); err != nil {
        return nil, err
    }

    // Publish NATS event → all services init tenant schema
    uc.publisher.Publish(ctx, "zep.admin.project.created", ProjectCreatedEvent{
        ProjectUUID: project.UUID,
        Name:        project.Name,
    })

    return project, nil
}

// services/zep-admin/internal/usecase/project/delete_project.go

func (uc *DeleteProjectUseCase) Execute(ctx context.Context, projectUUID string) error {
    // Soft delete project
    if err := uc.repo.SoftDelete(ctx, projectUUID); err != nil {
        return err
    }

    // Publish NATS cascade event → all services delete tenant data
    // Services: thread, memory, graph, search, user receive and clean up
    uc.publisher.Publish(ctx, "zep.admin.project.deleted", ProjectDeletedEvent{
        ProjectUUID: projectUUID,
        DeletedAt:   time.Now(),
    })

    return nil
}
```

### 3.5. API Key Management (Immediate Revocation)

```go
// services/zep-admin/internal/usecase/apikey/create_api_key.go

func (uc *CreateAPIKeyUseCase) Execute(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
    // Generate: "vnp_" + base62(32 bytes)
    plaintext, hash, prefix := GenerateAPIKey()

    apiKey := &APIKey{
        UUID:        newUUID(),
        ProjectUUID: req.ProjectUUID,
        Name:        req.Name,
        Hash:        hash,
        Prefix:      prefix,
        CreatedAt:   time.Now(),
    }

    if err := uc.repo.Create(ctx, apiKey); err != nil {
        return nil, err
    }

    return &CreateAPIKeyResponse{
        UUID:      apiKey.UUID,
        Plaintext: plaintext,  // returned ONCE, never stored
        Prefix:    prefix,     // for future identification
        Name:      apiKey.Name,
    }, nil
}

// services/zep-admin/internal/usecase/apikey/revoke_api_key.go

func (uc *RevokeAPIKeyUseCase) Execute(ctx context.Context, apiKeyUUID, projectUUID string) error {
    now := time.Now()
    if err := uc.repo.Revoke(ctx, apiKeyUUID, projectUUID, now); err != nil {
        return err
    }

    // Bust Redis cache for this key's hash → immediate invalidation
    // (Auth service caches key validation for 5 min)
    uc.cacheInvalidator.InvalidateAPIKey(ctx, apiKeyUUID)

    return nil
}
```

---

## 4. Database Schema

```sql
-- projects table
CREATE TABLE zep_projects (
    uuid         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR NOT NULL,
    settings     JSONB NOT NULL DEFAULT '{"max_request_size_mb":5,"timeout_seconds":30,"telemetry_enabled":true}',
    created_at   TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    deleted_at   TIMESTAMPTZ     -- soft delete
);

-- api_keys table (Zep format: vnp_ prefix)
CREATE TABLE zep_api_keys (
    uuid          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_uuid  UUID NOT NULL REFERENCES zep_projects(uuid),
    name          VARCHAR NOT NULL,
    hash          VARCHAR(64) NOT NULL UNIQUE,    -- SHA-256 hex
    prefix        VARCHAR(8) NOT NULL,            -- first 8 chars for identification
    created_at    TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    last_used_at  TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ                     -- nil = active
);

-- Indexes
CREATE INDEX zep_api_keys_project_idx ON zep_api_keys(project_uuid);
CREATE INDEX zep_api_keys_hash_idx ON zep_api_keys(hash) WHERE revoked_at IS NULL;
```

---

## 5. API Endpoints

```go
// gateway/adapter/handler/zep_admin_handler.go

func (h *ZepAdminHandler) Register(mux *http.ServeMux) {
    // Health (public)
    mux.HandleFunc("GET /healthz",                       h.HealthCheck)

    // Projects (admin auth required)
    mux.HandleFunc("GET /api/v2/admin/projects",         h.ListProjects)
    mux.HandleFunc("POST /api/v2/admin/projects",        h.CreateProject)
    mux.HandleFunc("GET /api/v2/admin/projects/{id}",    h.GetProject)
    mux.HandleFunc("DELETE /api/v2/admin/projects/{id}", h.DeleteProject)

    // API Keys
    mux.HandleFunc("POST /api/v2/admin/api-keys",        h.CreateAPIKey)
    mux.HandleFunc("GET /api/v2/admin/api-keys",         h.ListAPIKeys)
    mux.HandleFunc("DELETE /api/v2/admin/api-keys/{id}", h.RevokeAPIKey)
}
```

**GET /healthz response:**
```json
{
  "status": "healthy",
  "uptime": "72h15m30s",
  "checked_at": "2026-06-17T05:00:00Z",
  "services": {
    "zep-user":   {"status": "up", "latency": 2},
    "zep-thread": {"status": "up", "latency": 1},
    "zep-memory": {"status": "up", "latency": 3},
    "zep-graph":  {"status": "degraded", "latency": 150, "error": "slow response"},
    "zep-search": {"status": "up", "latency": 4}
  }
}
```

**POST /api/v2/admin/api-keys response (key shown once):**
```json
{
  "uuid": "key_abc123",
  "plaintext": "vnp_a7Kj3mN2pQ9xR4vW8yB5cD0eF6gH1iL3oP",
  "prefix": "vnp_a7Kj",
  "name": "Production Key",
  "created_at": "2026-06-17T05:00:00Z"
}
```

---

## 6. NATS Event Contract

```go
// Event: project created → services init tenant schema
type ProjectCreatedEvent struct {
    ProjectUUID string    `json:"project_uuid"`
    Name        string    `json:"name"`
    Settings    any       `json:"settings"`
    CreatedAt   time.Time `json:"created_at"`
}

// Event: project deleted → cascade delete all tenant data
type ProjectDeletedEvent struct {
    ProjectUUID string    `json:"project_uuid"`
    DeletedAt   time.Time `json:"deleted_at"`
}

// Subscriber in each service (thread, memory, graph, search, user):
func (s *AdminEventSubscriber) Start(ctx context.Context) {
    s.nats.Subscribe(ctx, "zep.admin.project.created", func(e ProjectCreatedEvent) {
        // Init tenant-specific resources (e.g. Neo4j label, schema partitions)
    })
    s.nats.Subscribe(ctx, "zep.admin.project.deleted", func(e ProjectDeletedEvent) {
        // Cascade delete all data for project_uuid
        s.deleteAllTenantData(ctx, e.ProjectUUID)
    })
}
```

---

## 7. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Domain model + DB schema | 1 ngày |
| **P2** | APIKey generation (vnp_ + base62) | 0.5 ngày |
| **P3** | Health aggregation (fan-out gRPC checks) | 1.5 ngày |
| **P4** | Project CRUD use cases | 1 ngày |
| **P5** | NATS cascade events (created + deleted) | 1 ngày |
| **P6** | APIKey management (create + list + revoke) | 1 ngày |
| **P7** | Cache invalidation on revoke | 0.5 ngày |
| **P8** | Gateway handlers + tests | 1.5 ngày |

**Tổng:** ~8 ngày (Wave 2)

---

## 8. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| GET /healthz → status từ tất cả services | Fan-out gRPC health checks + aggregate |
| Tạo project → NATS event → services nhận | ProjectCreatedEvent → subscribers |
| Delete project → cascade delete tất cả data | ProjectDeletedEvent → deleteAllTenantData() |
| Tạo API key → plaintext vnp_xxx 1 lần | GenerateAPIKey() trả về plaintext, store hash only |
| Revoke key → 401 ngay | Cache invalidation + revoked_at IS NOT NULL check |
