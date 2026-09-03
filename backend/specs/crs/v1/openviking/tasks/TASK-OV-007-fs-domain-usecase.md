# TASK-OV-007 — `services/openviking-fs` Domain Model & Core Use Cases

**Wave:** 3 (Storage)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-OV-001, TASK-OV-002, TASK-OV-003, TASK-OV-005 (crypto client)  
**Ước tính:** 4 giờ  
**Solution tham chiếu:** [SOL-OV-002 §3, §4](../solutions/SOL-OV-002-Filesystem-Service.md)  
**Port gRPC:** 9011

---

## Mục tiêu

Tạo phần cốt lõi của `services/openviking-fs/` — Domain model và các use cases chính (Read với L0/L1/L2 tiering + transparent decryption, Write với encryption + NATS, Mkdir/Rm/Mv/Cp/Stat/Exists, Tree listing).

---

## Cấu trúc thư mục

```
services/openviking-fs/
├── cmd/server/main.go
├── api/proto/fs/v1/fs.proto
├── internal/
│   ├── domain/
│   │   ├── file.go            # FileEntry, DirEntry, TreeNode
│   │   ├── relation.go        # ContextRelation, RelationType, RelationsFile
│   │   ├── privacy_config.go  # UserPrivacyConfig
│   │   ├── grep_result.go     # GrepMatch, GlobResult
│   │   └── errors.go
│   ├── usecase/
│   │   ├── read_file.go       # L0/L1/L2 tiered read + transparent decrypt
│   │   ├── write_file.go      # Encrypt + write + NATS emit
│   │   ├── directory_ops.go   # ls, tree, mkdir
│   │   ├── file_ops.go        # rm, mv, cp, stat, exists
│   │   └── port/
│   │       ├── input.go       # UseCase interfaces
│   │       └── output.go      # FileStore, CryptoClient, EventPublisher
```

---

## 1. Domain Models

**File: `internal/domain/file.go`**

```go
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
    Abstract    string
    ActiveCount int
    OwnerAccount string
    OwnerUser    string
    OwnerAgent   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
    IsEncrypted bool
    Meta        map[string]any
}

type TreeNode struct {
    Entry    FileEntry
    Children []*TreeNode
}

func IsOVE1Data(data []byte) bool  // Check OVE1 magic bytes
```

**File: `internal/domain/relation.go`**

```go
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

type RelationsFile struct {
    Version   int               `json:"version"`
    UpdatedAt time.Time         `json:"updated_at"`
    Relations []ContextRelation `json:"relations"`
}
```

---

## 2. Port Interfaces

**File: `internal/usecase/port/output.go`**

```go
package port

type FileStore interface {
    ReadRaw(ctx context.Context, uri string) ([]byte, error)
    WriteRaw(ctx context.Context, uri string, data []byte) error
    Mkdir(ctx context.Context, uri string, existOK bool) error
    Rm(ctx context.Context, uri string, recursive bool) error
    Mv(ctx context.Context, oldURI, newURI string) error
    Cp(ctx context.Context, srcURI, dstURI string) error
    Stat(ctx context.Context, uri string) (*domain.FileEntry, error)
    Exists(ctx context.Context, uri string) (bool, error)
    Ls(ctx context.Context, uri string) ([]domain.FileEntry, error)
    ListAll(ctx context.Context, uri string, opts ListOptions) ([]domain.FileEntry, error)
    AppendLines(ctx context.Context, uri string, lines [][]byte) error
}

type CryptoClient interface {
    Encrypt(ctx context.Context, plaintext []byte, accountID string) ([]byte, error)
    Decrypt(ctx context.Context, ciphertext []byte, accountID string) ([]byte, error)
    IsEnabled() bool
}

type EventPublisher interface {
    PublishContentWritten(ctx context.Context, payload ContentWrittenPayload)
    PublishContentDeleted(ctx context.Context, payload ContentDeletedPayload)
}

type ContentWrittenPayload struct {
    URI         string `json:"uri"`
    AccountID   string `json:"account_id"`
    ContextType string `json:"context_type"`
    Level       int    `json:"level"`
}

type ContentDeletedPayload struct {
    URI       string `json:"uri"`
    AccountID string `json:"account_id"`
}
```

---

## 3. Use Case: ReadFile

**File: `internal/usecase/read_file.go`**

```go
type ReadFileUseCase struct {
    fileStore    port.FileStore
    cryptoClient port.CryptoClient
}

type ReadRequest struct {
    URI       string
    Level     domain.ContextLevel  // 0=Abstract, 1=Overview, 2=Detail
}

type ReadResponse struct {
    URI     string
    Content []byte
    Exists  bool
}

func (uc *ReadFileUseCase) Execute(ctx context.Context, req ReadRequest) (*ReadResponse, error) {
    // 1. Level-based URI rewriting:
    //    Level=0 → uri + ".abstract.md"
    //    Level=1 → uri + ".overview.md"
    //    Level=2 → uri as-is (raw file)
    //
    // 2. Validate URI (pkg/viking.ValidateURI)
    //
    // 3. ReadRaw from FileStore
    //    If not found AND Level <= 1 → return {Exists: false, Content: ""} (L0/L1 optional)
    //    If not found AND Level == 2 → return ErrNotFound
    //
    // 4. Transparent decryption:
    //    if IsOVE1(raw) → cryptoClient.Decrypt(raw, accountID)
    //
    // 5. Return content
}

// ReadBatch: reads multiple URIs, same Level, max 10 concurrent
func (uc *ReadFileUseCase) ExecuteBatch(ctx context.Context, uris []string, level domain.ContextLevel) ([]ReadResponse, error)
```

---

## 4. Use Case: WriteFile

**File: `internal/usecase/write_file.go`**

```go
type WriteFileUseCase struct {
    fileStore    port.FileStore
    cryptoClient port.CryptoClient
    publisher    port.EventPublisher
    lock         *vikingfs.PathLock
}

type WriteRequest struct {
    URI      string
    Content  []byte
    AccountID string  // required for encryption
}

func (uc *WriteFileUseCase) Execute(ctx context.Context, req WriteRequest) error {
    // 1. ValidateURI
    // 2. Infer ContextType from URI prefix
    // 3. Encrypt if crypto enabled:
    //    data = cryptoClient.Encrypt(req.Content, req.AccountID)
    // 4. AcquirePoint lock on URI
    // 5. fileStore.WriteRaw(uri, data)
    // 6. Release lock
    // 7. Publish ov.content.written (AFTER successful write, async)
    //    → include Level (inferred: .abstract.md→0, .overview.md→1, else→2)
}

func inferContextType(uri string) domain.ContextType
// "viking://user/" → MEMORY
// "viking://resources/" → RESOURCE
// "viking://agent/" → SKILL
// "viking://session/" → SESSION

func inferLevel(uri string) int
// IsAbstractURI → 0; IsOverviewURI → 1; else → 2
```

---

## 5. Use Case: DirectoryOps

**File: `internal/usecase/directory_ops.go`**

```go
type DirectoryOpsUseCase struct {
    fileStore port.FileStore
}

// Mkdir — validate + create directory
func (uc *DirectoryOpsUseCase) Mkdir(ctx context.Context, uri string, existOK bool) error

// Ls — list directory (non-recursive)
func (uc *DirectoryOpsUseCase) Ls(ctx context.Context, uri string) ([]domain.FileEntry, error)

// Tree — depth-limited recursive tree
func (uc *DirectoryOpsUseCase) Tree(ctx context.Context, req TreeRequest) (*TreeResponse, error)
// req.MaxDepth defaults to 3; max allowed = 10
// Filters out .abstract.md / .overview.md from tree output (tiered files are noise)
// Loads dir.abstract.md for each directory node (efficient for agent context)
// Format "agent": compact XML-like representation; default: JSON tree
```

---

## 6. Use Case: FileOps

**File: `internal/usecase/file_ops.go`**

```go
type FileOpsUseCase struct {
    fileStore port.FileStore
    lock      *vikingfs.PathLock
    publisher port.EventPublisher
}

// Rm — remove file or directory
func (uc *FileOpsUseCase) Rm(ctx context.Context, uri string, recursive bool) error
// Lock: AcquireSubtree if directory; AcquirePoint if file
// After rm: publish ov.content.deleted (async)

// Mv — rename/move
func (uc *FileOpsUseCase) Mv(ctx context.Context, oldURI, newURI string) error
// Lock: AcquireMv(oldURI, parent(newURI))

// Cp — copy file/directory
func (uc *FileOpsUseCase) Cp(ctx context.Context, srcURI, dstURI string) error
// No lock needed for copy (read-only on src)

// Stat — file metadata
func (uc *FileOpsUseCase) Stat(ctx context.Context, uri string) (*domain.FileEntry, error)

// Exists — check existence
func (uc *FileOpsUseCase) Exists(ctx context.Context, uri string) (bool, error)
```

---

## 7. Adapter: FileStore (wraps pkg/vikingfs)

**File: `internal/adapter/repository/vikingfs/fs_adapter.go`**

```go
// FSAdapter wraps pkg/vikingfs.LocalFileSystem and converts between
// viking:// URIs and FileEntry domain types

type FSAdapter struct {
    fs   *vikingfs.LocalFileSystem
    lock *vikingfs.PathLock
}

func (a *FSAdapter) ReadRaw(ctx context.Context, uri string) ([]byte, error)
func (a *FSAdapter) WriteRaw(ctx context.Context, uri string, data []byte) error
func (a *FSAdapter) Stat(ctx context.Context, uri string) (*domain.FileEntry, error)
// Convert vikingfs.FileInfo → domain.FileEntry
// Fill ContextType from URI prefix
// IsEncrypted: peek first 4 bytes → "OVE1" → true
```

---

## Unit Tests

```
TestReadFile_DetailLevel_ReturnsContent     → L2 read → content returned
TestReadFile_AbstractLevel_ReturnsAbstract  → L0 → reads .abstract.md
TestReadFile_AbstractLevel_NotExists_NoErr  → L0, no .abstract.md → {Exists:false}
TestReadFile_DetailLevel_NotExists_Error    → L2, no file → ErrNotFound
TestReadFile_TransparentDecrypt             → OVE1 file → crypto.Decrypt called
TestReadFile_PlaintextFile_NoCrypto         → non-OVE1 → crypto not called
TestReadBatch_10Concurrent                  → 10 URIs → all succeed, ≤10 concurrent
TestWriteFile_EncryptsBeforeWrite           → crypto.Encrypt called → encrypted bytes written
TestWriteFile_NATSPublishedAfterSuccess     → write ok → publisher.PublishContentWritten called
TestWriteFile_NATSNotPublishedOnError       → WriteRaw fails → no NATS event
TestWriteFile_PathLockPreventsRace          → 2 goroutines same URI → sequential
TestMkdir_CreatesDirectory                  → Mkdir → Exists returns true
TestRm_File                                 → Write + Rm → Exists=false
TestRm_Directory_Recursive                  → Mkdir + Write + Rm(recursive) → gone
TestRm_Directory_NonRecursive_Error         → non-empty dir + !recursive → error
TestMv_FileExists                           → Mv → old gone, new exists
TestCp_BothExist                            → Cp → both exist, same content
TestTree_MaxDepth                           → depth=2, deep tree → stops at level 2
TestTree_ExcludesTieredFiles                → .abstract.md not in children list
TestInferContextType_AllPrefixes            → all 4 URI prefixes → correct ContextType
TestInferLevel_AbstractSuffix               → .abstract.md → 0
TestInferLevel_OverviewSuffix               → .overview.md → 1
TestInferLevel_RegularFile                  → foo.md → 2
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory

# Build (proto gen needed first for full service, but domain + usecase compile standalone)
go build ./services/openviking-fs/internal/...

# Test
go test ./services/openviking-fs/internal/... -v -count=1 -race
```

---

## Ghi chú triển khai

- CryptoClient trong usecase là **interface** — unit tests dùng mock
- PathLock instance là **single shared instance** trong DI wire
- NATS publish bất đồng bộ (goroutine) để không block write path
- EventPublisher dùng `log.Warn` khi publish fail (không fail write operation)
- `Rm` recursive emit batch event: `PublishContentDeleted` một lần với list URIs
