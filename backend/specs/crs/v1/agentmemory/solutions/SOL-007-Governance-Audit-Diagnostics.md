# Solution: SOL-007 — Governance, Audit & Diagnostics

**CR ID:** CR-AM-007  
**Solution ID:** SOL-007  
**Priority:** Medium (Wave 4)  
**Architecture:** EXTEND `services/memory-service/` + EXTEND `services/vnp-platform/`

---

## 1. Phân tích kiến trúc hiện tại

Từ `specs/architecture.md §5.1`:
- `services/vnp-platform/` có `admin/` domain: Tenant, User, APIKey, HealthStatus.
- `GET /v1/health` hiện tại chỉ trả về `{status: "ok"}` — thiếu chi tiết.
- `services/obs-service/` (observability) đã có domain nhưng chưa implement đầy đủ.
- Privacy redaction đã được implement trong `pkg/privacy/redact.go` (CR-AM-001/SOL-001).

**Chiến lược:**
1. **Cascade Delete + Audit Trail** → Thêm vào `services/memory-service/` (vì đây là data governance cho memories).
2. **Health Monitor + Doctor** → Thêm vào `services/vnp-platform/` admin domain.
3. **Git Snapshots** → `services/vnp-platform/` admin feature.
4. **Privacy Redaction** → Đã done trong SOL-001, chỉ cần ensure `pkg/privacy/` được dùng.
5. **Circuit Breaker** → Đã done trong SOL-006 (`consolidation/circuit_breaker.go`), promote lên `pkg/resilience/`.

---

## 2. Giải pháp

### 2.1. [EXTEND] Cascade Delete + Audit Trail — `services/memory-service/`

```go
// services/memory-service/internal/domain/agentmemory/audit.go

type AuditEntry struct {
    ID           string         `json:"id" db:"id"`
    Timestamp    time.Time      `json:"timestamp" db:"timestamp"`
    Operation    string         `json:"operation" db:"operation"`
    TargetIDs    []string       `json:"target_ids" db:"target_ids"`
    PerformedBy  string         `json:"performed_by" db:"performed_by"`
    Project      string         `json:"project" db:"project"`
    TenantID     string         `json:"tenant_id" db:"tenant_id"`
    Details      map[string]any `json:"details,omitempty" db:"details"`
    Reason       string         `json:"reason,omitempty" db:"reason"`
}

// Operation types (40+)
const (
    AuditObserve           = "observe"
    AuditRemember          = "remember"
    AuditSupersede         = "supersede"
    AuditForget            = "forget"
    AuditGovernanceDelete  = "governance_delete"
    AuditEvict             = "evict"
    AuditAutoForget        = "auto_forget"
    AuditCompress          = "compress"
    AuditSummarize         = "summarize"
    AuditConsolidate       = "consolidate"
    AuditSlotWrite         = "slot_write"
    AuditSlotDelete        = "slot_delete"
    AuditSessionStart      = "session_start"
    AuditSessionEnd        = "session_end"
    AuditSessionDelete     = "session_delete"
    AuditImportTranscript  = "import_transcript"
    AuditSearchQuery       = "search_query"
    AuditContextBuild      = "context_build"
    AuditSignalSend        = "signal_send"
    AuditLeaseAcquire      = "lease_acquire"
    AuditLeaseRelease      = "lease_release"
    AuditCheckpointCreate  = "checkpoint_create"
    AuditCheckpointResolve = "checkpoint_resolve"
    AuditSnapshotCreate    = "snapshot_create"
    AuditIndexRebuild      = "index_rebuild"
    AuditDecaySweep        = "decay_sweep"
    AuditEvictionRun       = "eviction_run"
    AuditLessonDecay       = "lesson_decay"
    // ... total 40+ types
)
```

```go
// services/memory-service/internal/usecase/agentmemory/governance_delete.go

type GovernanceDeleteUseCase struct {
    memRepo      port.IAgentMemoryRepo
    searchClient port.ISearchNotifier   // HTTP to am-search
    graphClient  port.IGraphClient      // HTTP to cognee/graphiti
    auditRepo    port.IAuditRepo
    publisher    port.IEventPublisher
}

func (uc *GovernanceDeleteUseCase) Execute(ctx context.Context, req GovernanceDeleteRequest) error {
    mem, err := uc.memRepo.Get(ctx, req.MemoryID)
    if err != nil { return err }
    
    // 1. Delete from PostgreSQL (cascade: related observations via FK)
    if err := uc.memRepo.Delete(ctx, req.MemoryID); err != nil { return err }
    
    // 2. Remove from BM25 index (HTTP call to am-search)
    if err := uc.searchClient.RemoveMemory(ctx, req.MemoryID); err != nil {
        log.Warn("BM25 index removal failed", "memID", req.MemoryID, "err", err)
        // Non-fatal: log but continue cascade
    }
    
    // 3. Remove from Vector index (HTTP call to am-search)
    if err := uc.searchClient.RemoveVector(ctx, req.MemoryID); err != nil {
        log.Warn("Vector index removal failed", "memID", req.MemoryID, "err", err)
    }
    
    // 4. Remove graph edges (HTTP call to graph service — cognee/graphiti)
    if err := uc.graphClient.RemoveBySourceID(ctx, req.MemoryID); err != nil {
        log.Warn("Graph edge removal failed", "memID", req.MemoryID, "err", err)
    }
    
    // 5. Create audit entry
    uc.auditRepo.Save(ctx, AuditEntry{
        ID:          newID(),
        Timestamp:   time.Now(),
        Operation:   AuditGovernanceDelete,
        TargetIDs:   []string{req.MemoryID},
        PerformedBy: req.PerformedBy,
        Project:     mem.Project,
        TenantID:    req.TenantID,
        Reason:      req.Reason,
        Details:     map[string]any{"memory_type": string(mem.Type), "title": mem.Title},
    })
    
    return nil
}
```

### 2.2. Audit Repository

```go
// services/memory-service/internal/adapter/repository/postgres/audit_repo.go

type AuditRepo struct{ db *sql.DB }

func (r *AuditRepo) Save(ctx context.Context, entry AuditEntry) error {
    // INSERT INTO audit_entries
}

func (r *AuditRepo) List(ctx context.Context, filter AuditFilter) ([]AuditEntry, error) {
    // SELECT with optional WHERE operation, tenant_id, project, date range
}

type AuditFilter struct {
    TenantID  string
    Operation string
    Project   string
    FromTime  *time.Time
    ToTime    *time.Time
    Limit     int
    Offset    int
}
```

### 2.3. [EXTEND] Health Monitor — `services/vnp-platform/`

```go
// services/vnp-platform/internal/domain/admin/health_snapshot.go

type HealthSnapshot struct {
    Status        string          `json:"status"`   // "healthy" | "degraded" | "critical"
    Timestamp     time.Time       `json:"timestamp"`
    UptimeSeconds float64         `json:"uptime_seconds"`
    Workers       []WorkerStatus  `json:"workers"`
    Memory        MemoryUsage     `json:"memory"`
    CPU           CPUUsage        `json:"cpu"`
    Indexes       IndexHealth     `json:"indexes"`
    Connections   ConnHealth      `json:"connections"`
    Alerts        []string        `json:"alerts,omitempty"`
}

type WorkerStatus struct {
    Name     string    `json:"name"`
    Status   string    `json:"status"`  // "running" | "stopped" | "error"
    LastRun  time.Time `json:"last_run,omitempty"`
    Error    string    `json:"error,omitempty"`
}

type IndexHealth struct {
    BM25Documents    int    `json:"bm25_documents"`
    VectorDocuments  int    `json:"vector_documents"`
    LastPersisted    time.Time `json:"last_persisted"`
    Status           string `json:"status"`
}

type ConnHealth struct {
    PostgreSQL  string `json:"postgresql"`   // "ok" | "error"
    NATS        string `json:"nats"`
    Bifrost     string `json:"bifrost"`
    ObserveSearch string `json:"observe_search"`
}
```

```go
// services/vnp-platform/internal/usecase/admin/health_monitor.go

type HealthMonitorUseCase struct {
    db          *sql.DB
    nats        *nats.Conn
    bifrostURL  string
    searchURL   string
    startTime   time.Time
    workers     map[string]*WorkerTracker  // track worker goroutines
}

func (uc *HealthMonitorUseCase) GetSnapshot(ctx context.Context) *HealthSnapshot {
    snap := &HealthSnapshot{
        Timestamp:     time.Now(),
        UptimeSeconds: time.Since(uc.startTime).Seconds(),
    }
    
    // Check connections
    snap.Connections = uc.checkConnections(ctx)
    
    // Check workers
    snap.Workers = uc.checkWorkers()
    
    // Check indexes (call am-search service)
    snap.Indexes = uc.checkIndexes(ctx)
    
    // Resource usage
    snap.Memory = uc.getMemoryUsage()
    snap.CPU = uc.getCPUUsage()
    
    // Determine overall status
    snap.Status = uc.determineStatus(snap)
    
    // Build alerts
    snap.Alerts = uc.buildAlerts(snap)
    
    return snap
}

func (uc *HealthMonitorUseCase) checkConnections(ctx context.Context) ConnHealth {
    conn := ConnHealth{}
    
    // PostgreSQL: ping
    if err := uc.db.PingContext(ctx); err != nil {
        conn.PostgreSQL = "error: " + err.Error()
    } else {
        conn.PostgreSQL = "ok"
    }
    
    // NATS: check status
    if uc.nats.Status() != nats.CONNECTED {
        conn.NATS = "error: not connected"
    } else {
        conn.NATS = "ok"
    }
    
    // Bifrost: HTTP health check
    if err := uc.httpCheck(ctx, uc.bifrostURL+"/health"); err != nil {
        conn.Bifrost = "error: " + err.Error()
    } else {
        conn.Bifrost = "ok"
    }
    
    return conn
}

func (uc *HealthMonitorUseCase) determineStatus(snap *HealthSnapshot) string {
    if snap.Connections.PostgreSQL != "ok" { return "critical" }
    for _, w := range snap.Workers {
        if w.Status == "error" { return "degraded" }
    }
    return "healthy"
}
```

### 2.4. Doctor Command

```go
// services/vnp-platform/internal/usecase/admin/doctor.go

type DiagnosticCheck struct {
    Check      string `json:"check"`
    Status     string `json:"status"`   // "ok" | "warning" | "error"
    Message    string `json:"message"`
    Suggestion string `json:"suggestion,omitempty"`
}

type DoctorUseCase struct {
    db         *sql.DB
    nats       *nats.Conn
    serviceURLs map[string]string
}

func (d *DoctorUseCase) Run(ctx context.Context) []DiagnosticCheck {
    checks := []DiagnosticCheck{}
    
    // Database connectivity
    if err := d.db.PingContext(ctx); err != nil {
        checks = append(checks, DiagnosticCheck{
            Check: "database.postgresql", Status: "error",
            Message: err.Error(),
            Suggestion: "Check VNP_MEMORY_POSTGRES_DSN and ensure PostgreSQL is running",
        })
    } else {
        checks = append(checks, DiagnosticCheck{Check: "database.postgresql", Status: "ok", Message: "connected"})
    }
    
    // Service connectivity checks
    for name, url := range d.serviceURLs {
        if err := d.httpCheck(ctx, url+"/health"); err != nil {
            checks = append(checks, DiagnosticCheck{
                Check: "service." + name, Status: "error", Message: err.Error(),
                Suggestion: fmt.Sprintf("Start %s service or check its URL in config", name),
            })
        } else {
            checks = append(checks, DiagnosticCheck{Check: "service." + name, Status: "ok"})
        }
    }
    
    // Data directory write access
    dir := os.Getenv("AGENTMEMORY_DATA_DIR")
    if dir == "" { dir = os.ExpandEnv("$HOME/.agentmemory") }
    if err := testWriteAccess(dir); err != nil {
        checks = append(checks, DiagnosticCheck{
            Check: "storage.data_dir", Status: "error",
            Message: "cannot write to " + dir,
            Suggestion: "Check directory permissions: chmod 755 " + dir,
        })
    } else {
        checks = append(checks, DiagnosticCheck{Check: "storage.data_dir", Status: "ok", Message: dir})
    }
    
    // NATS connectivity
    if d.nats.Status() != nats.CONNECTED {
        checks = append(checks, DiagnosticCheck{
            Check: "nats", Status: "error", Message: "not connected",
            Suggestion: "Set VNP_MEMORY_NATS_MODE=embedded or check external NATS config",
        })
    }
    
    return checks
}
```

### 2.5. Git Snapshots

```go
// services/vnp-platform/internal/usecase/admin/snapshot.go

type SnapshotUseCase struct {
    db      *sql.DB
    dataDir string
    enabled bool
}

type SnapshotMeta struct {
    CommitHash string        `json:"commit_hash"`
    Stats      SnapshotStats `json:"stats"`
    CreatedAt  time.Time     `json:"created_at"`
}

type SnapshotStats struct {
    Sessions     int `json:"sessions"`
    Observations int `json:"observations"`
    Memories     int `json:"memories"`
    GraphNodes   int `json:"graph_nodes"`
}

func (uc *SnapshotUseCase) Create(ctx context.Context) (*SnapshotMeta, error) {
    if !uc.enabled { return nil, ErrSnapshotDisabled }
    
    // 1. Collect stats
    stats := uc.collectStats(ctx)
    
    // 2. Export data to snapshot directory
    snapshotDir := filepath.Join(uc.dataDir, "snapshots", time.Now().Format("20060102T150405"))
    uc.exportData(ctx, snapshotDir, stats)
    
    // 3. Git commit
    commitMsg := fmt.Sprintf("snapshot: %d sessions, %d memories (%s)",
        stats.Sessions, stats.Memories, time.Now().Format(time.RFC3339))
    
    cmds := [][]string{
        {"git", "-C", uc.dataDir, "add", "."},
        {"git", "-C", uc.dataDir, "commit", "-m", commitMsg},
    }
    var commitHash string
    for _, cmd := range cmds {
        out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).Output()
        if err != nil { return nil, fmt.Errorf("git error: %w", err) }
        if strings.HasPrefix(string(cmd[len(cmd)-1]), "-m") {
            commitHash = extractCommitHash(out)
        }
    }
    
    return &SnapshotMeta{CommitHash: commitHash, Stats: stats, CreatedAt: time.Now()}, nil
}
```

### 2.6. `pkg/resilience/` — Promote Circuit Breaker to Shared Package

```go
// pkg/resilience/circuit_breaker.go — NEW shared package

// Promote the circuit breaker from consolidation to a reusable pkg
// Used by: consolidation pipeline (LLM), observe-search (embedding provider)

type CircuitBreaker struct { ... } // same as SOL-006 implementation
```

### 2.7. PostgreSQL Schema

```sql
-- Migration: 0014_audit_governance.up.sql

CREATE TABLE audit_entries (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   TEXT NOT NULL,
    timestamp   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    operation   TEXT NOT NULL,
    target_ids  TEXT[] DEFAULT '{}',
    performed_by TEXT,
    project     TEXT,
    details     JSONB,
    reason      TEXT
);

CREATE INDEX idx_audit_entries_tenant_op ON audit_entries(tenant_id, operation, timestamp DESC);
CREATE INDEX idx_audit_entries_timestamp ON audit_entries(timestamp DESC);
```

### 2.8. Gateway Routes

```go
// Governance
r.Delete("/v1/memory/agent/{id}/governance",  h.ForwardTo("memory-service", "GovernanceService/Delete"))
r.Get("/v1/memory/audit",                      h.ForwardTo("memory-service", "GovernanceService/ListAudit"))

// Health + Doctor
r.Get("/v1/health",           h.ForwardTo("vnp-platform", "AdminService/GetHealthSnapshot"))
r.Get("/v1/admin/doctor",     h.ForwardTo("vnp-platform", "AdminService/Doctor"))
r.Post("/v1/admin/snapshot",  h.ForwardTo("vnp-platform", "AdminService/CreateSnapshot"))
r.Get("/v1/admin/snapshots",  h.ForwardTo("vnp-platform", "AdminService/ListSnapshots"))

// Plugin configs (CR-AM-008 prep)
r.Get("/v1/admin/plugin/claude-code", h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Get("/v1/admin/plugin/codex",       h.ForwardTo("vnp-platform", "AdminService/GetPluginConfig"))
r.Post("/v1/admin/plugin/install",    h.ForwardTo("vnp-platform", "AdminService/InstallPlugin"))
```

---

## 3. Files

### [NEW]

| File | Mô tả |
|------|-------|
| `services/memory-service/internal/domain/agentmemory/audit.go` | AuditEntry + operation consts |
| `services/memory-service/internal/usecase/agentmemory/governance_delete.go` | Cascade delete |
| `services/memory-service/internal/adapter/repository/postgres/audit_repo.go` | Audit repo |
| `services/vnp-platform/internal/usecase/admin/health_monitor.go` | HealthSnapshot builder |
| `services/vnp-platform/internal/usecase/admin/doctor.go` | Diagnostic checks |
| `services/vnp-platform/internal/usecase/admin/snapshot.go` | Git snapshot |
| `pkg/resilience/circuit_breaker.go` | Shared circuit breaker |
| `db/migrations/0014_audit_governance.up.sql` | audit_entries table |

### [MODIFY]

| File | Thay đổi |
|------|---------|
| `gateway/internal/adapter/handler/router.go` | Governance + health + doctor + snapshot routes |
| `services/memory-service/internal/usecase/agentmemory/remember_agent.go` | Add audit on remember |
| `services/memory-service/internal/usecase/agentmemory/evict.go` | Add audit on evict |
| `services/memory-service/internal/usecase/agentmemory/auto_forget.go` | Add audit on TTL expire |
| `services/vnp-platform/internal/domain/admin/entity.go` | Add HealthSnapshot types |
| `services/vnp-platform/internal/adapter/grpc/admin_handler.go` | Add health/doctor/snapshot RPCs |

---

## 4. Acceptance Criteria Mapping

| AC từ CR-AM-007 | Covered by |
|-----------------|------------|
| DELETE governance → xóa khỏi PostgreSQL + BM25 + vector | governance_delete.go |
| Governance delete → AuditEntry created | auditRepo.Save() |
| GET /audit?operation=governance_delete | audit_repo.List(filter) |
| Hook payload → [REDACTED] | pkg/privacy/redact.go (SOL-001) |
| GET /health → {status, workers, indexes} | health_monitor.go |
| PostgreSQL down → {status: "critical", alerts} | checkConnections() |
| GET /admin/doctor → full diagnostic report | doctor.go |
| Snapshot → Git commit với đúng stats | snapshot.go |
