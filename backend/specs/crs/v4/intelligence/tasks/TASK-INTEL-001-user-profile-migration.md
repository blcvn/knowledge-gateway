# TASK-INTEL-001 — User Profile Assembly: tạo database schema cho user_profiles

| Field | Value |
|---|---|
| **Task ID** | TASK-INTEL-001 |
| **Wave** | 1 |
| **Solution** | [SOL-INTEL-001](../solutions/SOL-INTEL-001-User-Profile-Assembly.md) §2 |
| **Component** | `deployment/dev/migrations/` |
| **Priority** | 🔴 Critical |
| **Depends On** | — |
| **Estimated** | 1h |

---

## Mục tiêu

User Profile Assembly: tạo database schema cho user_profiles

---

## Công việc cụ thể

### `deployment/dev/migrations/0041_user_profiles.sql` [NEW]

```sql
-- +migrate Up

CREATE TABLE IF NOT EXISTS user_profiles (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    TEXT        NOT NULL,
    user_id      TEXT        NOT NULL,
    categories   JSONB       NOT NULL DEFAULT '{}',
    -- categories format: {"preferences": [...], "skills": [...], "work_context": [...]}
    summary      TEXT,                           -- LLM-generated profile summary
    version      INTEGER     NOT NULL DEFAULT 1,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_profiles_tenant_user ON user_profiles(tenant_id, user_id);

-- Profile history (for audit/rollback)
CREATE TABLE IF NOT EXISTS user_profile_versions (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id   UUID        NOT NULL REFERENCES user_profiles(id),
    version      INTEGER     NOT NULL,
    categories   JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +migrate Down
DROP TABLE IF EXISTS user_profile_versions;
DROP TABLE IF EXISTS user_profiles;
```

---

## Acceptance Criteria

- [ ] `user_profiles` table with UNIQUE(tenant_id, user_id)
- [ ] `user_profile_versions` for history tracking
- [ ] Indexes on tenant_id, user_id
- [ ] Migration runs cleanly

## Files

```
deployment/dev/migrations/0041_user_profiles.sql  [NEW]
```
