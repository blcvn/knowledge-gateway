# TASK-BE-007 — Console Adaptive Handler (Supermemory)

| Field | Value |
|---|---|
| **Task ID** | TASK-BE-007 |
| **Layer** | Backend — Go |
| **Status** | ✅ Done |
| **Solution Ref** | [SOL-006 CR-005](../solutions/SOL-006-Adaptive-to-Org-Solutions.md) + [SOL-007 §5.1](../solutions/SOL-007-Gap-Fixes.md) |
| **Priority** | 🟠 P1 |
| **Depends On** | — |
| **Estimated** | 3h |

---

## Target Files

| Action | File Path |
|---|---|
| CREATE | `gateway/internal/adapter/handler/console_adaptive_handler.go` |
| MODIFY | `gateway/internal/adapter/handler/router.go` |

---

## Implementation

```go
package handler

type ConsoleAdaptiveHandler struct {
    smMemory    SMMemoryClient
    smConnector SMConnectorClient
    smAnalytics SMAnalyticsClient
    nats        *nats.Conn
}

// GET /v1/console/adaptive/memories
func (h *ConsoleAdaptiveHandler) ListMemories(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    memories, err := h.smMemory.ListMemories(r.Context(), &sm.ListRequest{
        TenantID: tenantID, IsLatest: true,
    })
    if err != nil { httputil.Error(w, "Failed", "SM_ERROR", 500); return }
    httputil.JSON(w, 200, memories)
}

// GET /v1/console/adaptive/memories/{id}/versions
func (h *ConsoleAdaptiveHandler) GetVersions(w http.ResponseWriter, r *http.Request) {
    id := r.PathValue("id")
    versions, _ := h.smMemory.GetVersionChain(r.Context(), id, authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, versions)
}

// GET /v1/console/adaptive/connectors
func (h *ConsoleAdaptiveHandler) ListConnectors(w http.ResponseWriter, r *http.Request) {
    connectors, _ := h.smConnector.ListConnectors(r.Context(), authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, connectors)
}

// POST /v1/console/adaptive/connectors
func (h *ConsoleAdaptiveHandler) CreateConnector(w http.ResponseWriter, r *http.Request) {
    var cfg map[string]any
    json.NewDecoder(r.Body).Decode(&cfg)
    connector, _ := h.smConnector.CreateConnector(r.Context(), authctx.TenantID(r.Context()), cfg)
    httputil.JSON(w, 201, connector)
}

// POST /v1/console/adaptive/connectors/{id}/sync
func (h *ConsoleAdaptiveHandler) SyncConnector(w http.ResponseWriter, r *http.Request) {
    connID := r.PathValue("id")
    // Publish NATS event → sm-connector picks up and starts ingest job
    h.nats.Publish("sm.connector.sync."+connID, []byte(`{"tenant_id":"`+authctx.TenantID(r.Context())+`"}`))
    httputil.JSON(w, 200, map[string]string{"status": "sync_triggered"})
}

// GET /v1/console/adaptive/analytics
func (h *ConsoleAdaptiveHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
    tenantID := authctx.TenantID(r.Context())
    stats, _ := h.smAnalytics.GetStats(r.Context(), tenantID)
    // Map to AdaptiveAnalyticsResponse (all 5 fields: creation_rate, deletion_rate, contradiction_count, connector_sync_count, storage_usage_bytes)
    httputil.JSON(w, 200, map[string]any{
        "creation_rate":       stats.CreationRatePerHour,
        "deletion_rate":       stats.DeletionRatePerHour,
        "contradiction_count": stats.ContradictionCount,
        "connector_sync_count": stats.ConnectorSyncCount24h,
        "storage_usage_bytes": stats.StorageUsageBytes,
    })
}

// GET /v1/console/adaptive/forget-rules
func (h *ConsoleAdaptiveHandler) GetForgetRules(w http.ResponseWriter, r *http.Request) {
    rules, _ := h.smMemory.GetForgetRules(r.Context(), authctx.TenantID(r.Context()))
    httputil.JSON(w, 200, rules)
}

// PUT /v1/console/adaptive/forget-rules
func (h *ConsoleAdaptiveHandler) UpdateForgetRules(w http.ResponseWriter, r *http.Request) {
    var rules map[string]any
    json.NewDecoder(r.Body).Decode(&rules)
    updated, _ := h.smMemory.SetForgetRules(r.Context(), authctx.TenantID(r.Context()), rules)
    httputil.JSON(w, 200, updated)
}
```

### Routes

```go
mux.HandleFunc("GET /v1/console/adaptive/memories",                  authMiddleware(adp.ListMemories))
mux.HandleFunc("GET /v1/console/adaptive/memories/{id}/versions",    authMiddleware(adp.GetVersions))
mux.HandleFunc("GET /v1/console/adaptive/connectors",                authMiddleware(adp.ListConnectors))
mux.HandleFunc("POST /v1/console/adaptive/connectors",               authMiddleware(adp.CreateConnector))
mux.HandleFunc("POST /v1/console/adaptive/connectors/{id}/sync",     authMiddleware(adp.SyncConnector))
mux.HandleFunc("GET /v1/console/adaptive/analytics",                 authMiddleware(adp.GetAnalytics))
mux.HandleFunc("GET /v1/console/adaptive/forget-rules",              authMiddleware(adp.GetForgetRules))
mux.HandleFunc("PUT /v1/console/adaptive/forget-rules",              authMiddleware(adp.UpdateForgetRules))
```
