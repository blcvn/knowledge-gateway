# TASK-PLAT-005 — API Key Use Cases (Create / Revoke / Rotate / List)

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-005 |
| **Wave** | 2 (Auth Flows) |
| **Solution** | [SOL-PLAT-001](../solutions/SOL-PLAT-001-Auth-API-Key-JWT.md) §2.2 |
| **Component** | `services/vnp-platform/internal/usecase/` |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-PLAT-001, TASK-PLAT-002 |
| **Estimated** | 4h |

---

## Mục tiêu

Implement use case layer cho API key lifecycle: Create, List, Revoke, Rotate. Mỗi operation phải ghi audit log.

---

## Công việc cụ thể

### 1. Tạo `services/vnp-platform/internal/usecase/apikey_uc.go` [NEW]

```go
package usecase

type APIKeyUseCase struct {
    repo  port.APIKeyRepository
    audit port.AuditRepository
}

type CreateKeyRequest struct {
    TenantID string
    UserID   string
    Name     string
    TTL      *time.Duration // nil = no expiry
    ActorID  string
    IP       string
}

// Create generates new key, stores hashed, returns rawToken ONCE
func (uc *APIKeyUseCase) Create(ctx context.Context, req CreateKeyRequest) (rawToken string, key *domain.APIKey, err error) {
    rawToken, newKey, err := domain.Generate(req.TenantID, req.UserID, req.Name, req.TTL)
    if err != nil {
        return "", nil, fmt.Errorf("generate api key: %w", err)
    }
    newKey.ID = uuid.NewString()
    if err = uc.repo.Insert(ctx, &newKey); err != nil {
        return "", nil, fmt.Errorf("insert api key: %w", err)
    }
    uc.audit.Record(ctx, domain.APIKeyAuditEvent{
        ID: uuid.NewString(), KeyID: newKey.ID, TenantID: newKey.TenantID,
        Action: "created", ActorID: req.ActorID, IP: req.IP, CreatedAt: time.Now().UTC(),
    })
    return rawToken, &newKey, nil
}

// List returns all non-rotated keys for tenant (secret never exposed)
func (uc *APIKeyUseCase) List(ctx context.Context, tenantID string) ([]*domain.APIKey, error) {
    return uc.repo.ListByTenant(ctx, tenantID)
}

// Revoke marks a key as revoked
func (uc *APIKeyUseCase) Revoke(ctx context.Context, keyID, actorID, ip string) error {
    key, err := uc.repo.Get(ctx, keyID)
    if err != nil { return fmt.Errorf("get api key: %w", err) }
    if key.Status != domain.APIKeyStatusActive {
        return fmt.Errorf("key is not active (status=%s)", key.Status)
    }
    key.Status = domain.APIKeyStatusRevoked
    key.UpdatedAt = time.Now().UTC()
    if err = uc.repo.Update(ctx, key); err != nil { return err }
    uc.audit.Record(ctx, domain.APIKeyAuditEvent{
        ID: uuid.NewString(), KeyID: keyID, TenantID: key.TenantID,
        Action: "revoked", ActorID: actorID, IP: ip, CreatedAt: time.Now().UTC(),
    })
    return nil
}

// Rotate: creates new key + invalidates old (linked via rotated_to)
func (uc *APIKeyUseCase) Rotate(ctx context.Context, oldKeyID, actorID, ip string) (rawToken string, newKey *domain.APIKey, err error) {
    oldKey, err := uc.repo.Get(ctx, oldKeyID)
    if err != nil { return "", nil, fmt.Errorf("get old key: %w", err) }

    rawToken, generated, err := domain.Generate(oldKey.TenantID, oldKey.UserID, oldKey.Name+" (rotated)", nil)
    if err != nil { return "", nil, err }
    generated.ID = uuid.NewString()
    if err = uc.repo.Insert(ctx, &generated); err != nil { return "", nil, err }

    // Link old key to new
    oldKey.Status = domain.APIKeyStatusRotated
    oldKey.RotatedTo = &generated.ID
    oldKey.UpdatedAt = time.Now().UTC()
    uc.repo.Update(ctx, oldKey)

    uc.audit.Record(ctx, domain.APIKeyAuditEvent{
        ID: uuid.NewString(), KeyID: oldKeyID, TenantID: oldKey.TenantID,
        Action: "rotated", ActorID: actorID, IP: ip, CreatedAt: time.Now().UTC(),
    })
    return rawToken, &generated, nil
}

// AuthenticateAPIKey validates a raw API key token (from X-API-Key header)
// Returns APIKey domain object if valid
func (uc *APIKeyUseCase) Authenticate(ctx context.Context, rawToken string) (*domain.APIKey, error) {
    // Extract prefix from "vnp_{prefix}.{secret}"
    parts := strings.SplitN(strings.TrimPrefix(rawToken, "vnp_"), ".", 2)
    if len(parts) != 2 {
        return nil, fmt.Errorf("invalid api key format")
    }
    prefix := parts[0]

    key, err := uc.repo.FindByPrefix(ctx, prefix)
    if err != nil { return nil, port.ErrNotFound }
    if !key.Verify(rawToken) {
        return nil, fmt.Errorf("invalid or expired api key")
    }
    return key, nil
}
```

### 2. Unit tests `services/vnp-platform/internal/usecase/apikey_uc_test.go` [NEW]

```go
func TestCreate_ReturnsRawTokenOnce(t *testing.T) {
    repo := &mockAPIKeyRepo{}
    audit := &mockAuditRepo{}
    uc := &APIKeyUseCase{repo: repo, audit: audit}

    rawToken, key, err := uc.Create(ctx, CreateKeyRequest{
        TenantID: "t1", UserID: "u1", Name: "my-key",
    })
    assert.NoError(t, err)
    assert.True(t, strings.HasPrefix(rawToken, "vnp_"))
    assert.NotEmpty(t, key.ID)
    assert.Empty(t, key.SecretHash[:3]) // hash present but raw not re-exposed via domain key
    assert.Equal(t, 1, len(audit.events)) // audit recorded
}

func TestRevoke_NonActiveKey_Error(t *testing.T) {
    repo := &mockAPIKeyRepo{key: &domain.APIKey{Status: domain.APIKeyStatusRevoked}}
    uc := &APIKeyUseCase{repo: repo, audit: &mockAuditRepo{}}
    err := uc.Revoke(ctx, "key-id", "actor", "ip")
    assert.Error(t, err)
}
```

---

## Acceptance Criteria

- [ ] `Create()` returns rawToken once; rawToken not stored in DB (only hash)
- [ ] `List()` returns keys without SecretHash exposed in response
- [ ] `Revoke()` marks status=revoked, records audit event
- [ ] `Rotate()` creates new key, marks old as status=rotated, links via rotated_to
- [ ] `Authenticate()` validates `vnp_{prefix}.{secret}` format, hashes and compares
- [ ] Every operation records an audit event
- [ ] `go test ./services/vnp-platform/internal/usecase/...` passes

## Files

```
services/vnp-platform/internal/usecase/apikey_uc.go       [NEW]
services/vnp-platform/internal/usecase/apikey_uc_test.go  [NEW]
```
