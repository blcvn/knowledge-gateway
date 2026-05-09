---
id: DOC-S02
service: sm-connector
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-10
---

# sm-connector — API Reference

## gRPC Service Definition

```protobuf
service SmConnectorService {
  rpc CreateConnection(CreateConnectionRequest) returns (Connection);
  rpc SyncConnection(SyncConnectionRequest) returns (SyncResponse);
  rpc GetSyncStatus(GetStatusRequest) returns (SyncStatus);
  rpc DeleteConnection(DeleteConnectionRequest) returns (Empty);
}
```

## RPCs: CreateConnection, SyncConnection, GetSyncStatus, DeleteConnection

## NATS Events

Published: `sm.connection.synced` → sm-document (batch ingest synced files).
