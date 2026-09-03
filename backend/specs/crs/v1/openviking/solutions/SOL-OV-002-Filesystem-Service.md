# Solution: SOL-OV-002 — Filesystem Service (VikingFS & Transparent Encryption)

**CR:** [CR-OV-002](../CR-OV-002-Filesystem-Service.md)  
**Wave:** 3 (Storage — sau pkg/ và Crypto/Admin)  
**Priority:** Critical  
**Status:** Draft  
**Date:** 2026-06-17

---

## 1. Tổng quan Giải pháp

Xây dựng `services/openviking-fs` — hệ thống tệp ảo `viking://` hoàn toàn bằng Go, thay thế RAGFS (Rust FFI). Đây là **central storage service** — mọi service khác đều read/write qua đây.

### Chiến lược chính

| Vấn đề | Giải pháp |
|---|---|
| RAGFS Rust FFI phức tạp | Go-native VikingFS dùng `pkg/vikingfs/` |
| Transparent encryption phải hoạt động với grep | Decrypt-in-memory trước khi regex match |
| Concurrent write race conditions | PathLock (`pkg/vikingfs/lock.go`) per operation type |
| L0/L1/L2 tiering phải transparent | URI rewriting trong UseCase layer |
| Event emission phải sau write complete | Publish sau `WriteRaw()` returns successfully |
| `.relations.json` format | Append-only JSONB per URI, versioned |

---

## 2. Codebase Structure

```
services/openviking-fs/
├── cmd/server/main.go
├── api/proto/fs/v1/fs.proto
├── internal/
│   ├── domain/
│   │   ├── file.go                # FileEntry, DirEntry, TreeNode
│   │   ├── relation.go            # ContextRelation, RelationType
│   │   ├── privacy_config.go      # UserPrivacyConfig, ConfigVersion
│   │   ├── grep_result.go         # GrepMatch, GlobResult
│   │   └── errors.go
│   ├── usecase/
│   │   ├── read_file.go           # L0/L1/L2 tiered read + transparent decrypt
│   │   ├── write_file.go          # Encrypt + write + emit NATS
│   │   ├── directory_ops.go       # ls, tree, mkdir
│   │   ├── file_ops.go            # rm, mv, cp, stat, exists
│   │   ├── grep.go                # Parallel goroutine pool grep
│   │   ├── glob.go                # Filename pattern matching
│   │   ├── relations.go           # CRUD .relations.json
│   │   ├── privacy.go             # Privacy config CRUD + version history
│   │   ├── pack.go                # Context export
│   │   └── port/
│   │       ├── input.go
│   │       └── output.go          # FileStore, CryptoClient, EventPublisher
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go         # gRPC server handlers
│   │   │   └── mapper.go          # Proto ↔ Domain
│   │   ├── repository/
│   │   │   └── vikingfs/
│   │   │       ├── fs_adapter.go  # wraps pkg/vikingfs/LocalFileSystem
│   │   │       └── lock_adapter.go
│   │   ├── client/
│   │   │   └── crypto_client.go   # gRPC client → openviking-crypto:9015
│   │   └── event/
│   │       ├── publisher.go       # NATS publisher
│   │       └── subscriber.go      # Consume admin.account.* events
│   └── infra/
│       ├── config/
│       ├── server/grpc.go
│       ├── migrations/            # If using DB for metadata
│       └── wire/
```

---

## 3. Domain Model

### 3.1 FileEntry

```go
// internal/domain/file.go

type ContextType string
const (
    ContextTypeMemory   ContextType = "MEMORY"
    ContextTypeResource ContextType = "RESOURCE"
    ContextTypeSkill    ContextType = "SKILL"
    ContextTypeSession  ContextType = "SESSION"
)

type FileEntry struct {
    URI         string
    ParentURI   string
    Name        string
    ContextType ContextType
    Level       int      // 0=Abstract, 1=Overview, 2=Detail
    IsDirectory bool
    Size        int64
    Abstract    string   // Content of .abstract.md (L0), if exists
    ActiveCount int      // Usage counter for hotness
    OwnerAccount string
    OwnerUser    string
    OwnerAgent   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    IsEncrypted bool     // Whether file is OVE1 encrypted
    Meta        map[string]any
}

type TreeNode struct {
    Entry    FileEntry
    Children []*TreeNode
}

// Three-tier URI mapping:
// viking://resources/myrepo/README.md
//    L0 → viking://resources/myrepo/README.md.abstract.md
//    L1 → viking://resources/myrepo/README.md.overview.md
//    L2 → viking://resources/myrepo/README.md  (raw file)
```

### 3.2 Context Relation

```go
// internal/domain/relation.go

type RelationType string
const (
    RelationTypeRelated    RelationType = "related"
    RelationTypeRequires   RelationType = "requires"
    RelationTypeSupersedes RelationType = "supersedes"
    RelationTypeExtends    RelationType = "extends"
)

type ContextRelation struct {
    SourceURI   string
    TargetURI   string
    Relation    RelationType
    Description string
    CreatedAt   time.Time
}

// Stored in: {parent_dir}/.relations.json
// Format: JSON array of ContextRelation
// Versioned: append-only with timestamp

type RelationsFile struct {
    Version   int               `json:"version"`
    UpdatedAt time.Time         `json:"updated_at"`
    Relations []ContextRelation `json:"relations"`
}
```

### 3.3 Privacy Config

```go
// internal/domain/privacy_config.go

type UserPrivacyConfig struct {
    UserID         string
    AccountID      string
    ExcludeTopics  []string  // Topics that should NOT be memorized
    ExcludeKeywords []string // Keywords that trigger privacy filtering
    RedactPII      bool      // Whether to auto-redact PII before storage
    Version        int
    UpdatedAt      time.Time
}

type PrivacyConfigVersion struct {
    Config    UserPrivacyConfig
    UpdatedAt time.Time
    UpdatedBy string  // API key ID
}
// Stored in: viking://user/{account}/{user_id}/.privacy.json
// Version history: viking://user/{account}/{user_id}/.privacy.history.jsonl
```

---

## 4. Use Cases — Chi tiết Implementation

### 4.1 ReadFile — L0/L1/L2 Tiered

```go
// internal/usecase/read_file.go

type ReadFileUseCase struct {
    fileStore    port.FileStore
    cryptoClient port.CryptoClient
    lock         *vikingfs.PathLock
}

func (uc *ReadFileUseCase) Execute(ctx context.Context, req dto.ReadRequest) (*dto.ReadResponse, error) {
    targetURI := req.URI
    
    // Level-based URI rewriting
    switch req.Level {
    case domain.LevelAbstract:
        targetURI = viking.ToAbstractURI(req.URI)  // file.md → file.md.abstract.md
    case domain.LevelOverview:
        targetURI = viking.ToOverviewURI(req.URI)  // file.md → file.md.overview.md
    case domain.LevelDetail:
        // Use req.URI as-is
    default:
        return nil, &viking.OpenVikingError{Code: viking.ErrInvalidArgument, Message: "invalid level"}
    }
    
    // Validate access
    rc, ok := viking.FromContext(ctx)
    if !ok {
        return nil, &viking.OpenVikingError{Code: viking.ErrUnauthenticated}
    }
    if !viking.IsAccessible(targetURI, rc) {
        return nil, &viking.OpenVikingError{Code: viking.ErrPermissionDenied}
    }
    
    // Read raw bytes
    raw, err := uc.fileStore.ReadRaw(ctx, targetURI)
    if err != nil {
        if errors.Is(err, fs.ErrNotExist) {
            // L0/L1 files are optional — return empty string (not error)
            if req.Level != domain.LevelDetail {
                return &dto.ReadResponse{Content: "", Exists: false}, nil
            }
            return nil, &viking.OpenVikingError{Code: viking.ErrNotFound, Message: "file not found: " + targetURI}
        }
        return nil, fmt.Errorf("read raw: %w", err)
    }
    
    // Transparent decryption
    content := raw
    if domain.IsOVE1(raw) {
        accountID, _, _, _ := viking.ResolveOwner(targetURI)
        if accountID == "" {
            accountID = rc.User.AccountID
        }
        content, err = uc.cryptoClient.Decrypt(ctx, raw, accountID)
        if err != nil {
            return nil, fmt.Errorf("decrypt: %w", err)
        }
    }
    
    return &dto.ReadResponse{
        URI:     targetURI,
        Content: content,
        Exists:  true,
    }, nil
}

// ReadBatch: reads multiple URIs with specified level
func (uc *ReadFileUseCase) ExecuteBatch(ctx context.Context, req dto.ReadBatchRequest) (*dto.ReadBatchResponse, error) {
    results := make([]dto.ReadResponse, len(req.URIs))
    var mu sync.Mutex
    
    g, gCtx := errgroup.WithContext(ctx)
    sem := make(chan struct{}, 10)  // Max 10 concurrent reads
    
    for i, uri := range req.URIs {
        i, uri := i, uri
        g.Go(func() error {
            sem <- struct{}{}
            defer func() { <-sem }()
            
            resp, err := uc.Execute(gCtx, dto.ReadRequest{URI: uri, Level: req.Level})
            if err != nil {
                return err
            }
            mu.Lock()
            results[i] = *resp
            mu.Unlock()
            return nil
        })
    }
    
    if err := g.Wait(); err != nil {
        return nil, err
    }
    return &dto.ReadBatchResponse{Results: results}, nil
}
```

### 4.2 WriteFile — Encrypt + NATS

```go
// internal/usecase/write_file.go

type WriteFileUseCase struct {
    fileStore    port.FileStore
    cryptoClient port.CryptoClient
    publisher    port.EventPublisher
    lock         *vikingfs.PathLock
}

func (uc *WriteFileUseCase) Execute(ctx context.Context, req dto.WriteRequest) (*dto.WriteResponse, error) {
    // Validate
    if err := viking.ValidateURI(req.URI); err != nil {
        return nil, err
    }
    
    accountID, _, _, _ := viking.ResolveOwner(req.URI)
    if accountID == "" {
        rc, _ := viking.FromContext(ctx)
        accountID = rc.User.AccountID
    }
    
    // Determine ContextType from URI
    contextType := inferContextType(req.URI)
    
    // Encrypt if crypto client configured
    data := req.Content
    var err error
    if uc.cryptoClient.IsEnabled() {
        data, err = uc.cryptoClient.Encrypt(ctx, req.Content, accountID)
        if err != nil {
            return nil, fmt.Errorf("encrypt: %w", err)
        }
    }
    
    // Acquire point lock
    release, err := uc.lock.AcquirePoint(ctx, req.URI)
    if err != nil {
        return nil, err
    }
    defer release()
    
    // Write to VikingFS
    if err := uc.fileStore.WriteRaw(ctx, req.URI, data); err != nil {
        return nil, fmt.Errorf("write: %w", err)
    }
    
    // Emit NATS event (after successful write)
    level := inferLevel(req.URI)
    uc.publisher.PublishContentWritten(ctx, port.ContentWrittenPayload{
        URI:         req.URI,
        AccountID:   accountID,
        ContextType: contextType,
        Level:       level,
    })
    
    return &dto.WriteResponse{URI: req.URI}, nil
}

func inferContextType(uri string) domain.ContextType {
    if strings.HasPrefix(uri, "viking://user/") {
        return domain.ContextTypeMemory
    }
    if strings.HasPrefix(uri, "viking://resources/") {
        return domain.ContextTypeResource
    }
    if strings.HasPrefix(uri, "viking://agent/") {
        return domain.ContextTypeSkill
    }
    if strings.HasPrefix(uri, "viking://session/") {
        return domain.ContextTypeSession
    }
    return domain.ContextTypeResource
}

func inferLevel(uri string) int {
    if viking.IsAbstractURI(uri) {
        return 0
    }
    if viking.IsOverviewURI(uri) {
        return 1
    }
    return 2
}
```

### 4.3 Grep — Parallel Goroutine Pool

```go
// internal/usecase/grep.go

type GrepUseCase struct {
    fileStore    port.FileStore
    cryptoClient port.CryptoClient
    maxWorkers   int  // default: 20
}

func (uc *GrepUseCase) Execute(ctx context.Context, req dto.GrepRequest) (*dto.GrepResponse, error) {
    pattern, err := regexp.Compile(req.Pattern)
    if err != nil {
        return nil, &viking.OpenVikingError{Code: viking.ErrInvalidArgument, Message: "invalid regex: " + err.Error()}
    }
    
    // Get all files recursively
    files, err := uc.fileStore.ListAll(ctx, req.URI, dto.ListOptions{
        Recursive:     true,
        ExcludeHidden: true,                    // Skip .abstract.md, .overview.md, .meta.json
        ExcludeDirs:   true,
        MaxDepth:      req.MaxDepth,
        FileTypes:     req.FileTypes,           // Optional filter by extension
    })
    if err != nil {
        return nil, fmt.Errorf("list files: %w", err)
    }
    
    type grepJob struct {
        file domain.FileEntry
    }
    type grepResult struct {
        matches []domain.GrepMatch
        err     error
    }
    
    jobs := make(chan grepJob, len(files))
    results := make(chan grepResult, len(files))
    
    // Start worker pool
    for i := 0; i < uc.maxWorkers; i++ {
        go func() {
            for job := range jobs {
                matches, err := uc.grepFile(ctx, job.file, pattern, req.AccountID)
                results <- grepResult{matches, err}
            }
        }()
    }
    
    // Feed jobs
    for _, file := range files {
        jobs <- grepJob{file}
    }
    close(jobs)
    
    // Collect results
    var allMatches []domain.GrepMatch
    var grepErrors []error
    for range files {
        r := <-results
        if r.err != nil {
            grepErrors = append(grepErrors, r.err)
            continue
        }
        allMatches = append(allMatches, r.matches...)
    }
    
    return &dto.GrepResponse{
        Matches:     allMatches,
        TotalScanned: len(files),
        Errors:      grepErrors,
    }, nil
}

func (uc *GrepUseCase) grepFile(ctx context.Context, file domain.FileEntry, pattern *regexp.Regexp, accountID string) ([]domain.GrepMatch, error) {
    raw, err := uc.fileStore.ReadRaw(ctx, file.URI)
    if err != nil {
        return nil, err
    }
    
    // Decrypt in memory (never write plaintext to disk)
    content := raw
    if domain.IsOVE1(raw) {
        content, err = uc.cryptoClient.Decrypt(ctx, raw, accountID)
        if err != nil {
            return nil, err
        }
    }
    
    // Line-by-line matching
    var matches []domain.GrepMatch
    for lineNum, line := range strings.Split(string(content), "\n") {
        if locs := pattern.FindAllStringIndex(line, -1); len(locs) > 0 {
            matches = append(matches, domain.GrepMatch{
                URI:     file.URI,
                Line:    lineNum + 1,
                Content: line,
                Matches: locs,
            })
        }
    }
    return matches, nil
}
```

### 4.4 Tree — Depth-Limited Recursive Listing

```go
// internal/usecase/directory_ops.go

func (uc *DirectoryOpsUseCase) Tree(ctx context.Context, req dto.TreeRequest) (*dto.TreeResponse, error) {
    maxDepth := req.MaxDepth
    if maxDepth <= 0 || maxDepth > 10 {
        maxDepth = 3  // Safety default
    }
    
    root, err := uc.buildTreeNode(ctx, req.URI, 0, maxDepth, req.Format)
    if err != nil {
        return nil, err
    }
    
    if req.Format == "agent" {
        // Agent format: compact XML-like representation for LLM context
        return &dto.TreeResponse{AgentFormat: toAgentFormat(root)}, nil
    }
    // Original format: JSON tree
    return &dto.TreeResponse{Tree: root}, nil
}

func (uc *DirectoryOpsUseCase) buildTreeNode(ctx context.Context, uri string, depth, maxDepth int, format string) (*domain.TreeNode, error) {
    stat, err := uc.fileStore.Stat(ctx, uri)
    if err != nil {
        return nil, err
    }
    
    node := &domain.TreeNode{Entry: toDomainEntry(stat)}
    
    // Load L0 abstract for directory nodes (efficient for agent context)
    abstractURI := viking.ToAbstractURI(uri + ".directory")  // convention: directory.abstract.md
    abstract, _ := uc.fileStore.ReadRaw(ctx, abstractURI)
    node.Entry.Abstract = string(abstract)
    
    if depth >= maxDepth || !stat.IsDirectory {
        return node, nil
    }
    
    entries, err := uc.fileStore.Ls(ctx, uri)
    if err != nil {
        return node, nil  // Return partial tree on error
    }
    
    for _, entry := range entries {
        // Skip tiered files in tree view (avoid .abstract.md / .overview.md noise)
        if viking.IsAbstractURI(entry.URI) || viking.IsOverviewURI(entry.URI) {
            continue
        }
        child, err := uc.buildTreeNode(ctx, entry.URI, depth+1, maxDepth, format)
        if err != nil {
            continue
        }
        node.Children = append(node.Children, child)
    }
    
    return node, nil
}
```

### 4.5 Relations Management

```go
// internal/usecase/relations.go

type RelationsUseCase struct {
    fileStore port.FileStore
    lock      *vikingfs.PathLock
}

func (uc *RelationsUseCase) GetRelations(ctx context.Context, uri string) ([]domain.ContextRelation, error) {
    // .relations.json lives in the parent directory
    parentURI := filepath.Dir(uri) + "/"
    relationsURI := parentURI + ".relations.json"
    
    raw, err := uc.fileStore.ReadRaw(ctx, relationsURI)
    if errors.Is(err, fs.ErrNotExist) {
        return []domain.ContextRelation{}, nil
    }
    
    var relFile domain.RelationsFile
    json.Unmarshal(raw, &relFile)
    
    // Filter only relations involving the requested URI
    var result []domain.ContextRelation
    for _, r := range relFile.Relations {
        if r.SourceURI == uri || r.TargetURI == uri {
            result = append(result, r)
        }
    }
    return result, nil
}

func (uc *RelationsUseCase) AddRelation(ctx context.Context, req dto.AddRelationRequest) error {
    parentURI := filepath.Dir(req.SourceURI) + "/"
    relationsURI := parentURI + ".relations.json"
    
    release, _ := uc.lock.AcquirePoint(ctx, relationsURI)
    defer release()
    
    raw, _ := uc.fileStore.ReadRaw(ctx, relationsURI)
    var relFile domain.RelationsFile
    json.Unmarshal(raw, &relFile)
    
    relFile.Relations = append(relFile.Relations, domain.ContextRelation{
        SourceURI:   req.SourceURI,
        TargetURI:   req.TargetURI,
        Relation:    req.Relation,
        Description: req.Description,
        CreatedAt:   time.Now(),
    })
    relFile.UpdatedAt = time.Now()
    relFile.Version++
    
    data, _ := json.MarshalIndent(relFile, "", "  ")
    return uc.fileStore.WriteRaw(ctx, relationsURI, data)
}
```

---

## 5. NATS Events

### Published
```go
// port/event_publisher.go

type ContentWrittenPayload struct {
    URI         string          `json:"uri"`
    AccountID   string          `json:"account_id"`
    ContextType domain.ContextType `json:"context_type"`
    Level       int             `json:"level"`
}

type ContentDeletedPayload struct {
    URI       string `json:"uri"`
    AccountID string `json:"account_id"`
}

// Subjects:
// "ov.content.written"  → Search service indexes new content
// "ov.content.deleted"  → Search service removes from index
```

### Consumed (Subscriber)
```go
// adapter/event/subscriber.go

// admin.account.created → Init root directory structure
func (s *Subscriber) HandleAccountCreated(msg *nats.Msg) {
    var payload struct {
        AccountID string `json:"account_id"`
    }
    json.Unmarshal(msg.Data, &payload)
    
    // Create root namespaces for account
    for _, root := range []string{"resources", "user/" + payload.AccountID, "agent/" + payload.AccountID, "session/" + payload.AccountID} {
        s.fsUC.Mkdir(context.Background(), "viking://"+root+"/", true)
    }
    msg.Ack()
}

// admin.account.deleted → Cascade delete ALL account data
func (s *Subscriber) HandleAccountDeleted(msg *nats.Msg) {
    var payload struct{ AccountID string `json:"account_id"` }
    json.Unmarshal(msg.Data, &payload)
    
    toDelete := []string{
        "viking://user/" + payload.AccountID + "/",
        "viking://agent/" + payload.AccountID + "/",
        "viking://session/" + payload.AccountID + "/",
    }
    for _, uri := range toDelete {
        s.fileOpsUC.Rm(context.Background(), uri, true)
    }
    msg.Ack()
}

// ov.session.memory.extracted → Write extracted memories (already done in Session Phase 2 via FS gRPC)
// ov.crypto.key.rotated → No FS action needed (crypto rotates headers directly)
```

---

## 6. gRPC API — Complete Service Definition

```protobuf
syntax = "proto3";
package openviking.fs.v1;

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

  // Directory Operations
  rpc Ls(LsRequest) returns (LsResponse);
  rpc Tree(TreeRequest) returns (TreeResponse);

  // Tiered Context (L0/L1/L2)
  rpc Abstract(AbstractRequest) returns (AbstractResponse);    // L0: .abstract.md
  rpc Overview(OverviewRequest) returns (OverviewResponse);    // L1: .overview.md
  rpc ReadBatch(ReadBatchRequest) returns (ReadBatchResponse); // Batch with level

  // Pattern Matching
  rpc Grep(GrepRequest) returns (GrepResponse);
  rpc Glob(GlobRequest) returns (GlobResponse);

  // Relations
  rpc GetRelations(GetRelationsRequest) returns (GetRelationsResponse);
  rpc AddRelation(AddRelationRequest) returns (AddRelationResponse);
  rpc RemoveRelation(RemoveRelationRequest) returns (RemoveRelationResponse);

  // Privacy Config
  rpc GetPrivacyConfig(GetPrivacyConfigRequest) returns (GetPrivacyConfigResponse);
  rpc UpsertPrivacyConfig(UpsertPrivacyConfigRequest) returns (UpsertPrivacyConfigResponse);

  // Context Packing/Export
  rpc Pack(PackRequest) returns (PackResponse);
}

message ReadRequest {
  string uri = 1;
  int32 level = 2;  // 0=Abstract, 1=Overview, 2=Detail
}

message GrepRequest {
  string uri = 1;       // Root URI to search under
  string pattern = 2;   // Regex pattern
  string account_id = 3;
  int32 max_depth = 4;  // default: unlimited
  repeated string file_types = 5;  // filter by extension
}

message GrepResponse {
  repeated GrepMatch matches = 1;
  int32 total_scanned = 2;
}

message GrepMatch {
  string uri = 1;
  int32 line = 2;
  string content = 3;
}
```

---

## 7. Configuration

```yaml
fs:
  grpc:
    port: 9011
  health:
    port: 9091
  storage:
    workspace: "~/.openviking/data"   # Root of VikingFS local storage
    max_tree_depth: 10
    max_grep_goroutines: 20
    max_file_size_mb: 100
  crypto:
    service_url: "openviking-crypto:9015"
    enabled: true                      # false = plaintext mode
    timeout: 5s
  nats:
    url: "nats://nats:4222"
    stream: "openviking"
    subjects:
      publish_content_written: "ov.content.written"
      publish_content_deleted: "ov.content.deleted"
      subscribe_account_created: "admin.account.created"
      subscribe_account_deleted: "admin.account.deleted"
      subscribe_key_rotated: "ov.crypto.key.rotated"
  telemetry:
    service_name: "openviking-fs"
    otel_endpoint: "otel-collector:4317"
```

---

## 8. Testing Strategy

### Unit Tests
- `TestReadFile_AbstractLevel` — URI rewriting to `.abstract.md`, graceful empty if not exists
- `TestReadFile_TransparentDecrypt` — OVE1 file → crypto.Decrypt called → plaintext returned
- `TestReadFile_PlaintextFile` — non-OVE1 → crypto not called, content returned as-is
- `TestWriteFile_EncryptsContent` — crypto.Encrypt called before WriteRaw
- `TestWriteFile_NATSPublishedAfterSuccess` — event published with correct subject
- `TestWriteFile_NATSNotPublishedOnError` — WriteRaw fails → no NATS event
- `TestGrepUseCase_DecryptsBeforeMatch` — encrypted files → decrypted in-memory, grep matches returned
- `TestGrepUseCase_SkipsHiddenFiles` — .abstract.md not scanned
- `TestGrepUseCase_ConcurrentWorkers` — 20 files → max 20 goroutines
- `TestPathLock_PreventsConcurrentWrite` — 2 goroutines same path → sequential
- `TestTreeUseCase_MaxDepth` — depth=2 → stops at level 2, no deeper children
- `TestAddRelation_AppendOnly` — existing relations preserved, new one appended

### Integration Tests
- `TestWriteReadE2E_WithEncryption` — real crypto service
- `TestGrepE2E_EncryptedFiles` — write encrypted → grep → find content
- `TestAccountLifecycleFS` — NATS account.created → dirs created; account.deleted → dirs removed
- `TestMvConcurrencyE2E` — mv + rm same path → PathLock prevents race

---

## 9. Rủi ro & Biện pháp

| Rủi ro | Mức độ | Biện pháp |
|---|---|---|
| Crypto service down → all writes fail | Cao | Passthrough mode nếu `crypto.enabled=false`; circuit breaker để fail-fast |
| Grep trên large dir (10K files) → memory spike | Trung bình | Worker pool limit (max 20), stream results từng file, không load all |
| .relations.json concurrent write → corruption | Thấp | PathLock trên `.relations.json` path trước mỗi write |
| Rm recursive slow (deep tree) | Thấp | os.RemoveAll Go stdlib + emit NATS per file có thể nhiều events |
| NATS event flood khi rm large dir | Trung bình | Batch events: emit `ov.content.deleted` với list URIs thay vì per-file |
| VikingFS local path traversal | Thấp | `pkg/viking.ValidateURI()` trước mọi operation; filepath.Clean() sau khi map |
