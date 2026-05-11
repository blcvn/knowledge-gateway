---
id: TASK-CON-003
title: Data Models & Repositories
service: sm-connector
status: Done
priority: P0
created: 2026-05-11
---

# Data Models & Repositories

## Objective
Implement the storage and persistence adapters.

## Specs Mapping
Please refer to the following content from `specs/tdd.md` to implement this task:

```markdown
---
id: TDD-sm-connector
title: Technical Design — sm-connector
service: sm-connector
version: 1.1.0
status: Ready
created: 2026-05-09
updated: 2026-05-10
group: Supermemory
---

# Technical Design — sm-connector

> **Group**: Supermemory | **gRPC Port**: 9075 | **Health Port**: 9120

## 1. Service Overview

External data source sync: OAuth2 connection management for Google Drive, Notion, OneDrive. Incremental sync with document limit enforcement and batch ingestion via NATS.

## 2. Domain Layer

- **Connection**: id, provider (notion|google-drive|onedrive), org_id, user_id, email, document_limit (default 10000), container_tags[], access_token (encrypted), refresh_token (encrypted), expires_at, metadata (JSONB), created_at
- **ConnectionState**: state_token, provider, org_id, user_id, connection_id, document_limit, redirect_url, metadata, container_tags[], created_at, expires_at
- **ConnectionProvider**: enum — `notion` | `google-drive` | `onedrive`
- **SyncStatus**: connection_id, status (idle|syncing|completed|failed), documents_synced, last_sync_at, error

## 3. gRPC API

```protobuf
service SmConnectorService {
  rpc CreateConnection(CreateConnectionRequest) returns (ConnectionResponse);
  rpc SyncConnection(SyncConnectionRequest) returns (SyncResponse);
  rpc GetSyncStatus(GetStatusRequest) returns (SyncStatus);
  rpc GetConnection(GetConnectionRequest) returns (ConnectionResponse);
  rpc ListConnections(ListConnectionsRequest) returns (ListConnectionsResponse);
  rpc DeleteConnection(DeleteConnectionRequest) returns (DeleteConnectionResponse);
}
```

## 4. Auth & Sync Algorithms

### OAuth2 State Management
- `state_token` is generated as a secure random hex and stored in Redis/Postgres with a short TTL (10m).
- Protects against CSRF during the OAuth callback phase.
- Contains the `redirect_url` and `document_limit` requested by the client.

### Incremental Sync Strategy (Cursor-based)
- Maintains a `last_sync_cursor` or `last_sync_timestamp` for each connection.
- During periodic syncs, only queries documents modified after the cursor.
- Handles provider pagination explicitly (e.g. `next_cursor` in Notion, `nextPageToken` in Google Drive).
- Batches ingestion by publishing `sm.connection.synced` to NATS (e.g. 100 docs per event) to prevent overwhelming `sm-engine`.

## 5. NATS Events

| Direction | Subject | Payload |
|-----------|---------|---------|
| Publish | `sm.connection.synced` | `{connection_id, org_id, document_urls[], provider}` |

## 6. Data Model

### Tables
- `connections`: id(PK), provider, org_id, user_id, email, document_limit, container_tags(TEXT[]), access_token(BYTEA encrypted), refresh_token(BYTEA encrypted), expires_at, metadata(JSONB), created_at
- `connection_states`: state_token(PK), provider, org_id, user_id, connection_id, document_limit, redirect_url, metadata(JSONB), container_tags(TEXT[]), created_at, expires_at
- `sync_history`: id(PK), connection_id(FK), status, documents_synced, started_at, completed_at, error

### Key Indexes
- `idx_conn_org` (org_id) — list by org
- `idx_conn_provider` (org_id, provider) — list by provider
- `idx_state_token` (state_token) — OAuth callback lookup (TTL: 10min)

## 7. Observability

- **Metrics**: connections_total, sync_total, sync_duration_seconds, documents_synced_total, oauth_errors_total
- **Health**: gRPC + HTTP /healthz on port 9120

---

> **Next Steps**: FEAT-001 (Google Drive OAuth + Sync), FEAT-002 (Notion OAuth + Sync), FEAT-003 (OneDrive OAuth + Sync), SEC-001 (Token Encryption)

## Task Specs Registry

_To be populated during implementation._

| ID | Title | Status | Priority |
|----|-------|--------|----------|
| TASK-CON-001 | Implement Domain Models | Pending | P0 |
| TASK-CON-002 | Implement Usecases | Pending | P0 |
| TASK-CON-003 | Implement Adapters and Repositories | Pending | P0 |
| TASK-CON-004 | Infrastructure and Telemetry setup | Pending | P1 |

```

## Acceptance Criteria
- [x] Database schema / migrations created.
- [x] Repository implementations accurately query the data models.
