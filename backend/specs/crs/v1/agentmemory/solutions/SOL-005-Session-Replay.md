# Solution: SOL-005 — Session Replay & Real-Time Viewer

**CR ID:** CR-AM-005  
**Solution ID:** SOL-005  
**Priority:** Medium (Wave 3)  
**Architecture:** EXTEND `services/observe-service/` + EXTEND `apps/memory` UI

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md`:
- `apps/memory` là Memory Console UI (frontend).
- Gateway route `/v1/console/sessions/*` (FEAT-014) đã được define nhưng chưa implement đủ: chỉ có `list`, `live`, `get`, `timeline`, `diff`, `working-memory`, `user-summary`.
- `observe-service` (SOL-001) đã có `StreamBroker` — cần expose qua HTTP SSE.
- Gateway hiện dùng `net/http` — hỗ trợ SSE natively (text/event-stream).

---

## 2. Giải pháp

### 2.1. [EXTEND] `services/observe-service` — SSE Stream Endpoint

```go
// services/observe-service/internal/adapter/http/sse_handler.go

type SSEHandler struct {
    broker *observe.StreamBroker
}

func (h *SSEHandler) ServeSSE(w http.ResponseWriter, r *http.Request) {
    // Headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no")
    
    sessionFilter := r.URL.Query().Get("session_id")
    
    // Last-Event-ID support for reconnection
    lastEventID := r.Header.Get("Last-Event-ID")
    
    ch, cancel := h.broker.Subscribe(sessionFilter)
    defer cancel()
    
    // Flush initial snapshot if lastEventID provided (replay missed events)
    if lastEventID != "" {
        h.flushMissedEvents(w, lastEventID, sessionFilter)
    }
    
    flusher, ok := w.(http.Flusher)
    if !ok { http.Error(w, "streaming not supported", 500); return }
    
    // Heartbeat ticker: keep connection alive
    heartbeat := time.NewTicker(15 * time.Second)
    defer heartbeat.Stop()
    
    for {
        select {
        case event := <-ch:
            data, _ := json.Marshal(event)
            fmt.Fprintf(w, "id: %s\ndata: %s\n\n", event.ID, data)
            flusher.Flush()
        case <-heartbeat.C:
            fmt.Fprintf(w, ": heartbeat\n\n")
            flusher.Flush()
        case <-r.Context().Done():
            return
        }
    }
}
```

**Gateway SSE route:** SSE cần bypass gRPC proxy vì yêu cầu HTTP streaming trực tiếp:

```go
// gateway/internal/adapter/handler/router.go
// SSE endpoint: route thẳng đến SSEHandler (không qua gRPC proxy)
r.Get("/v1/stream", sseProxy.ServeHTTP)  // HTTP reverse proxy to observe-service SSE endpoint

// Replay + Import routes (thông qua gRPC proxy bình thường)
r.Get("/v1/sessions/{id}/replay",  h.ForwardTo("am-observe", "ObserveService/GetReplayData"))
r.Post("/v1/sessions/import",       h.ForwardTo("am-observe", "ObserveService/ImportTranscript"))
```

### 2.2. Replay Data Endpoint

```go
// services/observe-service/internal/usecase/get_replay.go

type ReplayData struct {
    Session     Session         `json:"session"`
    Events      []ReplayEvent   `json:"events"`
    TotalEvents int             `json:"total_events"`
    DurationMs  int64           `json:"duration_ms"`
}

type ReplayEvent struct {
    Timestamp   time.Time               `json:"timestamp"`
    SequenceIdx int                     `json:"seq_idx"`
    HookType    string                  `json:"hook_type"`
    Title       string                  `json:"title"`
    RawPayload  any                     `json:"raw_payload,omitempty"`
    Compressed  *CompressedObservation  `json:"compressed,omitempty"`
}

func (uc *GetReplayUseCase) Execute(ctx context.Context, sessionID string, includeRaw bool) (*ReplayData, error) {
    session, _ := uc.sessionRepo.Get(ctx, sessionID)
    rawObs, _ := uc.obsRepo.ListRaw(ctx, sessionID)
    compObs, _ := uc.obsRepo.ListCompressed(ctx, sessionID)
    
    // Map compressed by rawObsID for O(1) lookup
    compMap := map[string]*CompressedObservation{}
    for i := range compObs { compMap[compObs[i].SourceRawID] = &compObs[i] }
    
    events := make([]ReplayEvent, 0, len(rawObs))
    for i, raw := range rawObs {
        evt := ReplayEvent{
            Timestamp:   raw.Timestamp,
            SequenceIdx: i,
            HookType:    string(raw.HookType),
            Title:       raw.ToolName,
            Compressed:  compMap[raw.ID],
        }
        if includeRaw { evt.RawPayload = raw.Raw }
        if evt.Compressed != nil { evt.Title = evt.Compressed.Title }
        events = append(events, evt)
    }
    
    var durationMs int64
    if len(events) > 0 {
        durationMs = events[len(events)-1].Timestamp.Sub(events[0].Timestamp).Milliseconds()
    }
    
    return &ReplayData{Session: *session, Events: events, TotalEvents: len(events), DurationMs: durationMs}, nil
}
```

### 2.3. JSONL Transcript Import

```go
// services/observe-service/internal/usecase/import_transcript.go

type ImportTranscriptUseCase struct {
    sessionRepo port.ISessionRepo
    obsRepo     port.IObservationRepo
    pipeline    *observe.Pipeline
}

// Claude Code JSONL format:
// {"type": "user", "message": {...}, "timestamp": "2024-01-01T00:00:00Z"}
// {"type": "assistant", "message": {...}, "timestamp": "2024-01-01T00:00:05Z"}

func (uc *ImportTranscriptUseCase) Execute(ctx context.Context, req ImportRequest) (*ImportResponse, error) {
    // Create session
    session := Session{
        ID:        newID(),
        Project:   req.Project,
        Status:    "completed",
        StartedAt: time.Now(),
        TenantID:  req.TenantID,
    }
    if req.SessionName != "" { session.Project = req.SessionName }
    uc.sessionRepo.Save(ctx, session)
    
    // Parse JSONL
    lines := strings.Split(req.Transcript, "\n")
    count := 0
    for _, line := range lines {
        if line == "" { continue }
        
        var entry map[string]any
        if err := json.Unmarshal([]byte(line), &entry); err != nil { continue }
        
        entryType := entry["type"].(string)
        hookType := HookPromptSubmit
        if entryType == "assistant" { hookType = HookPostToolUse }
        
        raw := RawObservation{
            ID:        newID(),
            SessionID: session.ID,
            HookType:  hookType,
            TenantID:  req.TenantID,
        }
        if ts, ok := entry["timestamp"].(string); ok {
            raw.Timestamp, _ = time.Parse(time.RFC3339, ts)
        }
        if msg, ok := entry["message"].(map[string]any); ok {
            raw.AssistantResponse = fmt.Sprint(msg["content"])
        }
        uc.obsRepo.SaveRaw(ctx, raw)
        
        // Synthetic compress + index
        compressed := syntheticCompress(raw)
        uc.obsRepo.SaveCompressed(ctx, compressed)
        count++
    }
    
    uc.sessionRepo.UpdateObsCount(ctx, session.ID, count)
    return &ImportResponse{SessionID: session.ID, ObservationCount: count, Indexed: true}, nil
}
```

### 2.4. [NEW] `apps/memory` — Session Replay Tab

**Mô tả UI tab mới** (React/Next.js hoặc vanilla JS tùy tech stack của `apps/memory`):

```
Session Replay Tab Layout:
┌─────────────────────────────────────────────────────────┐
│  [Live Stream]  [Past Sessions ▼]  [Import Transcript]  │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  LIVE STREAM (khi không chọn session):                   │
│  ┌─────────────────────────────────────────┐            │
│  │ ● claude-code · write_file              │            │
│  │   Created /src/auth/middleware.go        │            │
│  │ ● claude-code · bash                    │            │
│  │   npm test -- --watch                   │            │
│  └─────────────────────────────────────────┘            │
│                                                          │
│  REPLAY (khi chọn session):                              │
│  Timeline:                                               │
│  ├─●──●──●──●──●──●──●──●──●──●──●──● (scrubber)        │
│  0s                                    4m 32s           │
│                                                          │
│  Controls: [⏮] [◀] [▶/⏸] [▶▶] [0.5× 1× 2× 4×]         │
│                                                          │
│  Event Detail:                                           │
│  {title, facts[], files[], raw payload}                  │
└─────────────────────────────────────────────────────────┘
```

**API calls từ UI:**
```javascript
// Live stream (SSE)
const evtSource = new EventSource('/v1/stream');
evtSource.onmessage = (e) => addEvent(JSON.parse(e.data));

// Load replay
const replay = await fetch(`/v1/sessions/${sessionId}/replay`).then(r => r.json());

// Import transcript
const formData = new FormData();
formData.append('transcript', file);
await fetch('/v1/sessions/import', { method: 'POST', body: formData });
```

**Replay player logic (JS):**
```javascript
class ReplayPlayer {
  constructor(events) { this.events = events; this.idx = 0; this.speed = 1.0; }
  
  play() {
    if (this.idx >= this.events.length) return;
    const cur = this.events[this.idx];
    const next = this.events[this.idx + 1];
    
    this.onEvent(cur);
    
    if (next) {
      const realDelay = next.timestamp - cur.timestamp; // ms
      const delay = realDelay / this.speed;
      this.timer = setTimeout(() => { this.idx++; this.play(); }, delay);
    }
  }
  
  setSpeed(s) { this.speed = s; }
  pause() { clearTimeout(this.timer); }
  stepForward() { this.idx++; this.onEvent(this.events[this.idx]); }
  stepBack() { if (this.idx > 0) { this.idx--; this.onEvent(this.events[this.idx]); } }
}
```

### 2.5. Console API Routes (FEAT-014 completion)

```go
// gateway/internal/adapter/handler/router.go — thêm vào FEAT-014 group

// SSE Live Stream (special HTTP proxy)
r.Get("/v1/console/sessions/live", sseProxy.ServeHTTP)  // Forward SSE from observe-service

// Replay routes
r.Get("/v1/console/sessions",           h.ForwardTo("am-observe", "ObserveService/ListSessions"))
r.Get("/v1/console/sessions/{id}",      h.ForwardTo("am-observe", "ObserveService/GetSession"))
r.Get("/v1/console/sessions/{id}/replay", h.ForwardTo("am-observe", "ObserveService/GetReplayData"))
r.Post("/v1/sessions/import",           h.ForwardTo("am-observe", "ObserveService/ImportTranscript"))
```

---

## 3. Files

### [EXTEND]

| File | Thay đổi |
|------|---------|
| `services/observe-service/internal/adapter/http/sse_handler.go` | SSE endpoint + heartbeat |
| `services/observe-service/internal/usecase/get_replay.go` | [NEW] Replay data builder |
| `services/observe-service/internal/usecase/import_transcript.go` | [NEW] JSONL importer |
| `services/observe-service/internal/domain/entity.go` | Thêm ReplayData, ReplayEvent types |
| `gateway/internal/adapter/handler/router.go` | SSE proxy route + replay routes |
| `apps/memory` | Thêm Session Replay tab (UI) |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-AM-005 | Covered by |
|-----------------|------------|
| GET /stream → text/event-stream + keep connection | sse_handler.go |
| Observation mới → SSE client nhận < 200ms | broker.Broadcast() non-blocking |
| GET /sessions/{id}/replay → chronological events[] | get_replay.go |
| events[].duration_ms = timestamp cuối - đầu | duration calc |
| Viewer UI replay at 2× speed | ReplayPlayer.setSpeed(2) |
| POST /sessions/import → session indexed BM25 | import → pipeline step 14 |
| Space/arrow keyboard shortcuts | JS keyboard event handlers |
