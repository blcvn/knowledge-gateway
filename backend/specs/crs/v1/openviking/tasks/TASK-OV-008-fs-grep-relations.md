# TASK-OV-008 — `services/openviking-fs` Grep, Glob, Relations & Privacy

**Wave:** 3 (Storage)  
**Ưu tiên:** High  
**Phụ thuộc:** TASK-OV-007 (fs domain + core use cases)  
**Ước tính:** 3 giờ  
**Solution tham chiếu:** [SOL-OV-002 §4.3, §4.4, §4.5](../solutions/SOL-OV-002-Filesystem-Service.md)

**Trạng thái:** ✅ Implemented  
**Ghi chú:** ov-fs grep/relations implemented  
---

## Mục tiêu

Hoàn thiện `services/openviking-fs/` với 4 use cases bổ sung: Grep (parallel goroutine pool với transparent decryption), Glob (filename pattern matching), Relations (CRUD `.relations.json`), và Privacy Config (user privacy settings versioned).

---

## Các file cần tạo

### 1. `internal/usecase/grep.go` — Parallel Grep Pool

```go
type GrepUseCase struct {
    fileStore    port.FileStore
    cryptoClient port.CryptoClient
    maxWorkers   int  // default: 20
}

type GrepRequest struct {
    URI       string   // Root URI to search under
    Pattern   string   // Regex pattern
    AccountID string
    MaxDepth  int      // 0 = unlimited
    FileTypes []string // Optional: filter by extension (e.g., [".go", ".md"])
}

type GrepResponse struct {
    Matches      []domain.GrepMatch
    TotalScanned int
    Errors       []error
}

func (uc *GrepUseCase) Execute(ctx context.Context, req GrepRequest) (*GrepResponse, error) {
    // 1. Compile regex (return ErrInvalidArgument if invalid)
    // 2. ListAll files recursively from req.URI
    //    ExcludeHidden=true (skip .abstract.md, .overview.md, .meta.json, .relations.json)
    //    MaxDepth=req.MaxDepth
    //    Filter by FileTypes if non-empty
    // 3. Worker pool (max 20 goroutines)
    // 4. Per file: ReadRaw → decrypt if OVE1 → line-by-line regex match
    // 5. Collect matches (thread-safe)
    // 6. Return sorted by URI
}

type GrepMatch = domain.GrepMatch  // {URI, Line, Content, Matches [][]int}
```

**domain.GrepMatch:**
```go
type GrepMatch struct {
    URI     string
    Line    int       // 1-indexed
    Content string    // Full line content
    Matches [][]int   // Byte offsets of each match
}
```

### 2. `internal/usecase/glob.go` — Filename Pattern Matching

```go
type GlobUseCase struct {
    fileStore port.FileStore
}

type GlobRequest struct {
    URI     string  // Root to search
    Pattern string  // Glob pattern: e.g., "*.go", "**/test_*.md", "src/**.ts"
}

type GlobResponse struct {
    URIs []string
}

func (uc *GlobUseCase) Execute(ctx context.Context, req GlobRequest) (*GlobResponse, error) {
    // 1. ListAll files recursively
    // 2. For each file: filepath.Match(pattern, file.Name) OR path.Match for ** patterns
    // 3. Support ** glob syntax (match any path depth)
    // 4. Return matching URIs sorted alphabetically
}

// Double-star (**) matching: "**/*.go" matches any .go file at any depth
// Single-star (*) matching: "*.go" matches only in current directory level
```

### 3. `internal/usecase/relations.go` — Context Relations CRUD

```go
type RelationsUseCase struct {
    fileStore port.FileStore
    lock      *vikingfs.PathLock
}

// GetRelations — load .relations.json for the parent directory of uri
// Returns relations where SourceURI == uri OR TargetURI == uri
func (uc *RelationsUseCase) GetRelations(ctx context.Context, uri string) ([]domain.ContextRelation, error)

// AddRelation — append new relation to .relations.json (atomic with PathLock)
func (uc *RelationsUseCase) AddRelation(ctx context.Context, req AddRelationRequest) error
// Storage: {parent_dir}/.relations.json
// Format: JSON RelationsFile{Version++, UpdatedAt, Relations: [...]}

// RemoveRelation — remove matching relation
func (uc *RelationsUseCase) RemoveRelation(ctx context.Context, sourceURI, targetURI string, relType domain.RelationType) error

type AddRelationRequest struct {
    SourceURI   string
    TargetURI   string
    Relation    domain.RelationType
    Description string
}
```

**Storage convention:**
- `.relations.json` stored in parent directory of SourceURI
- Path: `{parent_of_sourceURI}/.relations.json`
- Example: source=`viking://user/a/alice/notes/foo.md` → stored in `viking://user/a/alice/notes/.relations.json`
- File is an append-only array with version counter
- PathLock acquired on `.relations.json` URI before every write

### 4. `internal/usecase/privacy.go` — Privacy Config CRUD

```go
type PrivacyUseCase struct {
    fileStore port.FileStore
}

type GetPrivacyConfigRequest struct {
    AccountID string
    UserID    string
}

type UpsertPrivacyConfigRequest struct {
    AccountID      string
    UserID         string
    ExcludeTopics  []string
    ExcludeKeywords []string
    RedactPII      bool
    UpdatedBy      string  // API Key ID for audit
}

// GetPrivacyConfig — read .privacy.json for user
func (uc *PrivacyUseCase) GetPrivacyConfig(ctx context.Context, req GetPrivacyConfigRequest) (*domain.UserPrivacyConfig, error)
// URI: viking://user/{accountID}/{userID}/.privacy.json
// If not exists → return default config (RedactPII=false, empty lists)

// UpsertPrivacyConfig — write .privacy.json + append to .privacy.history.jsonl
func (uc *PrivacyUseCase) UpsertPrivacyConfig(ctx context.Context, req UpsertPrivacyConfigRequest) error
// 1. Load existing config (for version increment)
// 2. Write new config to .privacy.json (version++)
// 3. Append to .privacy.history.jsonl: {timestamp, updatedBy, config}
```

**domain.UserPrivacyConfig:**
```go
type UserPrivacyConfig struct {
    UserID          string    `json:"user_id"`
    AccountID       string    `json:"account_id"`
    ExcludeTopics   []string  `json:"exclude_topics"`
    ExcludeKeywords []string  `json:"exclude_keywords"`
    RedactPII       bool      `json:"redact_pii"`
    Version         int       `json:"version"`
    UpdatedAt       time.Time `json:"updated_at"`
}
```

---

## Unit Tests

```
// Grep
TestGrep_FindsMatch                  → write "hello world" → grep "hello" → 1 match
TestGrep_RegexPattern                → write "foo123" → grep "[0-9]+" → match found
TestGrep_InvalidRegex                → pattern "[[invalid" → ErrInvalidArgument
TestGrep_SkipsHiddenFiles            → .abstract.md not scanned
TestGrep_DecryptsBeforeMatch         → encrypted file → decrypted in memory → grep matches
TestGrep_WorkerPoolBounded           → 30 files, max 20 workers → ≤20 concurrent reads
TestGrep_FileTypeFilter              → [".go"] filter → only .go files scanned
TestGrep_MaxDepth                    → depth=1 → only top-level files scanned
TestGrep_EmptyDir                    → no files → TotalScanned=0, empty Matches
TestGrep_MultilineNoFalsePositive    → match only in lines containing pattern

// Glob
TestGlob_StarPattern                 → "*.md" → matches only .md in root
TestGlob_DoubleStarPattern           → "**/*.go" → matches .go at any depth
TestGlob_SpecificFile                → "README.md" → matches exactly
TestGlob_NoMatches                   → "*.xyz" → empty URIs list
TestGlob_SortedOutput                → results alphabetically sorted

// Relations
TestAddRelation_AppendsToFile        → add 2 relations → file has 2
TestAddRelation_IncreasesVersion     → add → version++
TestGetRelations_FiltersByURI        → 10 relations → only related ones returned
TestRemoveRelation_RemovesFromFile   → add 3 → remove 1 → 2 remain
TestAddRelation_ConcurrentSafe       → 5 goroutines add simultaneously → all 5 present
TestAddRelation_PathLockUsed         → lock acquired before write

// Privacy
TestGetPrivacyConfig_Default         → no file → returns default config
TestUpsertPrivacyConfig_SavesFile    → upsert → .privacy.json readable
TestUpsertPrivacyConfig_VersionBump  → upsert 3x → version=3
TestUpsertPrivacyConfig_HistoryAppended → .privacy.history.jsonl has 3 lines
TestGetPrivacyConfig_ReturnsLatest   → upsert → get → same config
```

---

## Lệnh kiểm tra hoàn thành

```bash
cd /Users/binhnt/Work/blockchain/vnp-memory
go build ./services/openviking-fs/...
go test ./services/openviking-fs/internal/usecase/... -v -count=1 -run "TestGrep|TestGlob|TestRelation|TestPrivacy"
```

---

## Ghi chú triển khai

- Glob `**` support: Go `filepath.Match` không hỗ trợ `**`; cần tự implement hoặc dùng `github.com/bmatcuk/doublestar`
- Relations file: read-modify-write pattern, PathLock bắt buộc để tránh lost update
- Privacy history: `.privacy.history.jsonl` dùng `AppendLines()` từ `FileStore` interface
- Grep: errors từng file (file not readable, decrypt fail) không làm fail toàn bộ request — collect trong `Errors []error`
