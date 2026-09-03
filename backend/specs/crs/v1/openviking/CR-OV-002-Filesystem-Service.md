# Change Request: CR-OV-002 — Filesystem Service (VikingFS & Transparent Encryption)

**CR ID:** CR-OV-002  
**Component:** `services/openviking-fs` [NEW SERVICE]  
**Priority:** Critical  
**Status:** Implemented
**Reference:** OpenViking PRD §4.1, SRS §2.1-2.2, specs/services/02-fs-service.md  
**Maps from Python/Rust:** `storage/viking_fs.py` (2199 lines), `ragfs` Rust crate

---

## 1. Mô tả

Xây dựng **openviking-fs** — hệ thống tệp ảo `viking://` viết thuần Go, thay thế hoàn toàn RAGFS Rust:

1. **File/Directory CRUD**: `read`, `write_file`, `mkdir`, `rm`, `mv`, `cp`, `stat`, `exists`.
2. **Directory Operations**: `ls`, `tree` (original format + agent format, depth/node limits).
3. **Three-Tier Context (L0/L1/L2)**: `abstract` (L0 ~100 tokens), `overview` (L1 ~2K tokens), `read` (L2 full), `read_batch` với level-aware loading.
4. **Pattern Matching**: `grep` (regex, parallel goroutines) và `glob` (filename pattern) — hoạt động trên encrypted files (decrypt-in-memory).
5. **Context Relations**: `get_relations`, `add_relation`, `remove_relation` — lưu trong `.relations.json`.
6. **Privacy Config**: Per-user privacy configuration với version history.
7. **Transparent Encryption**: Tự động encrypt-on-write / decrypt-on-read qua `openviking-crypto` service (invisible to caller).
8. **Event Emission**: NATS `ov.content.written`, `ov.content.deleted` để sync với VectorDB.
9. **PathLock**: Distributed lock theo path để đảm bảo data consistency.

---

## 2. Vấn đề hiện tại

- Python dùng RAGFS viết bằng Rust qua FFI → khó debug, khó deploy, compile chain phức tạp.
- Không có Go-native VikingFS → cần RAGFS như một external dependency.
- Transparent encryption chưa được tách ra service riêng → business logic bị vướng với crypto.
- Thiếu NATS event emission để sync vector index.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `services/openviking-fs/` (Port gRPC: 9011)

### 3.2. Clean Architecture Layout

```
services/openviking-fs/
├── cmd/server/main.go
├── internal/
│   ├── domain/
│   │   ├── file.go                     # FileEntry, DirectoryEntry, TreeNode
│   │   ├── relation.go                 # ContextRelation, RelationType
│   │   ├── privacy_config.go           # UserPrivacyConfig, ConfigVersion
│   │   ├── grep_result.go              # GrepMatch, GlobResult
│   │   ├── context_level.go            # ContextLevel enum (Abstract=0, Overview=1, Detail=2)
│   │   └── errors.go
│   ├── usecase/
│   │   ├── read_file.go                # Read with tiered level (L0/L1/L2)
│   │   ├── write_file.go               # Write + auto-encrypt + emit NATS event
│   │   ├── directory_ops.go            # ls, tree, mkdir
│   │   ├── file_ops.go                 # rm, mv, cp, stat, exists
│   │   ├── grep.go                     # Parallel grep (goroutine pool)
│   │   ├── glob.go                     # Filename pattern matching
│   │   ├── relations.go                # Relation CRUD (.relations.json)
│   │   ├── privacy.go                  # Privacy config CRUD + versioning
│   │   ├── pack.go                     # Context packing/export
│   │   ├── port/
│   │   │   ├── input.go               # FSUseCase interfaces
│   │   │   └── output.go             # FileStore, CryptoClient, EventPublisher
│   │   └── dto/
│   │       ├── request.go
│   │       └── response.go
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go             # gRPC service implementation
│   │   │   └── mapper.go             # Proto ↔ Domain
│   │   ├── repository/
│   │   │   ├── vikingfs/              # Go-native VikingFS
│   │   │   │   ├── fs_adapter.go      # Wraps pkg/vikingfs/
│   │   │   │   └── lock_adapter.go    # PathLock operations
│   │   │   └── local/                 # Local filesystem fallback
│   │   ├── client/
│   │   │   └── crypto_client.go       # gRPC client to openviking-crypto
│   │   └── event/
│   │       ├── publisher.go            # NATS: ov.content.written/deleted
│   │       └── subscriber.go          # NATS: ov.session.memory.extracted
│   └── infra/
│       ├── config/config.go
│       ├── server/grpc.go
│       ├── telemetry/
│       └── wire/wire.go
```

### 3.3. Domain Model

```go
// domain/file.go
type FileEntry struct {
    URI          string
    ParentURI    string
    Name         string
    ContextType  ContextType   // MEMORY | RESOURCE | SKILL | SESSION
    Level        int           // 0=Abstract, 1=Overview, 2=Detail
    IsDirectory  bool
    Size         int64
    Abstract     string        // L0 summary (~100 tokens) — from .abstract.md
    ActiveCount  int           // Usage counter (hotness)
    OwnerAccount string
    OwnerUser    string
    OwnerAgent   string
    CreatedAt    time.Time
    UpdatedAt    time.Time
    Meta         map[string]any
}

// domain/context_level.go — Tiered Context Mapping
// L0 → filename.abstract.md  (~100 tokens, one-sentence summary)
// L1 → filename.overview.md  (~2K tokens, core info + usage scenarios)
// L2 → filename               (full content, read only when needed)

// URI Namespace:
// viking://resources/            → project docs, repos, web pages
// viking://user/{account}/{id}/  → user memories, privacy configs
// viking://agent/{account}/{user}/{agent}/ → agent skills, memories, instructions
// viking://session/{session_id}/ → active session data
```

### 3.4. Tiered Read Use Cases

```go
// usecase/read_file.go
func (uc *ReadFileUseCase) Execute(ctx context.Context, req dto.ReadRequest) (*dto.ReadResponse, error) {
    uri := req.URI
    switch req.Level {
    case domain.LevelAbstract:
        uri = toAbstractURI(req.URI)  // "file.md" → "file.abstract.md"
    case domain.LevelOverview:
        uri = toOverviewURI(req.URI)  // "file.md" → "file.overview.md"
    // LevelDetail: use raw uri
    }
    
    raw, err := uc.fileStore.ReadRaw(ctx, uri)
    if err != nil { return nil, err }
    
    // Transparent decryption if file is encrypted
    if isEncrypted(raw) {
        plain, err := uc.cryptoClient.Decrypt(ctx, raw, req.AccountID)
        if err != nil { return nil, err }
        return &dto.ReadResponse{Content: plain}, nil
    }
    return &dto.ReadResponse{Content: raw}, nil
}
```

### 3.5. Transparent Encryption Flow

```
Write Flow:
  Gateway → FS.Write(uri, plaintext)
    → FS UseCase: gRPC → Crypto.Encrypt(plaintext, account_id) → ciphertext
    → FS Adapter: VikingFS.WriteRaw(uri, ciphertext)
    → Emit NATS: ov.content.written{uri, account_id, level}

Read Flow:
  Gateway → FS.Read(uri)
    → FS Adapter: VikingFS.ReadRaw(uri) → ciphertext (or plaintext if not encrypted)
    → FS UseCase: if encrypted → gRPC → Crypto.Decrypt(ciphertext, account_id) → plaintext
    → Return plaintext

Grep Flow (encrypted files):
  → Spawn goroutine pool (N workers)
  → Per file: ReadRaw → Decrypt (in-memory only) → Regex match → Collect results
  → Return GrepMatch list
```

### 3.6. PathLock (Data Consistency)

```go
// pkg/vikingfs/lock.go
type LockMode string
const (
    LockModePoint   LockMode = "point"   // Lock single path
    LockModeSubtree LockMode = "subtree" // Lock path + all children
    LockModeMv      LockMode = "mv"      // Lock source + destination parent
)

// Operations and their lock modes:
// rm directory → subtree lock
// mv source → dest → mv lock (source + dest parent)
// session commit write → point lock on session directory
// concurrent grep → no lock (read-only)
```

### 3.7. gRPC Service Definition

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

  // Tiered Read (L0/L1/L2)
  rpc Abstract(AbstractRequest) returns (AbstractResponse);   // L0
  rpc Overview(OverviewRequest) returns (OverviewResponse);   // L1
  rpc ReadBatch(ReadBatchRequest) returns (ReadBatchResponse);// Batch with level

  // Pattern Matching
  rpc Grep(GrepRequest) returns (GrepResponse);
  rpc Glob(GlobRequest) returns (GlobResponse);

  // Relations
  rpc GetRelations(GetRelationsRequest) returns (GetRelationsResponse);
  rpc AddRelation(AddRelationRequest) returns (AddRelationResponse);
  rpc RemoveRelation(RemoveRelationRequest) returns (RemoveRelationResponse);

  // Privacy Config (version-controlled)
  rpc GetPrivacyConfig(GetPrivacyConfigRequest) returns (GetPrivacyConfigResponse);
  rpc UpsertPrivacyConfig(UpsertPrivacyConfigRequest) returns (UpsertPrivacyConfigResponse);

  // Pack/Export
  rpc Pack(PackRequest) returns (PackResponse);
}
```

### 3.8. NATS Events

| Subject | Payload | Subscribers |
|---------|---------|-------------|
| `ov.content.written` | `{uri, account_id, context_type, level}` | Search (embed + upsert vector index) |
| `ov.content.deleted` | `{uri, account_id}` | Search (remove from vector index) |

**Consumed:**
| Subject | Source | Action |
|---------|--------|--------|
| `ov.session.memory.extracted` | Session | Write extracted memories to `viking://user/{id}/memories/` |
| `admin.account.created` | Admin | Init root directory structure for new account |
| `admin.account.deleted` | Admin | Cascade delete all account data |

### 3.9. Configuration

```yaml
fs:
  grpc:
    port: 9011
  health:
    port: 9091
  storage:
    workspace: "~/.openviking/data"    # Root of VikingFS
    max_tree_depth: 10
    max_grep_goroutines: 20
  crypto:
    service_url: "openviking-crypto:9015"
    enabled: true                       # false = plaintext mode
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
  telemetry:
    service_name: "openviking-fs"
    otel_endpoint: "otel-collector:4317"
```

---

## 4. Acceptance Criteria

- [ ] `Mkdir` + `Write` với `viking://user/test/hello.txt` → file được tạo đúng trên disk.
- [ ] `Write` (encryption enabled) → nội dung trên disk là ciphertext OVE1; `Read` → nhận plaintext.
- [ ] `Grep(pattern="TODO", uri="viking://resources/myrepo/")` → scan tất cả files (kể cả encrypted), trả về line matches.
- [ ] `Tree(uri="viking://resources/", depth=3)` → trả về directory structure đúng với max 3 levels.
- [ ] `Abstract(uri="viking://resources/README.md")` → đọc `README.abstract.md` tự động.
- [ ] `Write` → trigger NATS `ov.content.written` event; Search service nhận và index.
- [ ] `Rm(uri="viking://user/alice/memories/", recursive=true)` → xóa toàn bộ subtree; NATS `ov.content.deleted` được emit cho mỗi file.
- [ ] Concurrent `Mv` và `Rm` cùng path → PathLock đảm bảo không race condition.
- [ ] `GetRelations(uri)` → trả về list URIs từ `.relations.json` đúng format.
