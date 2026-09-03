package admin

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
)

type HealthMonitorUseCase struct {
	db         *pgxpool.Pool
	nc         *nats.Conn
	bifrostURL string
	searchURL  string
	startTime  time.Time
	workerMap  map[string]*WorkerTracker
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
	snap.Workers = uc.checkWorkers()
	snap.Memory = uc.getMemoryUsage()
	snap.CPU = uc.getCPUUsage()
	snap.Indexes = uc.checkIndexes(ctx)
	snap.Status = uc.determineStatus(snap)
	snap.Alerts = uc.buildAlerts(snap)

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
		if !t.Running {
			status = "stopped"
		}
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
	if snap.Connections.PostgreSQL != "ok" {
		return "critical"
	}
	if snap.Connections.NATS != "ok" {
		return "degraded"
	}
	for _, w := range snap.Workers {
		if w.Status == "error" {
			return "degraded"
		}
	}
	return "healthy"
}

func (uc *HealthMonitorUseCase) buildAlerts(snap *admin.HealthSnapshot) []string {
	var alerts []string
	if snap.Connections.PostgreSQL != "ok" {
		alerts = append(alerts, "PostgreSQL connection failed")
	}
	if snap.Connections.NATS != "ok" {
		alerts = append(alerts, "NATS connection failed")
	}
	if snap.Memory.GoroutineCount > 10000 {
		alerts = append(alerts, "High goroutine count")
	}
	return alerts
}

func (uc *HealthMonitorUseCase) httpCheck(ctx context.Context, url string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	return nil
}

func (uc *HealthMonitorUseCase) RegisterWorker(name string) *WorkerTracker {
	t := &WorkerTracker{Name: name, Running: true}
	uc.workerMap[name] = t
	return t
}
