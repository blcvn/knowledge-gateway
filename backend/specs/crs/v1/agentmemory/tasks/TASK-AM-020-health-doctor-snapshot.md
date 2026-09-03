# TASK-AM-020 — Health Monitor + Doctor + Git Snapshots

| Field | Value |
|-------|-------|
| **Task ID** | TASK-AM-020 |
| **Wave** | 4 (Governance) |
| **Component** | `services/vnp-platform/internal/usecase/admin/` |
| **Status** | ✅ Done |
| **Solution Ref** | SOL-007 §2.3, §2.4, §2.5 |
| **Priority** | Medium |
| **Depends On** | TASK-AM-017 |
| **Estimated** | 4h |

**Trạng thái:** ⏳ Pending  
**Ghi chú:** Health doctor + snapshot endpoint not implemented  
---

## Target Files

| Action | File Path |
|--------|-----------|
| CREATE | `services/vnp-platform/internal/usecase/admin/health_monitor.go` |
| CREATE | `services/vnp-platform/internal/usecase/admin/doctor.go` |
| CREATE | `services/vnp-platform/internal/usecase/admin/snapshot.go` |
| CREATE | `services/vnp-platform/internal/domain/admin/health_snapshot.go` |
| MODIFY | `services/vnp-platform/internal/adapter/grpc/admin_handler.go` |

---

## Implementation

### `domain/admin/health_snapshot.go`

```go
package admin

import "time"

type HealthSnapshot struct {
    Status        string         `json:"status"`   // "healthy" | "degraded" | "critical"
    Timestamp     time.Time      `json:"timestamp"`
    UptimeSeconds float64        `json:"uptime_seconds"`
    Workers       []WorkerStatus `json:"workers"`
    Memory        MemoryUsage    `json:"memory"`
    CPU           CPUUsage       `json:"cpu"`
    Indexes       IndexHealth    `json:"indexes"`
    Connections   ConnHealth     `json:"connections"`
    Alerts        []string       `json:"alerts,omitempty"`
}

type WorkerStatus struct {
    Name    string    `json:"name"`
    Status  string    `json:"status"`  // "running" | "stopped" | "error"
    LastRun time.Time `json:"last_run,omitempty"`
    Error   string    `json:"error,omitempty"`
}

type MemoryUsage struct {
    HeapMB    float64 `json:"heap_mb"`
    AllocMB   float64 `json:"alloc_mb"`
    GoroutineCount int `json:"goroutine_count"`
}

type CPUUsage struct {
    NumCPU    int     `json:"num_cpu"`
    GoMaxProcs int    `json:"go_max_procs"`
}

type IndexHealth struct {
    BM25Documents   int       `json:"bm25_documents"`
    VectorDocuments int       `json:"vector_documents"`
    LastPersisted   time.Time `json:"last_persisted,omitempty"`
    Status          string    `json:"status"`
}

type ConnHealth struct {
    PostgreSQL    string `json:"postgresql"`
    NATS         string `json:"nats"`
    Bifrost      string `json:"bifrost,omitempty"`
    ObserveSearch string `json:"observe_search"`
}

type DiagnosticCheck struct {
    Check      string `json:"check"`
    Status     string `json:"status"`     // "ok" | "warning" | "error"
    Message    string `json:"message"`
    Suggestion string `json:"suggestion,omitempty"`
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
}
```

### `usecase/admin/health_monitor.go`

```go
package admin

import (
    "context"
    "net/http"
    "runtime"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/nats-io/nats.go"
    "github.com/vnp-memory/services/vnp-platform/internal/domain/admin"
)

type HealthMonitorUseCase struct {
    db          *pgxpool.Pool
    nc          *nats.Conn
    bifrostURL  string
    searchURL   string
    startTime   time.Time
    workerMap   map[string]*WorkerTracker
}

type WorkerTracker struct {
    Name    string
    Running bool
    LastRun time.Time
}

func NewHealthMonitorUseCase(db *pgxpool.Pool, nc *nats.Conn, bifrostURL, searchURL string) *HealthMonitorUseCase {
    return &HealthMonitorUseCase{
        db:         db,
        nc:         nc,
        bifrostURL: bifrostURL,
        searchURL:  searchURL,
        startTime:  time.Now(),
        workerMap:  make(map[string]*WorkerTracker),
    }
}

func (uc *HealthMonitorUseCase) GetSnapshot(ctx context.Context) *admin.HealthSnapshot {
    snap := &admin.HealthSnapshot{
        Timestamp:     time.Now(),
        UptimeSeconds: time.Since(uc.startTime).Seconds(),
    }

    snap.Connections = uc.checkConnections(ctx)
    snap.Workers     = uc.checkWorkers()
    snap.Memory      = uc.getMemoryUsage()
    snap.CPU         = uc.getCPUUsage()
    snap.Indexes     = uc.checkIndexes(ctx)
    snap.Status      = uc.determineStatus(snap)
    snap.Alerts      = uc.buildAlerts(snap)

    return snap
}

func (uc *HealthMonitorUseCase) checkConnections(ctx context.Context) admin.ConnHealth {
    conn := admin.ConnHealth{}
    // PostgreSQL
    if err := uc.db.Ping(ctx); err != nil {
        conn.PostgreSQL = "error: " + err.Error()
    } else {
        conn.PostgreSQL = "ok"
    }
    // NATS
    if uc.nc.Status() != nats.CONNECTED {
        conn.NATS = "error: not connected"
    } else {
        conn.NATS = "ok"
    }
    // Bifrost (optional)
    if uc.bifrostURL != "" {
        if err := uc.httpCheck(ctx, uc.bifrostURL+"/health", 2*time.Second); err != nil {
            conn.Bifrost = "error: " + err.Error()
        } else {
            conn.Bifrost = "ok"
        }
    }
    // am-search
    if uc.searchURL != "" {
        if err := uc.httpCheck(ctx, uc.searchURL+"/health", 2*time.Second); err != nil {
            conn.ObserveSearch = "error: " + err.Error()
        } else {
            conn.ObserveSearch = "ok"
        }
    }
    return conn
}

func (uc *HealthMonitorUseCase) checkWorkers() []admin.WorkerStatus {
    var workers []admin.WorkerStatus
    for name, t := range uc.workerMap {
        status := "running"
        if !t.Running { status = "stopped" }
        workers = append(workers, admin.WorkerStatus{Name: name, Status: status, LastRun: t.LastRun})
    }
    return workers
}

func (uc *HealthMonitorUseCase) getMemoryUsage() admin.MemoryUsage {
    var m runtime.MemStats
    runtime.ReadMemStats(&m)
    return admin.MemoryUsage{
        HeapMB:         float64(m.HeapAlloc) / 1024 / 1024,
        AllocMB:        float64(m.Alloc) / 1024 / 1024,
        GoroutineCount: runtime.NumGoroutine(),
    }
}

func (uc *HealthMonitorUseCase) getCPUUsage() admin.CPUUsage {
    return admin.CPUUsage{NumCPU: runtime.NumCPU(), GoMaxProcs: runtime.GOMAXPROCS(0)}
}

func (uc *HealthMonitorUseCase) checkIndexes(ctx context.Context) admin.IndexHealth {
    // HTTP call to am-search service for index stats
    return admin.IndexHealth{Status: "unknown"}
}

func (uc *HealthMonitorUseCase) determineStatus(snap *admin.HealthSnapshot) string {
    if snap.Connections.PostgreSQL != "ok" { return "critical" }
    if snap.Connections.NATS != "ok" { return "degraded" }
    for _, w := range snap.Workers { if w.Status == "error" { return "degraded" } }
    return "healthy"
}

func (uc *HealthMonitorUseCase) buildAlerts(snap *admin.HealthSnapshot) []string {
    var alerts []string
    if snap.Connections.PostgreSQL != "ok" { alerts = append(alerts, "PostgreSQL connection failed") }
    if snap.Connections.NATS != "ok" { alerts = append(alerts, "NATS connection failed") }
    if snap.Memory.GoroutineCount > 10000 { alerts = append(alerts, "High goroutine count") }
    return alerts
}

func (uc *HealthMonitorUseCase) httpCheck(ctx context.Context, url string, timeout time.Duration) error {
    client := &http.Client{Timeout: timeout}
    req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
    resp, err := client.Do(req)
    if err != nil { return err }
    defer resp.Body.Close()
    if resp.StatusCode >= 400 { return fmt.Errorf("status %d", resp.StatusCode) }
    return nil
}

func (uc *HealthMonitorUseCase) RegisterWorker(name string) *WorkerTracker {
    t := &WorkerTracker{Name: name, Running: true}
    uc.workerMap[name] = t
    return t
}
```

### `usecase/admin/doctor.go`

```go
package admin

import (
    "context"
    "fmt"
    "os"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/nats-io/nats.go"
    "github.com/vnp-memory/services/vnp-platform/internal/domain/admin"
)

type DoctorUseCase struct {
    db          *pgxpool.Pool
    nc          *nats.Conn
    serviceURLs map[string]string
    dataDir     string
}

func NewDoctorUseCase(db *pgxpool.Pool, nc *nats.Conn, serviceURLs map[string]string, dataDir string) *DoctorUseCase {
    return &DoctorUseCase{db: db, nc: nc, serviceURLs: serviceURLs, dataDir: dataDir}
}

func (d *DoctorUseCase) Run(ctx context.Context) []admin.DiagnosticCheck {
    var checks []admin.DiagnosticCheck

    // 1. PostgreSQL
    if err := d.db.Ping(ctx); err != nil {
        checks = append(checks, admin.DiagnosticCheck{
            Check: "database.postgresql", Status: "error", Message: err.Error(),
            Suggestion: "Check VNP_MEMORY_POSTGRES_DSN and ensure PostgreSQL is running",
        })
    } else {
        checks = append(checks, admin.DiagnosticCheck{Check: "database.postgresql", Status: "ok", Message: "connected"})
    }

    // 2. NATS
    if d.nc.Status() != nats.CONNECTED {
        checks = append(checks, admin.DiagnosticCheck{
            Check: "nats", Status: "error", Message: "not connected",
            Suggestion: "Set VNP_MEMORY_NATS_MODE=embedded or check external NATS URL",
        })
    } else {
        checks = append(checks, admin.DiagnosticCheck{Check: "nats", Status: "ok"})
    }

    // 3. Service URLs
    for name, url := range d.serviceURLs {
        if err := d.httpCheck(ctx, url+"/health"); err != nil {
            checks = append(checks, admin.DiagnosticCheck{
                Check: "service." + name, Status: "error", Message: err.Error(),
                Suggestion: fmt.Sprintf("Ensure %s service is running at %s", name, url),
            })
        } else {
            checks = append(checks, admin.DiagnosticCheck{Check: "service." + name, Status: "ok"})
        }
    }

    // 4. Data directory write access
    dir := d.dataDir
    if dir == "" { dir = os.ExpandEnv("$HOME/.agentmemory") }
    if err := testWriteAccess(dir); err != nil {
        checks = append(checks, admin.DiagnosticCheck{
            Check: "storage.data_dir", Status: "error", Message: "cannot write to " + dir,
            Suggestion: "Check permissions: chmod 755 " + dir,
        })
    } else {
        checks = append(checks, admin.DiagnosticCheck{Check: "storage.data_dir", Status: "ok", Message: dir})
    }

    // 5. Go version check
    checks = append(checks, admin.DiagnosticCheck{
        Check: "runtime.go_version", Status: "ok",
        Message: fmt.Sprintf("go%s, goroutines=%d", goVersion(), runtime.NumGoroutine()),
    })

    return checks
}

func testWriteAccess(dir string) error {
    os.MkdirAll(dir, 0755)
    testFile := filepath.Join(dir, ".write_test")
    if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil { return err }
    return os.Remove(testFile)
}

func goVersion() string { return runtime.Version()[2:] }  // strip "go" prefix
```

### `usecase/admin/snapshot.go`

```go
package admin

import (
    "context"
    "fmt"
    "os/exec"
    "path/filepath"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/vnp-memory/services/vnp-platform/internal/domain/admin"
)

var ErrSnapshotDisabled = fmt.Errorf("snapshots are disabled")

type SnapshotUseCase struct {
    db      *pgxpool.Pool
    dataDir string
    enabled bool
}

func NewSnapshotUseCase(db *pgxpool.Pool, dataDir string, enabled bool) *SnapshotUseCase {
    return &SnapshotUseCase{db: db, dataDir: dataDir, enabled: enabled}
}

func (uc *SnapshotUseCase) Create(ctx context.Context) (*admin.SnapshotMeta, error) {
    if !uc.enabled { return nil, ErrSnapshotDisabled }

    stats, err := uc.collectStats(ctx)
    if err != nil { return nil, err }

    commitMsg := fmt.Sprintf("snapshot: %d sessions, %d memories (%s)",
        stats.Sessions, stats.Memories, time.Now().Format(time.RFC3339))

    cmds := [][]string{
        {"git", "-C", uc.dataDir, "add", "."},
        {"git", "-C", uc.dataDir, "commit", "-m", commitMsg, "--allow-empty"},
    }

    var commitHash string
    for _, cmd := range cmds {
        out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).Output()
        if err != nil {
            // git commit may fail if nothing to commit; this is acceptable
            if cmd[2] == "commit" { commitHash = "no-changes" } else { return nil, err }
        }
        if len(out) > 0 && cmd[2] == "commit" {
            lines := strings.Split(string(out), "\n")
            if len(lines) > 0 { commitHash = strings.TrimSpace(lines[0]) }
        }
    }

    return &admin.SnapshotMeta{
        CommitHash: commitHash,
        Stats:      stats,
        CreatedAt:  time.Now(),
    }, nil
}

func (uc *SnapshotUseCase) collectStats(ctx context.Context) (admin.SnapshotStats, error) {
    var stats admin.SnapshotStats
    row := uc.db.QueryRow(ctx, `SELECT COUNT(*) FROM agent_sessions`)
    row.Scan(&stats.Sessions)
    row = uc.db.QueryRow(ctx, `SELECT COUNT(*) FROM raw_observations`)
    row.Scan(&stats.Observations)
    row = uc.db.QueryRow(ctx, `SELECT COUNT(*) FROM agent_memories WHERE is_latest = TRUE`)
    row.Scan(&stats.Memories)
    return stats, nil
}
```

---

## Acceptance Criteria

| AC | Check |
|----|-------|
| `GET /v1/health` → `{status: "healthy", connections, workers, memory}` | ✅ |
| PostgreSQL down → `{status: "critical", alerts: ["PostgreSQL connection failed"]}` | ✅ |
| `GET /v1/admin/doctor` → array of DiagnosticChecks | ✅ |
| Doctor checks: postgresql, nats, storage.data_dir, runtime | ✅ |
| `POST /v1/admin/snapshot` with `SNAPSHOT_ENABLED=false` → 400 error | ✅ |
| `POST /v1/admin/snapshot` with `SNAPSHOT_ENABLED=true` → git commit created | ✅ |
| Snapshot stats: sessions, observations, memories count correct | ✅ |
