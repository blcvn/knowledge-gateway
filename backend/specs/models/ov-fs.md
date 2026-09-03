# ov-fs — Data Models

> **Service**: `services/ov-fs`
> **Role**: OpenViking virtual filesystem — hierarchical file/directory tree with optimistic concurrency control.

---

## FSNode

```go
type FSNode struct {
    ID             string    `json:"id"`
    TenantID       string    `json:"tenant_id"`     // Multi-tenant isolation
    ParentID       string    `json:"parent_id"`     // Nullable for root
    Name           string    `json:"name"`          // Base name of the file/folder
    Type           FileType  `json:"type"`          // FILE | DIRECTORY | SYMLINK
    Size           int64     `json:"size"`          // File size in bytes
    MimeType       string    `json:"mime_type"`     // E.g., application/pdf
    ChecksumSHA256 string    `json:"checksum"`      // Integrity hash
    Version        int32     `json:"version"`       // Optimistic concurrency control
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}

type FileType string
// FILE | DIRECTORY | SYMLINK
```

---

## TreeNode

```go
type TreeNode struct {
    Path       string
    IsDir      bool
    L0Abstract string
    Children   []*TreeNode
}

type TreeOptions struct {
    MaxDepth         int32
    IncludeAbstracts bool
}
```

---

## FileRelation

```go
type FileRelation struct {
    ID           string
    SourceFileID string
    TargetFileID string
    RelationType RelationType
    AccountID    string
    Metadata     map[string]interface{}
    CreatedAt    time.Time
}

type RelationType string
// references | extracted_from | summarizes
```

---

## Sources
- [`services/ov-fs/internal/domain/model/file.go`](../../services/ov-fs/internal/domain/model/file.go)
- [`services/ov-fs/internal/domain/model/tree.go`](../../services/ov-fs/internal/domain/model/tree.go)
- [`services/ov-fs/internal/domain/model/relation.go`](../../services/ov-fs/internal/domain/model/relation.go)
