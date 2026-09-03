# TASK-BE-001 — PostgreSQL Migrations: `users` + `refresh_tokens` tables

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-001 |
| **Layer** | Backend — Go / PostgreSQL |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-002 §2.3](../solutions/SOL-002-Auth-Solution.md) |
| **Priority** | 🔴 P0 — Critical |
| **Depends On** | — |
| **Estimated** | 1h |

---

## Context

Auth system cần bảng `users` (login, bcrypt hash) và `refresh_tokens` (JWT rotation). DB schema cần được tạo trước khi implement handler.

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `vnp-platform/migrations/0001_create_users.sql` |
| CREATE | `vnp-platform/migrations/0002_create_refresh_tokens.sql` |

---

## Implementation

### File: `vnp-platform/migrations/0001_create_users.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL,
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT,                        -- bcrypt hash, NULL for SSO-only users
    role          TEXT        NOT NULL DEFAULT 'user',
                                               -- admin | editor | viewer | user
    tenant_id     UUID        NOT NULL,
    avatar_url    TEXT,
    is_active     BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email     ON users(email);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- +migrate Down
DROP TABLE IF EXISTS users;
```

### File: `vnp-platform/migrations/0002_create_refresh_tokens.sql`

```sql
-- +migrate Up
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash    TEXT        UNIQUE NOT NULL,  -- SHA-256 of raw refresh token
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked       BOOLEAN     NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);

-- +migrate Down
DROP TABLE IF EXISTS refresh_tokens;
```

---

## Verification

```bash
# Chạy migrations
cd vnp-platform
go run ./cmd/migrate/main.go up

# Verify tables tồn tại
psql $DATABASE_URL -c "\d users"
psql $DATABASE_URL -c "\d refresh_tokens"
```

**Expected**: Cả hai tables xuất hiện với đúng columns và indexes.
