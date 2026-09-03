# SOL-AM-001 — Solution: Observe Service (Hook Capture Pipeline)

| Field | Value |
|---|---|
| **Solution ID** | SOL-AM-001 |
| **CR** | CR-AM-001 |
| **TDD ref** | [12-agentmemory-services.md](../../../tdd/architecture/12-agentmemory-services.md) |
| **Status** | Open |
| **Priority** | 🔴 Critical |
| **Component** | `services/observe-service` |

---

## 1. Phân tích

observe-service đã implement 13-step pipeline. Cần thêm:
1. HLD canonical endpoint: `POST /v1/observe/hooks` (hiện tại là session-based)
2. Step 10: Embed → pgvector (pending per HLD)

### 1.1 `POST /v1/observe/hooks` endpoint [NEW] — HLD canonical

```go
// gateway/adapter/handler/agentmemory_handler.go [MODIFY]
// Simple hook endpoint: không cần pre-create session
func (h *AgentMemoryHandler) ObserveHook(w http.ResponseWriter, r *http.Request) {
    var req ObserveHookRequest
    json.NewDecoder(r.Body).Decode(&req)
    
    // Auto-create session if not exists
    if req.SessionID == "" {
        req.SessionID, _ = h.sessionUC.GetOrCreate(r.Context(), req.AgentID)
    }
    
    // Pipeline: 14 steps
    obs, err := h.pipeline.Process(r.Context(), &HookEvent{
        SessionID: req.SessionID,
        HookType:  req.HookType,
        Payload:   req.Payload,
        TenantID:  tenant.FromContext(r.Context()),
    })
    writeJSON(w, 201, obs)
}
```

### 1.2 Step 10 — Embed [NEW]

```go
// observe/pipeline/step_embed.go [NEW]
type EmbedStep struct { embedder port.EmbedderClient; vectorRepo port.VectorRepository }

func (s *EmbedStep) Process(ctx context.Context, obs *Observation) (*Observation, error) {
    embedding, err := s.embedder.Embed(ctx, obs.Content)
    if err != nil { return obs, nil } // non-fatal: skip if embedder unavailable
    s.vectorRepo.StoreObservation(ctx, obs.ID, obs.TenantID, embedding)
    return obs, nil
}
```

## 2. File Changes

| File | Action |
|---|---|
| `gateway/adapter/handler/agentmemory_handler.go` | MODIFY — add ObserveHook endpoint |
| `gateway/adapter/handler/router.go` | MODIFY — register POST /v1/observe/hooks |
| `services/observe-service/pipeline/step_embed.go` | NEW — embed step |

## 3. Acceptance Criteria

- [ ] `POST /v1/observe/hooks` works without pre-created session
- [ ] Session auto-created if sessionID empty
- [ ] Embed step stores observation vectors in pgvector
- [ ] All 14 pipeline steps execute in order

