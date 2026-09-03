# Change Request: CR-OV-006 — Crypto & Admin Services

**CR ID:** CR-OV-006  
**Component:** `services/openviking-crypto` + `services/openviking-admin` [NEW SERVICES]  
**Priority:** High  
**Status:** Implemented
**Reference:** OpenViking PRD §4.7, SRS §2.5-2.6, specs/services/06-crypto-service.md + 07-admin-service.md  
**Maps from Python:** `crypto/encryptor.py`, `crypto/providers.py`, Admin routes

---

## 1. Mô tả

Xây dựng 2 services cross-cutting:

### 1A. openviking-crypto (Port 9015)
Envelope encryption service với OVE1 binary format:
1. **Per-file AES-256-GCM encryption**: Mỗi file có File Key ngẫu nhiên riêng.
2. **Key Hierarchy**: Root Key → Account Key (HKDF) → File Key (random 32B).
3. **KMS Adapters**: Local file, HashiCorp Vault, Cloud KMS (AWS/GCP/Volcengine).
4. **Key Rotation**: Re-wrap File Keys với Root Key mới (không cần re-encrypt content).
5. **OVE1 Binary Format**: Binary-compatible với Python implementation.

### 1B. openviking-admin (Port 9030)
Admin service quản lý multi-tenant infrastructure:
1. **Account Management**: CRUD tenants — mỗi account là một namespace isolation boundary.
2. **User Management**: CRUD users trong account scope.
3. **API Key Management**: Create, list, revoke keys với role assignment (ROOT/ADMIN/USER/BOT).
4. **Health Aggregation**: Parallel health check tất cả downstream services.
5. **System Info**: Version, uptime, configuration diagnostics.
6. **Maintenance**: Storage cleanup, vector index maintenance.
7. **Usage Stats**: Token usage, storage usage per tenant.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có per-file envelope encryption.
- API key management phân tán trong nhiều services.
- Thiếu account lifecycle events (cascade delete khi xóa tenant).
- Chưa có usage statistics per tenant.

---

## 3. Thay đổi đề xuất — Crypto Service

### 3.1. [NEW] `services/openviking-crypto/` (Port gRPC: 9015)

### 3.2. OVE1 Binary Format

```
┌──────────┬─────────┬──────────┬───────────┬──────────┬──────────┐
│ Magic 4B │ Ver 1B  │ Prov 1B  │ EFK Len 2B│ KIV Len 2B│ DIV Len 2B│
├──────────┴─────────┴──────────┴───────────┴──────────┴──────────┤
│     Encrypted File Key     │   Key IV   │   Data IV              │
├────────────────────────────────────────────────────────────────-─┤
│                 AES-GCM Ciphertext + 16B Auth Tag                │
└──────────────────────────────────────────────────────────────────┘

Magic:    "OVE1" (4 bytes, ASCII)
Version:  0x01
Provider: 0x01=Local, 0x02=Vault, 0x03=Cloud
```

### 3.3. Key Hierarchy

```
Root Key (from KMS Provider — stored securely)
  └── Account Key (derived via HKDF(root_key, account_id, "openviking-account-key"))
       └── File Key (random 32 bytes per file)
            └── AES-256-GCM encrypt(plaintext) → ciphertext
```

**Key Rotation (re-wrap only):**
```
1. Get new Root Key from KMS
2. For each file in account:
   a. Read OVE1 header
   b. Decrypt File Key with OLD Account Key
   c. Re-encrypt File Key with NEW Account Key
   d. Rewrite OVE1 header (content unchanged)
3. Publish ov.crypto.key.rotated → FS re-wrap complete
```

### 3.4. Domain Model

```go
// domain/envelope.go
type OVE1Header struct {
    Magic            [4]byte  // "OVE1"
    Version          byte     // 0x01
    ProviderType     byte     // 0x01 | 0x02 | 0x03
    EncKeyLen        uint16
    KeyIVLen         uint16
    DataIVLen        uint16
    EncryptedFileKey []byte
    KeyIV            []byte
    DataIV           []byte
}

// domain/key_hierarchy.go
type KeyScope struct {
    AccountID string
    UserID    string   // optional: per-user key isolation
}
```

### 3.5. KMS Provider Interface

```go
type KMSProvider interface {
    GetRootKey(ctx context.Context) ([]byte, error)
    DeriveAccountKey(ctx context.Context, accountID string) ([]byte, error)
    RotateRootKey(ctx context.Context) error
    ProviderType() byte
}

// Implementations:
// LocalFileProvider: key file at encryption.local_key_path
// VaultProvider:    HashiCorp Vault KV engine (vault_addr + vault_token)
// CloudKMSProvider: AWS KMS | GCP KMS | Volcengine KMS
```

### 3.6. Crypto gRPC Service

```protobuf
service CryptoService {
  rpc Encrypt(EncryptRequest) returns (EncryptResponse);    // plaintext → OVE1 bytes
  rpc Decrypt(DecryptRequest) returns (DecryptResponse);    // OVE1 bytes → plaintext
  rpc RotateKeys(RotateKeysRequest) returns (RotateKeysResponse); // Re-wrap all file keys
  rpc GetKeyStatus(GetKeyStatusRequest) returns (GetKeyStatusResponse);
  rpc Bootstrap(BootstrapRequest) returns (BootstrapResponse);
}

message EncryptRequest {
  bytes plaintext = 1;
  string account_id = 2;
}

message EncryptResponse {
  bytes ciphertext = 1;   // OVE1 format binary
}
```

---

## 4. Thay đổi đề xuất — Admin Service

### 4.1. [NEW] `services/openviking-admin/` (Port gRPC: 9030)

### 4.2. Domain Model

```go
// domain/account.go
type Account struct {
    ID        string
    Name      string
    Config    AccountConfig
    CreatedAt time.Time
    IsActive  bool
}

type AccountConfig struct {
    MaxStorageMB     int
    MaxUsers         int
    EncryptionEnabled bool
}

// domain/api_key.go
type APIKey struct {
    ID         string
    AccountID  string
    UserID     string     // empty for ADMIN keys
    Role       Role       // ROOT | ADMIN | USER | BOT
    Name       string     // human-readable label
    KeyHash    []byte     // bcrypt hash (plaintext NOT stored)
    KeyPrefix  string     // first 8 chars for identification
    CreatedAt  time.Time
    ExpiresAt  *time.Time
    LastUsedAt *time.Time
    IsActive   bool
}

// ROOT key: format "ovr_" + base62(32 bytes)
// USER key: format "ovu_" + base62(32 bytes)
// BOT  key: format "ovb_" + base62(32 bytes)

// domain/health.go
type AggregatedHealth struct {
    OverallStatus string
    Services      map[string]ServiceHealth
    CheckedAt     time.Time
}

type ServiceHealth struct {
    Status    string    // "serving" | "not_serving" | "unknown"
    LatencyMs int64
    Error     string
}
```

### 4.3. Admin gRPC Service

```protobuf
service AdminService {
  // Accounts (tenants)
  rpc CreateAccount(CreateAccountRequest) returns (CreateAccountResponse);
  rpc ListAccounts(ListAccountsRequest) returns (ListAccountsResponse);
  rpc GetAccount(GetAccountRequest) returns (GetAccountResponse);
  rpc DeleteAccount(DeleteAccountRequest) returns (DeleteAccountResponse);

  // Users
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse);
  rpc DeleteUser(DeleteUserRequest) returns (DeleteUserResponse);

  // API Keys
  rpc CreateAPIKey(CreateAPIKeyRequest) returns (CreateAPIKeyResponse);
  rpc ListAPIKeys(ListAPIKeysRequest) returns (ListAPIKeysResponse);
  rpc RevokeAPIKey(RevokeAPIKeyRequest) returns (RevokeAPIKeyResponse);
  rpc ResolveAPIKey(ResolveAPIKeyRequest) returns (ResolveAPIKeyResponse); // Called by Gateway

  // Health
  rpc GetAggregatedHealth(GetAggregatedHealthRequest) returns (AggregatedHealthResponse);

  // Tasks (cross-service)
  rpc GetTask(GetTaskRequest) returns (GetTaskResponse);
  rpc ListTasks(ListTasksRequest) returns (ListTasksResponse);

  // System
  rpc GetSystemInfo(GetSystemInfoRequest) returns (GetSystemInfoResponse);
  rpc WaitReady(WaitReadyRequest) returns (WaitReadyResponse);

  // Maintenance
  rpc RunMaintenance(RunMaintenanceRequest) returns (RunMaintenanceResponse);

  // Stats
  rpc GetUsageStats(GetUsageStatsRequest) returns (GetUsageStatsResponse);
}
```

### 4.4. Health Aggregation (Parallel fan-out)

```go
// usecase/aggregate_health.go
func (uc *AggregateHealthUseCase) Execute(ctx context.Context) (*AggregatedHealthResponse, error) {
    services := []string{"fs", "search", "session", "resource", "crypto"}
    
    var wg sync.WaitGroup
    results := make(map[string]ServiceHealth)
    
    for _, svc := range services {
        wg.Add(1)
        go func(name string) {
            defer wg.Done()
            start := time.Now()
            // gRPC Health check per service (grpc.health.v1)
            resp, err := uc.clients[name].Check(ctx, &healthpb.HealthCheckRequest{})
            results[name] = ServiceHealth{
                Status:    resp.GetStatus().String(),
                LatencyMs: time.Since(start).Milliseconds(),
            }
        }(svc)
    }
    wg.Wait()
    
    // Aggregate: ALL healthy → SERVING; ANY unhealthy → DEGRADED; ALL down → NOT_SERVING
    return aggregateResults(results), nil
}
```

### 4.5. Account Lifecycle Events (NATS)

```
CreateAccount → Publish admin.account.created
  → FS: init root dirs (viking://resources/, viking://user/, viking://agent/, viking://session/)
  → Search: create vector collection for account ("openviking_{account_id}")
  → Crypto: derive account encryption key

DeleteAccount → Publish admin.account.deleted
  → FS: cascade delete ALL account data
  → Search: drop vector collection
  → Session: delete all account sessions
  → Crypto: revoke account key
```

### 4.6. API Key Security

```
Key Generation:
  plaintext = "ovr_" + base62(rand.Read(32))  // Never stored
  hash      = bcrypt.GenerateFromPassword(plaintext, 12)
  prefix    = plaintext[:8]                    // Stored for identification only
  → Return plaintext once, store ONLY hash+prefix

Key Resolution (per request in Gateway):
  → Admin.ResolveAPIKey(presented_key)
  → bcrypt.CompareHashAndPassword(stored_hash, presented_key)
  → Return AccountID, UserID, Role (cached in Redis, TTL=5min)
```

### 4.7. Configuration

```yaml
crypto:
  grpc:
    port: 9015
  health:
    port: 9095
  encryption:
    enabled: false
    provider: "local"           # local | vault | cloud
    local_key_path: "~/.openviking/root.key"
    vault_addr: "http://vault:8200"
    vault_token: ""
  nats:
    url: "nats://nats:4222"

admin:
  grpc:
    port: 9030
  health:
    port: 9099
  storage:
    backend: "redis"            # redis | local
    redis_url: "redis://redis:6379/0"
  auth:
    key_bcrypt_cost: 12
    key_cache_ttl: 300s         # Redis cache for resolved keys
  nats:
    url: "nats://nats:4222"
    stream: "admin"
  clients:                      # For health aggregation
    fs: "openviking-fs:9011"
    search: "openviking-search:9012"
    session: "openviking-session:9013"
    resource: "openviking-resource:9014"
    crypto: "openviking-crypto:9015"
```

---

## 5. Acceptance Criteria

### Crypto Service
- [ ] `Encrypt(plaintext)` → bytes start with "OVE1" magic; binary-compatible với Python OVE1 format.
- [ ] `Decrypt(OVE1_bytes)` → trả về đúng plaintext gốc.
- [ ] Key rotation: sau RotateKeys, mọi file vẫn decrypt được với key mới (re-wrapped, content không đổi).
- [ ] Vault provider: kết nối HashiCorp Vault, GetRootKey thành công.
- [ ] Encryption disabled: `Encrypt(plaintext)` → trả về plaintext nguyên vẹn (passthrough mode).

### Admin Service
- [ ] `CreateAccount("acme")` → NATS `admin.account.created` → FS tạo root directories cho account "acme".
- [ ] `DeleteAccount("acme")` → NATS `admin.account.deleted` → tất cả data của "acme" bị xóa.
- [ ] `CreateAPIKey(role=USER)` → trả về plaintext key `ovu_xxx...` một lần duy nhất; sau đó `ListAPIKeys` chỉ thấy prefix và hash.
- [ ] `ResolveAPIKey(wrong_key)` → 401; `ResolveAPIKey(correct_key)` → AccountID + Role đúng.
- [ ] `GetAggregatedHealth()` → parallel check 5 services trong < 200ms total.
- [ ] Service down → `GetAggregatedHealth()` trả về status=DEGRADED với detail per-service.
- [ ] `RevokeAPIKey(key_id)` → key đó bị reject ngay lập tức (cache invalidated).
