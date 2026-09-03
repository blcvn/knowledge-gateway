---
id: DOC-S04
service: sm-connector
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-connector — Data Model

> **Database**: PostgreSQL

## Tables

### connections

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | Connection ID |
| provider | VARCHAR(20) | NOT NULL | notion/google-drive/onedrive |
| org_id | VARCHAR(36) | NOT NULL, INDEX | Organization |
| user_id | VARCHAR(36) | NOT NULL | Connection owner |
| email | VARCHAR(255) | | Provider email |
| document_limit | INT | DEFAULT 10000 | Max docs to sync |
| container_tags | TEXT[] | | Scope tags |
| access_token | BYTEA | | Encrypted OAuth token |
| refresh_token | BYTEA | | Encrypted refresh token |
| expires_at | TIMESTAMPTZ | | Token expiry |
| metadata | JSONB | NOT NULL, DEFAULT '{}' | Provider-specific |
| created_at | TIMESTAMPTZ | NOT NULL | |

### connection_states

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| state_token | VARCHAR(64) | PK | OAuth state |
| provider | VARCHAR(20) | NOT NULL | |
| org_id | VARCHAR(36) | NOT NULL | |
| user_id | VARCHAR(36) | NOT NULL | |
| connection_id | UUID | | Pre-allocated |
| document_limit | INT | DEFAULT 10000 | |
| redirect_url | TEXT | | Callback URL |
| metadata | JSONB | | |
| container_tags | TEXT[] | | |
| created_at | TIMESTAMPTZ | NOT NULL | |
| expires_at | TIMESTAMPTZ | | TTL (10 min) |

### sync_history

| Column | Type | Constraints | Description |
|--------|------|-------------|-------------|
| id | UUID | PK | |
| connection_id | UUID | FK → connections | |
| status | VARCHAR(20) | | idle/syncing/completed/failed |
| documents_synced | INT | DEFAULT 0 | |
| started_at | TIMESTAMPTZ | | |
| completed_at | TIMESTAMPTZ | | |
| error | TEXT | | Error details |

## Index Strategy

| Index | Columns | Purpose |
|-------|---------|---------|
| idx_conn_org | org_id | List by org |
| idx_conn_provider | (org_id, provider) | List by provider |
| idx_state_expiry | (expires_at) | State cleanup |
| idx_sync_conn | connection_id | Sync history |
