# Solution: SOL-OV-006 — Crypto & Admin Services

**CR:** [CR-OV-006](../CR-OV-006-Crypto-Admin-Services.md)  
**Wave:** 2 (Security — xây sau `pkg/`, trước Filesystem)  
**Priority:** High  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng 2 services bảo mật nền tảng:
- **`openviking-crypto` (port 9015)** — OVE1 envelope encryption engine
- **`openviking-admin` (port 9030)** — Multi-tenant account/user/key management + health aggregation

### Tại sao phải làm trước Filesystem?

Filesystem service cần gọi `crypto.Encrypt/Decrypt` cho mỗi file write/read. Admin service cần tồn tại trước để Gateway có thể `ResolveAPIKey`. Cả 2 services là foundation của hệ thống.

---

## 2. `services/openviking-crypto/`

### 2.1 OVE1 Binary Format — Chi tiết Implementation

```
Binary layout (big-endian):

Offset  Size  Field
------  ----  -----
0       4     Magic bytes: "OVE1" (0x4F 0x56 0x45 0x31)
4       1     Version: 0x01
5       1     Provider type: 0x01=Local, 0x02=Vault, 0x03=Cloud
6       2     EFK_LEN: length of Encrypted File Key (bytes)
8       2     KEY_IV_LEN: length of Key IV (bytes, always 12 for GCM)
10      2     DATA_IV_LEN: length of Data IV (bytes, always 12 for GCM)
12      EFK_LEN      Encrypted File Key (AES-256-GCM, wrapped by Account Key)
12+EFK  KEY_IV_LEN   Key IV (12 bytes for AESGCM nonce)
...     DATA_IV_LEN  Data IV (12 bytes for AESGCM nonce)  
...     variable     AES-GCM Ciphertext || 16-byte Auth Tag
```

```go
// internal/domain/envelope.go

const OVE1Magic = "OVE1"
const OVE1Version = 0x01

type OVE1Header struct {
    Magic            [4]byte
    Version          byte
    ProviderType     byte
    EncKeyLen        uint16
    KeyIVLen         uint16
    DataIVLen        uint16
    EncryptedFileKey []byte  // len = EncKeyLen
    KeyIV            []byte  // len = KeyIVLen (12 bytes)
    DataIV           []byte  // len = DataIVLen (12 bytes)
}

func ParseOVE1Header(data []byte) (*OVE1Header, int, error) {
    if len(data) < 12 {
        return nil, 0, fmt.Errorf("data too short for OVE1 header")
    }
    if string(data[0:4]) != OVE1Magic {
        return nil, 0, &viking.OpenVikingError{
            Code:    viking.ErrInvalidArgument,
            Message: "not an OVE1 encoded file",
        }
    }
    h := &OVE1Header{}
    copy(h.Magic[:], data[0:4])
    h.Version      = data[4]
    h.ProviderType = data[5]
    h.EncKeyLen    = binary.BigEndian.Uint16(data[6:8])
    h.KeyIVLen     = binary.BigEndian.Uint16(data[8:10])
    h.DataIVLen    = binary.BigEndian.Uint16(data[10:12])
    
    offset := 12
    h.EncryptedFileKey = data[offset : offset+int(h.EncKeyLen)]
    offset += int(h.EncKeyLen)
    h.KeyIV            = data[offset : offset+int(h.KeyIVLen)]
    offset += int(h.KeyIVLen)
    h.DataIV           = data[offset : offset+int(h.DataIVLen)]
    offset += int(h.DataIVLen)
    
    return h, offset, nil
}

func IsOVE1(data []byte) bool {
    return len(data) >= 4 && string(data[0:4]) == OVE1Magic
}
```

### 2.2 Key Hierarchy Implementation

```go
// internal/usecase/encrypt.go

type EncryptUseCase struct {
    kms     adapters.KMSProvider
    metrics *observability.CryptoMetrics
}

func (uc *EncryptUseCase) Execute(ctx context.Context, plaintext []byte, accountID string) ([]byte, error) {
    // 1. Get Account Key (derived from Root Key via HKDF)
    accountKey, err := uc.kms.DeriveAccountKey(ctx, accountID)
    if err != nil {
        return nil, fmt.Errorf("derive account key: %w", err)
    }
    defer zeroBytesSlice(accountKey)  // Security: zero key after use
    
    // 2. Generate random File Key (32 bytes)
    fileKey := make([]byte, 32)
    if _, err := rand.Read(fileKey); err != nil {
        return nil, fmt.Errorf("generate file key: %w", err)
    }
    defer zeroBytesSlice(fileKey)
    
    // 3. Encrypt File Key with Account Key (AES-256-GCM)
    keyIV := make([]byte, 12)
    rand.Read(keyIV)
    encFileKey, err := aesGCMEncrypt(accountKey, keyIV, fileKey)
    if err != nil {
        return nil, fmt.Errorf("encrypt file key: %w", err)
    }
    
    // 4. Encrypt plaintext with File Key (AES-256-GCM)
    dataIV := make([]byte, 12)
    rand.Read(dataIV)
    ciphertext, err := aesGCMEncrypt(fileKey, dataIV, plaintext)
    if err != nil {
        return nil, fmt.Errorf("encrypt data: %w", err)
    }
    
    // 5. Assemble OVE1 binary
    return assembleOVE1(OVE1Header{
        Magic:            [4]byte{'O', 'V', 'E', '1'},
        Version:          OVE1Version,
        ProviderType:     uc.kms.ProviderType(),
        EncKeyLen:        uint16(len(encFileKey)),
        KeyIVLen:         uint16(len(keyIV)),
        DataIVLen:        uint16(len(dataIV)),
        EncryptedFileKey: encFileKey,
        KeyIV:            keyIV,
        DataIV:           dataIV,
    }, ciphertext), nil
}

// aesGCMEncrypt: AES-256-GCM encrypt with auth tag appended
func aesGCMEncrypt(key, iv, plaintext []byte) ([]byte, error) {
    block, err := aes.NewCipher(key)
    if err != nil {
        return nil, err
    }
    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }
    // gcm.Seal appends ciphertext + 16-byte auth tag
    return gcm.Seal(nil, iv, plaintext, nil), nil
}
```

### 2.3 Key Rotation (Re-wrap Only)

```go
// internal/usecase/rotate_keys.go

func (uc *RotateKeysUseCase) Execute(ctx context.Context, accountID string, fileURIs []string) error {
    // 1. Get OLD Account Key (before rotation)
    oldAccountKey, err := uc.kms.DeriveAccountKey(ctx, accountID)
    
    // 2. Rotate root key in KMS
    if err := uc.kms.RotateRootKey(ctx); err != nil {
        return err
    }
    
    // 3. Get NEW Account Key (after rotation)
    newAccountKey, err := uc.kms.DeriveAccountKey(ctx, accountID)
    
    // 4. Re-wrap each file's File Key (parallel, bounded goroutines)
    g, gCtx := errgroup.WithContext(ctx)
    sem := semaphore.NewWeighted(10)  // Max 10 concurrent re-wraps
    
    for _, uri := range fileURIs {
        uri := uri
        g.Go(func() error {
            sem.Acquire(gCtx, 1)
            defer sem.Release(1)
            return uc.rewrapFile(gCtx, uri, oldAccountKey, newAccountKey)
        })
    }
    
    if err := g.Wait(); err != nil {
        return err
    }
    
    // 5. Publish NATS event
    uc.publisher.Publish(ctx, "ov.crypto.key.rotated", map[string]string{
        "account_id": accountID,
    })
    return nil
}

func (uc *RotateKeysUseCase) rewrapFile(ctx context.Context, uri string, oldKey, newKey []byte) error {
    // Read OVE1 binary
    ove1Data, _ := uc.fsClient.ReadRaw(ctx, uri)
    if !IsOVE1(ove1Data) {
        return nil  // Not encrypted, skip
    }
    
    header, ciphertextOffset, _ := ParseOVE1Header(ove1Data)
    
    // Decrypt File Key with OLD Account Key
    fileKey, _ := aesGCMDecrypt(oldKey, header.KeyIV, header.EncryptedFileKey)
    
    // Re-encrypt File Key with NEW Account Key
    newKeyIV := make([]byte, 12)
    rand.Read(newKeyIV)
    newEncFileKey, _ := aesGCMEncrypt(newKey, newKeyIV, fileKey)
    
    // Rewrite ONLY the OVE1 header (content/ciphertext unchanged)
    newOVE1 := assembleOVE1(OVE1Header{
        Magic:            header.Magic,
        Version:          header.Version,
        ProviderType:     header.ProviderType,
        EncKeyLen:        uint16(len(newEncFileKey)),
        KeyIVLen:         uint16(len(newKeyIV)),
        DataIVLen:        header.DataIVLen,
        EncryptedFileKey: newEncFileKey,
        KeyIV:            newKeyIV,
        DataIV:           header.DataIV,
    }, ove1Data[ciphertextOffset:])  // Re-use existing ciphertext
    
    return uc.fsClient.WriteRaw(ctx, uri, newOVE1)
}
```

### 2.4 Passthrough Mode

```go
// Khi encryption.enabled=false:
// Encrypt(plaintext) → trả về plaintext nguyên vẹn
// Decrypt(data) → nếu IsOVE1(data) → error; else → trả về data nguyên vẹn

type PassthroughCryptoService struct{}
func (s *PassthroughCryptoService) Encrypt(_ context.Context, data []byte, _ string) ([]byte, error) {
    return data, nil
}
func (s *PassthroughCryptoService) Decrypt(_ context.Context, data []byte, _ string) ([]byte, error) {
    if IsOVE1(data) {
        return nil, &viking.OpenVikingError{
            Code:    viking.ErrInvalidArgument,
            Message: "cannot decrypt OVE1 data: encryption is disabled",
        }
    }
    return data, nil
}
```

---

## 3. `services/openviking-admin/`

### 3.1 API Key Management

```go
// internal/domain/api_key.go

// Key formats:
// ROOT:  "ovr_" + base62(rand 32 bytes)  → "ovr_abc123..."
// ADMIN: "ova_" + base62(rand 32 bytes)  → "ova_def456..."
// USER:  "ovu_" + base62(rand 32 bytes)  → "ovu_xyz789..."
// BOT:   "ovb_" + base62(rand 32 bytes)  → "ovb_qrs012..."

type APIKey struct {
    ID         string
    AccountID  string
    UserID     string        // empty for ADMIN/ROOT keys
    Role       viking.Role
    Name       string        // human-readable label
    KeyHash    []byte        // bcrypt(plaintext, cost=12) — NEVER store plaintext
    KeyPrefix  string        // first 8 chars for UI identification
    CreatedAt  time.Time
    ExpiresAt  *time.Time
    LastUsedAt *time.Time
    IsActive   bool
}

// internal/usecase/create_api_key.go

func (uc *CreateAPIKeyUseCase) Execute(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResult, error) {
    // 1. Generate random 32 bytes
    secret := make([]byte, 32)
    rand.Read(secret)
    secretB62 := base62Encode(secret)
    
    // 2. Build plaintext key
    prefix := keyPrefix(req.Role)
    plaintext := prefix + "_" + secretB62  // e.g., "ovu_abc123xyz..."
    
    // 3. bcrypt hash (cost=12 for security; ~200ms, acceptable for key creation)
    hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), 12)
    
    // 4. Store key metadata (hash + prefix, NEVER plaintext)
    key := &APIKey{
        ID:        uuid.New().String(),
        AccountID: req.AccountID,
        UserID:    req.UserID,
        Role:      req.Role,
        Name:      req.Name,
        KeyHash:   hash,
        KeyPrefix: plaintext[:8],  // "ovu_abc1" for identification
        IsActive:  true,
    }
    uc.keyRepo.Save(ctx, key)
    
    // 5. Return PLAINTEXT only once — cannot be retrieved later
    return &CreateAPIKeyResult{
        KeyID:     key.ID,
        Plaintext: plaintext,  // User must store this!
    }, nil
}
```

### 3.2 ResolveAPIKey (Called by Gateway per request)

```go
// internal/usecase/resolve_api_key.go

func (uc *ResolveAPIKeyUseCase) Execute(ctx context.Context, presentedKey string) (*ResolvedKey, error) {
    // 1. Extract prefix to find candidate keys efficiently
    if len(presentedKey) < 8 {
        return nil, &viking.OpenVikingError{Code: viking.ErrUnauthenticated}
    }
    prefix := presentedKey[:8]
    
    // 2. Load candidate keys by prefix
    candidates, err := uc.keyRepo.FindByPrefix(ctx, prefix)
    if err != nil || len(candidates) == 0 {
        return nil, &viking.OpenVikingError{Code: viking.ErrUnauthenticated}
    }
    
    // 3. bcrypt comparison for each candidate (usually 1)
    for _, candidate := range candidates {
        if !candidate.IsActive {
            continue
        }
        if err := bcrypt.CompareHashAndPassword(candidate.KeyHash, []byte(presentedKey)); err == nil {
            // Match found
            // Update LastUsedAt (async, don't block)
            go uc.keyRepo.UpdateLastUsedAt(context.Background(), candidate.ID)
            return &ResolvedKey{
                AccountID: candidate.AccountID,
                UserID:    candidate.UserID,
                Role:      candidate.Role,
                KeyID:     candidate.ID,
            }, nil
        }
    }
    
    return nil, &viking.OpenVikingError{Code: viking.ErrUnauthenticated}
}
```

**bcrypt overhead mitigation:** Gateway caches resolved keys in Redis (`TTL=5min`). Most requests hit cache → ~1ms. Only first request per key triggers bcrypt (~200ms).

### 3.3 Account Lifecycle — NATS Events

```go
// internal/usecase/create_account.go

func (uc *CreateAccountUseCase) Execute(ctx context.Context, req CreateAccountRequest) (*Account, error) {
    account := &Account{
        ID:        req.ID,   // User-provided unique name (DNS-safe)
        Name:      req.Name,
        IsActive:  true,
        Config: AccountConfig{
            MaxStorageMB:      req.MaxStorageMB,     // default: 10240 (10GB)
            MaxUsers:          req.MaxUsers,          // default: 100
            EncryptionEnabled: req.EncryptionEnabled, // default: false
        },
    }
    if err := uc.accountRepo.Save(ctx, account); err != nil {
        return nil, err
    }
    
    // Broadcast — các services init account resources
    uc.publisher.Publish(ctx, "admin.account.created", map[string]any{
        "account_id":         account.ID,
        "encryption_enabled": account.Config.EncryptionEnabled,
    })
    // Subscribers:
    // → FS:     Mkdir viking://resources/, viking://user/, viking://agent/, viking://session/
    // → Search: CreateCollection("openviking_{account_id}")
    // → Crypto: DeriveAccountKey(account_id) → pre-compute + cache
    
    return account, nil
}

// internal/usecase/delete_account.go
func (uc *DeleteAccountUseCase) Execute(ctx context.Context, accountID string) error {
    if err := uc.accountRepo.Delete(ctx, accountID); err != nil {
        return err
    }
    uc.publisher.Publish(ctx, "admin.account.deleted", map[string]string{"account_id": accountID})
    // Subscribers:
    // → FS:      Rm(viking://user/{accountID}/, recursive=true) + viking://resources/...
    // → Search:  DropCollection("openviking_{accountID}")
    // → Session: Delete all sessions for accountID
    // → Crypto:  Revoke account key from cache
    return nil
}
```

### 3.4 Health Aggregation — Parallel Fan-out

```go
// internal/usecase/aggregate_health.go

func (uc *AggregateHealthUseCase) Execute(ctx context.Context) (*domain.AggregatedHealth, error) {
    services := map[string]healthpb.HealthClient{
        "fs":       uc.clients.FS,
        "search":   uc.clients.Search,
        "session":  uc.clients.Session,
        "resource": uc.clients.Resource,
        "crypto":   uc.clients.Crypto,
    }
    
    type result struct {
        name   string
        health domain.ServiceHealth
    }
    
    // Budget 5 seconds for all health checks
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    
    results := make(chan result, len(services))
    
    for name, client := range services {
        go func(n string, c healthpb.HealthClient) {
            start := time.Now()
            resp, err := c.Check(ctx, &healthpb.HealthCheckRequest{Service: ""})
            latency := time.Since(start).Milliseconds()
            
            h := domain.ServiceHealth{LatencyMs: latency}
            if err != nil {
                h.Status = "error"
                h.Error  = err.Error()
            } else {
                h.Status = strings.ToLower(resp.Status.String())
            }
            results <- result{n, h}
        }(name, client)
    }
    
    agg := &domain.AggregatedHealth{
        Services:  make(map[string]domain.ServiceHealth),
        CheckedAt: time.Now(),
    }
    
    for i := 0; i < len(services); i++ {
        r := <-results
        agg.Services[r.name] = r.health
    }
    
    // Compute overall status
    allHealthy := true
    for _, svc := range agg.Services {
        if svc.Status != "serving" {
            allHealthy = false
            break
        }
    }
    if allHealthy {
        agg.OverallStatus = "serving"
    } else {
        agg.OverallStatus = "degraded"
    }
    
    return agg, nil
}
```

### 3.5 Usage Statistics

```go
// Redis-based usage tracking (no DB write per request)

// Key: "ov_usage:{account_id}:{date}:tokens_in"  → INCRBY
// Key: "ov_usage:{account_id}:{date}:tokens_out" → INCRBY
// Key: "ov_usage:{account_id}:{date}:searches"   → INCR

// Daily aggregation job (midnight cron):
// SCAN "ov_usage:{account_id}:{yesterday}:*"
// → Persist to PostgreSQL usage_records table
// → DELETE Redis keys
```

### 3.6 Storage Backend

Admin service lưu account/user/key metadata trong **PostgreSQL** (không chỉ Redis):

```sql
-- services/openviking-admin/internal/infra/migrations/001_init.up.sql

CREATE TABLE IF NOT EXISTS accounts (
    id                VARCHAR NOT NULL PRIMARY KEY,
    name              VARCHAR NOT NULL,
    config            JSONB   NOT NULL DEFAULT '{}',
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id         UUID        NOT NULL,
    account_id VARCHAR     NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    meta       JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, account_id)
);

CREATE TABLE IF NOT EXISTS api_keys (
    id           UUID        NOT NULL PRIMARY KEY,
    account_id   VARCHAR     NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id      UUID,
    role         INTEGER     NOT NULL,
    name         VARCHAR     NOT NULL,
    key_hash     BYTEA       NOT NULL,
    key_prefix   VARCHAR(8)  NOT NULL,
    is_active    BOOLEAN     NOT NULL DEFAULT true,
    expires_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_api_keys_account ON api_keys(account_id);
```

---

## 4. Testing Strategy

### Unit Tests — Crypto
- `TestEncryptDecrypt_RoundTrip` — `Encrypt(plain)` → `Decrypt()` → same plain
- `TestOVE1Header_ParseAndReassemble` — binary format stability
- `TestIsOVE1_ValidMagic` — bytes starting "OVE1" → true
- `TestIsOVE1_InvalidMagic` — plaintext → false
- `TestKeyRotation_ContentUnchanged` — re-wrap: ciphertext bytes identical, but EFK different
- `TestPassthroughEncrypt_ReturnsPlaintext`
- `TestKMSLocal_LoadKeyFile` — read key from file

### Unit Tests — Admin
- `TestCreateAPIKey_PrefixStoredNotPlaintext` — repo.Save never called with plaintext
- `TestResolveAPIKey_bcryptVerification` — correct key → resolved; wrong key → ErrUnauthenticated
- `TestResolveAPIKey_InactiveKey` — revoked key → ErrUnauthenticated
- `TestCreateAccount_NATSPublished` — event published with correct payload
- `TestDeleteAccount_CascadeNATSPublished`
- `TestAggregateHealth_AllHealthy` — all services return SERVING → status=serving
- `TestAggregateHealth_OneDown` — 1 service error → status=degraded
- `TestAggregateHealth_Timeout` — context timeout 5s → returns partial results

### Integration Tests
- `TestEncryptDecryptE2E_VaultKMS` — real Vault in testcontainer
- `TestAccountLifecycleE2E` — create → list → delete, verify NATS events fired

---

## 5. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| Root key file lost → all data inaccessible | Cao | Backup root.key securely; document recovery procedure; recommend Vault for production |
| bcrypt cost=12 throttles key creation | Thấp | Key creation is rare; bcrypt slow is INTENDED for security |
| API key resolution bottleneck (many uncached requests) | Trung bình | Redis cache TTL=5min reduces bcrypt to <1% of requests |
| NATS publish fail on account create/delete | Trung bình | Transactional outbox: write NATS payload to DB, publish async |
| OVE1 format not binary-compatible với Python | Trung bình | Write unit test khớp với Python test vectors từ Python implementation |
| Key rotation fails mid-way (partial re-wrap) | Trung bình | Track rotation progress in DB; idempotent rewrapFile; resume-safe |
