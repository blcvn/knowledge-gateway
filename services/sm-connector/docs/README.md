---
id: DOC-S01
service: sm-connector
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
owner: VNP Memory — Supermemory Team
---

# sm-connector

> **Group**: Supermemory | **gRPC Port**: 9075 | **Health Port**: 9120 | **Origin**: Supermemory

## Purpose

External data source synchronization service. Manages **OAuth2 flows**, **incremental sync**, and **document limit enforcement** for Google Drive, Notion, and OneDrive connectors. Handles connection lifecycle including token refresh, state management, and batch document ingestion.

### Business Capability

- **Connection CRUD**: Create/list/get/delete connections per provider
- **OAuth2 Flow**: State token generation, redirect handling, token exchange + refresh
- **Incremental Sync**: Detect changes since last sync, ingest new/updated documents
- **Document Limit**: Per-connection document count limits (default: 10,000)
- **Container Tag Scoping**: Connections scoped by container tags
- **Metadata Enrichment**: Provider-specific metadata (file type, last modified, shared status)

## API Surface

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

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v3/connections/:provider` | Initiate OAuth2 connection |
| GET | `/v3/connections` | List all connections |
| GET | `/v3/connections/:connectionId` | Get connection details |
| DELETE | `/v3/connections/:connectionId` | Delete (optionally with documents) |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| PostgreSQL | SQL | Connection + state persistence |
| sm-document | NATS pub | `sm.connection.synced` → batch ingest |
| Google/Notion/OneDrive APIs | HTTPS | External provider data sync |

## Owner

- **Team**: VNP Memory — Supermemory
