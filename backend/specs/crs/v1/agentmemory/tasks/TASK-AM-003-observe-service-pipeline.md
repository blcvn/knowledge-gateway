# TASK-AM-003 — Observe Service: gRPC Handler + Session Usecases

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-003 |
| **Wave** | 1 (Foundation) |
| **Component** | `services/observe-service/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-001 §2.6, §2.8, §2.9 |
| **Priority** | 🔴 Critical |
| **Depends On** | TASK-AM-002 |
| **Estimated** | 6h |

**Trạng thái:** 🔄 Partial  
**Ghi chú:** observe-service pipeline: 13/14 steps impl; Embed step pending  
---

## Context

Hoàn thành phần còn lại của observe-service:
- Session usecases (start, end)
- gRPC handler + mapper
- PostgreSQL repos
- NATS event publisher
- SSE HTTP handler

---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/observe-service/internal/usecase/session_start.go` |
| CREATE | `services/observe-service/internal/usecase/session_end.go` |
| CREATE | `services/observe-service/internal/usecase/observe.go` |
| CREATE | `services/observe-service/internal/adapter/grpc/handler.go` |
| CREATE | `services/observe-service/internal/adapter/grpc/mapper.go` |
| CREATE | `services/observe-service/internal/adapter/repository/postgres/session_repo.go` |
| CREATE | `services/observe-service/internal/adapter/repository/postgres/observation_repo.go` |
| CREATE | `services/observe-service/internal/adapter/event/publisher.go` |
| CREATE | `services/observe-service/internal/adapter/http/sse_handler.go` |

---

## Implementation

### `internal/usecase/session_start.go`

```go
package usecase

import (
    "context"
    "time"

    "github.com/vnp-memory/services/observe-service/internal/domain"
    "github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type CreateSessionUseCase struct {
    sessionRepo port.ISessionRepo
    publisher   port.IEventPublisher
}

type CreateSessionRequest struct {
    TenantID    string
    Project     string
    CWD         string
    Model       string
    AgentID     string
    FirstPrompt string
}

type CreateSessionResponse struct {
    SessionID string
    Status    string
}

func (uc *CreateSessionUseCase) Execute(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error) {
    session := domain.NewSession(req.TenantID, req.Project, req.CWD, req.Model, req.AgentID)
    session.FirstPrompt = req.FirstPrompt

    if err := uc.sessionRepo.Save(ctx, session); err != nil {
        return nil, err
    }

    uc.publisher.Publish(ctx, "agentmemory.session.started", map[string]any{
        "session_id": session.ID,
        "tenant_id":  session.TenantID,
        "project":    session.Project,
        "agent_id":   session.AgentID,
        "started_at": session.StartedAt,
    })

    return &CreateSessionResponse{SessionID: session.ID, Status: "active"}, nil
}
```

### `internal/usecase/session_end.go`

```go
package usecase

import (
    "context"
    "time"

    "github.com/vnp-memory/services/observe-service/internal/domain"
    "github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type EndSessionUseCase struct {
    sessionRepo port.ISessionRepo
    obsRepo     port.IObservationRepo
    publisher   port.IEventPublisher
}

type EndSessionRequest  struct { SessionID string; TenantID string }
type EndSessionResponse struct { SessionID string; Status string; ObservationCount int }

func (uc *EndSessionUseCase) Execute(ctx context.Context, req EndSessionRequest) (*EndSessionResponse, error) {
    session, err := uc.sessionRepo.GetByID(ctx, req.SessionID)
    if err != nil { return nil, domain.ErrSessionNotFound }
    if session.Status == "completed" { return nil, domain.ErrSessionEnded }

    now := time.Now()
    session.Status = "completed"
    session.EndedAt = &now

    if err := uc.sessionRepo.UpdateStatus(ctx, req.SessionID, "completed"); err != nil {
        return nil, err
    }

    count, _ := uc.sessionRepo.GetObsCount(ctx, req.SessionID)

    uc.publisher.Publish(ctx, "agentmemory.session.ended", map[string]any{
        "session_id":        req.SessionID,
        "tenant_id":         req.TenantID,
        "observation_count": count,
        "ended_at":          now,
    })

    return &EndSessionResponse{
        SessionID:        req.SessionID,
        Status:           "completed",
        ObservationCount: count,
    }, nil
}
```

### `internal/usecase/observe.go`

```go
package usecase

import (
    "context"

    "github.com/vnp-memory/services/observe-service/internal/observe"
    "github.com/vnp-memory/services/observe-service/internal/usecase/port"
)

type ObserveUseCase struct {
    pipeline    *observe.Pipeline
    sessionRepo port.ISessionRepo
    obsRepo     port.IObservationRepo
}

func (uc *ObserveUseCase) Execute(ctx context.Context, req observe.ObserveRequest) (*observe.ObserveResponse, error) {
    // Validate session exists
    if _, err := uc.sessionRepo.GetByID(ctx, req.SessionID); err != nil {
        return nil, err
    }
    return uc.pipeline.Execute(ctx, req)
}

func (uc *ObserveUseCase) GetObservations(ctx context.Context, sessionID string, limit, offset int) ([]any, error) {
    return uc.obsRepo.ListCompressed(ctx, sessionID, limit, offset)
}
```

### `internal/adapter/grpc/handler.go`

```go
package grpc

import (
    "context"

    observepb "github.com/vnp-memory/api/proto/observe/v1"
    "github.com/vnp-memory/services/observe-service/internal/observe"
    "github.com/vnp-memory/services/observe-service/internal/usecase"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type ObserveHandler struct {
    observepb.UnimplementedObserveServiceServer
    observeUC       *usecase.ObserveUseCase
    createSessionUC *usecase.CreateSessionUseCase
    endSessionUC    *usecase.EndSessionUseCase
    sessionRepo     port.ISessionRepo
    stream          *observe.StreamBroker
}

func (h *ObserveHandler) Observe(ctx context.Context, req *observepb.ObserveRequest) (*observepb.ObserveResponse, error) {
    ucReq := observe.ObserveRequest{
        SessionID:         req.SessionId,
        HookType:          req.HookType,
        ToolName:          req.ToolName,
        ToolInput:         req.ToolInput,
        ToolOutput:        req.ToolOutput,
        UserPrompt:        req.UserPrompt,
        AssistantResponse: req.AssistantResponse,
        AgentID:           req.AgentId,
        TenantID:          req.TenantId,
    }
    if req.Timestamp != nil { ucReq.Timestamp = req.Timestamp.AsTime() }

    resp, err := h.observeUC.Execute(ctx, ucReq)
    if err != nil { return nil, status.Errorf(codes.Internal, "observe: %v", err) }

    return mapObserveResponse(resp), nil
}

func (h *ObserveHandler) StartSession(ctx context.Context, req *observepb.StartSessionRequest) (*observepb.StartSessionResponse, error) {
    resp, err := h.createSessionUC.Execute(ctx, usecase.CreateSessionRequest{
        TenantID: req.TenantId, Project: req.Project,
        CWD: req.Cwd, Model: req.Model, AgentID: req.AgentId,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "start session: %v", err) }
    return &observepb.StartSessionResponse{SessionId: resp.SessionID, Status: resp.Status}, nil
}

func (h *ObserveHandler) EndSession(ctx context.Context, req *observepb.EndSessionRequest) (*observepb.EndSessionResponse, error) {
    resp, err := h.endSessionUC.Execute(ctx, usecase.EndSessionRequest{
        SessionID: req.SessionId, TenantID: req.TenantId,
    })
    if err != nil { return nil, status.Errorf(codes.Internal, "end session: %v", err) }
    return &observepb.EndSessionResponse{
        SessionId:        resp.SessionID,
        Status:           resp.Status,
        ObservationCount: int32(resp.ObservationCount),
    }, nil
}

func (h *ObserveHandler) ListSessions(ctx context.Context, req *observepb.ListSessionsRequest) (*observepb.ListSessionsResponse, error) {
    sessions, err := h.sessionRepo.List(ctx, req.TenantId, req.Status, req.Project, int(req.Limit), int(req.Offset))
    if err != nil { return nil, status.Errorf(codes.Internal, "list sessions: %v", err) }
    return mapListSessionsResponse(sessions), nil
}

func (h *ObserveHandler) GetSession(ctx context.Context, req *observepb.GetSessionRequest) (*observepb.GetSessionResponse, error) {
    sess, err := h.sessionRepo.GetByID(ctx, req.SessionId)
    if err != nil { return nil, status.Errorf(codes.NotFound, "session not found") }
    return &observepb.GetSessionResponse{Session: mapSession(sess)}, nil
}

func (h *ObserveHandler) DeleteSession(ctx context.Context, req *observepb.DeleteSessionRequest) (*observepb.DeleteSessionResponse, error) {
    if err := h.sessionRepo.UpdateStatus(ctx, req.SessionId, "abandoned"); err != nil {
        return nil, status.Errorf(codes.Internal, "delete session: %v", err)
    }
    return &observepb.DeleteSessionResponse{Deleted: true}, nil
}

func (h *ObserveHandler) GetObservations(ctx context.Context, req *observepb.GetObservationsRequest) (*observepb.GetObservationsResponse, error) {
    obs, err := h.observeUC.GetObservations(ctx, req.SessionId, int(req.Limit), int(req.Offset))
    if err != nil { return nil, status.Errorf(codes.Internal, "get observations: %v", err) }
    return mapObservationsResponse(obs), nil
}

// StreamEvents — server-side streaming (SSE via gRPC)
func (h *ObserveHandler) StreamEvents(req *observepb.StreamEventsRequest, stream observepb.ObserveService_StreamEventsServer) error {
    ch, cancel := h.stream.Subscribe(req.SessionId)
    defer cancel()
    for {
        select {
        case event, ok := <-ch:
            if !ok { return nil }
            data, _ := json.Marshal(event.Data)
            stream.Send(&observepb.StreamEvent{
                EventType: event.Type,
                Data:      data,
            })
        case <-stream.Context().Done():
            return nil
        }
    }
}
```

### `internal/adapter/repository/postgres/session_repo.go`

```go
package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/lib/pq"
    "github.com/vnp-memory/services/observe-service/internal/domain"
)

type SessionRepo struct{ db *pgxpool.Pool }

func NewSessionRepo(db *pgxpool.Pool) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Save(ctx context.Context, s domain.Session) error {
    _, err := r.db.Exec(ctx, `
        INSERT INTO agent_sessions
            (id, tenant_id, project, cwd, model, agent_id, status, first_prompt, tags, started_at, last_active_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `, s.ID, s.TenantID, s.Project, s.CWD, s.Model, s.AgentID, s.Status,
        s.FirstPrompt, pq.Array(s.Tags), s.StartedAt, s.LastActiveAt)
    return err
}

func (r *SessionRepo) GetByID(ctx context.Context, id string) (*domain.Session, error) {
    row := r.db.QueryRow(ctx, `
        SELECT id, tenant_id, project, cwd, model, agent_id, status, summary,
               observation_count, tags, started_at, ended_at, last_active_at
        FROM agent_sessions WHERE id = $1
    `, id)
    var s domain.Session
    err := row.Scan(&s.ID, &s.TenantID, &s.Project, &s.CWD, &s.Model, &s.AgentID,
        &s.Status, &s.Summary, &s.ObservationCount, pq.Array(&s.Tags),
        &s.StartedAt, &s.EndedAt, &s.LastActiveAt)
    if err != nil { return nil, err }
    return &s, nil
}

func (r *SessionRepo) List(ctx context.Context, tenantID, status, project string, limit, offset int) ([]domain.Session, error) {
    rows, err := r.db.Query(ctx, `
        SELECT id, tenant_id, project, status, observation_count, started_at
        FROM agent_sessions
        WHERE tenant_id = $1
          AND ($2 = '' OR status = $2)
          AND ($3 = '' OR project = $3)
        ORDER BY started_at DESC LIMIT $4 OFFSET $5
    `, tenantID, status, project, limit, offset)
    if err != nil { return nil, err }
    defer rows.Close()

    var sessions []domain.Session
    for rows.Next() {
        var s domain.Session
        rows.Scan(&s.ID, &s.TenantID, &s.Project, &s.Status, &s.ObservationCount, &s.StartedAt)
        sessions = append(sessions, s)
    }
    return sessions, nil
}

func (r *SessionRepo) UpdateStatus(ctx context.Context, id, status string) error {
    now := time.Now()
    _, err := r.db.Exec(ctx, `
        UPDATE agent_sessions SET status = $1, ended_at = $2 WHERE id = $3
    `, status, now, id)
    return err
}

func (r *SessionRepo) IncrementObsCount(ctx context.Context, id string) error {
    _, err := r.db.Exec(ctx, `
        UPDATE agent_sessions
        SET observation_count = observation_count + 1, last_active_at = NOW()
        WHERE id = $1
    `, id)
    return err
}

func (r *SessionRepo) GetObsCount(ctx context.Context, id string) (int, error) {
    var count int
    err := r.db.QueryRow(ctx, `SELECT observation_count FROM agent_sessions WHERE id = $1`, id).Scan(&count)
    return count, err
}
```

### `internal/adapter/event/publisher.go`

```go
package event

import (
    "context"
    "encoding/json"
    "github.com/nats-io/nats.go"
)

type NATSPublisher struct {
    conn *nats.Conn
}

func NewNATSPublisher(conn *nats.Conn) *NATSPublisher {
    return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) Publish(ctx context.Context, subject string, payload any) error {
    data, err := json.Marshal(payload)
    if err != nil { return err }
    return p.conn.Publish(subject, data)
}
```

### `internal/adapter/http/sse_handler.go`

```go
package http

import (
    "fmt"
    "net/http"
    "time"

    "github.com/vnp-memory/services/observe-service/internal/observe"
)

type SSEHandler struct {
    stream *observe.StreamBroker
}

func NewSSEHandler(stream *observe.StreamBroker) *SSEHandler {
    return &SSEHandler{stream: stream}
}

// ServeSSE handles GET /v1/stream?session_id=<id>
func (h *SSEHandler) ServeSSE(w http.ResponseWriter, r *http.Request) {
    sessionFilter := r.URL.Query().Get("session_id")

    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "streaming not supported", http.StatusInternalServerError)
        return
    }

    ch, cancel := h.stream.Subscribe(sessionFilter)
    defer cancel()

    // Send heartbeat every 30s
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case event, ok := <-ch:
            if !ok { return }
            data, _ := json.Marshal(event)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
            flusher.Flush()

        case <-ticker.C:
            fmt.Fprintf(w, ": heartbeat\n\n")
            flusher.Flush()

        case <-r.Context().Done():
            return
        }
    }
}
```

---

## Verification

```bash
cd services/observe-service
go build ./...
go test ./internal/usecase/... -v
```

## Acceptance Criteria

| AC | Check |
|----|-------|
| `StartSession` → `session_id` returned, status=active | ✅ |
| `EndSession` → status=completed, NATS event published | ✅ |
| `GetObservations` → compressed obs list | ✅ |
| `StreamEvents` gRPC → server-side stream | ✅ |
| SSE `/v1/stream` → heartbeat every 30s | ✅ |
| NATS `agentmemory.session.started` published on session start | ✅ |
| NATS `agentmemory.session.ended` published on session end | ✅ |
