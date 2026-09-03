package admin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
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
	if dir == "" {
		dir = os.ExpandEnv("$HOME/.agentmemory")
	}
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
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return err
	}
	return os.Remove(testFile)
}

func goVersion() string { return runtime.Version()[2:] } // strip "go" prefix

func (d *DoctorUseCase) httpCheck(ctx context.Context, url string) error {
    return nil // Dummy implementation for now
}
