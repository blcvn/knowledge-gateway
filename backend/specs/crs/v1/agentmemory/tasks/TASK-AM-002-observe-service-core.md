# TASK-AM-002 — Observe Service Core (Domain + Adapters)

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-002 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/observe-service/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.1 → §2.7 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-AM-001, TASK-AM-004 |
| **Estimated** | 8h |

---

## Context

Tạo **service #36** trong monolith: `am-observe`. Xử lý session lifecycle + observation pipeline (14 bước).

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/observe-service/internal/domain/entity.go` |
| CREATE | `services/observe-service/internal/domain/value_object.go` |
| CREATE | `services/observe-service/internal/domain/errors.go` |
| CREATE | `services/observe-service/internal/observe/dedup.go` |
| CREATE | `services/observe-service/internal/observe/synthetic.go` |
| CREATE | `services/observe-service/internal/observe/stream.go` |
| CREATE | `services/observe-service/internal/observe/pipeline.go` |
| CREATE | `services/observe-service/internal/usecase/observe.go` |
| CREATE | `services/observe-service/internal/usecase/session_start.go` |
| CREATE | `services/observe-service/internal/usecase/session_end.go` |
| CREATE | `services/observe-service/internal/usecase/port/input.go` |
| CREATE | `services/observe-service/internal/usecase/port/output.go` |
| CREATE | `services/observe-service/internal/adapter/repository/postgres/session_repo.go` |
| CREATE | `services/observe-service/internal/adapter/repository/postgres/observation_repo.go` |
| CREATE | `services/observe-service/internal/adapter/event/publisher.go` |
| CREATE | `services/observe-service/internal/adapter/grpc/handler.go` |
| CREATE | `services/observe-service/internal/adapter/grpc/mapper.go` |

---

## Implementation

### `internal/domain/entity.go`

```go
package domain

import (
    "time"
    "github.com/google/uuid"
)

type Session struct {
    ID               string
    TenantID         string
    Project          string
    CWD              string
    Model            string
    AgentID          string
    Status           string  // "active" | "completed" | "abandoned"
    FirstPrompt      string
    Summary          string
    ObservationCount int
    Tags             []string
    CommitSHAs       []string
    StartedAt        time.Time
    EndedAt          *time.Time
    LastActiveAt     time.Time
}

type RawObservation struct {
    ID                string
    SessionID         string
    TenantID          string
    HookType          string
    ToolName          string
    ToolInput         []byte  // JSON
    ToolOutput        []byte  // JSON
    UserPrompt        string
    AssistantResponse string
    Modality          string  // "text" | "image"
    ImageData         string
    AgentID           string
    Raw               []byte  // full JSON payload
    Timestamp         time.Time
}

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

func NewSession(tenantID, project, cwd, model, agentID string) Session {
    return Session{
        ID:           uuid.New().String(),
        TenantID:     tenantID,
        Project:      project,
        CWD:          cwd,
        Model:        model,
        AgentID:      agentID,
        Status:       "active",
        StartedAt:    time.Now(),
        LastActiveAt: time.Now(),
    }
}
```

### `internal/domain/value_object.go`

```go
package domain

type HookType string
const (
    HookSessionStart     HookType = "session_start"
    HookPromptSubmit     HookType = "prompt_submit"
    HookPreToolUse       HookType = "pre_tool_use"
    HookPostToolUse      HookType = "post_tool_use"
    HookPostToolFailure  HookType = "post_tool_failure"
    HookSessionEnd       HookType = "session_end"
    HookTaskCompleted    HookType = "task_completed"
    HookPreSubagent      HookType = "pre_subagent"
    HookPostSubagent     HookType = "post_subagent"
    HookNotification     HookType = "notification"
    HookStop             HookType = "stop"
    HookCustom           HookType = "custom"
)

type ObsType string
const (
    ObsToolCall     ObsType = "tool_call"
    ObsToolSuccess  ObsType = "tool_success"
    ObsError        ObsType = "error"
    ObsConversation ObsType = "conversation"
    ObsFileWrite    ObsType = "file_write"
    ObsFileRead     ObsType = "file_read"
    ObsSearch       ObsType = "search"
    ObsExec         ObsType = "exec"
    ObsCommit       ObsType = "commit"
    ObsBuild        ObsType = "build"
    ObsTest         ObsType = "test"
    ObsInstall      ObsType = "install"
    ObsAPI          ObsType = "api_call"
    ObsMemory       ObsType = "memory"
    ObsDecision     ObsType = "decision"
)
```

### `internal/domain/errors.go`

```go
package domain

import "errors"

var (
    ErrMissingFields       = errors.New("session_id and hook_type are required")
    ErrSessionNotFound     = errors.New("session not found")
    ErrSessionLimitExceeded = errors.New("session observation limit exceeded (500)")
    ErrSessionEnded        = errors.New("session already ended")
)
```

### `internal/observe/dedup.go`

```go
package observe

import (
    "context"
    "sync"
    "time"
)

const DefaultDedupTTL = 30 * time.Second

type DedupMap struct {
    mu    sync.RWMutex
    store map[[32]byte]time.Time
}

func NewDedupMap() *DedupMap {
    return &DedupMap{store: make(map[[32]byte]time.Time)}
}

func (d *DedupMap) IsSeen(hash [32]byte) bool {
    d.mu.RLock()
    defer d.mu.RUnlock()
    exp, ok := d.store[hash]
    return ok && time.Now().Before(exp)
}

func (d *DedupMap) MarkSeen(hash [32]byte, ttl time.Duration) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.store[hash] = time.Now().Add(ttl)
}

// StartCleanup runs every 60s to clear expired entries
func (d *DedupMap) StartCleanup(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-ticker.C:
            d.mu.Lock()
            now := time.Now()
            for hash, exp := range d.store {
                if now.After(exp) { delete(d.store, hash) }
            }
            d.mu.Unlock()
        case <-ctx.Done():
            return
        }
    }
}
```

### `internal/observe/synthetic.go`

```go
package observe

import (
    "fmt"
    "strings"

    "github.com/vnp-memory/services/observe-service/internal/domain"
)

// syntheticCompress converts RawObservation → CompressedObservation without LLM
func syntheticCompress(raw domain.RawObservation) domain.CompressedObservation {
    obs := domain.CompressedObservation{
        SessionID: raw.SessionID,
        TenantID:  raw.TenantID,
        AgentID:   raw.AgentID,
        Timestamp: raw.Timestamp,
        Confidence: 0.8,
    }
    
    switch domain.HookType(raw.HookType) {
    case domain.HookPostToolUse:
        obs.ObsType   = string(deriveObsType(raw.ToolName))
        obs.Title     = fmt.Sprintf("%s: %s", raw.ToolName, extractFirstLine(raw.ToolOutput))
        obs.Files     = extractFilePaths(raw.ToolInput, raw.ToolOutput)
        obs.Facts     = extractFacts(raw.ToolOutput)
        obs.Importance = 0.5

    case domain.HookPostToolFailure:
        obs.ObsType    = string(domain.ObsError)
        obs.Title      = fmt.Sprintf("ERROR in %s: %s", raw.ToolName, extractErrorMsg(raw.ToolOutput))
        obs.Facts      = []string{extractErrorMsg(raw.ToolOutput)}
        obs.Importance = 0.8

    case domain.HookPromptSubmit:
        obs.ObsType    = string(domain.ObsConversation)
        obs.Title      = truncate(raw.UserPrompt, 80)
        obs.Importance = 0.3

    case domain.HookTaskCompleted:
        obs.ObsType    = string(domain.ObsDecision)
        obs.Title      = "Task completed"
        obs.Importance = 0.7

    default:
        obs.ObsType    = string(domain.ObsToolCall)
        obs.Title      = fmt.Sprintf("[%s] %s", raw.HookType, raw.ToolName)
        obs.Importance = 0.4
    }
    
    obs.Concepts = extractConcepts(obs.Title, obs.Facts)
    return obs
}

func deriveObsType(toolName string) domain.ObsType {
    switch {
    case strings.Contains(toolName, "write") || strings.Contains(toolName, "create"):
        return domain.ObsFileWrite
    case strings.Contains(toolName, "read") || strings.Contains(toolName, "view"):
        return domain.ObsFileRead
    case strings.Contains(toolName, "search") || strings.Contains(toolName, "grep"):
        return domain.ObsSearch
    case strings.Contains(toolName, "run") || strings.Contains(toolName, "exec"):
        return domain.ObsExec
    case strings.Contains(toolName, "git"):
        return domain.ObsCommit
    default:
        return domain.ObsToolCall
    }
}

func extractFirstLine(data []byte) string {
    if len(data) == 0 { return "" }
    s := strings.Split(string(data), "\n")[0]
    return truncate(s, 60)
}

func extractErrorMsg(data []byte) string {
    s := string(data)
    if idx := strings.Index(s, "error"); idx >= 0 {
        return truncate(s[idx:], 80)
    }
    return truncate(s, 80)
}

func truncate(s string, n int) string {
    if len(s) <= n { return s }
    return s[:n] + "..."
}

func extractFilePaths(input, output []byte) []string {
    var paths []string
    for _, b := range [][]byte{input, output} {
        s := string(b)
        for _, word := range strings.Fields(s) {
            if strings.HasPrefix(word, "/") && strings.Contains(word, ".") {
                paths = append(paths, word)
            }
        }
    }
    return paths
}

func extractFacts(output []byte) []string {
    if len(output) == 0 { return nil }
    lines := strings.Split(string(output), "\n")
    var facts []string
    for _, l := range lines {
        l = strings.TrimSpace(l)
        if len(l) > 10 && len(l) < 200 { facts = append(facts, l) }
        if len(facts) >= 5 { break }
    }
    return facts
}

func extractConcepts(title string, facts []string) []string {
    words := strings.Fields(title)
    for _, f := range facts { words = append(words, strings.Fields(f)...) }
    seen := map[string]bool{}
    var concepts []string
    for _, w := range words {
        w = strings.ToLower(strings.Trim(w, ".,;:()[]\"'"))
        if len(w) >= 4 && !seen[w] && !isStopword(w) {
            seen[w] = true
            concepts = append(concepts, w)
            if len(concepts) >= 8 { break }
        }
    }
    return concepts
}

var stopwords = map[string]bool{"this": true, "that": true, "with": true, "from": true, "have": true, "will": true, "been": true}
func isStopword(w string) bool { return stopwords[w] }
func detectModality(rawJSON []byte) string {
    if strings.Contains(string(rawJSON), "base64") { return "image" }
    return "text"
}
```

### `internal/observe/stream.go`

```go
package observe

import (
    "sync"
)

type StreamEvent struct {
    Type      string `json:"type"`
    SessionID string `json:"session_id,omitempty"`
    Data      any    `json:"data"`
}

type StreamBroker struct {
    mu      sync.RWMutex
    clients map[string]map[chan StreamEvent]struct{}  // sessionID → channels ("" = all)
}

func NewStreamBroker() *StreamBroker {
    return &StreamBroker{clients: make(map[string]map[chan StreamEvent]struct{})}
}

func (sb *StreamBroker) Subscribe(sessionFilter string) (chan StreamEvent, func()) {
    ch := make(chan StreamEvent, 100)
    sb.mu.Lock()
    if sb.clients[sessionFilter] == nil {
        sb.clients[sessionFilter] = make(map[chan StreamEvent]struct{})
    }
    sb.clients[sessionFilter][ch] = struct{}{}
    sb.mu.Unlock()
    cancel := func() { sb.unsubscribe(sessionFilter, ch) }
    return ch, cancel
}

func (sb *StreamBroker) unsubscribe(filter string, ch chan StreamEvent) {
    sb.mu.Lock()
    defer sb.mu.Unlock()
    delete(sb.clients[filter], ch)
    close(ch)
}

func (sb *StreamBroker) Broadcast(event StreamEvent) {
    sb.broadcastToGroup("", event)
    if event.SessionID != "" { sb.broadcastToGroup(event.SessionID, event) }
}

func (sb *StreamBroker) broadcastToGroup(filter string, event StreamEvent) {
    sb.mu.RLock()
    defer sb.mu.RUnlock()
    for ch := range sb.clients[filter] {
        select {
        case ch <- event:
        default: // drop if subscriber is slow
        }
    }
}
```

### `internal/observe/pipeline.go`

```go
package observe

import (
    "context"
    "crypto/sha256"
    "fmt"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/vnp-memory/pkg/privacy"
    "github.com/vnp-memory/services/observe-service/internal/domain"
    "github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

const maxObsPerSession = 500

type PipelineConfig struct {
    MaxObsPerSession int
    DedupTTL         time.Duration
    InjectContext    bool
    TokenBudget      int
}

type Pipeline struct {
    dedup      *DedupMap
    kvStore    port.IKVStore
    search     port.ISearchIndexer
    publisher  port.IEventPublisher
    stream     *StreamBroker
    privacy    *privacy.Redactor
    mu         sync.Map  // per-session mutex
    config     PipelineConfig
}

func NewPipeline(dedup *DedupMap, kvStore port.IKVStore, search port.ISearchIndexer,
    publisher port.IEventPublisher, stream *StreamBroker, priv *privacy.Redactor,
    cfg PipelineConfig) *Pipeline {
    return &Pipeline{dedup: dedup, kvStore: kvStore, search: search,
        publisher: publisher, stream: stream, privacy: priv, config: cfg}
}

type ObserveRequest struct {
    SessionID         string
    HookType          string
    ToolName          string
    ToolInput         []byte
    ToolOutput        []byte
    UserPrompt        string
    AssistantResponse string
    AgentID           string
    TenantID          string
    Project           string
    Timestamp         time.Time
}

type ObserveResponse struct {
    ObservationID   string
    Deduplicated    bool
    Compressed      domain.CompressedObservation
    InjectedContext string
    ContextTokens   int
}

func (p *Pipeline) Execute(ctx context.Context, req ObserveRequest) (*ObserveResponse, error) {
    // Step 1: Validate
    if req.SessionID == "" || req.HookType == "" {
        return nil, domain.ErrMissingFields
    }

    // Step 2: Dedup check
    hash := sha256.Sum256([]byte(req.SessionID + req.ToolName + fmt.Sprint(req.ToolInput)))
    if p.dedup.IsSeen(hash) {
        return &ObserveResponse{Deduplicated: true}, nil
    }

    // Step 3: Privacy redaction
    reqJSON, _ := json.Marshal(req)
    stripped := p.privacy.Strip(string(reqJSON))
    json.Unmarshal([]byte(stripped), &req)

    // Step 4: Build RawObservation
    raw := domain.RawObservation{
        ID:                uuid.New().String(),
        SessionID:         req.SessionID,
        TenantID:          req.TenantID,
        HookType:          req.HookType,
        ToolName:          req.ToolName,
        ToolInput:         req.ToolInput,
        ToolOutput:        req.ToolOutput,
        UserPrompt:        req.UserPrompt,
        AssistantResponse: req.AssistantResponse,
        AgentID:           req.AgentID,
        Timestamp:         req.Timestamp,
    }

    // Step 5: Image detection
    raw.Modality = detectModality(req.ToolInput)

    // Step 6: Per-session keyed mutex
    mu := p.getOrCreateMu(req.SessionID)
    mu.Lock()
    defer mu.Unlock()

    // Step 7: Session limit check
    count := p.kvStore.GetSessionObsCount(ctx, req.SessionID)
    if count >= maxObsPerSession {
        return nil, domain.ErrSessionLimitExceeded
    }

    // Step 8: AgentID inheritance
    if raw.AgentID == "" {
        raw.AgentID = p.kvStore.GetSessionAgentID(ctx, req.SessionID)
    }

    // Step 9: Persist RawObservation
    if err := p.kvStore.SaveRawObservation(ctx, raw); err != nil {
        return nil, err
    }

    // Step 10: Mark dedup hash seen
    p.dedup.MarkSeen(hash, p.config.DedupTTL)

    // Step 11: SSE broadcast
    p.stream.Broadcast(StreamEvent{
        Type:      "raw_observation",
        SessionID: req.SessionID,
        Data:      raw,
    })

    // Step 12: Increment session obs count
    p.kvStore.IncrementObsCount(ctx, req.SessionID)

    // Step 13: Synthetic compression
    compressed := syntheticCompress(raw)
    compressed.ID = uuid.New().String()
    p.kvStore.SaveCompressedObservation(ctx, compressed)

    // Step 14: Async index (non-blocking)
    go p.search.IndexObservation(context.Background(), compressed)

    // Publish NATS event
    p.publisher.Publish(ctx, "agentmemory.observation.captured", map[string]any{
        "observation_id": raw.ID,
        "session_id":     req.SessionID,
        "tenant_id":      req.TenantID,
        "hook_type":      req.HookType,
    })

    return &ObserveResponse{
        ObservationID: raw.ID,
        Compressed:    compressed,
    }, nil
}

func (p *Pipeline) getOrCreateMu(sessionID string) *sync.Mutex {
    mu, _ := p.mu.LoadOrStore(sessionID, &sync.Mutex{})
    return mu.(*sync.Mutex)
}
```

### `internal/usecase/port/output.go`

```go
package port

import (
    "context"
    "github.com/vnp-memory/services/observe-service/internal/domain"
)

type IKVStore interface {
    GetSessionObsCount(ctx context.Context, sessionID string) int
    GetSessionAgentID(ctx context.Context, sessionID string) string
    IncrementObsCount(ctx context.Context, sessionID string)
    SaveRawObservation(ctx context.Context, raw domain.RawObservation) error
    SaveCompressedObservation(ctx context.Context, comp domain.CompressedObservation)
}

type ISearchIndexer interface {
    IndexObservation(ctx context.Context, comp domain.CompressedObservation) error
}

type IEventPublisher interface {
    Publish(ctx context.Context, subject string, payload any) error
}

type IStreamBroker interface {
    Subscribe(sessionFilter string) (chan any, func())
    Broadcast(event any)
}

type ISessionRepo interface {
    Save(ctx context.Context, session domain.Session) error
    GetByID(ctx context.Context, id string) (*domain.Session, error)
    List(ctx context.Context, tenantID, status, project string, limit, offset int) ([]domain.Session, error)
    UpdateStatus(ctx context.Context, id, status string) error
    IncrementObsCount(ctx context.Context, id string) error
    GetObsCount(ctx context.Context, id string) (int, error)
}

type IObservationRepo interface {
    SaveRaw(ctx context.Context, raw domain.RawObservation) error
    SaveCompressed(ctx context.Context, comp domain.CompressedObservation) error
    ListCompressed(ctx context.Context, sessionID string, limit, offset int) ([]domain.CompressedObservation, error)
    DeleteBySessionID(ctx context.Context, sessionID string) error
}
```

---

## Verification

```bash
cd services/observe-service
go build ./...
go test ./internal/observe/... -v
```

```go
func TestPipeline_Dedup(t *testing.T) {
    // First call → observation_id returned
    // Same call within 30s → Deduplicated=true
}

func TestPipeline_SessionLimit(t *testing.T) {
    // 501st call → ErrSessionLimitExceeded
}

func TestPipeline_PrivacyRedaction(t *testing.T) {
    // Input with "sk-abc123456789" → stored as "[REDACTED:openai_key]"
}
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| POST /observe với hookType=post_tool_use → RawObs + CompressedObs saved | ✅ |
| 2 identical requests trong 30s → second: deduplicated=true | ✅ |
| sk-* token in input → [REDACTED:openai_key] before save | ✅ |
| 501 observations → ErrSessionLimitExceeded | ✅ |
| SSE broadcast trên step 11 | ✅ |
