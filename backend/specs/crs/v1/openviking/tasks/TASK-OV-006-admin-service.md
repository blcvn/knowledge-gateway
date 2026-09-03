# TASK-OV-006 — `services/openviking-admin` Admin & Multi-Tenant Service

**Wave:** 2 (Security)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-001, TASK-OV-004 (pkg/nats, pkg/auth)  
**Ước tính:** 5 giờ  
**Solution tham chiếu:** [SOL-OV-006 §3](../solutions/SOL-OV-006-Crypto-Admin-Services.md)  
**Port gRPC:** 9030

**Trạng thái:** ✅ Implemented  
**Ghi chú:** ov-admin: 21 .go - admin + tenant management  
---

## Mục tiêu

Tạo `services/openviking-admin/` — quản lý multi-tenant: accounts, users, API keys với bcrypt hashing, health aggregation của tất cả downstream services, và account lifecycle NATS events.

---

## Cấu trúc thư mục

```
services/openviking-admin/
├── cmd/server/main.go
├── api/proto/admin/v1/admin.proto
├── internal/
│   ├── domain/
│   │   ├── account.go         # Account, AccountConfig
│   │   ├── user.go            # User
│   │   ├── api_key.go         # APIKey, Role, key format
│   │   └── health.go          # AggregatedHealth, ServiceHealth
│   ├── usecase/
│   │   ├── create_account.go
│   │   ├── delete_account.go
│   │   ├── list_accounts.go
│   │   ├── create_user.go
│   │   ├── delete_user.go
│   │   ├── create_api_key.go  # bcrypt hash, prefix generation
│   │   ├── list_api_keys.go
│   │   ├── revoke_api_key.go
│   │   ├── resolve_api_key.go # bcrypt verify + Redis cache
│   │   ├── aggregate_health.go # Parallel fan-out
│   │   ├── get_system_info.go
│   │   ├── get_usage_stats.go
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go      # AccountRepo, APIKeyRepo, UserRepo, HealthClients
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go
│   │   │   └── mapper.go
│   │   ├── repository/
│   │   │   ├── account_postgres.go  # PostgreSQL CRUD
│   │   │   ├── user_postgres.go
│   │   │   └── api_key_postgres.go
│   │   ├── cache/
│   │   │   └── redis_key_cache.go   # API key resolution cache
│   │   └── event/
│   │       └── publisher.go   # admin.account.created/deleted
│   └── infra/
│       ├── config/config.go
│       ├── db/migrations/001_init.up.sql
│       ├── server/grpc.go
│       └── wire/wire.go
```

---

## 1. Domain Models

**File: `internal/domain/api_key.go`**

```go
// Key format:
// ROOT:  "ovr_" + base62(32 random bytes)
// ADMIN: "ova_" + base62(32 random bytes)
// USER:  "ovu_" + base62(32 random bytes)
// BOT:   "ovb_" + base62(32 random bytes)

type APIKey struct {
    ID         string
    AccountID  string
    UserID     string        // empty for ADMIN/ROOT
    Role       viking.Role
    Name       string        // human-readable label
    KeyHash    []byte        // bcrypt hash (plaintext NEVER stored)
    KeyPrefix  string        // first 8 chars for UI
    IsActive   bool
    ExpiresAt  *time.Time
    LastUsedAt *time.Time
    CreatedAt  time.Time
}

func RoleKeyPrefix(role viking.Role) string {
    switch role {
    case viking.RoleRoot:  return "ovr"
    case viking.RoleAdmin: return "ova"
    case viking.RoleUser:  return "ovu"
    case viking.RoleBot:   return "ovb"
    }
    return "ovu"
}
```

**File: `internal/domain/health.go`**

```go
type ServiceHealth struct {
    Status    string // "serving" | "not_serving" | "unknown" | "error"
    LatencyMs int64
    Error     string
}

type AggregatedHealth struct {
    OverallStatus string  // "serving" | "degraded" | "not_serving"
    Services      map[string]ServiceHealth
    CheckedAt     time.Time
}
```

---

## 2. Use Cases

**File: `internal/usecase/create_api_key.go`**

```go
type CreateAPIKeyResult struct {
    KeyID     string
    Plaintext string  // Returned ONCE — not stored
    Prefix    string  // First 8 chars for identification
}

func (uc *CreateAPIKeyUseCase) Execute(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResult, error) {
    // 1. Generate: secret = base62(rand.Read(32))
    // 2. Plaintext: prefix + "_" + secret  (e.g., "ovu_abc123...")
    // 3. Hash: bcrypt(plaintext, cost=12)
    // 4. Save APIKey{KeyHash: hash, KeyPrefix: plaintext[:8]}
    // 5. Return plaintext ONCE (caller must store)
}
```

**File: `internal/usecase/resolve_api_key.go`**

```go
func (uc *ResolveAPIKeyUseCase) Execute(ctx context.Context, presentedKey string) (*ResolvedKey, error) {
    // 1. Check Redis cache (TTL=5min)
    // 2. Cache miss: FindByPrefix(presentedKey[:8]) → candidates
    // 3. bcrypt.CompareHashAndPassword for each candidate
    // 4. On match: UpdateLastUsedAt (async), cache result, return
    // 5. No match: return ErrUnauthenticated
}
```

**File: `internal/usecase/aggregate_health.go`**

```go
func (uc *AggregateHealthUseCase) Execute(ctx context.Context) (*AggregatedHealth, error) {
    // Budget: 5 second timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    services := []string{"fs", "search", "session", "resource", "crypto"}
    results := make(chan struct{ name string; health ServiceHealth }, len(services))
    
    for _, svc := range services {
        go func(name string) {
            start := time.Now()
            resp, err := uc.clients[name].Check(ctx, &healthpb.HealthCheckRequest{})
            h := ServiceHealth{LatencyMs: time.Since(start).Milliseconds()}
            if err != nil { h.Status = "error"; h.Error = err.Error() } else { h.Status = "serving" }
            results <- struct { ... }{name, h}
        }(svc)
    }
    
    // Aggregate: ALL serving → "serving"; ANY error → "degraded"; ALL error → "not_serving"
}
```

**File: `internal/usecase/create_account.go`**

```go
func (uc *CreateAccountUseCase) Execute(ctx context.Context, req CreateAccountRequest) (*Account, error) {
    // 1. Validate: account ID must be DNS-safe (lowercase, alphanumeric, hyphens)
    // 2. Save account to PostgreSQL
    // 3. Publish admin.account.created{account_id, encryption_enabled}
    // Subscribers:
    //   → FS: init viking:// namespaces
    //   → Search: create vector collection
    //   → Crypto: pre-derive account key
}
```

---

## 3. Database Migrations

**File: `internal/infra/db/migrations/001_init.up.sql`**

```sql
CREATE TABLE IF NOT EXISTS accounts (
    id           VARCHAR(63)  NOT NULL PRIMARY KEY,  -- DNS-safe, lowercase
    name         VARCHAR(255) NOT NULL,
    config       JSONB        NOT NULL DEFAULT '{}',
    is_active    BOOLEAN      NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id         UUID        NOT NULL,
    account_id VARCHAR(63) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    meta       JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, account_id)
);

CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID        NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id   VARCHAR(63) NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id      UUID,
    role         SMALLINT    NOT NULL,  -- 0=USER, 1=BOT, 2=ADMIN, 3=ROOT
    name         VARCHAR(255) NOT NULL,
    key_hash     BYTEA       NOT NULL,
    key_prefix   VARCHAR(8)  NOT NULL,
    is_active    BOOLEAN     NOT NULL DEFAULT true,
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix  ON api_keys(key_prefix)   WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_api_keys_account ON api_keys(account_id)   WHERE is_active = true;
```

---

## 4. gRPC Proto

**File: `api/proto/admin/v1/admin.proto`**

```protobuf
syntax = "proto3";
package openviking.admin.v1;

service AdminService {
  // Accounts
  rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse);
  rpc ListAccounts(ListAccountsRequest)   returns (ListAccountsResponse);
  rpc GetAccount(GetAccountRequest)       returns (GetAccountResponse);
  rpc DeleteAccount(DeleteAccountRequest) returns (DeleteAccountResponse);

  // Users
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc ListUsers(ListUsersRequest)   returns (ListUsersResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);

  // API Keys
  rpc CreateAPIKey(CreateAPIKeyRequest)   returns (CreateAPIKeyResponse);
  rpc ListAPIKeys(ListAPIKeysRequest)     returns (ListAPIKeysResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest)   returns (RevokeAPIKeyResponse);
  rpc ResolveAPIKey(ResolveAPIKeyRequest) returns (ResolveAPIKeyResponse);

  // Health
  rpc GetAggregatedHealth(GetAggregatedHealthRequest) returns (AggregatedHealthResponse);

  // System
  rpc GetSystemInfo(GetSystemInfoRequest) returns (GetSystemInfoResponse);
  rpc WaitReady(WaitReadyRequest)         returns (WaitReadyResponse);

  // Stats
  rpc GetUsageStats(GetUsageStatsRequest) returns (GetUsageStatsResponse);
}

message CreateAPIKeyResponse {
  string key_id   = 1;
  string plaintext = 2;  // Returned ONCE
  string prefix   = 3;   // First 8 chars
}

message ResolveAPIKeyRequest { string key = 1; }
message ResolveAPIKeyResponse {
  string account_id = 1;
  string user_id    = 2;
  int32  role       = 3;
  string key_id     = 4;
}

message AggregatedHealthResponse {
  string overall_status = 1;
  map<string, ServiceHealthProto> services = 2;
  google.protobuf.Timestamp checked_at = 3;
}

message ServiceHealthProto {
  string status     = 1;
  int64  latency_ms = 2;
  string error      = 3;
}
```

---

## 5. Config

```yaml
admin:
  grpc:
    port: 9030
  health:
    port: 9099
  database:
    url: "${DATABASE_URL}"
    max_connections: 20
  redis:
    url: "redis://redis:6379/0"
    key_cache_ttl: 300s   # 5 minutes
  auth:
    key_bcrypt_cost: 12   # ~200ms per hash
  nats:
    url: "nats://nats:4222"
    stream: "admin"
  clients:
    fs:       "openviking-fs:9011"
    search:   "openviking-search:9012"
    session:  "openviking-session:9013"
    resource: "openviking-resource:9014"
    crypto:   "openviking-crypto:9015"
```

---

## Unit Tests

```
TestCreateAPIKey_HashNotPlaintext    → SavedKey.KeyHash ≠ plaintext
TestCreateAPIKey_PrefixCorrect       → prefix = plaintext[:8]
TestCreateAPIKey_PlaintextReturned   → result.Plaintext starts with "ovu_"
TestResolveAPIKey_CorrectKey         → bcrypt match → returns AccountID, Role
TestResolveAPIKey_WrongKey           → no match → ErrUnauthenticated
TestResolveAPIKey_InactiveKey        → is_active=false → ErrUnauthenticated
TestResolveAPIKey_CacheHit           → 2nd call → no bcrypt (cache hit)
TestRevokeAPIKey_CacheInvalidated    → revoke → cache cleared → next resolve fails
TestAggregateHealth_AllHealthy       → all return SERVING → "serving"
TestAggregateHealth_OneDown          → 1 error → "degraded"
TestAggregateHealth_Timeout5s        → 5s budget → returns partial
TestCreateAccount_PublishesNATS      → event published with correct accountID
TestDeleteAccount_PublishesNATS      → account.deleted event published
TestBase62Encode_Deterministic       → same input → same output
TestBase62Encode_Length32            → 32 bytes input → expected length output
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

# Generate protobuf
buf generate services/openviking-admin/

# Migrations (requires PostgreSQL)
# psql $DATABASE_URL < services/openviking-admin/internal/infra/db/migrations/001_init.up.sql

# Build
go build ./services/openviking-admin/...

# Test (unit tests chạy không cần PostgreSQL)
go test ./services/openviking-admin/... -v -count=1
```

---

## Ghi chú triển khai

- `golang.org/x/crypto/bcrypt` cho key hashing
- base62 encoding: `0-9A-Za-z` (62 ký tự), tự implement hoặc dùng `github.com/jxskiss/base62`
- PostgreSQL connection: `github.com/jackc/pgx/v5`
- Database migrations: sử dụng `golang-migrate/migrate` hoặc `pressly/goose`
- Account ID validation: `regexp.MustCompile("^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$")`
- `UpdateLastUsedAt` gọi async (go routine) để không block authentication path
