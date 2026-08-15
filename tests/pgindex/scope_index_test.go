// Package pgindex_test verifies that the graph-scope predicates introduced by migration 000017
// are actually served by an index on a table large enough for the planner to have a choice.
//
// These assertions cannot be made against the in-memory store or a toy dataset: on a small table
// Postgres correctly prefers a sequential scan, so a passing test would prove nothing. The suite
// therefore seeds a six-figure row count and inspects the real plan.
//
// Opt-in: set KG_TEST_POSTGRES_DSN to a database with the full migration set applied, e.g.
//
//	docker run -d --name kgpg -e POSTGRES_USER=kg_service -e POSTGRES_PASSWORD=secret \
//	  -e POSTGRES_DB=kg_service -p 55432:5432 pgvector/pgvector:pg16
//	docker run --rm --network host -v "$PWD/migrations:/migrations:ro" migrate/migrate:v4.17.1 \
//	  -path /migrations -database "postgres://kg_service:secret@127.0.0.1:55432/kg_service?sslmode=disable" up
//	KG_TEST_POSTGRES_DSN="postgres://kg_service:secret@127.0.0.1:55432/kg_service?sslmode=disable" \
//	  go test ./tests/pgindex/...
package pgindex_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	seedRows       = 120_000
	testDomainID   = "pgindex_scope_domain"
	testTenantID   = "11111111-1111-1111-1111-111111111111"
	targetScope    = "bas:kg:doc-under-test"
	noiseScopeStem = "bas:kg:doc-noise-"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("KG_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("KG_TEST_POSTGRES_DSN not set; see package doc for how to run this suite")
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
	// Registered here, before seed() registers its row cleanup, so it runs LAST: t.Cleanup is LIFO
	// and the cleanup queries still need an open handle. A plain `defer db.Close()` in each test
	// would close the pool before those cleanups ran.
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seed builds a table where the target scope is a small slice of a large whole. If every row shared
// one scope the planner would sequential-scan regardless of indexes, and the test would be
// measuring nothing.
func seed(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()

	cleanup(t, db)
	t.Cleanup(func() { cleanup(t, db) })

	if _, err := db.ExecContext(ctx, `
		INSERT INTO domains (id, name, owner_tenant_id, status, version, visibility)
		VALUES ($1, 'pgindex scope domain', $2, 'active', 1, 'private')
		ON CONFLICT (id) DO NOTHING
	`, testDomainID, testTenantID); err != nil {
		t.Fatalf("seed domain: %v", err)
	}

	// 1 row in 400 lands in the target scope; the rest spread over 500 other scopes.
	if _, err := db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO kg_nodes (
			id, node_type, domain_id, owner_tenant_id, visibility, properties, domain_version,
			external_ref, is_deleted
		)
		SELECT
			gen_random_uuid(),
			CASE WHEN i %% 7 = 0 THEN 'FEATURE' ELSE 'SCREEN' END,
			$1,
			$2,
			'private',
			jsonb_build_object(
				'_kg_graph_scope', CASE WHEN i %% 400 = 0 THEN $3 ELSE $4 || (i %% 500)::text END,
				'kg_level',        CASE WHEN i %% 3 = 0 THEN 'product' ELSE 'feature' END,
				'feature_ref',     'F-' || lpad(((i %% 50) + 1)::text, 3, '0'),
				'document_id',     'doc-under-test'
			),
			1,
			'bas/doc-' || (i %% 500)::text || '/n/node-' || i::text,
			false
		FROM generate_series(1, %d) AS s(i)
	`, seedRows), testDomainID, testTenantID, targetScope, noiseScopeStem); err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	if _, err := db.ExecContext(ctx, `ANALYZE kg_nodes`); err != nil {
		t.Fatalf("analyze: %v", err)
	}
}

func cleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `DELETE FROM kg_nodes WHERE domain_id = $1`, testDomainID); err != nil {
		t.Fatalf("cleanup nodes: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM domains WHERE id = $1`, testDomainID); err != nil {
		t.Fatalf("cleanup domain: %v", err)
	}
}

func explain(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), "EXPLAIN (ANALYZE, BUFFERS) "+query, args...)
	if err != nil {
		t.Fatalf("explain: %v", err)
	}
	defer rows.Close()
	var plan strings.Builder
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			t.Fatalf("scan plan: %v", err)
		}
		plan.WriteString(line)
		plan.WriteString("\n")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("plan rows: %v", err)
	}
	return plan.String()
}

func assertIndexed(t *testing.T, plan, indexName, table string) {
	t.Helper()
	if !strings.Contains(plan, indexName) {
		t.Errorf("plan does not use %s:\n%s", indexName, plan)
	}
	if strings.Contains(plan, "Seq Scan on "+table) {
		t.Errorf("plan falls back to a sequential scan of %s:\n%s", table, plan)
	}
}

// TestScopeQueryUsesGraphScopeIndex covers the read-by-scope predicate with an empty level filter:
// everything belonging to one document.
func TestScopeQueryUsesGraphScopeIndex(t *testing.T) {
	db := openDB(t)
	seed(t, db)

	plan := explain(t, db, `
		SELECT id
		FROM kg_nodes
		WHERE NOT is_deleted
		  AND properties ->> '_kg_graph_scope' = $1
		ORDER BY id
		LIMIT 1000
	`, targetScope)
	assertIndexed(t, plan, "idx_kg_nodes_graph_scope", "kg_nodes")
}

// TestScopeLevelQueryUsesGraphScopeIndex covers the product-slice read, which adds the level column
// of the composite index.
func TestScopeLevelQueryUsesGraphScopeIndex(t *testing.T) {
	db := openDB(t)
	seed(t, db)

	plan := explain(t, db, `
		SELECT id
		FROM kg_nodes
		WHERE NOT is_deleted
		  AND properties ->> '_kg_graph_scope' = $1
		  AND properties ->> 'kg_level' = 'product'
		ORDER BY id
		LIMIT 1000
	`, targetScope)
	assertIndexed(t, plan, "idx_kg_nodes_graph_scope", "kg_nodes")
}

// TestScopeFeatureQueryUsesGraphScopeIndex covers the feature-slice read: product union one
// feature, which is the shape the executor's scoped load issues on every feature skill.
func TestScopeFeatureQueryUsesGraphScopeIndex(t *testing.T) {
	db := openDB(t)
	seed(t, db)

	plan := explain(t, db, `
		SELECT id
		FROM kg_nodes
		WHERE NOT is_deleted
		  AND properties ->> '_kg_graph_scope' = $1
		  AND (
				properties ->> 'kg_level' = 'product'
				OR (properties ->> 'kg_level' = 'feature' AND properties ->> 'feature_ref' = $2)
		  )
		ORDER BY id
		LIMIT 1000
	`, targetScope, "F-001")
	assertIndexed(t, plan, "idx_kg_nodes_graph_scope", "kg_nodes")
}

// TestExternalRefPrefixUsesPatternIndex covers DELETE /v1/kg/write/nodes:by-external-ref-prefix.
// Without text_pattern_ops this predicate cannot use a plain btree index under most collations.
func TestExternalRefPrefixUsesPatternIndex(t *testing.T) {
	db := openDB(t)
	seed(t, db)

	plan := explain(t, db, `
		SELECT id
		FROM kg_nodes
		WHERE external_ref IS NOT NULL
		  AND NOT is_deleted
		  AND external_ref LIKE $1
		LIMIT 1000
	`, "bas/doc-42/%")
	assertIndexed(t, plan, "idx_kg_nodes_external_ref_prefix", "kg_nodes")
}
