# TASK-PLAT-001 — API Key Domain Model

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-001 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-PLAT-001](../solutions/SOL-PLAT-001-Auth-API-Key-JWT.md) §2.1 |
| **Component** | `services/vnp-platform/internal/domain/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 2h |

---

## Mục tiêu

Tạo domain model `APIKey` và `APIKeyAuditEvent` với logic `Generate()` và `Verify()`.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/domain/apikey.go` [NEW]

```go
package domain

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "time"
)

type APIKeyStatus string

const (
    APIKeyStatusActive  APIKeyStatus = "active"
    APIKeyStatusRevoked APIKeyStatus = "revoked"
    APIKeyStatusExpired APIKeyStatus = "expired"
    APIKeyStatusRotated APIKeyStatus = "rotated"
)

type APIKey struct {
    ID         string
    TenantID   string
    UserID     string
    Prefix     string       // 8-char — visible in logs
    SecretHash string       // SHA-256(raw_token) — never re-exposed
    Name       string
    Status     APIKeyStatus
    ExpiresAt  *time.Time
    CreatedAt  time.Time
    UpdatedAt  time.Time
    RotatedTo  *string
}

type APIKeyAuditEvent struct {
    ID        string
    KeyID     string
    TenantID  string
    Action    string // "created" | "revoked" | "rotated" | "expired"
    ActorID   string
    IP        string
    CreatedAt time.Time
}

// Generate creates a raw token + domain object
// Format: vnp_{8char_prefix}.{32char_secret}
func Generate(tenantID, userID, name string, ttl *time.Duration) (rawToken string, key APIKey, err error) {
    prefixBytes := make([]byte, 6)
    if _, err = rand.Read(prefixBytes); err != nil {
        return
    }
    prefix := base64.RawURLEncoding.EncodeToString(prefixBytes)[:8]

    secretBytes := make([]byte, 24)
    if _, err = rand.Read(secretBytes); err != nil {
        return
    }
    secret := base64.RawURLEncoding.EncodeToString(secretBytes)

    rawToken = fmt.Sprintf("vnp_%s.%s", prefix, secret)

    hash := sha256.Sum256([]byte(rawToken))
    secretHash := base64.StdEncoding.EncodeToString(hash[:])

    var expiresAt *time.Time
    if ttl != nil {
        t := time.Now().UTC().Add(*ttl)
        expiresAt = &t
    }

    key = APIKey{
        TenantID:   tenantID,
        UserID:     userID,
        Prefix:     prefix,
        SecretHash: secretHash,
        Name:       name,
        Status:     APIKeyStatusActive,
        ExpiresAt:  expiresAt,
        CreatedAt:  time.Now().UTC(),
        UpdatedAt:  time.Now().UTC(),
    }
    return
}

// Verify checks raw token against stored hash
func (k *APIKey) Verify(rawToken string) bool {
    hash := sha256.Sum256([]byte(rawToken))
    return base64.StdEncoding.EncodeToString(hash[:]) == k.SecretHash &&
        k.Status == APIKeyStatusActive &&
        (k.ExpiresAt == nil || k.ExpiresAt.After(time.Now().UTC()))
}
```

### 2. Tạo `services/vnp-platform/internal/domain/apikey_test.go` [NEW]

```go
package domain_test

func TestGenerate_Format(t *testing.T) {
    rawToken, key, err := Generate("tenant-1", "user-1", "test-key", nil)
    assert.NoError(t, err)
    assert.True(t, strings.HasPrefix(rawToken, "vnp_"))
    assert.Contains(t, rawToken, ".")
    assert.Equal(t, 8, len(key.Prefix))
    assert.NotEmpty(t, key.SecretHash)
    assert.Nil(t, key.ExpiresAt)
}

func TestVerify_ValidToken(t *testing.T) {
    rawToken, key, _ := Generate("tenant-1", "user-1", "test-key", nil)
    assert.True(t, key.Verify(rawToken))
}

func TestVerify_WrongToken(t *testing.T) {
    _, key, _ := Generate("tenant-1", "user-1", "test-key", nil)
    assert.False(t, key.Verify("vnp_wrongtoken"))
}

func TestVerify_RevokedKey(t *testing.T) {
    rawToken, key, _ := Generate("tenant-1", "user-1", "test-key", nil)
    key.Status = APIKeyStatusRevoked
    assert.False(t, key.Verify(rawToken))
}

func TestVerify_ExpiredKey(t *testing.T) {
    rawToken, key, _ := Generate("tenant-1", "user-1", "test-key", nil)
    past := time.Now().Add(-1 * time.Hour)
    key.ExpiresAt = &past
    assert.False(t, key.Verify(rawToken))
}
```

### 3. Tạo `services/vnp-platform/internal/port/apikey_repository.go` [NEW]

```go
package port

type APIKeyRepository interface {
    Insert(ctx context.Context, key *domain.APIKey) error
    Get(ctx context.Context, id string) (*domain.APIKey, error)
    FindByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
    ListByTenant(ctx context.Context, tenantID string) ([]*domain.APIKey, error)
    Update(ctx context.Context, key *domain.APIKey) error
}

type AuditRepository interface {
    Record(ctx context.Context, event domain.APIKeyAuditEvent) error
    List(ctx context.Context, tenantID string, limit int) ([]*domain.APIKeyAuditEvent, error)
}
```

---

## Acceptance Criteria

- [ ] `Generate()` returns rawToken với format `vnp_{8char}.{32char}`
- [ ] `Verify(rawToken)` returns true với đúng token
- [ ] `Verify("wrong")` returns false
- [ ] Revoked/Expired key → Verify returns false
- [ ] `go test ./services/vnp-platform/internal/domain/...` passes

## Files

```
services/vnp-platform/internal/domain/apikey.go       [NEW]
services/vnp-platform/internal/domain/apikey_test.go  [NEW]
services/vnp-platform/internal/port/apikey_repository.go [NEW]
```
