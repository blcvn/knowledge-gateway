---
id: DOC-S01
service: ov-fs
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
owner: VNP Memory — OpenViking Team
---

# ov-fs

> **Group**: OpenViking | **gRPC Port**: 9051 | **Health Port**: 9104 | **Origin**: OpenViking

## Purpose

Go-native **VikingFS** filesystem service replacing the Python RAGFS/PyAGFS implementation. Provides file CRUD, directory tree traversal, `grep`, `glob`, file relations, and `viking://` URI resolution. Integrates with **ov-crypto** for transparent envelope encryption and publishes content events to **ov-search** for indexing.

### Business Capability

- **File CRUD**: Read/Write/Delete files with `viking://` URI scheme
- **Directory Operations**: MkDir, ListDir, Tree (recursive), Move
- **Search Primitives**: Grep (content search), Glob (pattern matching)
- **File Relations**: Track cross-references between files (e.g., memories referencing sessions)
- **Tiered Context**: L0 (abstract ~100 tokens), L1 (overview ~2K tokens), L2 (full content)
- **PathLock**: Concurrent access control with point/subtree/mv locking
- **Transparent Encryption**: Delegates to ov-crypto for AES-256-GCM envelope encrypt/decrypt

## Tech Stack

- **Language**: Go 1.23+
- **Framework**: gRPC server
- **Storage**: Go-native VikingFS (`pkg/vikingfs/`) backed by PostgreSQL/SurrealDB
- **Architecture**: 4-layer Clean Architecture (Domain → Usecase → Adapter → Infra)
- **DI**: Google Wire

## Quick Start

```bash
# From monorepo root
make build-ov-fs
make run-ov-fs

# Or with Docker
docker compose up ov-fs
```

## API Surface

### gRPC Service

```protobuf
service OvFsService {
  rpc ReadFile(ReadFileRequest) returns (ReadFileResponse);
  rpc WriteFile(WriteFileRequest) returns (WriteFileResponse);
  rpc DeleteFile(DeleteFileRequest) returns (google.protobuf.Empty);
  rpc MkDir(MkDirRequest) returns (google.protobuf.Empty);
  rpc ListDir(ListDirRequest) returns (ListDirResponse);
  rpc Tree(TreeRequest) returns (TreeResponse);
  rpc Grep(GrepRequest) returns (GrepResponse);
  rpc Glob(GlobRequest) returns (GlobResponse);
  rpc Move(MoveRequest) returns (google.protobuf.Empty);
  rpc GetRelations(GetRelationsRequest) returns (RelationsResponse);
}
```

### REST (via vnp-gateway)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/ov/files/{path}` | Read file content |
| PUT | `/v1/ov/files/{path}` | Write file content |
| DELETE | `/v1/ov/files/{path}` | Delete file |
| GET | `/v1/ov/tree/{path}` | Directory tree |
| POST | `/v1/ov/grep` | Content search |

## Cross-Service Dependencies

| Service | Protocol | Purpose |
|---------|----------|---------|
| ov-crypto | gRPC | Envelope encrypt/decrypt for transparent file encryption |
| ov-search | NATS | Publish `ov.content.written` / `ov.content.deleted` for indexing |
| PostgreSQL/SurrealDB | SQL | Metadata persistence (paths, relations, tiered abstracts) |

## NATS Events

| Event | Direction | Subscriber |
|-------|-----------|------------|
| `ov.content.written` | Publish | ov-search (embed + upsert index) |
| `ov.content.deleted` | Publish | ov-search (remove from index) |
| `ov.crypto.key.rotated` | Subscribe | Re-wrap encrypted files with new key |
| `ov.session.memory.extracted` | Subscribe | Write extracted memory files |

## Links

- [API Reference](./api.md) · [Architecture](./architecture.md) · [Data Model](./data-model.md)
- [Configuration](./configuration.md) · [Runbook](./runbook.md) · [Changelog](./changelog.md)
- [Architecture Spec](../../../specs/architecture/05-openviking-services.md)

## Owner

- **Team**: VNP Memory — OpenViking
