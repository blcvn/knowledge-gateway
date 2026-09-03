# observe-service — Data Models

> **Service**: `services/observe-service`
> **Role**: Agent observation pipeline — captures, compresses, and stores AI agent sessions and raw observations.

---

## Session

```go
type Session struct {
    ID               string
    TenantID         string
    Project          string
    CWD              string
    Model            string
    AgentID          string
    Status           string     // "active" | "completed" | "abandoned"
    FirstPrompt      string
    Summary          string
    ObservationCount int
    Tags             []string
    CommitSHAs       []string
    StartedAt        time.Time
    EndedAt          *time.Time
    LastActiveAt     time.Time
}
```

---

## RawObservation

```go
type RawObservation struct {
    ID                string
    SessionID         string
    TenantID          string
    HookType          string
    ToolName          string
    ToolInput         []byte   // JSON
    ToolOutput        []byte   // JSON
    UserPrompt        string
    AssistantResponse string
    Modality          string   // "text" | "image"
    ImageData         string
    AgentID           string
    Raw               []byte   // full JSON payload
    Timestamp         time.Time
}
```

---

## CompressedObservation

```go
type CompressedObservation struct {
    ID         string
    SessionID  string
    TenantID   string
    ObsType    string
    Title      string
    Subtitle   string
    Facts      []string
    Narrative  string
    Concepts   []string
    Files      []string
    Importance float64
    Confidence float64
    ImageRef   string
    AgentID    string
    Timestamp  time.Time
}
```

---

## Sources
- [`services/observe-service/internal/domain/entity.go`](../../services/observe-service/internal/domain/entity.go)
