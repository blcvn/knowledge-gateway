# SOL-PLAT-001 — Solution: Auth Backend — API Key Lifecycle & JWT RS256

| Field | Value |
|---|---|
| **Solution ID** | SOL-PLAT-001 |
| **CR** | [CR-PLAT-001](../../../../docs/crs/v3/platform/CR-PLAT-001-Auth-API-Key-JWT.md) |
| **TDD ref** | [01-gateway.md §2-Auth](../../../tdd/architecture/01-gateway.md) · [08-platform-services.md §VNP-Admin](../../../tdd/architecture/08-platform-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |

---

## 1. Phân tích kiến trúc

Theo TDD `01-gateway.md §2`, auth middleware hiện hỗ trợ **X-API-Key** và **Bearer JWT** nhưng thiếu:
- Full API key lifecycle (rotation, expiry, immutable audit trail)
- RSA-2048 / RS256 JWT signing (hiện dùng HMAC)
- JWK endpoint để public key discovery
- Dev mode guard (chỉ localhost)

**Các service liên quan:**
- `gateway/internal/infra/middleware/auth.go` — enforce auth
- `services/vnp-platform` (hoặc `vnp-admin`) — gRPC backend quản lý keys
- `shared/pkg/resilience` — circuit breaker wrap gRPC calls

---

## 2. Giải pháp

### 2.1 `services/vnp-platform/internal/domain/apikey.go` [NEW]

```go
package domain

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "time"
)

// APIKey format: vnp_{prefix}.{secret}
// prefix:  8-char alphanumeric (safe to log)
// secret:  32-char random (hashed with SHA-256 before storage)

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
    Prefix     string       // vnp_{prefix} — visible in logs
    SecretHash string       // SHA-256(secret) — stored, never re-exposed
    Name       string       // human-readable label
    Status     APIKeyStatus
    ExpiresAt  *time.Time   // nil = no expiry
    CreatedAt  time.Time
    UpdatedAt  time.Time
    RotatedTo  *string      // points to new key ID after rotation
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

// Generate creates a new raw API key and returns the visible token + domain object
func Generate(tenantID, userID, name string, ttl *time.Duration) (rawToken string, key APIKey, err error) {
    // prefix: 8 random bytes → base64url → take 8 chars
    prefixBytes := make([]byte, 6)
    if _, err = rand.Read(prefixBytes); err != nil {
        return
    }
    prefix := base64.RawURLEncoding.EncodeToString(prefixBytes)[:8]

    // secret: 24 random bytes → base64url → 32 chars
    secretBytes := make([]byte, 24)
    if _, err = rand.Read(secretBytes); err != nil {
        return
    }
    secret := base64.RawURLEncoding.EncodeToString(secretBytes)

    rawToken = fmt.Sprintf("vnp_%s.%s", prefix, secret)

    // hash the secret
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
    }
    return
}

// Verify checks if a raw token matches the stored hash
func (k *APIKey) Verify(rawToken string) bool {
    hash := sha256.Sum256([]byte(rawToken))
    return base64.StdEncoding.EncodeToString(hash[:]) == k.SecretHash &&
        k.Status == APIKeyStatusActive &&
        (k.ExpiresAt == nil || k.ExpiresAt.After(time.Now().UTC()))
}
```

### 2.2 `services/vnp-platform/internal/usecase/apikey_uc.go` [NEW]

```go
package usecase

// APIKeyUseCase implements full lifecycle: Create, List, Revoke, Rotate
type APIKeyUseCase struct {
    repo  port.APIKeyRepository
    audit port.AuditRepository
    nats  port.NATSPublisher
}

func (uc *APIKeyUseCase) Create(ctx context.Context, req CreateKeyRequest) (string, *domain.APIKey, error) {
    rawToken, key, err := domain.Generate(req.TenantID, req.UserID, req.Name, req.TTL)
    if err != nil {
        return "", nil, err
    }
    if err = uc.repo.Insert(ctx, &key); err != nil {
        return "", nil, err
    }
    // Emit immutable audit event
    uc.audit.Record(ctx, domain.APIKeyAuditEvent{
        KeyID: key.ID, TenantID: key.TenantID, Action: "created", ActorID: req.ActorID,
    })
    return rawToken, &key, nil // rawToken: only exposed ONCE on creation
}

func (uc *APIKeyUseCase) Revoke(ctx context.Context, keyID, actorID string) error {
    key, err := uc.repo.Get(ctx, keyID)
    if err != nil { return err }
    key.Status = domain.APIKeyStatusRevoked
    if err = uc.repo.Update(ctx, key); err != nil { return err }
    uc.audit.Record(ctx, domain.APIKeyAuditEvent{
        KeyID: keyID, TenantID: key.TenantID, Action: "revoked", ActorID: actorID,
    })
    return nil
}

func (uc *APIKeyUseCase) Rotate(ctx context.Context, oldKeyID, actorID string) (string, *domain.APIKey, error) {
    oldKey, err := uc.repo.Get(ctx, oldKeyID)
    if err != nil { return "", nil, err }

    // Create new key
    rawToken, newKey, err := domain.Generate(oldKey.TenantID, oldKey.UserID, oldKey.Name+"_rotated", nil)
    if err != nil { return "", nil, err }
    if err = uc.repo.Insert(ctx, &newKey); err != nil { return "", nil, err }

    // Invalidate old key
    oldKey.Status = domain.APIKeyStatusRotated
    oldKey.RotatedTo = &newKey.ID
    uc.repo.Update(ctx, oldKey)

    uc.audit.Record(ctx, domain.APIKeyAuditEvent{
        KeyID: oldKeyID, TenantID: oldKey.TenantID, Action: "rotated", ActorID: actorID,
    })
    return rawToken, &newKey, nil
}
```

### 2.3 JWT RS256 — `gateway/internal/infra/middleware/jwt_rsa.go` [MODIFY]

```go
package middleware

import (
    "crypto/rsa"
    "github.com/golang-jwt/jwt/v5"
)

// JWTValidator validates RS256 tokens using rotating RSA public keys
type JWTValidator struct {
    publicKeys []*rsa.PublicKey  // support multiple active keys for rotation
}

type VNPClaims struct {
    jwt.RegisteredClaims
    TenantID string   `json:"tenant_id"`
    Roles    []string `json:"roles"`
    Scopes   []string `json:"scopes"`
    RateTier string   `json:"rate_tier"`
}

func (v *JWTValidator) Validate(tokenStr string) (*VNPClaims, error) {
    var lastErr error
    for _, pubKey := range v.publicKeys {
        token, err := jwt.ParseWithClaims(tokenStr, &VNPClaims{}, func(t *jwt.Token) (interface{}, error) {
            if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
            }
            return pubKey, nil
        })
        if err != nil {
            lastErr = err
            continue
        }
        if claims, ok := token.Claims.(*VNPClaims); ok && token.Valid {
            return claims, nil
        }
    }
    return nil, lastErr
}

// JWKSHandler serves GET /.well-known/jwks.json
func JWKSHandler(privateKey *rsa.PrivateKey) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        jwks := buildJWKS(&privateKey.PublicKey)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(jwks)
    }
}
```

### 2.4 Dev Mode Guard — `gateway/internal/infra/middleware/devmode.go` [NEW]

```go
package middleware

// DevModeGuard: allows bypass only from loopback (127.0.0.1, ::1)
func DevModeGuard(devMode bool) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        if !devMode {
            return next
        }
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            host, _, _ := net.SplitHostPort(r.RemoteAddr)
            if host != "127.0.0.1" && host != "::1" {
                http.Error(w, "dev mode only accepts localhost", http.StatusForbidden)
                return
            }
            // Inject mock auth context
            ctx := context.WithValue(r.Context(), authCtxKey, &AuthContext{
                TenantID: "dev-tenant",
                UserID:   "dev-user",
                Roles:    []string{"admin"},
                RateTier: "enterprise",
            })
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### 2.5 DB Migration — `deployment/dev/migrations/xxx_api_keys_audit.up.sql` [NEW]

```sql
-- API Keys table
CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    prefix      TEXT NOT NULL,       -- vnp_{prefix} — loggable
    secret_hash TEXT NOT NULL,       -- SHA-256(raw_token) — never exposed
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active', -- active|revoked|expired|rotated
    expires_at  TIMESTAMPTZ,
    rotated_to  UUID REFERENCES api_keys(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE UNIQUE INDEX idx_api_keys_prefix ON api_keys(prefix);

-- Audit log (append-only)
CREATE TABLE api_key_audit_log (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id     UUID NOT NULL,
    tenant_id  TEXT NOT NULL,
    action     TEXT NOT NULL,   -- created|revoked|rotated|expired
    actor_id   TEXT,
    ip_address TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_key_id ON api_key_audit_log(key_id);
CREATE INDEX idx_audit_log_tenant ON api_key_audit_log(tenant_id);
```

---

## 3. File Changes

| File | Action | Mô tả |
|---|---|---|
| `services/vnp-platform/internal/domain/apikey.go` | NEW | APIKey domain model + Generate/Verify |
| `services/vnp-platform/internal/usecase/apikey_uc.go` | NEW | Create, List, Revoke, Rotate use cases |
| `services/vnp-platform/internal/adapter/grpc/apikey_handler.go` | NEW | gRPC handler for key CRUD |
| `gateway/internal/infra/middleware/jwt_rsa.go` | MODIFY | Switch from HMAC to RSA-2048 RS256 |
| `gateway/internal/infra/middleware/devmode.go` | NEW | Localhost-only dev mode guard |
| `gateway/adapter/handler/auth.go` | MODIFY | Add key lifecycle endpoints |
| `gateway/adapter/handler/router.go` | MODIFY | Register `GET /.well-known/jwks.json` |
| `deployment/dev/migrations/xxx_api_keys_audit.up.sql` | NEW | api_keys + audit_log tables |

---

## 4. API Endpoints

| Method | Path | Handler | Description |
|---|---|---|---|
| `POST` | `/v1/auth/login` | `auth.Login` | Email/password → JWT RS256 |
| `POST` | `/v1/auth/refresh` | `auth.Refresh` | Refresh token → new JWT |
| `POST` | `/v1/auth/logout` | `auth.Logout` | Revoke refresh token |
| `GET` | `/.well-known/jwks.json` | `auth.JWKSHandler` | JWK Set (public key discovery) |
| `GET` | `/v1/console/sdk/keys` | `auth.ListKeys` | List API keys (prefix visible, secret hidden) |
| `POST` | `/v1/console/sdk/keys` | `auth.CreateKey` | Create key → rawToken exposed ONCE |
| `DELETE` | `/v1/console/sdk/keys/{id}` | `auth.RevokeKey` | Revoke key |
| `POST` | `/v1/console/sdk/keys/{id}/rotate` | `auth.RotateKey` | Rotate: new key + invalidate old |

---

## 5. Acceptance Criteria

- [ ] API key format: `vnp_{8char_prefix}.{32char_secret}` — prefix loggable, secret never re-exposed
- [ ] Secret stored as SHA-256 hash, raw token returned only at creation
- [ ] JWT RS256: signed with RSA-2048 private key, verified with public key
- [ ] `GET /.well-known/jwks.json` serves current active public keys (JWK format)
- [ ] Key rotation: creates new key, marks old as `rotated`, points to new key ID
- [ ] Audit log: immutable append-only for every create/revoke/rotate/expire
- [ ] Dev mode (`AUTH_DEV_MODE=true`): bypasses auth ONLY for localhost (127.0.0.1 or ::1)
- [ ] Expired keys: background job marks status=`expired` after TTL

---

## 6. Dependencies

- `services/vnp-platform` deployed with PostgreSQL access
- `shared/pkg/resilience` circuit breaker wrapping gRPC key lookup
- RSA-2048 keypair generated at deploy time and stored in environment/secrets
