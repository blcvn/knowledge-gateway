# TASK-OV-002 — `pkg/vikingfs/` Go-Native Filesystem Engine

**Wave:** 1 (Foundation)  
**Ưu tiên:** Critical  
**Phụ thuộc:** TASK-OV-001 (pkg/viking)  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-OV-007 §4](../solutions/SOL-OV-007-Shared-Infrastructure.md)

---

## Mục tiêu

Tạo package `pkg/vikingfs/` — Go-native filesystem engine thay thế RAGFS Rust. Gồm 2 phần chính:
1. **FileSystem interface + LocalFileSystem implementation** — ánh xạ `viking://` URI → local disk path
2. **PathLock** — hierarchical concurrent locking để đảm bảo data consistency

---

## Ngữ cảnh

`pkg/vikingfs/` được dùng bởi `services/openviking-fs` như storage adapter. Package này hoạt động trên local disk. Trong tương lai có thể swap sang S3 hoặc GCS mà không cần thay đổi use cases.

URI mapping rule:
```
viking://resources/myrepo/     → {workspace}/data/resources/myrepo/
viking://user/acct1/alice/     → {workspace}/data/user/acct1/alice/
viking://agent/acct1/alice/b/  → {workspace}/data/agent/acct1/alice/b/
viking://session/abc123/       → {workspace}/data/session/abc123/
```

---

## Các file cần tạo

### 1. `pkg/vikingfs/fs.go` — Interface + Domain types

```go
// FileSystem interface — tất cả operations đều dùng viking:// URIs
type FileSystem interface {
    Read(ctx context.Context, uri string) ([]byte, error)
    ReadRange(ctx context.Context, uri string, offset, size int64) ([]byte, error)
    Write(ctx context.Context, uri string, data []byte) error
    Mkdir(ctx context.Context, uri string, existOK bool) error
    Rm(ctx context.Context, uri string, recursive bool) error
    Stat(ctx context.Context, uri string) (*FileInfo, error)
    Exists(ctx context.Context, uri string) (bool, error)
    Ls(ctx context.Context, uri string) ([]DirEntry, error)
    Mv(ctx context.Context, oldURI, newURI string) error
    Cp(ctx context.Context, srcURI, dstURI string) error
    ListAll(ctx context.Context, uri string, opts ListOptions) ([]DirEntry, error)  // Recursive list
    AppendLines(ctx context.Context, uri string, lines [][]byte) error  // Append to JSONL
}

type FileInfo struct {
    URI         string
    Name        string
    IsDirectory bool
    Size        int64
    ModTime     time.Time
    Mode        os.FileMode
}

type DirEntry struct {
    URI         string
    Name        string
    IsDirectory bool
    Size        int64
    ModTime     time.Time
}

type ListOptions struct {
    Recursive     bool
    ExcludeHidden bool  // Skip files starting with . (except .abstract.md, .overview.md)
    ExcludeDirs   bool
    MaxDepth      int
}
```

### 2. `pkg/vikingfs/local.go` — LocalFileSystem

```go
type LocalFileSystem struct {
    workspace string  // absolute path to data root, e.g., "/home/user/.openviking/data"
}

func NewLocalFileSystem(workspace string) (*LocalFileSystem, error)
// Must expand ~ and create workspace dir if not exists

// uriToPath: validates URI, maps to local path
// Security: filepath.Clean() + verify result is within workspace (no escape)
func (fs *LocalFileSystem) uriToPath(uri string) (string, error)

// Implement all FileSystem interface methods
// Write: os.MkdirAll for parent dirs + os.WriteFile (atomic via temp+rename)
// Rm: os.Remove (file) or os.RemoveAll (dir, only when recursive=true)
// Ls: os.ReadDir + convert to DirEntry
// ListAll: recursive filepath.WalkDir với ListOptions filter
// AppendLines: os.OpenFile(O_APPEND|O_CREATE) + write lines
```

**Atomic write pattern** (dùng cho Write để tránh partial writes):
```go
// 1. Write to temp file: {path}.tmp
// 2. os.Rename(tempPath, path)  ← atomic on POSIX
```

### 3. `pkg/vikingfs/lock.go` — PathLock

```go
type PathLock struct {
    mu    sync.Mutex
    locks map[string]*lockEntry
}

type lockEntry struct {
    mu       sync.RWMutex
    refCount int
}

type LockReleaser func()

func NewPathLock() *PathLock

// AcquirePoint: exclusive lock trên một path (session commit, file write)
// Context cancellation → trả về ErrResourceBusy
func (pl *PathLock) AcquirePoint(ctx context.Context, path string) (LockReleaser, error)

// AcquireSubtree: lock path + tất cả children (rm -rf directory)
// Convention: dùng path+"/*" làm lock key cho subtree
func (pl *PathLock) AcquireSubtree(ctx context.Context, path string) (LockReleaser, error)

// AcquireMv: lock source + destination parent để tránh deadlock
// Sort paths alphabetically trước khi lock (fixed order → no deadlock)
func (pl *PathLock) AcquireMv(ctx context.Context, srcPath, dstParentPath string) (LockReleaser, error)

// cleanup: decrement refCount, remove entry khi refCount = 0
func (pl *PathLock) cleanup(path string)
```

---

## Unit Tests

```
TestLocalFileSystem_WriteRead         → Write "hello" → Read → same content
TestLocalFileSystem_AtomicWrite       → Concurrent writes → no partial content
TestLocalFileSystem_Mkdir_ExistOK     → Mkdir existing dir + existOK=true → no error
TestLocalFileSystem_Mkdir_NotExistOK  → Mkdir existing + existOK=false → error
TestLocalFileSystem_Rm_File           → Write then Rm → Exists=false
TestLocalFileSystem_Rm_Dir_Recursive  → Mkdir + Write inside → Rm recursive → dir gone
TestLocalFileSystem_Rm_Dir_NonRecursive → non-empty dir + recursive=false → error
TestLocalFileSystem_Mv                → Write file → Mv → old not exists, new exists
TestLocalFileSystem_Cp                → Write file → Cp → both exist with same content
TestLocalFileSystem_Ls                → Mkdir + 3 files → Ls → 3 entries
TestLocalFileSystem_ListAll_Recursive → 3 levels → all files listed
TestLocalFileSystem_ListAll_MaxDepth  → depth=1 → only top-level files
TestLocalFileSystem_URIPathTraversal  → "viking://../escape" → error (security)
TestLocalFileSystem_URIEscapeWorkspace → uriToPath must stay within workspace
TestLocalFileSystem_AppendLines       → Append 3 lines → file has 3 lines
TestPathLock_Point_Sequential         → 2 goroutines same path → sequential execution
TestPathLock_Point_ContextCancel      → lock held → new lock with cancelled ctx → ErrResourceBusy
TestPathLock_Subtree_BlocksChild      → subtree lock on /foo/ → lock /foo/bar → blocks
TestPathLock_Mv_NoDeadlock            → AcquireMv with paths in reverse order → no deadlock
TestPathLock_Cleanup_NoLeak           → acquire+release 1000x → no memory leak in locks map
TestPathLock_Concurrent_20Goroutines  → 20 goroutines same path → all succeed in turn
```

---

## Cấu trúc thư mục kết quả

```
pkg/vikingfs/
├── fs.go          # Interface + types
├── local.go       # LocalFileSystem implementation
├── local_test.go
├── lock.go        # PathLock
└── lock_test.go
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./pkg/vikingfs/...
go test ./pkg/vikingfs/... -v -count=1 -race
```

Test phải pass với `-race` flag (race condition detector enabled).

---

## Ghi chú triển khai

- Package imports: `pkg/viking` (for ValidateURI, OpenVikingError)
- `Stat()` trả về `*FileInfo` không phải `os.FileInfo` (để abstraction gọn hơn)
- `ListAll()` với `ExcludeHidden=true` → bỏ qua files bắt đầu bằng `.` NGOẠI TRỪ `.abstract.md` và `.overview.md` (vì chúng là tiered context files, cần xuất hiện trong Ls)
- `AppendLines()` dùng `O_SYNC` để đảm bảo durability khi crash
