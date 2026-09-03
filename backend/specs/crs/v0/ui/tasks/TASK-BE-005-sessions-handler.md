# TASK-BE-005 — Console Sessions Handler

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-005 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-004](../solutions/SOL-004-Sessions-Solution.md) + [SOL-007 §3](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🔴 P0 |
| **Depends On** | TASK-BE-004 |
| **Estimated** | 4h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/console_sessions_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

```go
package handler

type ConsoleSessionsHandler struct {
    db      *sql.DB
    zepThrd ZepThreadClient   // gRPC bufconn → zep-thread
    zepMem  ZepMemoryClient   // gRPC bufconn → zep-memory
    eventSvc VNPEventClient   // gRPC bufconn → vnp-event
    memoCtx MemobaseContextClient // gRPC bufconn → memobase-context
}

// GET /v1/console/sessions
// Query params: status, page, page_size, search
func (h *ConsoleSessionsHandler) List(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    q := r.URL.Query()
    status   := q.Get("status")
    page, _  := strconv.Atoi(q.Get("page"))
    pageSize := 20
    if p, _ := strconv.Atoi(q.Get("page_size")); p > 0 { pageSize = p }
    search   := q.Get("search")
    if page < 1 { page = 1 }
    offset   := (page - 1) * pageSize

    // Build query with optional filters
    where := "WHERE tenant_id = $1"
    args  := []any{tenantID}
    if status != "" { where += " AND status = $2"; args = append(args, status) }
    if search != "" { where += " AND tsv @@ to_tsquery($3)"; args = append(args, search) }
    args = append(args, pageSize, offset)

    var total int
    h.db.QueryRowContext(r.Context(), "SELECT COUNT(*) FROM sessions " + where[:len(where)], args[:len(args)-2]...).Scan(&total)

    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, user_id, agent_id, engine, status, started_at, ended_at, metadata
         FROM sessions ` + where + ` ORDER BY created_at DESC LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)),
        args...,
    )
    defer rows.Close()

    sessions := []map[string]any{}
    for rows.Next() {
        var s struct { ID, UserID, AgentID, Engine, Status string; StartedAt, EndedAt *time.Time; Meta json.RawMessage }
        rows.Scan(&s.ID, &s.UserID, &s.AgentID, &s.Engine, &s.Status, &s.StartedAt, &s.EndedAt, &s.Meta)
        sessions = append(sessions, map[string]any{
            "id": s.ID, "user_id": s.UserID, "agent_id": s.AgentID,
            "engine": s.Engine, "status": s.Status,
            "started_at": s.StartedAt, "ended_at": s.EndedAt,
        })
    }
    httputil.JSON(w, 200, map[string]any{"data": sessions, "total": total, "page": page, "has_more": page*pageSize < total})
}

// GET /v1/console/sessions/live
func (h *ConsoleSessionsHandler) ListLive(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    rows, _ := h.db.QueryContext(r.Context(),
        `SELECT id, user_id, agent_id, engine, started_at
         FROM sessions WHERE tenant_id = $1 AND status = 'active'
         ORDER BY started_at DESC`, tenantID)
    // ... scan and return
}

// GET /v1/console/sessions/{id}
// → Fetch messages từ PostgreSQL + Zep thread metadata
func (h *ConsoleSessionsHandler) Detail(w http.ResponseWriter, r *http.Request) { /* ... */ }

// GET /v1/console/sessions/{id}/working-memory
// → Gọi gRPC zep-memory.GetWorkingMemory(sessionID)
func (h *ConsoleSessionsHandler) WorkingMemory(w http.ResponseWriter, r *http.Request) { /* ... */ }

// GET /v1/console/sessions/{id}/timeline
// → Gọi gRPC vnp-event.GetSessionTimeline(sessionID, tenantID)
func (h *ConsoleSessionsHandler) Timeline(w http.ResponseWriter, r *http.Request) { /* ... */ }

// GET /v1/console/sessions/{id}/diff
// → Gọi gRPC: compare working memory before/after session
func (h *ConsoleSessionsHandler) Diff(w http.ResponseWriter, r *http.Request) { /* ... */ }

// GET /v1/console/sessions/{id}/user-summary
// → Gọi gRPC memobase-context.AssembleContext(session.UserID, tenantID)
func (h *ConsoleSessionsHandler) UserSummary(w http.ResponseWriter, r *http.Request) { /* ... */ }
```

### Routes registration

```go
mux.HandleFunc("GET /v1/console/sessions",                   authMiddleware(sess.List))
mux.HandleFunc("GET /v1/console/sessions/live",              authMiddleware(sess.ListLive))
mux.HandleFunc("GET /v1/console/sessions/{id}",              authMiddleware(sess.Detail))
mux.HandleFunc("GET /v1/console/sessions/{id}/working-memory", authMiddleware(sess.WorkingMemory))
mux.HandleFunc("GET /v1/console/sessions/{id}/timeline",     authMiddleware(sess.Timeline))
mux.HandleFunc("GET /v1/console/sessions/{id}/diff",         authMiddleware(sess.Diff))
mux.HandleFunc("GET /v1/console/sessions/{id}/user-summary", authMiddleware(sess.UserSummary))
```

---

## Verification

```bash
curl "http://localhost:8080/v1/console/sessions?status=active&page=1" \
  -H "Authorization: Bearer <token>" -H "x-tenant-id: <tid>"
# Expected: {"data":[...],"total":N,"page":1,"has_more":false}
```
