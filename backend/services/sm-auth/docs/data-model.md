---
id: DOC-S04
service: sm-auth
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-auth — Data Model

> **Database**: PostgreSQL

## Tables

### api_keys

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Key ID |
| org_id | VARCHAR(36) | NOT NULL, INDEX | Organization |
| name | VARCHAR(255) | NOT NULL | Display name |
| key_hash | VARCHAR(64) | NOT NULL, UNIQUE | SHA-256 hash |
| prefix | VARCHAR(10) | DEFAULT 'sm_' | Key prefix |
| permissions | TEXT[] | | Allowed operations |
| expires_at | TIMESTAMPTZ | | Expiry (null=never) |
| revoked_at | TIMESTAMPTZ | | Revocation timestamp |
| last_used_at | TIMESTAMPTZ | | Last usage |
| created_at | TIMESTAMPTZ | NOT NULL | |

### organizations

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Org ID |
| name | VARCHAR(255) | NOT NULL | Org name |
| subscription_tier | VARCHAR(20) | | api_pro/api_scale/api_enterprise |
| metadata | JSONB | | Extra config |
| created_at | TIMESTAMPTZ | NOT NULL | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

### org_members

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| org_id | UUID | FK → organizations, PK | |
| user_id | VARCHAR(36) | PK | |
| role | VARCHAR(10) | NOT NULL | owner/admin/member |
| created_at | TIMESTAMPTZ | NOT NULL | |

### waitlist

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| org_id | UUID | PK | |
| in_waitlist | BOOL | DEFAULT true | |
| access_granted | BOOL | DEFAULT false | |
| created_at | TIMESTAMPTZ | NOT NULL | |

## Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_key_hash | key_hash (UNIQUE) | Fast key validation |
| idx_key_org | org_id | List by org |
| idx_member_user | user_id | User's orgs lookup |
