package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"kg-service/internal/write"
)

const stubDriverName = "postgres-repository-stub"

var registerStubDriverOnce sync.Once

type stubHarness struct {
	mu    sync.Mutex
	calls []stubCall
}

type stubCall struct {
	kind  string
	query string
	args  []any
}

func (h *stubHarness) record(kind, query string, args []driver.NamedValue) {
	h.mu.Lock()
	defer h.mu.Unlock()
	values := make([]any, 0, len(args))
	for _, arg := range args {
		values = append(values, arg.Value)
	}
	h.calls = append(h.calls, stubCall{kind: kind, query: query, args: values})
}

func (h *stubHarness) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.calls = nil
}

func (h *stubHarness) snapshot() []stubCall {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]stubCall, len(h.calls))
	copy(out, h.calls)
	return out
}

type stubDriver struct {
	harness *stubHarness
}

func (d stubDriver) Open(name string) (driver.Conn, error) {
	return &stubConn{harness: d.harness}, nil
}

type stubConn struct {
	harness *stubHarness
}

func (c *stubConn) Prepare(query string) (driver.Stmt, error) { return nil, driver.ErrSkip }
func (c *stubConn) Close() error                              { return nil }
func (c *stubConn) Begin() (driver.Tx, error)                 { return nil, driver.ErrSkip }

func (c *stubConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.harness.record("exec", query, args)
	return stubResult(1), nil
}

func (c *stubConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.harness.record("query", query, args)

	switch {
	case strings.Contains(query, "FROM kg_graph_identifiers"):
		return &stubRows{
			columns: []string{"identifier_id", "owner_tenant_id", "owner_app_id", "graph_scope", "head_version_number", "head_version_id", "created_at", "updated_at"},
			values: [][]driver.Value{{
				"identifier-1",
				"tenant-1",
				"app-1",
				"scope-1",
				int64(7),
				"version-1",
				time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
				time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC),
			}},
		}, nil
	case strings.Contains(query, "INSERT INTO kg_graph_versions"):
		return &stubRows{
			columns: []string{"version_id"},
			values:  [][]driver.Value{{"version-1"}},
		}, nil
	default:
		return &stubRows{columns: []string{}, values: nil}, nil
	}
}

func (c *stubConn) CheckNamedValue(*driver.NamedValue) error { return nil }

type stubRows struct {
	columns []string
	values  [][]driver.Value
	idx     int
}

func (r *stubRows) Columns() []string { return append([]string(nil), r.columns...) }

func (r *stubRows) Close() error { return nil }

func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.values) {
		return io.EOF
	}
	row := r.values[r.idx]
	r.idx++
	copy(dest, row)
	return nil
}

type stubResult int64

func (r stubResult) LastInsertId() (int64, error) { return 0, nil }
func (r stubResult) RowsAffected() (int64, error) { return int64(r), nil }

func openStubRepositoryDB(t *testing.T, harness *stubHarness) *sql.DB {
	t.Helper()

	registerStubDriverOnce.Do(func() {
		sql.Register(stubDriverName, stubDriver{harness: harness})
	})
	db, err := sql.Open(stubDriverName, "")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	return db
}

func TestSealGraphVersionBindsOwnerIdentityIntoGraphIdentifiers(t *testing.T) {
	harness := &stubHarness{}
	db := openStubRepositoryDB(t, harness)
	t.Cleanup(func() { _ = db.Close() })
	repo := NewRepository(db)

	_, _, err := repo.SealGraphVersion(context.Background(), write.GraphVersionSealRequest{
		OwnerTenantID: "tenant-1",
		OwnerAppID:    "app-1",
		GraphScope:    "scope-1",
		ReferenceID:   "ref-1",
		StorageClass:  "ONLINE",
		VersionStatus: "SEALED",
	})
	if err != nil {
		t.Fatalf("SealGraphVersion() error = %v", err)
	}

	calls := harness.snapshot()
	if len(calls) < 3 {
		t.Fatalf("recorded calls = %#v, want at least 3", calls)
	}

	firstExec := calls[0]
	if !strings.Contains(firstExec.query, "INSERT INTO kg_graph_identifiers") {
		t.Fatalf("first query = %q, want kg_graph_identifiers insert", firstExec.query)
	}
	if got := firstExec.args; len(got) < 3 || got[0] != "tenant-1" || got[1] != "app-1" || got[2] != "scope-1" {
		t.Fatalf("kg_graph_identifiers args = %#v, want tenant/app/scope", got)
	}
}

