# 02 — OpenViking Filesystem Service

> **Service**: `openviking-fs`  
> **Port**: 9011 (gRPC) · 9091 (Health/Metrics)  
> **Origin**: L2 FSService + L5 VikingFS + L5 VikingDBManager  
> **Role**: File CRUD, directory operations, grep/glob, relations, transparent encryption

---

## 1. Responsibilities

| Capability | Description |
|-----------|-------------|
| **File CRUD** | read, write, mkdir, rm, mv, cp, stat, exists |
| **Directory** | ls, tree (original/agent format), depth/node limits |
| **Tiered Read** | abstract (L0), overview (L1), read (L2), read_batch |
| **Pattern Match** | grep (regex), glob (filename) — works on encrypted files |
| **Relations** | get/add/remove context relations (.relations.json) |
| **Privacy** | Privacy config CRUD (version-controlled) |
| **Pack** | Context packing/export |
| **Encryption** | Transparent encrypt-on-write, decrypt-on-read via Crypto service |
| **Vector Sync** | Emit events when content changes for search index sync |

---

## 2. Clean Architecture Layout

```
services/openviking-fs/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── file.go                     # FileEntry, DirectoryEntry, TreeNode
│   │   ├── relation.go                 # ContextRelation, RelationType
│   │   ├── privacy_config.go           # UserPrivacyConfig, ConfigVersion
│   │   ├── grep_result.go              # GrepMatch, GlobResult
│   │   └── errors.go
│   ├── usecase/
│   │   ├── read_file.go                # Read with tiered level support
│   │   ├── write_file.go               # Write with auto-encrypt + emit event
│   │   ├── directory_ops.go            # ls, tree, mkdir
│   │   ├── file_ops.go                 # rm, mv, cp, stat
│   │   ├── grep.go                     # Parallel grep on encrypted files
│   │   ├── glob.go                     # Filename pattern matching
│   │   ├── relations.go                # Relation CRUD
│   │   ├── privacy.go                  # Privacy config CRUD
│   │   ├── pack.go                     # Context packing
│   │   ├── port/
│   │   │   ├── input.go               # FSUseCase interfaces
│   │   │   └── output.go             # FileStore, CryptoClient, EventPublisher
│   │   └── dto/
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go             # gRPC service implementation
│   │   │   └── mapper.go             # Proto ↔ Domain
│   │   ├── repository/
│   │   │   ├── vikingfs/              # Go-native VikingFS adapter
│   │   │   │   ├── fs_adapter.go
│   │   │   │   └── lock_adapter.go
│   │   │   └── local/                 # Local filesystem fallback
│   │   ├── client/
│   │   │   └── crypto_client.go       # gRPC client to Crypto service
│   │   └── event/
│   │       ├── publisher.go            # NATS: ov.content.written/deleted
│   │       └── subscriber.go          # NATS: ov.session.memory.extracted
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       ├── telemetry/
│       └── wire/
```

---

## 3. gRPC Service Definition

```protobuf
service FileSystemService {
  // Basic CRUD
  rpc Read(ReadRequest) returns (ReadResponse);
  rpc Write(WriteRequest) returns (WriteResponse);
  rpc Mkdir(MkdirRequest) returns (MkdirResponse);
  rpc Rm(RmRequest) returns (RmResponse);
  rpc Mv(MvRequest) returns (MvResponse);
  rpc Cp(CpRequest) returns (CpResponse);
  rpc Stat(StatRequest) returns (StatResponse);
  rpc Exists(ExistsRequest) returns (ExistsResponse);

  // Directory
  rpc Ls(LsRequest) returns (LsResponse);
  rpc Tree(TreeRequest) returns (TreeResponse);

  // Tiered Read
  rpc Abstract(AbstractRequest) returns (AbstractResponse);      // L0
  rpc Overview(OverviewRequest) returns (OverviewResponse);      // L1
  rpc ReadBatch(ReadBatchRequest) returns (ReadBatchResponse);   // Batch with level

  // Pattern
  rpc Grep(GrepRequest) returns (GrepResponse);
  rpc Glob(GlobRequest) returns (GlobResponse);

  // Relations
  rpc GetRelations(GetRelationsRequest) returns (GetRelationsResponse);
  rpc AddRelation(AddRelationRequest) returns (AddRelationResponse);
  rpc RemoveRelation(RemoveRelationRequest) returns (RemoveRelationResponse);

  // Privacy
  rpc GetPrivacyConfig(GetPrivacyConfigRequest) returns (GetPrivacyConfigResponse);
  rpc UpsertPrivacyConfig(UpsertPrivacyConfigRequest) returns (UpsertPrivacyConfigResponse);

  // Pack
  rpc Pack(PackRequest) returns (PackResponse);
}
```

---

## 4. Transparent Encryption Integration

```
Write Flow:
  Gateway → FS.Write(uri, plaintext)
    → FS UseCase: gRPC → Crypto.Encrypt(plaintext, account_id)
    → FS Adapter: VikingFS.WriteRaw(uri, ciphertext)
    → Emit: ov.content.written

Read Flow:
  Gateway → FS.Read(uri)
    → FS Adapter: VikingFS.ReadRaw(uri) → ciphertext
    → FS UseCase: gRPC → Crypto.Decrypt(ciphertext, account_id)
    → Return plaintext
```

---

## 5. Lock-Protected Operations

| Operation | Lock Mode | Scope |
|-----------|----------|-------|
| `rm` | subtree | Entire directory tree |
| `mv` | mv | Source path + destination parent |
| Session commit write | point | Session directory |
| Concurrent grep | none | Read-only, no locks needed |

---

## 6. Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| VikingFS rewritten in Go | Eliminate Python+Rust FFI complexity; Go's `os` package is sufficient |
| Crypto as separate service | Key management isolation; independent scaling; security boundary |
| Events for vector sync | Decouple FS writes from search index; eventual consistency is acceptable |
| Parallel grep | Go goroutines + semaphore for concurrent file grep on encrypted content |
