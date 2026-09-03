# registry-service — App Registry Service

> **Role:** Quản lý toàn bộ lifecycle của tenant applications: đăng ký, API Key management, Quota, và Audit Logs.

---

## 1. Trách Nhiệm (Single Responsibility)

`registry-service` là **nguồn sự thật duy nhất** cho:
- **App lifecycle**: Tạo, cấu hình, suspend, delete apps
- **API Key management**: Tạo, revoke, validate API keys
- **Quota management**: Định nghĩa và enforce resource limits per app
- **Audit log storage**: Nhận và lưu audit logs từ Gateway

---

## 2. Kiến Trúc Nội Tại

```
┌─────────────────────────────────────────────────────────┐
│                  registry-service                        │
│                                                         │
│  gRPC Server (port 9001)                                │
│  ┌─────────────────────────────────────────────────┐   │
│  │              RegistryServiceServer               │   │
│  │                                                 │   │
│  │  CreateApp()        GetApp()      ListApps()    │   │
│  │  UpdateApp()        DeleteApp()   SuspendApp()  │   │
│  │  IssueAPIKey()      RevokeAPIKey()              │   │
│  │  ValidateAPIKey()   ListAPIKeys()               │   │
│  │  SetQuota()         GetQuota()                  │   │
│  │  WriteAuditLog()    QueryAuditLogs()            │   │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
│  ┌────────────────────────▼───────────────────────┐    │
│  │              Business Logic                     │    │
│  │                                                 │    │
│  │  RegistryUsecase                                │    │
│  │  ├── App lifecycle state machine                │    │
│  │  ├── API Key: hash-based storage (SHA-256)      │    │
│  │  ├── Quota enforcement check                    │    │
│  │  └── Audit log async writer                     │    │
│  └─────────────────────────────────────────────────┘   │
│                          │                              │
│  ┌────────────────────────▼───────────────────────┐    │
│  │              Data Layer (PostgreSQL)             │    │
│  │                                                 │    │
│  │  Table: apps                                    │    │
│  │  Table: api_keys                                │    │
│  │  Table: quotas                                  │    │
│  │  Table: audit_logs                              │    │
│  └─────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

---

## 3. Data Models

### 3.1 App

```go
type App struct {
    AppID       string         `gorm:"primaryKey;type:varchar(50)"`
    AppName     string         `gorm:"type:varchar(200);not null"`
    Description string         `gorm:"type:text"`
    Owner       string         `gorm:"type:varchar(100);not null"`
    Status      string         `gorm:"type:varchar(20);default:'ACTIVE'"` // ACTIVE | INACTIVE | SUSPENDED | DELETED
    Config      datatypes.JSON `gorm:"type:jsonb"`                          // Extra config per app
    CreatedAt   time.Time
    UpdatedAt   time.Time
    DeletedAt   gorm.DeletedAt `gorm:"index"`

    APIKeys []APIKey `gorm:"foreignKey:AppID"`
    Quotas  []Quota  `gorm:"foreignKey:AppID"`
}
```

### 3.2 APIKey

```go
type APIKey struct {
    KeyHash   string     `gorm:"primaryKey;type:varchar(64)"` // SHA-256 hash
    AppID     string     `gorm:"type:varchar(50);not null;index"`
    KeyPrefix string     `gorm:"type:varchar(10);not null"` // kgs_abc12...
    Name      string     `gorm:"type:varchar(100)"`
    Scopes    string     `gorm:"type:varchar(500)"`          // "graph:read,graph:write"
    ExpiresAt *time.Time
    RevokedAt *time.Time
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

### 3.3 Quota

```go
type Quota struct {
    ID        uint   `gorm:"primaryKey"`
    AppID     string `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_quota_type"`
    QuotaType string `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_quota_type"`
    // Types: "requests_per_minute" | "max_nodes" | "max_edges" | "max_entity_types"
    Limit     int64  `gorm:"not null"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 3.4 AuditLog

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

## 4. gRPC API

### Proto Definition

```protobuf
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
  rpc ValidateAPIKey(ValidateAPIKeyRequest) returns (AppContext); // Called by Gateway

  // Quota Management
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
  map<string, int64> quotas = 4;
}
```

---

## 5. App Lifecycle State Machine

```
              ┌─────────┐
              │  (new)  │
              └────┬────┘
                   │ CreateApp
                   ▼
           ┌────────────┐
           │   ACTIVE   │◄───────────────┐
           └────┬───────┘                │
                │                        │
         SuspendApp                  ReactivateApp
                │                        │
                ▼                        │
          ┌────────────┐                 │
          │ SUSPENDED  │─────────────────┘
          └────┬───────┘
               │
           DeleteApp
               │
               ▼
         ┌──────────┐
         │ DELETED  │ (soft delete, data retained 30 days)
         └──────────┘
```

**Rules:**
- SUSPENDED app: API keys bị block, không nhận được request mới
- DELETED app: Soft delete, data Neo4j/Qdrant được schedule cleanup sau 30 ngày
- Chỉ Platform Admin (`admin` scope) được thực hiện các thao tác này

---

## 6. API Key Security

### Tạo API Key (IssueAPIKey)

```
1. Generate: plain_key = "kgs_" + random(32 bytes, base64url)
   Example: "kgs_abc123XYZ..."
2. Prefix: key_prefix = plain_key[:10]  → "kgs_abc123"
3. Hash:   key_hash   = SHA-256(plain_key)
4. Store:  {key_hash, key_prefix, app_id, scopes, expires_at}
5. Return: plain_key (CHỈ trả về một lần, không lưu lại)
```

### Validate API Key (ValidateAPIKey)

```
1. Receive: plain_key từ X-API-Key header
2. Compute: key_hash = SHA-256(plain_key)
3. Lookup:  SELECT * FROM api_keys WHERE key_hash = ? AND deleted_at IS NULL AND revoked_at IS NULL
4. Check:   expires_at > now() (nếu có expiry)
5. Check:   app.status == 'ACTIVE'
6. Return:  AppContext{app_id, tenant_id, scopes, quotas}
```

---

## 7. NATS Events Published

| Event | Topic | Trigger |
|-------|-------|---------|
| AppCreated | `registry.app.created` | Sau khi tạo app thành công |
| AppSuspended | `registry.app.suspended` | Khi app bị suspend |
| AppDeleted | `registry.app.deleted` | Khi app bị delete |
| APIKeyRevoked | `registry.apikey.revoked` | Khi revoke key → Gateway invalidate cache |

---

## 8. HTTP REST Endpoints (Exposed qua Gateway)

| Method | Path | gRPC Method | Scope Required |
|--------|------|------------|----------------|
| POST | `/v1/apps` | CreateApp | `admin` |
| GET | `/v1/apps` | ListApps | `admin` |
| GET | `/v1/apps/:app_id` | GetApp | `admin` |
| PUT | `/v1/apps/:app_id` | UpdateApp | `admin` |
| POST | `/v1/apps/:app_id/suspend` | SuspendApp | `admin` |
| DELETE | `/v1/apps/:app_id` | DeleteApp | `admin` |
| POST | `/v1/apps/:app_id/keys` | IssueAPIKey | `admin` |
| DELETE | `/v1/apps/:app_id/keys/:key_id` | RevokeAPIKey | `admin` |
| GET | `/v1/apps/:app_id/keys` | ListAPIKeys | `admin` |
| PUT | `/v1/apps/:app_id/quotas` | SetQuota | `admin` |
| GET | `/v1/apps/:app_id/quotas` | GetQuotas | `admin` |
| GET | `/v1/apps/:app_id/audit-logs` | QueryAuditLogs | `admin` |

---

## 9. Configuration

```yaml
# configs/registry.yaml
registry_service:
  grpc_port: 9001

  database:
    dsn: "postgres://kgs:password@postgres:5432/kgs_registry"
    max_open_conns: 20
    max_idle_conns: 5

  nats:
    addr: nats:4222

  api_key:
    prefix: "kgs_"
    default_expiry_days: 365

  observability:
    metrics_port: 9090
```

---

## 10. Observability

| Metric | Mô tả |
|--------|-------|
| `registry_apps_total` | Tổng số apps đang ACTIVE |
| `registry_api_key_validations_total{result}` | Số lần validate key (hit/miss/invalid) |
| `registry_audit_logs_written_total` | Số audit logs được ghi |
| `registry_quota_checks_total{app_id, exceeded}` | Quota check statistics |
