// Package pgsession_test covers giving up on a sync session, against real Postgres.
//
// This is the one behaviour the in-memory store cannot vouch for. That store assigns
// version_status as a plain string field, so it accepts any value; Postgres has a CHECK constraint
// on the column. Between 000012 and 000018 the constraint permitted PENDING_ENTITIES, SEALED and
// FAILED_FINALIZATION while both abandon paths wrote ABANDONED, so every abandon failed against a
// real database and the whole unit suite stayed green.
//
// The damage was in the sweep rather than the explicit abandon: cleanupExpiredSyncSessionInTx marks
// the version and deletes the scope lease in one transaction, so the rejected UPDATE took the DELETE
// down with it. A lease left by a writer that died mid-write became permanent, and every subsequent
// write to that graph scope failed with SYNC_SCOPE_LOCKED — unrecoverable from the client side,
// since releasing another session's lease is the service's job.
//
// Opt-in: set KG_TEST_POSTGRES_DSN to a database with the full migration set applied. See
// tests/pgmigrate for the container recipe.
package pgsession_test

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"kg-service/internal/platform/postgres"
	"kg-service/internal/write"
)

const (
	testDomainID = "pgsession_domain"
	testTenantID = "11111111-1111-1111-1111-111111111111"
	testAppID    = "11111111-1111-4111-8111-111111111111"
	testScope    = "bas:kg:pgsession-doc"
)

func newRepo(t *testing.T) (*postgres.Repository, *sql.DB) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("KG_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("KG_TEST_POSTGRES_DSN not set; see package doc")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return postgres.NewRepository(db), db
}

// openSession creates the identifier, version and lease a live sync session consists of, and
// returns the version id.
func openSession(t *testing.T, repo *postgres.Repository, db *sql.DB) string {
	t.Helper()
	ctx := context.Background()
	cleanup(t, db)
	t.Cleanup(func() { cleanup(t, db) })

	if _, err := db.ExecContext(ctx, `
		INSERT INTO domains (id, name, owner_tenant_id, status, version, visibility)
		VALUES ($1, 'pgsession domain', $2, 'active', 1, 'private')
		ON CONFLICT (id) DO NOTHING
	`, testDomainID, testTenantID); err != nil {
		t.Fatalf("seed domain: %v", err)
	}

	return openVersion(t, repo, "pgsession-ref")
}

// openVersion seals a PENDING_ENTITIES version on the shared scope and leases the scope to it,
// which together are what an open sync session is.
func openVersion(t *testing.T, repo *postgres.Repository, referenceID string) string {
	t.Helper()
	ctx := context.Background()

	_, version, err := repo.SealGraphVersion(ctx, write.GraphVersionSealRequest{
		OwnerTenantID: testTenantID,
		OwnerAppID:    testAppID,
		GraphScope:    testScope,
		ReferenceID:   referenceID,
		StorageClass:  "ONLINE",
		VersionStatus: "PENDING_ENTITIES",
	})
	if err != nil {
		t.Fatalf("seal pending version: %v", err)
	}
	if err := repo.AcquireScopeLease(ctx, testTenantID, testAppID, testScope, version.VersionID,
		time.Now().UTC().Add(2*time.Hour)); err != nil {
		t.Fatalf("acquire lease: %v", err)
	}
	return version.VersionID
}

// TestAbandonGraphVersionIsAcceptedByTheDatabase is the direct regression: before 000018 this
// returned a check-constraint violation and the version stayed PENDING_ENTITIES forever.
func TestAbandonGraphVersionIsAcceptedByTheDatabase(t *testing.T) {
	repo, db := newRepo(t)
	versionID := openSession(t, repo, db)

	if err := repo.AbandonGraphVersion(context.Background(), versionID); err != nil {
		t.Fatalf("AbandonGraphVersion() error = %v, want the abandon to be accepted", err)
	}
	if got := versionStatus(t, db, versionID); got != "ABANDONED" {
		t.Fatalf("version_status = %q, want ABANDONED", got)
	}
}

// TestCleanupExpiredSyncSessionReleasesTheLease is the behaviour that actually mattered.
//
// A client cannot release a lease held by a session it does not own, so if this sweep leaves the
// lease in place the graph scope is locked permanently. Asserting the lease is gone — not merely
// that the call returned nil — is the point: the update and the delete share a transaction, and the
// bug's whole signature was a silent rollback of the delete.
func TestCleanupExpiredSyncSessionReleasesTheLease(t *testing.T) {
	repo, db := newRepo(t)
	versionID := openSession(t, repo, db)

	if err := repo.CleanupExpiredSyncSession(context.Background(), versionID); err != nil {
		t.Fatalf("CleanupExpiredSyncSession() error = %v", err)
	}
	if got := versionStatus(t, db, versionID); got != "ABANDONED" {
		t.Fatalf("version_status = %q, want ABANDONED", got)
	}
	if leases := leaseCount(t, db, testScope); leases != 0 {
		t.Fatalf("leases on %s = %d, want 0: the scope stays locked for good if the sweep cannot release it", testScope, leases)
	}
}

// TestScopeIsWritableAgainAfterCleanup closes the loop: reclaiming a lease is only worth anything if
// the next writer can then open a session on that scope.
func TestScopeIsWritableAgainAfterCleanup(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()
	versionID := openSession(t, repo, db)

	if err := repo.CleanupExpiredSyncSession(ctx, versionID); err != nil {
		t.Fatalf("CleanupExpiredSyncSession() error = %v", err)
	}

	nextVersionID := openVersion(t, repo, "pgsession-ref-2")
	if _, ok := repo.GetGraphVersionByID(ctx, nextVersionID); !ok {
		t.Fatal("second session's version is missing after a successful open")
	}
}

func versionStatus(t *testing.T, db *sql.DB, versionID string) string {
	t.Helper()
	var status string
	if err := db.QueryRowContext(context.Background(),
		`SELECT version_status FROM kg_graph_versions WHERE version_id = $1`, versionID).Scan(&status); err != nil {
		t.Fatalf("read version status: %v", err)
	}
	return status
}

func leaseCount(t *testing.T, db *sql.DB, scope string) int {
	t.Helper()
	var count int
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM kg_graph_scope_leases WHERE graph_scope = $1`, scope).Scan(&count); err != nil {
		t.Fatalf("count leases: %v", err)
	}
	return count
}

func cleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	statements := []string{
		`DELETE FROM kg_graph_scope_leases WHERE graph_scope = $1`,
		`DELETE FROM kg_graph_versions WHERE identifier_id IN (SELECT identifier_id FROM kg_graph_identifiers WHERE graph_scope = $1)`,
		`DELETE FROM kg_graph_identifiers WHERE graph_scope = $1`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement, testScope); err != nil {
			t.Fatalf("cleanup %q: %v", statement, err)
		}
	}
}
