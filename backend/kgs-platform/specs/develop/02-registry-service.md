# registry-service — App Registry Service (NEW)

> **Strategy:** 🆕 NEW  
> **Source:** `kgs-platform/internal/biz/registry.go` + `registry_usecase.go`  
> **Target:** `kgs-platform/cmd/registry/`  
> **Priority:** P0 — Cần có trước kgs-gateway

---

## 1. Lý Do Tạo Mới

`registry-service` là service **hoàn toàn mới** vì:

1. **Không tồn tại trong monolith hiện tại** — `kgs-platform` không có App/API Key management. Apps giao tiếp trực tiếp bằng `tenant_id` từ JWT.
2. **Gateway cần nguồn xác thực** — Không thể tiếp tục dùng JWT để auth khi deploy nhiều services.
3. **Quota management** — Mỗi app cần có giới hạn riêng thay vì flat rate_tier.
4. **Audit trail** — Cần audit log tập trung per app.

### 1.1 Code Có Thể Tái Sử Dụng

Từ `kgs-platform/internal/biz/`:

| File | Tái Sử Dụng | Ghi Chú |
|------|-----------|---------|
| `registry.go` | ✅ Model definitions | App, APIKey structs |
| `registry_usecase.go` | ✅ Business logic | ValidateAPIKey, ListApps logic |

```go
// kgs-platform/internal/biz/registry.go — Existing code
type App struct {
    AppID    string
    AppName  string
    Status   string // ACTIVE | SUSPENDED | DELETED
    ...
}

// kgs-platform/internal/biz/registry_usecase.go — Existing code
type RegistryUsecase struct {
    repo RegistryRepo
    log  *log.Helper
}
```

---

## 2. Cấu Trúc Service Mới

```
kgs-platform/
├── cmd/
│   └── registry/
│       └── main.go              ← Entry point
├── internal/
│   └── registry/                ← NEW: registry service package
│       ├── biz/
│       │   ├── app.go           ← App lifecycle (từ biz/registry_usecase.go)
│       │   ├── apikey.go        ← API Key management
│       │   ├── quota.go         ← Quota management
│       │   └── audit.go         ← Audit log writer
│       ├── data/
│       │   ├── app_pg.go        ← PostgreSQL repos
│       │   ├── apikey_pg.go
│       │   └── models.go        ← App, APIKey, Quota, AuditLog models
│       └── server/
│           └── grpc.go          ← gRPC server implementation
└── api/
    └── registry/v1/
        └── registry.proto       ← NEW: Proto definition
```

---

## 3. Data Models (tái sử dụng và mở rộng)

### 3.1 App Model

```go
// Tái sử dụng từ biz/registry.go, thêm fields mới
type App struct {
    AppID       string         `gorm:"primaryKey;type:varchar(50)"`
    AppName     string         `gorm:"type:varchar(200);not null"`
    Description string         `gorm:"type:text"`
    Owner       string         `gorm:"type:varchar(100);not null"`
    Status      string         `gorm:"type:varchar(20);default:'ACTIVE'"` // ACTIVE|SUSPENDED|DELETED
    Config      datatypes.JSON `gorm:"type:jsonb"`
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`

    APIKeys []APIKey `gorm:"foreignKey:AppID"`
    Quotas  []Quota  `gorm:"foreignKey:AppID"`
}
```

### 3.2 APIKey Model

```go
type APIKey struct {
    KeyHash   string     `gorm:"primaryKey;type:varchar(64)"` // SHA-256
    AppID     string     `gorm:"type:varchar(50);not null;index"`
    KeyPrefix string     `gorm:"type:varchar(10);not null"` // kgs_abc12...
    Name      string     `gorm:"type:varchar(100)"`
    Scopes    string     `gorm:"type:varchar(500)"`         // "graph:read,graph:write"
    ExpiresAt *time.Time
    RevokedAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 3.3 Quota Model

```go
type Quota struct {
    ID        uint   `gorm:"primaryKey"`
    AppID     string `gorm:"type:varchar(50);uniqueIndex:idx_app_quota_type"`
    QuotaType string `gorm:"type:varchar(50);uniqueIndex:idx_app_quota_type"`
    // requests_per_minute | max_nodes | max_edges | max_entity_types
    Limit     int64
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 3.4 AuditLog Model

```go
type AuditLog struct {
    ID         uint      `gorm:"primaryKey"`
    RequestID  string    `gorm:"type:varchar(64);index"`
    AppID      string    `gorm:"type:varchar(50);index"`
    TenantID   string    `gorm:"type:varchar(50);index"`
    Method     string    `gorm:"type:varchar(10)"`
    Path       string    `gorm:"type:varchar(500)"`
    StatusCode int
    LatencyMs  int
    IPAddress  string    `gorm:"type:varchar(50)"`
    UserAgent  string    `gorm:"type:varchar(500)"`
    Timestamp  time.Time `gorm:"index"`
}
```

---

## 4. gRPC Server Implementation

### 4.1 Proto Definition (NEW)

```protobuf
// api/registry/v1/registry.proto
syntax = "proto3";
package registry.v1;

import "google/protobuf/empty.proto";
import "google/protobuf/timestamp.proto";

service RegistryService {
  // App Management
  rpc CreateApp(CreateAppRequest) returns (App);
  rpc GetApp(GetAppRequest) returns (App);
  rpc ListApps(ListAppsRequest) returns (ListAppsResponse);
  rpc UpdateApp(UpdateAppRequest) returns (App);
  rpc SuspendApp(SuspendAppRequest) returns (App);
  rpc DeleteApp(DeleteAppRequest) returns (google.protobuf.Empty);

  // API Key Management
  rpc IssueAPIKey(IssueAPIKeyRequest) returns (IssueAPIKeyResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (google.protobuf.Empty);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (AppContext); // Gateway calls this

  // Quota
  rpc SetQuota(SetQuotaRequest) returns (Quota);
  rpc GetQuotas(GetQuotasRequest) returns (GetQuotasResponse);

  // Audit Logs
  rpc WriteAuditLog(WriteAuditLogRequest) returns (google.protobuf.Empty);
  rpc QueryAuditLogs(QueryAuditLogsRequest) returns (QueryAuditLogsResponse);
}

message AppContext {
  string app_id = 1;
  string tenant_id = 2;
  repeated string scopes = 3;
  map<string, int64> quotas = 4; // {"requests_per_minute": 1000}
}

message ValidateAPIKeyRequest {
  string key_hash = 1; // SHA-256 hash, computed by gateway
}
```

### 4.2 Server Implementation

```go
// internal/registry/server/grpc.go
type RegistryServer struct {
    registrypb.UnimplementedRegistryServiceServer
    appUC    *biz.AppUsecase
    keyUC    *biz.APIKeyUsecase
    quotaUC  *biz.QuotaUsecase
    auditUC  *biz.AuditUsecase
    nats     *nats.Conn
    log      *slog.Logger
}

func (s *RegistryServer) ValidateAPIKey(ctx context.Context, req *registrypb.ValidateAPIKeyRequest) (*registrypb.AppContext, error) {
    appCtx, err := s.keyUC.ValidateKeyHash(ctx, req.KeyHash)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "invalid API key: %v", err)
    }
    return toProtoAppContext(appCtx), nil
}

func (s *RegistryServer) IssueAPIKey(ctx context.Context, req *registrypb.IssueAPIKeyRequest) (*registrypb.IssueAPIKeyResponse, error) {
    // Generate: plain_key = "kgs_" + random(32 bytes, base64url)
    plainKey := generateKGSKey()
    keyHash := sha256Hex(plainKey)
    keyPrefix := plainKey[:10]
    
    apiKey := &biz.APIKey{
        KeyHash:   keyHash,
        AppID:     req.AppId,
        KeyPrefix: keyPrefix,
        Name:      req.Name,
        Scopes:    strings.Join(req.Scopes, ","),
    }
    
    if err := s.keyUC.CreateKey(ctx, apiKey); err != nil {
        return nil, status.Errorf(codes.Internal, "failed to issue key: %v", err)
    }
    
    // Publish NATS event
    s.nats.Publish("registry.apikey.issued", marshalJSON(apiKey))
    
    return &registrypb.IssueAPIKeyResponse{
        PlainKey:  plainKey, // CHỈ trả về 1 lần
        KeyPrefix: keyPrefix,
        KeyId:     keyHash,
    }, nil
}

func (s *RegistryServer) RevokeAPIKey(ctx context.Context, req *registrypb.RevokeAPIKeyRequest) (*emptypb.Empty, error) {
    if err := s.keyUC.RevokeKey(ctx, req.KeyId); err != nil {
        return nil, status.Errorf(codes.Internal, "failed to revoke key: %v", err)
    }
    
    // Publish NATS event → Gateway invalidates cache
    s.nats.Publish("registry.apikey.revoked", marshalJSON(map[string]string{
        "key_hash": req.KeyId,
    }))
    
    return &emptypb.Empty{}, nil
}
```

---

## 5. Business Logic (tái sử dụng và mở rộng)

### 5.1 APIKey Usecase

```go
// internal/registry/biz/apikey.go
// Tái sử dụng logic từ biz/registry_usecase.go

type APIKeyUsecase struct {
    repo   APIKeyRepo
    appRepo AppRepo
    log    *slog.Logger
}

func (uc *APIKeyUsecase) ValidateKeyHash(ctx context.Context, keyHash string) (*AppContext, error) {
    // Lookup key
    key, err := uc.repo.FindByHash(ctx, keyHash)
    if err != nil || key == nil {
        return nil, ErrInvalidKey
    }
    
    // Check not revoked
    if key.RevokedAt != nil {
        return nil, ErrKeyRevoked
    }
    
    // Check not expired
    if key.ExpiresAt != nil && key.ExpiresAt.Before(time.Now()) {
        return nil, ErrKeyExpired
    }
    
    // Load app
    app, err := uc.appRepo.FindByID(ctx, key.AppID)
    if err != nil || app == nil {
        return nil, ErrAppNotFound
    }
    
    // Check app status
    if app.Status != "ACTIVE" {
        return nil, ErrAppNotActive
    }
    
    // Load quotas
    quotas, _ := uc.repo.GetQuotas(ctx, app.AppID)
    
    return &AppContext{
        AppID:    app.AppID,
        TenantID: app.TenantID, // Derived from app config
        Scopes:   strings.Split(key.Scopes, ","),
        Quotas:   quotasToMap(quotas),
    }, nil
}
```

### 5.2 App Lifecycle

```go
// internal/registry/biz/app.go
// Tái sử dụng từ biz/registry_usecase.go

type AppUsecase struct {
    repo AppRepo
    log  *slog.Logger
}

// State machine: ACTIVE → SUSPENDED → DELETED
func (uc *AppUsecase) SuspendApp(ctx context.Context, appID string) error {
    app, err := uc.repo.FindByID(ctx, appID)
    if err != nil {
        return err
    }
    if app.Status != "ACTIVE" {
        return ErrInvalidTransition
    }
    return uc.repo.UpdateStatus(ctx, appID, "SUSPENDED")
}
```

---

## 6. Database Migrations

```sql
-- migrations/001_registry_init.sql

CREATE TABLE IF NOT EXISTS apps (
    app_id      VARCHAR(50) PRIMARY KEY,
    app_name    VARCHAR(200) NOT NULL,
    description TEXT,
    owner       VARCHAR(100) NOT NULL,
    status      VARCHAR(20) DEFAULT 'ACTIVE',
    config      JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS api_keys (
    key_hash   VARCHAR(64) PRIMARY KEY,
    app_id     VARCHAR(50) NOT NULL REFERENCES apps(app_id),
    key_prefix VARCHAR(10) NOT NULL,
    name       VARCHAR(100),
    scopes     VARCHAR(500),
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX idx_apikey_app ON api_keys(app_id);

CREATE TABLE IF NOT EXISTS quotas (
    id         SERIAL PRIMARY KEY,
    app_id     VARCHAR(50) NOT NULL REFERENCES apps(app_id),
    quota_type VARCHAR(50) NOT NULL,
    quota_limit BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(app_id, quota_type)
);

CREATE TABLE IF NOT EXISTS audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    request_id  VARCHAR(64),
    app_id      VARCHAR(50),
    tenant_id   VARCHAR(50),
    method      VARCHAR(10),
    path        VARCHAR(500),
    status_code INT,
    latency_ms  INT,
    ip_address  VARCHAR(50),
    user_agent  VARCHAR(500),
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_app ON audit_logs(app_id, timestamp DESC);
```

---

## 7. NATS Events

| Event | Topic | Trigger |
|-------|-------|---------| 
| AppCreated | `registry.app.created` | CreateApp |
| AppSuspended | `registry.app.suspended` | SuspendApp |
| AppDeleted | `registry.app.deleted` | DeleteApp |
| APIKeyIssued | `registry.apikey.issued` | IssueAPIKey |
| APIKeyRevoked | `registry.apikey.revoked` | RevokeAPIKey → **Gateway invalidates cache** |

---

## 8. Entry Point

```go
// cmd/registry/main.go
package main

import (
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/transport/grpc"
    "kgs-platform/internal/registry/biz"
    "kgs-platform/internal/registry/data"
    "kgs-platform/internal/registry/server"
)

func main() {
    db := data.NewDB(conf.Database.DSN)
    nats := data.NewNATS(conf.NATS.Addr)
    
    appRepo := data.NewAppRepo(db)
    keyRepo := data.NewAPIKeyRepo(db)
    
    appUC := biz.NewAppUsecase(appRepo)
    keyUC := biz.NewAPIKeyUsecase(keyRepo, appRepo)
    auditUC := biz.NewAuditUsecase(data.NewAuditRepo(db))
    
    grpcSrv := grpc.NewServer(
        grpc.Address(":9001"),
    )
    
    registryServer := server.NewRegistryServer(appUC, keyUC, auditUC, nats)
    registrypb.RegisterRegistryServiceServer(grpcSrv, registryServer)
    
    app := kratos.New(
        kratos.Server(grpcSrv),
    )
    
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

---

## 9. Ước Tính Effort

| Task | Effort |
|------|--------|
| Proto definition + code gen | 0.5 ngày |
| Data models + migrations | 0.5 ngày |
| Business logic (tái sử dụng từ biz/registry*.go) | 1 ngày |
| gRPC server implementation | 1 ngày |
| NATS integration | 0.5 ngày |
| Unit tests | 1 ngày |
| **Total** | **4.5 ngày** |

---

## 10. Tại Sao Phải Tạo Mới

1. **Không tồn tại** trong codebase hiện tại — `kgs-platform` không có App/API Key management
2. **Prerequisite** cho tất cả services khác — Gateway cần registry để xác thực
3. **Security boundary** — API Key management phải tách biệt hoàn toàn
4. **Audit compliance** — Cần audit trail riêng biệt per app
5. **Quota control** — Mỗi app cần có rate limit riêng, không dùng chung flat tier
