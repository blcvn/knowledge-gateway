# Solution: SOL-CONSOLE-003 — Sessions Explorer Backend APIs

**CR:** CR-CONSOLE-003
**TDD refs:** `architecture/12-agentmemory-services.md §observe-service`
**Version:** v3/console

---

## 1. Architecture

All session data comes from `observe-service`. Gateway proxies to `observe-service` gRPC.

---

## 2. Implementation

```go
// gateway/adapter/handler/sessions_handler.go [NEW]
type SessionsHandler struct {
    registry port.GRPCRegistry
}

func (h *SessionsHandler) observeClient() observepb.ObserveServiceClient {
    conn, _ := h.registry.Get("observe-service")
    return observepb.NewObserveServiceClient(conn)
}

// GET /v1/console/sessions?status=live&limit=20&offset=0
func (h *SessionsHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
    tenantID := tenant.FromContext(r.Context())
    status := r.URL.Query().Get("status") // live|completed|failed
    limit, _  := strconv.Atoi(r.URL.Query().Get("limit"))
    offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
    if limit == 0 { limit = 20 }

    resp, err := h.observeClient().ListSessions(r.Context(), &observepb.ListSessionsRequest{
        TenantId: tenantID, Status: status,
        Limit: int32(limit), Offset: int32(offset),
    })
    if err != nil { writeError(w, 500, "list_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"sessions": resp.Sessions, "total": resp.Total})
}

// GET /v1/console/sessions/{id}
func (h *SessionsHandler) GetSession(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    tenantID  := tenant.FromContext(r.Context())
    resp, err := h.observeClient().GetSession(r.Context(), &observepb.GetSessionRequest{
        SessionId: sessionID, TenantId: tenantID,
    })
    if err != nil { writeError(w, 404, "not_found", ""); return }
    writeJSON(w, 200, resp)
}

// GET /v1/console/sessions/{id}/timeline
func (h *SessionsHandler) GetTimeline(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    tenantID  := tenant.FromContext(r.Context())
    resp, err := h.observeClient().GetObservations(r.Context(), &observepb.GetObservationsRequest{
        SessionId: sessionID, TenantId: tenantID, IncludePayloadSummary: true,
    })
    if err != nil { writeError(w, 500, "timeline_failed", err.Error()); return }
    writeJSON(w, 200, map[string]any{"session_id": sessionID, "events": resp.Observations})
}

// GET /v1/console/sessions/{id}/diff — memory before vs after session
func (h *SessionsHandler) GetMemoryDiff(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    tenantID  := tenant.FromContext(r.Context())
    // Call observe-service which tracks memory operations during session
    resp, err := h.observeClient().GetMemoryDiff(r.Context(), &observepb.MemoryDiffRequest{
        SessionId: sessionID, TenantId: tenantID,
    })
    if err != nil { writeError(w, 500, "diff_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}

// GET /v1/console/sessions/{id}/working-memory
func (h *SessionsHandler) GetWorkingMemory(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    tenantID  := tenant.FromContext(r.Context())
    resp, err := h.observeClient().GetWorkingMemory(r.Context(), &observepb.WorkingMemoryRequest{
        SessionId: sessionID, TenantId: tenantID,
    })
    if err != nil { writeError(w, 500, "working_memory_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}

// GET /v1/console/sessions/{id}/user-summary
func (h *SessionsHandler) GetUserSummary(w http.ResponseWriter, r *http.Request) {
    sessionID := chi.URLParam(r, "id")
    tenantID  := tenant.FromContext(r.Context())
    resp, err := h.observeClient().GetSessionSummary(r.Context(), &observepb.SessionSummaryRequest{
        SessionId: sessionID, TenantId: tenantID,
    })
    if err != nil { writeError(w, 500, "summary_failed", err.Error()); return }
    writeJSON(w, 200, resp)
}
```

---

## 3. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/sessions_handler.go` | **[NEW]** |
| `gateway/adapter/handler/router.go` | **[MODIFY]** `/v1/console/sessions/*` |
