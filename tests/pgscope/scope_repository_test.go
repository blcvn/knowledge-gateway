// Package pgscope_test exercises the scope read/delete repository methods against real Postgres.
//
// The in-memory store tests in internal/write and internal/read pin the contract, but they cannot
// prove the SQL: predicate construction, JSONB property extraction, cursor comparison semantics and
// RETURNING clauses only exist in the Postgres implementation. A divergence there would let the
// executor's scoped load and scoped delete behave differently in production than in CI.
//
// Opt-in: set KG_TEST_POSTGRES_DSN to a database with the full migration set applied. See
// tests/pgmigrate for the container recipe.
package pgscope_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"kg-service/internal/platform/postgres"
	"kg-service/internal/write"
)

const (
	testDomainID = "pgscope_domain"
	testTenantID = "11111111-1111-1111-1111-111111111111"
	graphScope   = "bas:kg:pgscope-doc"
	otherScope   = "bas:kg:pgscope-other"
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

type nodeSpec struct {
	id         string
	nodeType   string
	level      string
	featureRef string
	scope      string
}

type relSpec struct {
	id         string
	from, to   string
	level      string
	featureRef string
	scope      string
}

// seed builds a partitioned graph: product slice, two feature slices, and one node in a second
// graph scope that must never appear in any assertion below.
func seed(t *testing.T, repo *postgres.Repository, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	cleanup(t, db)
	t.Cleanup(func() { cleanup(t, db) })

	if _, err := db.ExecContext(ctx, `
		INSERT INTO domains (id, name, owner_tenant_id, status, version, visibility)
		VALUES ($1, 'pgscope domain', $2, 'active', 1, 'private')
		ON CONFLICT (id) DO NOTHING
	`, testDomainID, testTenantID); err != nil {
		t.Fatalf("seed domain: %v", err)
	}

	nodes := []nodeSpec{
		{"11111111-0000-0000-0000-00000000000a", "FEATURE", "product", "", graphScope},
		{"11111111-0000-0000-0000-00000000000b", "SCREEN", "product", "", graphScope},
		{"11111111-0000-0000-0000-00000000000c", "SCREEN", "feature", "F-001", graphScope},
		{"11111111-0000-0000-0000-00000000000d", "SCREEN", "feature", "F-001", graphScope},
		{"11111111-0000-0000-0000-00000000000e", "SCREEN", "feature", "F-002", graphScope},
		{"11111111-0000-0000-0000-00000000000f", "SCREEN", "product", "", otherScope},
	}
	records := make([]write.NodeRecord, 0, len(nodes))
	now := time.Now().UTC()
	for _, spec := range nodes {
		records = append(records, write.NodeRecord{
			ID:            spec.id,
			NodeType:      spec.nodeType,
			DomainID:      testDomainID,
			OwnerTenantID: testTenantID,
			Visibility:    "private",
			ExternalRef:   "bas/pgscope/n/" + spec.id,
			DomainVersion: 1,
			Properties: map[string]any{
				"_kg_graph_scope": spec.scope,
				"kg_level":        spec.level,
				"feature_ref":     spec.featureRef,
				"reference_id":    "REF-" + spec.id[len(spec.id)-1:],
				"summary":         "summary text that refs_only must drop",
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := repo.CreateNodesBulkWithOutbox(ctx, records, nil); err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	rels := []relSpec{
		{"22222222-0000-0000-0000-00000000000a", nodes[0].id, nodes[1].id, "product", "", graphScope},
		{"22222222-0000-0000-0000-00000000000b", nodes[0].id, nodes[2].id, "feature", "F-001", graphScope},
		{"22222222-0000-0000-0000-00000000000c", nodes[0].id, nodes[4].id, "feature", "F-002", graphScope},
	}
	relRecords := make([]write.RelationshipRecord, 0, len(rels))
	for _, spec := range rels {
		relRecords = append(relRecords, write.RelationshipRecord{
			ID:            spec.id,
			RelType:       "CONTAINS",
			FromNodeID:    spec.from,
			ToNodeID:      spec.to,
			DomainID:      testDomainID,
			OwnerTenantID: testTenantID,
			DomainVersion: 1,
			ExternalRef:   "bas/pgscope/e/" + spec.id,
			Properties: map[string]any{
				"_kg_graph_scope": spec.scope,
				"kg_level":        spec.level,
				"feature_ref":     spec.featureRef,
			},
			CreatedAt: now,
		})
	}
	if err := repo.CreateRelationshipsBulkWithOutbox(ctx, relRecords, nil); err != nil {
		t.Fatalf("seed relationships: %v", err)
	}
}

func cleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `DELETE FROM kg_relationships WHERE domain_id = $1`, testDomainID); err != nil {
		t.Fatalf("cleanup relationships: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM kg_nodes WHERE domain_id = $1`, testDomainID); err != nil {
		t.Fatalf("cleanup nodes: %v", err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM domains WHERE id = $1`, testDomainID); err != nil {
		t.Fatalf("cleanup domain: %v", err)
	}
}

func scopeQuery(levels ...write.ScopeLevel) write.ScopeQuery {
	return write.ScopeQuery{
		ScopeFilter: write.ScopeFilter{
			DomainID:   testDomainID,
			GraphScope: graphScope,
			Levels:     levels,
		},
	}
}

func idsOf(nodes []write.NodeRecord) []string {
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	sort.Strings(ids)
	return ids
}

func relIDsOf(rels []write.RelationshipRecord) []string {
	ids := make([]string, 0, len(rels))
	for _, rel := range rels {
		ids = append(ids, rel.ID)
	}
	sort.Strings(ids)
	return ids
}

func TestListNodesByScopeEmptyLevels(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)

	nodes, next, err := repo.ListNodesByScope(context.Background(), scopeQuery())
	if err != nil {
		t.Fatalf("ListNodesByScope() error = %v", err)
	}
	if len(nodes) != 5 {
		t.Fatalf("nodes = %d (%v), want 5 — the sixth belongs to another scope", len(nodes), idsOf(nodes))
	}
	if next != "" {
		t.Fatalf("next cursor = %q on a complete page, want empty", next)
	}
	for _, node := range nodes {
		if got := node.Properties["_kg_graph_scope"]; got != graphScope {
			t.Fatalf("node %s has scope %v", node.ID, got)
		}
	}
}

func TestListNodesByScopeProductLevel(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)

	nodes, _, err := repo.ListNodesByScope(context.Background(), scopeQuery(write.ScopeLevel{Level: "product"}))
	if err != nil {
		t.Fatalf("ListNodesByScope() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("nodes = %d (%v), want 2", len(nodes), idsOf(nodes))
	}
	for _, node := range nodes {
		if node.Properties["kg_level"] != "product" {
			t.Fatalf("node %s leaked at level %v", node.ID, node.Properties["kg_level"])
		}
	}
}

func TestListNodesByScopeProductUnionFeature(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)

	nodes, _, err := repo.ListNodesByScope(context.Background(), scopeQuery(
		write.ScopeLevel{Level: "product"},
		write.ScopeLevel{Level: "feature", FeatureRef: "F-001"},
	))
	if err != nil {
		t.Fatalf("ListNodesByScope() error = %v", err)
	}
	if len(nodes) != 4 {
		t.Fatalf("nodes = %d (%v), want 4 (2 product + 2 in F-001)", len(nodes), idsOf(nodes))
	}
	for _, node := range nodes {
		if node.Properties["feature_ref"] == "F-002" {
			t.Fatalf("sibling feature leaked: %s", node.ID)
		}
	}
}

// TestListNodesByScopePaginationIsComplete walks the scope one row per page. The cursor comparison
// happens in SQL (`id > $n`), so this is the only place that check is actually exercised.
func TestListNodesByScopePaginationIsComplete(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)

	seen := make([]string, 0, 5)
	cursor := ""
	for i := 0; i < 10; i++ {
		query := scopeQuery()
		query.Limit = 1
		query.Cursor = cursor
		page, next, err := repo.ListNodesByScope(context.Background(), query)
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
		for _, node := range page {
			seen = append(seen, node.ID)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if len(seen) != 5 {
		t.Fatalf("paged %d rows (%v), want 5", len(seen), seen)
	}
	unique := map[string]struct{}{}
	for _, id := range seen {
		if _, dup := unique[id]; dup {
			t.Fatalf("row %s returned twice while paging", id)
		}
		unique[id] = struct{}{}
	}
}

// TestListNodesByScopeRefsOnly proves the projection actually drops payload while keeping the keys
// a delta computation and a feature cascade need.
func TestListNodesByScopeRefsOnly(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)

	query := scopeQuery()
	query.RefsOnly = true
	nodes, _, err := repo.ListNodesByScope(context.Background(), query)
	if err != nil {
		t.Fatalf("ListNodesByScope() error = %v", err)
	}
	if len(nodes) != 5 {
		t.Fatalf("nodes = %d, want 5", len(nodes))
	}
	for _, node := range nodes {
		if _, present := node.Properties["summary"]; present {
			t.Fatalf("node %s still carries its payload in refs_only mode", node.ID)
		}
		if node.ExternalRef == "" {
			t.Fatalf("node %s lost its external_ref", node.ID)
		}
		if _, ok := node.Properties["kg_level"]; !ok {
			t.Fatalf("node %s lost kg_level", node.ID)
		}
		if _, ok := node.Properties["reference_id"]; !ok {
			t.Fatalf("node %s lost reference_id, which the feature cascade needs", node.ID)
		}
		if node.NodeType == "" {
			t.Fatalf("node %s lost node_type, which the feature cascade needs", node.ID)
		}
	}
}

func TestListRelationshipsByScopeFiltersByLevel(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)

	rels, _, err := repo.ListRelationshipsByScope(context.Background(), scopeQuery(write.ScopeLevel{Level: "feature", FeatureRef: "F-001"}))
	if err != nil {
		t.Fatalf("ListRelationshipsByScope() error = %v", err)
	}
	if len(rels) != 1 {
		t.Fatalf("relationships = %d (%v), want 1", len(rels), relIDsOf(rels))
	}
	if rels[0].Properties["feature_ref"] != "F-001" {
		t.Fatalf("wrong relationship returned: %v", rels[0].Properties)
	}
}

// TestSoftDeleteByScopeLeavesOtherLevels is the core partition guarantee, checked in SQL.
func TestSoftDeleteByScopeLeavesOtherLevels(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)
	ctx := context.Background()

	filter := write.ScopeFilter{
		DomainID:   testDomainID,
		GraphScope: graphScope,
		Levels:     []write.ScopeLevel{{Level: "feature", FeatureRef: "F-001"}},
	}
	deletedRels, err := repo.SoftDeleteRelationshipsByScope(ctx, filter)
	if err != nil {
		t.Fatalf("SoftDeleteRelationshipsByScope() error = %v", err)
	}
	deletedNodes, err := repo.SoftDeleteNodesByScope(ctx, filter, time.Now().UTC())
	if err != nil {
		t.Fatalf("SoftDeleteNodesByScope() error = %v", err)
	}
	if len(deletedNodes) != 2 {
		t.Fatalf("deleted nodes = %d, want 2", len(deletedNodes))
	}
	if len(deletedRels) != 1 {
		t.Fatalf("deleted relationships = %d, want 1", len(deletedRels))
	}

	remaining, _, err := repo.ListNodesByScope(ctx, scopeQuery())
	if err != nil {
		t.Fatalf("ListNodesByScope() after delete: %v", err)
	}
	if len(remaining) != 3 {
		t.Fatalf("remaining nodes = %d (%v), want 3 (2 product + F-002)", len(remaining), idsOf(remaining))
	}
	var otherScopeCount int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM kg_nodes
		WHERE domain_id = $1 AND NOT is_deleted AND properties ->> '_kg_graph_scope' = $2
	`, testDomainID, otherScope).Scan(&otherScopeCount); err != nil {
		t.Fatalf("count other scope: %v", err)
	}
	if otherScopeCount != 1 {
		t.Fatalf("other scope count = %d, want 1 — a scoped delete must not cross graph scopes", otherScopeCount)
	}
}

// TestSoftDeleteRelationshipsByExternalRefs covers the delta delete, including the idempotent
// treatment of a reference that does not resolve.
func TestSoftDeleteRelationshipsByExternalRefs(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)
	ctx := context.Background()

	target := "bas/pgscope/e/22222222-0000-0000-0000-00000000000b"
	deleted, err := repo.SoftDeleteRelationshipsByExternalRefs(ctx, []string{target, "bas/pgscope/e/never-existed"})
	if err != nil {
		t.Fatalf("SoftDeleteRelationshipsByExternalRefs() error = %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("deleted = %d, want 1 (the unknown ref must be ignored)", len(deleted))
	}
	if deleted[0].ExternalRef != target {
		t.Fatalf("deleted the wrong relationship: %s", deleted[0].ExternalRef)
	}

	again, err := repo.SoftDeleteRelationshipsByExternalRefs(ctx, []string{target})
	if err != nil {
		t.Fatalf("second delete error = %v", err)
	}
	if len(again) != 0 {
		t.Fatalf("second delete removed %d rows, want 0 — the call must be idempotent", len(again))
	}
}

// TestRelationshipExternalRefUpsertInPostgres proves the partial unique index and the ON CONFLICT
// clause agree: re-writing the same reference updates one row rather than raising a duplicate-key
// error or inserting a second edge.
func TestRelationshipExternalRefUpsertInPostgres(t *testing.T) {
	repo, db := newRepo(t)
	seed(t, repo, db)
	ctx := context.Background()

	existing, ok := repo.GetRelationshipByExternalRef("bas/pgscope/e/22222222-0000-0000-0000-00000000000a")
	if !ok {
		t.Fatal("seeded relationship not found by external ref")
	}
	existing.Properties["note"] = "rewritten"
	if err := repo.CreateRelationshipsBulkWithOutbox(ctx, []write.RelationshipRecord{existing}, nil); err != nil {
		t.Fatalf("re-upsert: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM kg_relationships WHERE external_ref = $1 AND NOT is_deleted
	`, existing.ExternalRef).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("live rows for external_ref = %d, want 1", count)
	}
	reloaded, _ := repo.GetRelationshipByExternalRef(existing.ExternalRef)
	if reloaded.Properties["note"] != "rewritten" {
		t.Fatalf("properties = %v, want the rewrite to have won", reloaded.Properties)
	}
	if reloaded.ID != existing.ID {
		t.Fatalf("row identity changed: %s != %s", reloaded.ID, existing.ID)
	}
}

// TestArchiveGraphVersionsInPostgres exercises the window function and the manifest prune.
func TestArchiveGraphVersionsInPostgres(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()

	identifierID := "33333333-0000-0000-0000-000000000001"
	if _, err := db.ExecContext(ctx, `
		INSERT INTO kg_graph_identifiers (identifier_id, owner_tenant_id, graph_scope, head_version_number)
		VALUES ($1, $2, $3, 5)
		ON CONFLICT (identifier_id) DO NOTHING
	`, identifierID, testTenantID, "bas:kg:archive-doc"); err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM kg_graph_identifiers WHERE identifier_id = $1`, identifierID)
	})

	old := time.Now().UTC().Add(-1000 * time.Hour)
	versionIDs := make([]string, 0, 5)
	for i := 1; i <= 5; i++ {
		versionID := fmt.Sprintf("44444444-0000-0000-0000-00000000000%d", i)
		versionIDs = append(versionIDs, versionID)
		if _, err := db.ExecContext(ctx, `
			INSERT INTO kg_graph_versions (version_id, identifier_id, version_number, reference_id, storage_class, version_status, created_at, sealed_at)
			VALUES ($1, $2, $3, $4, 'ONLINE', 'SEALED', $5, $5)
		`, versionID, identifierID, i, fmt.Sprintf("ref-%d", i), old); err != nil {
			t.Fatalf("seed version %d: %v", i, err)
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO kg_graph_version_entities (version_id, entity_kind, entity_id, change_kind)
			VALUES ($1, 'node', $2, 'UPSERT')
		`, versionID, fmt.Sprintf("55555555-0000-0000-0000-00000000000%d", i)); err != nil {
			t.Fatalf("seed manifest %d: %v", i, err)
		}
	}
	if _, err := db.ExecContext(ctx, `
		UPDATE kg_graph_identifiers SET head_version_id = $1 WHERE identifier_id = $2
	`, versionIDs[4], identifierID); err != nil {
		t.Fatalf("set head: %v", err)
	}

	archived, err := repo.ArchiveGraphVersions(ctx, 2, time.Now().UTC())
	if err != nil {
		t.Fatalf("ArchiveGraphVersions() error = %v", err)
	}
	// Versions 5 and 4 are inside the keep-count (5 is also head), leaving 3, 2 and 1.
	if len(archived) != 3 {
		t.Fatalf("archived = %d (%v), want 3", len(archived), archived)
	}

	var manifestRows int
	if err := db.QueryRowContext(ctx, `
		SELECT count(*) FROM kg_graph_version_entities WHERE version_id = ANY($1::uuid[])
	`, "{"+strings.Join(archived, ",")+"}").Scan(&manifestRows); err != nil {
		t.Fatalf("count manifests: %v", err)
	}
	if manifestRows != 0 {
		t.Fatalf("archived versions kept %d manifest rows, want 0", manifestRows)
	}

	var headClass string
	if err := db.QueryRowContext(ctx, `SELECT storage_class FROM kg_graph_versions WHERE version_id = $1`, versionIDs[4]).Scan(&headClass); err != nil {
		t.Fatalf("read head: %v", err)
	}
	if headClass != "ONLINE" {
		t.Fatalf("head storage_class = %s, want ONLINE", headClass)
	}
}
