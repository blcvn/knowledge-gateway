# memobase-* — Data Models

> **Services**: `services/memobase-admin`, `services/memobase-engine`, `services/memobase-context`, `services/memobase-ingestion`, `services/memobase-pipeline`, `services/memobase-event`
> **Role**: Memobase working memory engine — user profiling from conversation blobs with per-project configuration.

---

## memobase-admin — Project & User Management

### Project

```go
type Project struct {
    ProjectID     string
    ProjectSecret string        // bcrypt hash stored in DB
    ProfileConfig string        // YAML string
    Status        ProjectStatus // active | suspended
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type ProjectProfileConfig struct {
    MaxSubtopics     int      `yaml:"max_subtopics"`
    MaxSlotTokenSize int      `yaml:"max_slot_token_size"`
    StrictMode       bool     `yaml:"strict_mode"`
    ValidateMode     bool     `yaml:"validate_mode"`
    Language         string   `yaml:"language"` // "en" | "zh"
    AdditionalTopics []string `yaml:"additional_topics"`
}
```

### User (in project)

```go
type User struct {
    ID        string
    ProjectID string
    Metadata  map[string]any
    CreatedAt time.Time
    UpdatedAt time.Time
}

type UserStatus struct {
    ID         string
    ProjectID  string
    UserID     string
    StatusData map[string]any
    CreatedAt  time.Time
    UpdatedAt  time.Time
}

// ProjectContext is the resolved caller identity after token validation.
type ProjectContext struct {
    ProjectID string
}
```

---

## memobase-engine — Core Memory Engine

### Blob

```go
type Blob struct {
    ID        uuid.UUID
    UserID    string
    ProjectID string
    Type      BlobType
    Content   string
    CreatedAt time.Time
}

type BlobType string
// chat | doc | summary
```

### Profile

```go
type Profile struct {
    ID         uuid.UUID
    UserID     string
    ProjectID  string
    Topic      string
    SubTopic   string
    Content    string
    Attributes map[string]interface{}
    UpdatedAt  time.Time
}

type ProfileTopic struct {
    Topic       string `json:"topic"`
    SubTopic    string `json:"sub_topic"`
    Description string `json:"description"`
}
```

### UserEvent (Engine-level)

```go
type UserEvent struct {
    ID        uuid.UUID
    UserID    string
    ProjectID string
    EventData map[string]interface{}
    Embedding []float32
    CreatedAt time.Time
}

type EventGist struct {
    ID        uuid.UUID
    UserID    string
    ProjectID string
    EventID   uuid.UUID
    GistData  map[string]interface{}
    Embedding []float32
    CreatedAt time.Time
}
```

---

## memobase-context — Context Retrieval

### Profile (Context)

```go
type Profile struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    ProjectID string    `json:"project_id"`
    Content   string    `json:"content"`
    Topic     string    `json:"topic"`
    SubTopic  string    `json:"sub_topic"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### ContextResult

```go
type ContextResult struct {
    Context      string `json:"context"`
    ProfileCount int32  `json:"profile_count"`
    EventCount   int32  `json:"event_count"`
    TotalTokens  int32  `json:"total_tokens"`
}
```

### TruncationPolicy

```go
type TruncationPolicy struct {
    PreferTopics    []string
    OnlyTopics      []string
    MaxTokenSize    int32
    MaxSubtopicSize int32
}
```

---

## Sources
- [`services/memobase-admin/internal/domain/project.go`](../../services/memobase-admin/internal/domain/project.go)
- [`services/memobase-engine/internal/domain/model/blob.go`](../../services/memobase-engine/internal/domain/model/blob.go)
- [`services/memobase-engine/internal/domain/model/profile.go`](../../services/memobase-engine/internal/domain/model/profile.go)
- [`services/memobase-engine/internal/domain/model/event.go`](../../services/memobase-engine/internal/domain/model/event.go)
- [`services/memobase-context/domain/model/profile.go`](../../services/memobase-context/domain/model/profile.go)
- [`services/memobase-context/domain/model/context.go`](../../services/memobase-context/domain/model/context.go)
- [`services/memobase-context/domain/model/event_gist.go`](../../services/memobase-context/domain/model/event_gist.go)
