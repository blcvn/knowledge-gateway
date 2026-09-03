# TASK-PLAT-002 — API Key & Audit DB Migration

| Field | Value |
|---|---|
| **Task ID** | TASK-PLAT-002 |
| **Wave** | 1 (Foundation) |
| **Solution** | [SOL-PLAT-001](../solutions/SOL-PLAT-001-Auth-API-Key-JWT.md) §2.5 |
| **Component** | `deployment/dev/migrations/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 1h |

**Trạng thái:** ⏳ Pending  
**Ghi chú audit:** No api_keys DB migration found in deployment/dev/migrations/
---

## Mục tiêu

Tạo SQL migrations cho bảng `api_keys` và `api_key_audit_log`.

---

## Công việc cụ thể

### 1. Tạo migration file `deployment/dev/migrations/XXX_api_keys_audit.up.sql` [NEW]

> Thay XXX bằng sequence number tiếp theo sau migration cuối cùng hiện có.

```sql
-- API Keys table
-- Stores API keys with prefix visible, secret hashed (SHA-256), never re-exposed
CREATE TABLE IF NOT EXISTS api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    user_id     TEXT NOT NULL,
    prefix      TEXT NOT NULL,       -- vnp_{prefix} — safe to log
    secret_hash TEXT NOT NULL,       -- SHA-256(raw_token)
    name        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'revoked', 'expired', 'rotated')),
    expires_at  TIMESTAMPTZ,
    rotated_to  UUID REFERENCES api_keys(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(prefix);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(tenant_id, status);

-- Audit log — append-only, never updated/deleted
CREATE TABLE IF NOT EXISTS api_key_audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_id      UUID NOT NULL,
    tenant_id   TEXT NOT NULL,
    action      TEXT NOT NULL
                    CHECK (action IN ('created', 'revoked', 'rotated', 'expired')),
    actor_id    TEXT,
    ip_address  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_key_id  ON api_key_audit_log(key_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_tenant  ON api_key_audit_log(tenant_id, created_at DESC);
```

### 2. Tạo down migration `deployment/dev/migrations/XXX_api_keys_audit.down.sql` [NEW]

```sql
DROP TABLE IF EXISTS api_key_audit_log;
DROP TABLE IF EXISTS api_keys;
```

### 3. Implement `services/vnp-platform/internal/adapter/postgres/apikey_repo.go` [NEW]

```go
package postgres

type APIKeyRepo struct {
    db *pgxpool.Pool
}

func (r *APIKeyRepo) Insert(ctx context.Context, key *domain.APIKey) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO api_keys
            (id, tenant_id, user_id, prefix, secret_hash, name, status, expires_at, created_at, updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
        key.ID, key.TenantID, key.UserID, key.Prefix, key.SecretHash,
        key.Name, key.Status, key.ExpiresAt, key.CreatedAt, key.UpdatedAt,
    )
    return err
}

func (r *APIKeyRepo) FindByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error) {
    var key domain.APIKey
    err := r.db.QueryRow(ctx,
        `SELECT id, tenant_id, user_id, prefix, secret_hash, name, status, expires_at, rotated_to, created_at, updated_at
         FROM api_keys WHERE prefix = $1`, prefix,
    ).Scan(&key.ID, &key.TenantID, &key.UserID, &key.Prefix, &key.SecretHash,
           &key.Name, &key.Status, &key.ExpiresAt, &key.RotatedTo, &key.CreatedAt, &key.UpdatedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, port.ErrNotFound
    }
    return &key, err
}

func (r *APIKeyRepo) ListByTenant(ctx context.Context, tenantID string) ([]*domain.APIKey, error) {
    rows, err := r.db.Query(ctx,
        `SELECT id, tenant_id, user_id, prefix, name, status, expires_at, created_at
         FROM api_keys WHERE tenant_id = $1 AND status != 'rotated'
         ORDER BY created_at DESC`, tenantID,
    )
    if err != nil { return nil, err }
    defer rows.Close()
    // scan rows...
    return keys, nil
}

func (r *APIKeyRepo) Update(ctx context.Context, key *domain.APIKey) error {
    _, err := r.db.Exec(ctx,
        `UPDATE api_keys SET status=$1, rotated_to=$2, updated_at=NOW() WHERE id=$3`,
        key.Status, key.RotatedTo, key.ID,
    )
    return err
}
```

---

## Acceptance Criteria

- [ ] Migration runs without error: `goose up` or `migrate up`
- [ ] `api_keys` table created với correct columns và constraints
- [ ] `api_key_audit_log` table created (append-only, no update/delete rights)
- [ ] Unique constraint trên `prefix` column
- [ ] Down migration drops both tables cleanly
- [ ] `APIKeyRepo.Insert()` + `FindByPrefix()` + `Update()` implemented và build passes

## Files

```
deployment/dev/migrations/XXX_api_keys_audit.up.sql    [NEW]
deployment/dev/migrations/XXX_api_keys_audit.down.sql  [NEW]
services/vnp-platform/internal/adapter/postgres/apikey_repo.go [NEW]
```
