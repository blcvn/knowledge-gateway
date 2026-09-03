---
id: DOC-S02
service: ov-fs
version: 1.1.0
status: Active
created: 2026-05-09
updated: 2026-05-09
---

# ov-fs — API Reference

> **Protocol**: gRPC (internal) | **Port**: 9051

## gRPC Service Definition

```protobuf
// api/proto/openviking/fs/v1/service.proto
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

## Endpoints

### ReadFile

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | VikingFS path (e.g., `viking://account/user/memories/fact_01.md`) |
| `context_level` | ContextLevel | No | L0 (abstract), L1 (overview), L2 (full content). Default: L2 |

**Response**: `ReadFileResponse { content: bytes, metadata: FileMetadata, context_level: ContextLevel }`

### WriteFile

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Target file path |
| `content` | bytes | Yes | File content (auto-encrypted via ov-crypto) |
| `create_parents` | bool | No | Create parent directories if missing |
| `context_abstracts` | TieredAbstracts | No | Pre-computed L0/L1 abstracts |

**Response**: `WriteFileResponse { path: string, size_bytes: int64, encrypted: bool }`

**Side-effect**: Publishes `ov.content.written` to NATS.

### DeleteFile

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | File path to delete |
| `recursive` | bool | No | Delete directory and all children |

**Side-effect**: Publishes `ov.content.deleted` to NATS.

### MkDir

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Directory path to create |
| `create_parents` | bool | No | Create intermediate directories |

### ListDir

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | Directory path |
| `recursive` | bool | No | Include children recursively |
| `include_metadata` | bool | No | Include file metadata |

**Response**: `ListDirResponse { entries: []DirEntry }`

### Tree

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `root` | string | Yes | Root directory for tree |
| `max_depth` | int32 | No | Maximum tree depth (default: unlimited) |
| `include_abstracts` | bool | No | Include L0 abstracts in nodes |

**Response**: `TreeResponse { root: TreeNode }`

### Grep

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | Yes | Search pattern (regex or literal) |
| `path` | string | No | Scope search to directory |
| `case_insensitive` | bool | No | Case-insensitive matching |
| `max_results` | int32 | No | Maximum result count |

**Response**: `GrepResponse { matches: []GrepMatch }`

### Glob

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `pattern` | string | Yes | Glob pattern (e.g., `**/*.md`) |
| `root` | string | No | Root directory |

**Response**: `GlobResponse { paths: []string }`

### Move

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `source` | string | Yes | Source path |
| `destination` | string | Yes | Destination path |
| `overwrite` | bool | No | Overwrite existing files |

### GetRelations

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `path` | string | Yes | File path |
| `relation_type` | RelationType | No | Filter by relation type |

**Response**: `RelationsResponse { relations: []FileRelation }`

## Authentication

All requests require `x-tenant-id` (account_id) and `x-user-id` gRPC metadata propagated from vnp-gateway.

## Error Codes

| gRPC Code | HTTP Equiv | Description |
|-----------|-----------|-------------|
| `NOT_FOUND` | 404 | File or directory not found |
| `ALREADY_EXISTS` | 409 | File or directory already exists |
| `INVALID_ARGUMENT` | 400 | Invalid path or parameters |
| `PERMISSION_DENIED` | 403 | Insufficient namespace permissions |
| `RESOURCE_EXHAUSTED` | 429 | PathLock contention or quota exceeded |
| `INTERNAL` | 500 | Internal server error |
| `UNAVAILABLE` | 503 | Service or ov-crypto unavailable |
