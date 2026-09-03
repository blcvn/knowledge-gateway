# Solution: SOL-001 — Observe Service

**CR ID:** CR-AM-001  
**Solution ID:** SOL-001  
**Priority:** Critical (Wave 1)  
**Architecture:** VNP Memory Monolith (`apps/memory/`) + Gateway

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- VNP Memory dùng **InProcessRegistry** (gRPC bufconn) — mọi service giao tiếp qua in-memory transport.
- Gateway expose REST API, forward vào services qua `InProcessRegistry`.
- `services/memory-service/` có `IngestUseCase` nhưng không có hook pipeline.
- NATS JetStream embedded trong monolith — sẵn sàng cho event publishing.
- PostgreSQL là primary store — thêm tables mới cùng migration.

---

## 2. Giải pháp

### 2.1. Tạo `services/observe-service/` — Service mới

Observe Service sẽ là service thứ 36 trong monolith, đăng ký vào `InProcessRegistry`.

```
services/observe-service/
├── cmd/observe/main.go               # Standalone binary (dev/test)
├── internal/
│   ├── domain/
│   │   ├── entity.go                 # Session, RawObservation, CompressedObservation
│   │   ├── value_object.go           # HookType (12 types), ObservationType (15 types)
│   │   └── errors.go
│   ├── observe/
│   │   ├── pipeline.go               # 14-step pipeline entry point
│   │   ├── dedup.go                  # DedupMap[SHA256 → expireAt], 30s TTL
│   │   ├── synthetic.go              # Synthetic compression (zero LLM)
│   │   └── stream.go                 # StreamBroker (SSE fan-out)
│   ├── usecase/
│   │   ├── observe.go                # ObserveUseCase (orchestrates pipeline)
│   │   ├── session_start.go          # CreateSession
│   │   ├── session_end.go            # EndSession + trigger summarize
│   │   └── port/
│   │       ├── input.go              # IObserveUseCase interface
│   │       └── output.go             # IKVStore, ISearchIndexer, IEventPublisher, IStreamBroker
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── handler.go            # gRPC server (proto: ObserveService)
│   │   │   └── mapper.go
│   │   ├── http/
│   │   │   ├── handler.go            # HTTP endpoints (fallback / dev mode)
│   │   │   └── middleware.go         # HMAC bearer auth
│   │   ├── repository/
│   │   │   └── postgres/
│   │   │       ├── session_repo.go
│   │   │       └── observation_repo.go
│   │   └── event/
│   │       └── publisher.go          # NATS: agentmemory.session.* , agentmemory.observation.*
└── api/proto/observe/v1/observe.proto
```

### 2.2. Protobuf Service Definition

```protobuf
// api/proto/observe/v1/observe.proto
service ObserveService {
  rpc Observe(ObserveRequest) returns (ObserveResponse);
  rpc StartSession(StartSessionRequest) returns (StartSessionResponse);
  rpc EndSession(EndSessionRequest) returns (EndSessionResponse);
  rpc GetSession(GetSessionRequest) returns (GetSessionResponse);
  rpc ListSessions(ListSessionsRequest) returns (ListSessionsResponse);
  rpc GetObservations(GetObservationsRequest) returns (GetObservationsResponse);
  rpc DeleteSession(DeleteSessionRequest) returns (DeleteSessionResponse);
  rpc StreamEvents(StreamEventsRequest) returns (stream StreamEvent); // server-side stream
}

message ObserveRequest {
  string session_id = 1;
  string hook_type = 2;        // HookType enum value
  string tool_name = 3;
  bytes tool_input = 4;        // JSON
  bytes tool_output = 5;       // JSON
  string user_prompt = 6;
  string assistant_response = 7;
  string agent_id = 8;
  string tenant_id = 9;
  google.protobuf.Timestamp timestamp = 10;
}

message ObserveResponse {
  string observation_id = 1;
  bool deduplicated = 2;
  CompressedObservationProto compressed = 3;
}
```

### 2.3. 14-Step Pipeline — Chi tiết triển khai

```go
// internal/observe/pipeline.go

type Pipeline struct {
    dedup        *DedupMap
    kvStore      port.IKVStore
    searchClient port.ISearchIndexer    // HTTP client to am-search service
    publisher    port.IEventPublisher   // NATS
    stream       *StreamBroker
    privacyPkg   *privacy.Redactor     // pkg/privacy/redact.go
    mu           sync.Map              // per-session keyed mutex
}

func (p *Pipeline) Execute(ctx context.Context, req ObserveRequest) (*ObserveResponse, error) {
    // Step 1: Validate
    if req.SessionID == "" || req.HookType == "" { return nil, ErrMissingFields }
    
    // Step 2: Dedup check
    hash := sha256.Sum256([]byte(req.SessionID + req.ToolName + fmt.Sprint(req.ToolInput)))
    if p.dedup.IsSeen(hash) { return &ObserveResponse{Deduplicated: true}, nil }
    
    // Step 3: Privacy redaction
    req = p.privacyPkg.RedactRequest(req)
    
    // Step 4: Build RawObservation
    raw := buildRawObservation(req)
    
    // Step 5: Image detection (detect base64 in payload → set modality)
    raw.Modality = detectModality(req)
    
    // Step 6: Per-session keyed mutex (ordering guarantee)
    mu := p.getSessionMutex(req.SessionID)
    mu.Lock()
    defer mu.Unlock()
    
    // Step 7: Session limit check (MAX_OBS_PER_SESSION default 500)
    count := p.kvStore.GetSessionObsCount(ctx, req.SessionID)
    if count >= maxObsPerSession { return nil, ErrSessionLimitExceeded }
    
    // Step 8: AgentID inheritance
    if raw.AgentID == "" { raw.AgentID = p.kvStore.GetSessionAgentID(ctx, req.SessionID) }
    
    // Step 9: KV Write (persist RawObservation to PostgreSQL)
    if err := p.kvStore.SaveRawObservation(ctx, raw); err != nil { return nil, err }
    
    // Step 10: Mark dedup hash as seen (30s TTL)
    p.dedup.MarkSeen(hash, 30*time.Second)
    
    // Step 11: SSE broadcast (raw_observation event)
    p.stream.Broadcast(StreamEvent{Type: "raw_observation", Data: raw})
    
    // Step 12: Session update (increment count, update lastActive)
    p.kvStore.IncrementObsCount(ctx, req.SessionID)
    
    // Step 13: Synthetic compression
    compressed := syntheticCompress(raw)
    p.kvStore.SaveCompressedObservation(ctx, compressed)
    
    // Step 14: Async BM25 + Vector index (non-blocking)
    go p.searchClient.IndexObservation(ctx, compressed)
    
    // Publish NATS event
    p.publisher.Publish(ctx, "agentmemory.observation.captured", ObservationEvent{...})
    
    return &ObserveResponse{ObservationID: raw.ID, Compressed: compressed}, nil
}
```

### 2.4. DedupMap Implementation

```go
// internal/observe/dedup.go
type DedupMap struct {
    mu    sync.RWMutex
    store map[[32]byte]time.Time // hash → expireAt
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

// Cleanup goroutine: scan expired entries every 60s
func (d *DedupMap) StartCleanup(ctx context.Context) { ... }
```

### 2.5. Synthetic Compression (Zero LLM)

```go
// internal/observe/synthetic.go
// Rules: build CompressedObservation without LLM

func syntheticCompress(raw RawObservation) CompressedObservation {
    obs := CompressedObservation{
        ID:        newID(),
        SessionID: raw.SessionID,
        Timestamp: raw.Timestamp,
        AgentID:   raw.AgentID,
    }
    switch raw.HookType {
    case HookPostToolUse:
        obs.Type = deriveObsType(raw.ToolName)
        obs.Title = fmt.Sprintf("%s: %s", raw.ToolName, extractFirstLine(raw.ToolOutput))
        obs.Files = extractFilePaths(raw.ToolInput, raw.ToolOutput)
        obs.Facts = extractFacts(raw.ToolOutput)
        obs.Importance = 0.5
    case HookPostToolFailure:
        obs.Type = ObsError
        obs.Title = fmt.Sprintf("ERROR in %s: %s", raw.ToolName, extractErrorMsg(raw.ToolOutput))
        obs.Importance = 0.8 // Errors are important
    case HookPromptSubmit:
        obs.Type = ObsConversation
        obs.Title = truncate(raw.UserPrompt, 80)
        obs.Importance = 0.3
    // ... other hook types
    }
    obs.Concepts = extractConcepts(obs.Title, obs.Facts)
    return obs
}
```

### 2.6. SSE StreamBroker

```go
// internal/observe/stream.go
type StreamBroker struct {
    mu      sync.RWMutex
    clients map[string]map[chan StreamEvent]struct{}  // sessionId → channels (empty string = all)
}

func (sb *StreamBroker) Subscribe(sessionFilter string) (chan StreamEvent, func()) {
    ch := make(chan StreamEvent, 100)
    sb.mu.Lock()
    if sb.clients[sessionFilter] == nil { sb.clients[sessionFilter] = make(map[chan StreamEvent]struct{}) }
    sb.clients[sessionFilter][ch] = struct{}{}
    sb.mu.Unlock()
    cancel := func() { sb.Unsubscribe(sessionFilter, ch) }
    return ch, cancel
}

func (sb *StreamBroker) Broadcast(event StreamEvent) {
    // Broadcast to all-sessions subscribers AND session-specific subscribers
    sb.broadcastToGroup("", event)
    if event.SessionID != "" { sb.broadcastToGroup(event.SessionID, event) }
}
```

### 2.7. PostgreSQL Schema

```sql
-- Migration: 0010_observe_service.up.sql

CREATE TABLE agent_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       TEXT NOT NULL,
    project         TEXT,
    cwd             TEXT,
    model           TEXT,
    agent_id        TEXT,
    status          TEXT NOT NULL DEFAULT 'active',  -- active | completed | abandoned
    first_prompt    TEXT,
    summary         TEXT,
    observation_count INT DEFAULT 0,
    tags            TEXT[],
    commit_shas     TEXT[],
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ,
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE agent_raw_observations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    tenant_id           TEXT NOT NULL,
    hook_type           TEXT NOT NULL,
    tool_name           TEXT,
    tool_input          JSONB,
    tool_output         JSONB,
    user_prompt         TEXT,
    assistant_response  TEXT,
    modality            TEXT DEFAULT 'text',
    image_data          TEXT,
    agent_id            TEXT,
    raw                 JSONB,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE agent_compressed_observations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id  UUID NOT NULL REFERENCES agent_sessions(id) ON DELETE CASCADE,
    tenant_id   TEXT NOT NULL,
    obs_type    TEXT NOT NULL,
    title       TEXT NOT NULL,
    subtitle    TEXT,
    facts       TEXT[],
    narrative   TEXT,
    concepts    TEXT[],
    files       TEXT[],
    importance  FLOAT8 DEFAULT 0.5,
    confidence  FLOAT8 DEFAULT 0.8,
    image_ref   TEXT,
    agent_id    TEXT,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_sessions_tenant ON agent_sessions(tenant_id, status);
CREATE INDEX idx_raw_obs_session ON agent_raw_observations(session_id, timestamp);
CREATE INDEX idx_compressed_obs_session ON agent_compressed_observations(session_id, importance DESC);
```

### 2.8. Bootstrap Integration (Monolith)

```go
// apps/memory/internal/bootstrap/observe.go
package bootstrap

func InitObserveService(reg *bus.InProcessRegistry, db *sql.DB, nats *nats.Conn, cfg *config.Config) {
    // Build dependencies
    sessionRepo    := postgres.NewSessionRepo(db)
    obsRepo        := postgres.NewObservationRepo(db)
    searchClient   := httpclient.NewSearchClient(cfg.ObserveSearch.URL)
    publisher      := natevent.NewPublisher(nats, "agentmemory")
    streamBroker   := observestream.NewBroker()
    privacyRedactor := privacy.NewRedactor()
    dedup          := observe.NewDedupMap()
    
    pipeline := observe.NewPipeline(dedup, obsRepo, searchClient, publisher, streamBroker, privacyRedactor, cfg.Observe)
    
    usecase := usecase.NewObserveUseCase(pipeline, sessionRepo, publisher)
    
    // Register gRPC service
    grpcServer := grpc.NewServer()
    observepb.RegisterObserveServiceServer(grpcServer, grpchandler.NewHandler(usecase))
    reg.Register("am-observe", grpcServer)
    
    // Start background DedupMap cleanup
    go dedup.StartCleanup(context.Background())
}
```

### 2.9. Gateway Routes

```go
// gateway/internal/adapter/handler/router.go — thêm vào route registration

// Observe Service routes (proxy to am-observe via InProcessRegistry)
r.Post("/v1/observe",                        h.ForwardTo("am-observe", "ObserveService/Observe"))
r.Post("/v1/observe/session/start",          h.ForwardTo("am-observe", "ObserveService/StartSession"))
r.Post("/v1/observe/session/end",            h.ForwardTo("am-observe", "ObserveService/EndSession"))
r.Get("/v1/sessions",                        h.ForwardTo("am-observe", "ObserveService/ListSessions"))
r.Get("/v1/sessions/{id}",                   h.ForwardTo("am-observe", "ObserveService/GetSession"))
r.Get("/v1/sessions/{id}/observations",      h.ForwardTo("am-observe", "ObserveService/GetObservations"))
r.Delete("/v1/sessions/{id}",                h.ForwardTo("am-observe", "ObserveService/DeleteSession"))
// SSE: special handler (không dùng gRPC proxy vì SSE cần HTTP streaming)
r.Get("/v1/stream",                          sseHandler.ServeSSE)
```

### 2.10. Privacy Package

```go
// pkg/privacy/redact.go — NEW shared package

type Redactor struct {
    patterns []redactPattern
}

type redactPattern struct {
    name string
    re   *regexp.Regexp
    tag  string  // [REDACTED:name]
}

func NewRedactor() *Redactor {
    return &Redactor{patterns: []redactPattern{
        {"bearer_token", regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._-]{20,}`), ""},
        {"openai_key",   regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`), ""},
        {"aws_key",      regexp.MustCompile(`AKIA[A-Z0-9]{16}`), ""},
        {"private_key",  regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`), ""},
        {"jwt_token",    regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`), ""},
        {"github_pat",   regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`), ""},
        {"env_secret",   regexp.MustCompile(`(?m)^[A-Z_]+=["']?[A-Za-z0-9+/=]{20,}["']?$`), ""},
    }}
}

func (r *Redactor) Strip(input string) string {
    for _, p := range r.patterns {
        input = p.re.ReplaceAllString(input, "[REDACTED:"+p.name+"]")
    }
    return input
}

func (r *Redactor) RedactRequest(req ObserveRequest) ObserveRequest {
    // Convert to JSON, strip, convert back
    raw, _ := json.Marshal(req)
    stripped := r.Strip(string(raw))
    json.Unmarshal([]byte(stripped), &req)
    return req
}
```

### 2.11. Config

```go
// apps/memory/configs/config.yaml — thêm section
observe:
  max_obs_per_session: 500
  dedup_ttl_seconds: 30
  index_save_interval: "30s"
  secret: ""            # HMAC auth secret (empty = no auth in dev)

observe_search:
  url: "http://localhost:8082"  # am-search HTTP URL
```

```
Environment variables:
  VNP_MEMORY_OBSERVE_MAX_OBS_PER_SESSION=500
  VNP_MEMORY_OBSERVE_DEDUP_TTL_SECONDS=30
  VNP_MEMORY_OBSERVE_SECRET=<hmac-secret>
```

---

## 3. Files thay đổi

### [NEW] Files

| File | Mô tả |
|------|-------|
| `services/observe-service/internal/domain/entity.go` | Session, RawObservation, CompressedObservation |
| `services/observe-service/internal/domain/value_object.go` | HookType (12), ObservationType (15) |
| `services/observe-service/internal/observe/pipeline.go` | 14-step pipeline |
| `services/observe-service/internal/observe/dedup.go` | DedupMap (SHA256, 30s TTL) |
| `services/observe-service/internal/observe/synthetic.go` | Zero-LLM compression |
| `services/observe-service/internal/observe/stream.go` | SSE StreamBroker |
| `services/observe-service/internal/usecase/observe.go` | ObserveUseCase |
| `services/observe-service/internal/usecase/session_start.go` | CreateSession |
| `services/observe-service/internal/usecase/session_end.go` | EndSession |
| `services/observe-service/internal/adapter/grpc/handler.go` | gRPC server |
| `services/observe-service/internal/adapter/repository/postgres/` | Session + Obs repos |
| `services/observe-service/internal/adapter/event/publisher.go` | NATS publisher |
| `pkg/privacy/redact.go` | Shared privacy redaction |
| `apps/memory/internal/bootstrap/observe.go` | Bootstrap integration |
| `db/migrations/0010_observe_service.up.sql` | DB schema |
| `api/proto/observe/v1/observe.proto` | gRPC contract |

### [MODIFY] Files

| File | Thay đổi |
|------|---------|
| `apps/memory/internal/bootstrap/bootstrap.go` | Gọi `InitObserveService()` |
| `gateway/internal/adapter/handler/router.go` | Thêm `/v1/observe*` + `/v1/sessions*` routes |
| `apps/memory/configs/config.yaml` | Thêm `observe:` section |
| `go.mod` | Không cần thêm dependencies mới (chỉ dùng stdlib + existing libs) |

---

## 4. Dependencies

| Dependency | Loại | Đã có? |
|------------|------|--------|
| PostgreSQL | Storage | ✅ Existing |
| NATS JetStream (embedded) | Events | ✅ Existing |
| `pkg/privacy` | NEW shared pkg | ❌ Cần tạo |
| `go-chi/chi` (HTTP router) | HTTP | ✅ Existing (gateway) |
| `net/http` SSE | SSE streaming | ✅ stdlib |
| `encoding/gob` | Index serialization | ✅ stdlib |

---

## 5. Test Strategy

```go
// services/observe-service/internal/observe/pipeline_test.go
func TestPipeline_Dedup(t *testing.T) {
    // Send same request twice in 30s window → second returns Deduplicated=true
}

func TestPipeline_PrivacyRedaction(t *testing.T) {
    // Request with sk-abc123456789 → stored observation has [REDACTED:openai_key]
}

func TestPipeline_SessionLimit(t *testing.T) {
    // 501st observation → ErrSessionLimitExceeded
}

func TestSSEBroadcast(t *testing.T) {
    // Subscribe to /stream → capture observation → SSE event received in <200ms
}
```

---

## 6. Acceptance Criteria Mapping

| AC từ CR-AM-001 | Covered by |
|-----------------|------------|
| POST /observe với hookType: post_tool_use → RawObs + CompressedObs | pipeline.go step 4, 9, 13 |
| 2 request giống nhau trong 30s → deduplicated: true | dedup.go |
| sk-abc123456789 → [REDACTED] trước khi lưu | privacy.Strip() step 3 |
| SSE stream /stream nhận raw_observation real-time | stream.go step 11 |
| Session observationCount tăng đúng | step 12 |
| POST /observe/session/end → status=completed + NATS event | session_end.go |
| GET /sessions/{id}/observations → title, facts, files, importance | compressed obs schema |
| 501 observations → bị từ chối | step 7 |
