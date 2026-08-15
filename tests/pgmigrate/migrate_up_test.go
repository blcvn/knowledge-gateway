// Package pgmigrate_test applies the checked-in migration set to a genuinely empty database.
//
// tests/migrations guards the file names; this suite guards the SQL. A migration that references a
// table an earlier migration never created, or that is valid on its author's machine but not on a
// clean one, only fails here.
//
// Opt-in: set KG_TEST_POSTGRES_DSN to a superuser-capable connection on a Postgres with the vector
// extension available (migration 000008 does CREATE EXTENSION vector, so pgvector/pgvector:pg16 or
// equivalent). The suite creates and drops its own scratch database; it never touches the database
// named in the DSN.
//
//	docker run -d --name kgpg -e POSTGRES_USER=kg_service -e POSTGRES_PASSWORD=secret \
//	  -e POSTGRES_DB=kg_service -p 55432:5432 pgvector/pgvector:pg16
//	KG_TEST_POSTGRES_DSN="postgres://kg_service:secret@127.0.0.1:55432/kg_service?sslmode=disable" \
//	  go test ./tests/pgmigrate/...
package pgmigrate_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const migrationsDir = "../../migrations"

var upMigration = regexp.MustCompile(`^(\d{6})_([a-z0-9_]+)\.up\.sql$`)

type upFile struct {
	version int
	name    string
	path    string
}

func orderedUpMigrations(t *testing.T) []upFile {
	t.Helper()
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	files := make([]upFile, 0, len(entries))
	for _, entry := range entries {
		m := upMigration.FindStringSubmatch(entry.Name())
		if m == nil {
			continue
		}
		version, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("bad version in %q: %v", entry.Name(), err)
		}
		files = append(files, upFile{version: version, name: m[2], path: filepath.Join(migrationsDir, entry.Name())})
	}
	if len(files) == 0 {
		t.Fatalf("no up migrations found in %s", migrationsDir)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].version < files[j].version })
	return files
}

// scratchDatabase creates an empty database next to the one in the DSN and returns a connection to
// it. Running against a fresh database is the whole point: applying migrations to an
// already-migrated database would pass even if migration 1 depended on migration 9.
func scratchDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("KG_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("KG_TEST_POSTGRES_DSN not set; see package doc for how to run this suite")
	}

	admin, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open admin db: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := admin.PingContext(ctx); err != nil {
		admin.Close()
		t.Fatalf("ping admin db: %v", err)
	}

	name := fmt.Sprintf("kg_migrate_check_%d", time.Now().UnixNano())
	if _, err := admin.ExecContext(context.Background(), `CREATE DATABASE `+pgQuoteIdent(name)); err != nil {
		admin.Close()
		t.Fatalf("create scratch database: %v", err)
	}

	scratchDSN, err := replaceDatabase(dsn, name)
	if err != nil {
		admin.Close()
		t.Fatalf("build scratch dsn: %v", err)
	}
	scratch, err := sql.Open("pgx", scratchDSN)
	if err != nil {
		admin.Close()
		t.Fatalf("open scratch db: %v", err)
	}

	t.Cleanup(func() {
		scratch.Close()
		if _, err := admin.ExecContext(context.Background(), `DROP DATABASE IF EXISTS `+pgQuoteIdent(name)); err != nil {
			t.Errorf("drop scratch database %s: %v", name, err)
		}
		admin.Close()
	})
	return scratch
}

func pgQuoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func replaceDatabase(dsn, database string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", err
	}
	u.Path = "/" + database
	return u.String(), nil
}

// TestMigrateUpFromEmptyDatabase applies every up migration in version order. This is the check
// that caught nothing before the 000014 duplicate was removed, because golang-migrate refused to
// load the directory at all — the set was unrunnable rather than wrong.
func TestMigrateUpFromEmptyDatabase(t *testing.T) {
	db := scratchDatabase(t)

	for _, migration := range orderedUpMigrations(t) {
		sqlBytes, err := os.ReadFile(migration.path)
		if err != nil {
			t.Fatalf("read %s: %v", migration.path, err)
		}
		if _, err := db.ExecContext(context.Background(), string(sqlBytes)); err != nil {
			t.Fatalf("apply %06d_%s: %v", migration.version, migration.name, err)
		}
	}

	// Spot-check the objects the executor integration depends on, so a silently reordered or
	// dropped migration cannot pass this test.
	assertColumnExists(t, db, "kg_relationships", "external_ref")
	assertIndexExists(t, db, "idx_kg_relationships_external_ref_active")
	assertIndexExists(t, db, "idx_kg_nodes_graph_scope")
	assertIndexExists(t, db, "idx_kg_relationships_graph_scope")
	assertIndexExists(t, db, "idx_kg_nodes_external_ref_prefix")
}

func assertColumnExists(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	var exists bool
	if err := db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = $1 AND column_name = $2
		)
	`, table, column).Scan(&exists); err != nil {
		t.Fatalf("check column %s.%s: %v", table, column, err)
	}
	if !exists {
		t.Errorf("column %s.%s missing after migrate up", table, column)
	}
}

func assertIndexExists(t *testing.T, db *sql.DB, index string) {
	t.Helper()
	var exists bool
	if err := db.QueryRowContext(context.Background(), `
		SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)
	`, index).Scan(&exists); err != nil {
		t.Fatalf("check index %s: %v", index, err)
	}
	if !exists {
		t.Errorf("index %s missing after migrate up", index)
	}
}
