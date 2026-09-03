# ov-resource — Data Models

> **Service**: `services/ov-resource`
> **Role**: OpenViking resource ingestion — parses files into chunks for indexing, with filesystem watch support.

---

## Resource

```go
type Resource struct {
    ID              string
    AccountID       string
    SourcePath      string
    TargetPath      string
    Filename        string
    MimeType        string
    ParserType      ResourceType
    ChunkCount      int
    TotalTokens     int
    ContentHash     string
    Status          ResourceStatus
    ErrorMessage    string
    ParseDurationMs int
    IngestedAt      time.Time
    CreatedAt       time.Time
}

type ResourceType string
// code | document | markdown | default

type ResourceStatus string
// pending | processing | completed | failed
```

---

## IngestionResult

```go
type IngestionResult struct {
    ResourceID      string
    ChunksCount     int
    TotalTokens     int
    ParseDurationMs int
    TargetPath      string
}
```

---

## Chunk

```go
type Chunk struct {
    ID       string
    Content  string
    Metadata ChunkMetadata
}

type ChunkMetadata struct {
    StartLine   int
    EndLine     int
    TotalTokens int
    ASTNodeType string
    ASTNodePath string
}
```

---

## WatchTask

```go
type WatchTask struct {
    ID             string
    AccountID      string
    SourcePath     string
    TargetPath     string
    Patterns       []string
    PollIntervalMs int64
    Status         WatchStatus
    LastPollAt     time.Time
    FilesTracked   int
    CreatedAt      time.Time
    UpdatedAt      time.Time
}

type WatchStatus string
// active | paused | deleted

type WatchEvent struct {
    Type      EventType   // CREATED | MODIFIED | DELETED
    Path      string
    Timestamp time.Time
}
```

---

## Sources
- [`services/ov-resource/internal/domain/model/resource.go`](../../services/ov-resource/internal/domain/model/resource.go)
- [`services/ov-resource/internal/domain/model/chunk.go`](../../services/ov-resource/internal/domain/model/chunk.go)
- [`services/ov-resource/internal/domain/model/watch.go`](../../services/ov-resource/internal/domain/model/watch.go)
