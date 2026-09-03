---
id: TDD-ov-fs
title: Technical Design — ov-fs
service: ov-fs
version: 1.1.0
status: Draft
created: 2026-05-09
updated: 2026-05-09
group: OpenViking
---

# Technical Design — ov-fs

> **Group**: OpenViking | **gRPC Port**: 9051 | **Origin**: OpenViking (VikingFS / RAGFS)

## 1. Service Overview

Go-native VikingFS filesystem service. Provides file CRUD, directory tree, grep, glob, file relations, and tiered context abstraction. Integrates with ov-crypto for transparent envelope encryption and publishes content events for ov-search indexing.

**Origin mapping**: `openviking/storage/viking_fs.py` (82KB) + `openviking/pyagfs/` (filesystem client) + `openviking/core/` (URI, namespace, context).

## 2. Clean Architecture Layers

### 2.1 Domain Layer (Layer 1)

```
internal/domain/
├── model/
│   ├── file.go              # File, FileMetadata, DirEntry, FileContent
│   ├── tree.go              # TreeNode, TreeOptions (max_depth, include_abstracts)
│   ├── relation.go          # FileRelation, RelationType enum
│   ├── context_level.go     # ContextLevel{L0, L1, L2}
│   └── lock.go              # LockType{Point, Subtree, Move}, LockRequest
├── repository/
│   ├── file_repo.go         # FileRepository interface (CRUD + tree + grep + glob)
│   ├── relation_repo.go     # RelationRepository interface
│   └── abstract_repo.go     # AbstractRepository (L0/L1 tiered abstracts)
├── event.go                 # ContentWritten, ContentDeleted domain events
└── errors.go                # PathNotFound, PathAlreadyExists, LockContention
```

### 2.2 Usecase Layer (Layer 2)

```
internal/usecase/
├── file_ops.go              # ReadFile, WriteFile, DeleteFile
├── dir_ops.go               # MkDir, ListDir, Tree
├── search_ops.go            # Grep, Glob
├── move_ops.go              # Move (acquires mv lock atomically)
├── relation_ops.go          # GetRelations, AddRelation
├── port/
│   ├── input.go             # FileUseCase, DirUseCase, SearchUseCase interfaces
│   └── output.go            # EncryptionPort, EventPublisherPort, AbstractGeneratorPort
└── dto/
    └── file_dto.go          # Request/Response data transfer objects
```

### 2.3 Adapter Layer (Layer 3)

```
internal/adapter/
├── grpc/
│   ├── handler.go           # OvFsService gRPC implementation
│   └── mapper.go            # Proto ↔ Domain model mapping
├── event/
│   ├── publisher.go         # NATS publisher for ov.content.* events
│   └── subscriber.go        # Subscriber for ov.crypto.key.rotated, ov.session.memory.extracted
└── client/
    └── crypto_client.go     # ov-crypto gRPC client (EncryptionPort impl)
```

### 2.4 Infrastructure Layer (Layer 4)

```
internal/infra/
├── persistence/
│   ├── vikingfs_repo.go     # PostgreSQL/SurrealDB file repository
│   ├── relation_repo.go     # Relation repository implementation
│   └── abstract_repo.go     # L0/L1 abstract persistence
├── lock/
│   └── pathlock.go          # In-memory PathLock + PostgreSQL advisory locks
├── config/config.go         # Viper configuration
├── server/grpc.go           # gRPC server bootstrap
├── telemetry/               # OTel, Prometheus metrics
└── wire/wire.go             # Google Wire DI
```

## 3. gRPC API

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

## 4. NATS Events

### Published

| Subject | Payload | Trigger |
|---------|---------|---------|
| `ov.content.written` | `{path, account_id, size, checksum}` | After WriteFile succeeds |
| `ov.content.deleted` | `{path, account_id}` | After DeleteFile succeeds |

### Subscribed

| Subject | Action |
|---------|--------|
| `ov.crypto.key.rotated` | Re-wrap all encrypted files for affected account |
| `ov.session.memory.extracted` | Write extracted memory files to VikingFS |

## 5. Data Model

Core entities from `openviking/storage/viking_fs.py`:

- **File**: `{id, account_id, user_id, path, content(encrypted), l0_abstract, l1_abstract, metadata}`
- **DirEntry**: `{name, path, is_dir, size, modified_at}`
- **FileRelation**: `{source_id, target_id, type(references|extracted_from|summarizes)}`
- **TreeNode**: `{path, children[], is_dir, l0_abstract}`
- **GrepMatch**: `{path, line_number, content, score}`

## 6. Cross-Service Dependencies

| Service | Direction | Protocol | Purpose |
|---------|-----------|----------|---------|
| ov-crypto | Outbound | gRPC | Encrypt/decrypt file content |
| ov-search | Outbound (NATS) | Async | Content change notifications for indexing |
| ov-session | Inbound (NATS) | Async | Write extracted memories from sessions |
| ov-resource | Inbound | gRPC | Resource ingestion writes parsed files |

## 7. Observability

- **Metrics**: Prometheus counters/histograms for all RPCs + PathLock + encryption
- **Traces**: OTel spans: `ov-fs.ReadFile`, `ov-fs.WriteFile`, `ov-fs.Tree`, etc.
- **Logs**: Structured JSON via slog with `request_id`, `account_id`, `path`
- **Health**: gRPC Health v1 + HTTP `/healthz` on port 9104

## 8. Multi-Tenancy

- **Isolation Key**: `account_id` (from gRPC metadata `x-tenant-id`)
- **Path Namespace**: `viking://{account_id}/{user_id}/{agent_id}/...`
- **DB Isolation**: All queries filtered by `account_id`
- **RBAC**: Namespace policy restricts cross-user access

---

> **Next Steps**: Decompose this TDD into individual FEAT/ARCH specs in `specs/features/` and `specs/architecture/` before implementation.
