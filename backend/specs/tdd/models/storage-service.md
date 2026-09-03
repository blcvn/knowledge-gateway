# storage-service — Data Models

> **Service**: `services/storage-service`
> **Role**: Unified storage service — absorbs `ov-fs`, `ov-crypto`, `ov-resource`, `ov-session`.
> Manages encrypted file storage, resource ingestion, and agent sessions.

---

## fs — Virtual Filesystem

```go
type File struct {
    Path      string    `json:"path"`
    Content   []byte    `json:"content,omitempty"`
    Size      int64     `json:"size"`
    MimeType  string    `json:"mime_type"`
    Encrypted bool      `json:"encrypted"`
    ModTime   time.Time `json:"mod_time"`
}

type Directory struct {
    Path     string     `json:"path"`
    Children []TreeNode `json:"children"`
}

type TreeNode struct {
    Name     string     `json:"name"`
    Path     string     `json:"path"`
    IsDir    bool       `json:"is_dir"`
    Size     int64      `json:"size,omitempty"`
    Children []TreeNode `json:"children,omitempty"`
}

type GrepResult struct {
    Path    string `json:"path"`
    Line    int    `json:"line"`
    Content string `json:"content"`
    Match   string `json:"match"`
}
```

---

## resource — Resource Ingestion

```go
type Resource struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    URI       string    `json:"uri"`    // file:// | http:// | s3://
    Type      string    `json:"type"`   // "document" | "image" | "code" | "web"
    Status    string    `json:"status"` // "pending" | "processing" | "indexed" | "failed"
    EmbedPath string    `json:"embed_path,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type IngestJob struct {
    ResourceID string        `json:"resource_id"`
    URI        string        `json:"uri"`
    TenantID   string        `json:"tenant_id"`
    Options    IngestOptions `json:"options"`
    CreatedAt  time.Time     `json:"created_at"`
}

type IngestOptions struct {
    ChunkSize  int    `json:"chunk_size"`    // token chunk size
    Overlap    int    `json:"overlap"`       // overlap between chunks
    Language   string `json:"language"`      // hint for code syntax
    ExtractPDF bool   `json:"extract_pdf"`   // extract text from PDF
}
```

---

## session — Agent Sessions

```go
type Session struct {
    ID                 string
    AccountID          string
    UserID             string
    AgentID            string
    Title              string
    Status             SessionStatus  // active | committed | archived
    ArchivePath        string
    MemoriesCount      int
    CompressionVersion string
    Metadata           map[string]interface{}
    CreatedAt          time.Time
    CommittedAt        *time.Time
}

type SessionMeta struct {
    ID        string
    Title     string
    Status    SessionStatus
    CreatedAt time.Time
}

type SessionStatus string
// active | committed | archived
```

### Working Memory

```go
type WorkingMemory struct {
    SessionID string                 `json:"session_id"`
    Title     string                 `json:"title"`
    State     WMState                `json:"state"`
    Goals     []string               `json:"goals"`
    Facts     []Fact                 `json:"facts"`
    Errors    []ErrorState           `json:"errors"`
    Context   map[string]interface{} `json:"context"`
    UpdatedAt time.Time              `json:"updated_at"`
}

type WMState string
// ongoing | paused | completed

type Fact struct {
    Key        string  `json:"key"`
    Value      string  `json:"value"`
    Confidence float64 `json:"confidence"`
}

type ErrorState struct {
    Message  string `json:"message"`
    Resolved bool   `json:"resolved"`
}
```

### Message

```go
type Message struct {
    ID         string
    SessionID  string
    Role       MessageRole   // user | assistant | system | tool
    Content    string
    ToolCalls  []ToolCall
    TokenCount int
    Sequence   int
    CreatedAt  time.Time
}

type ToolCall struct {
    ID       string `json:"id"`
    Type     string `json:"type"`
    Function struct {
        Name      string `json:"name"`
        Arguments string `json:"arguments"`
    } `json:"function"`
}
```

### CandidateMemory

```go
type CandidateMemory struct {
    ID          string
    SessionID   string
    AccountID   string
    Category    MemoryCategory
    Content     string
    Confidence  float64
    DedupAction DedupAction
    FSPath      string
    CreatedAt   time.Time
}

type MemoryCategory string
// fact | preference | skill | procedure | tool_skill

type DedupAction string
// CREATE | MERGE | SKIP | ARCHIVE
```

---

## Sources
- [`services/storage-service/internal/domain/fs/entity.go`](../../services/storage-service/internal/domain/fs/entity.go)
- [`services/storage-service/internal/domain/resource/entity.go`](../../services/storage-service/internal/domain/resource/entity.go)
- [`services/storage-service/internal/domain/session/entity.go`](../../services/storage-service/internal/domain/session/entity.go)
- [`services/ov-session/domain/model/session.go`](../../services/ov-session/domain/model/session.go)
- [`services/ov-session/domain/model/message.go`](../../services/ov-session/domain/model/message.go)
- [`services/ov-session/domain/model/working_memory.go`](../../services/ov-session/domain/model/working_memory.go)
- [`services/ov-session/domain/model/memory.go`](../../services/ov-session/domain/model/memory.go)
