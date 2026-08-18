// Package pglookup_test covers looking entities up by their primary key, against real Postgres.
//
// This is the second behaviour the in-memory store cannot vouch for, and it cost more than the
// first. MemoryStore keys its maps by string, so GetNodesByIDs there is a map lookup that cannot
// fail. In Postgres, kg_nodes.id is a uuid column and the query compared it to a text array —
// `operator does not exist: uuid = text` on every single call. The function has no error in its
// signature and reported failure as an empty map, so the projection worker concluded that none of
// the nodes in a sealed version existed and marked every one of them already-projected.
//
// The result was a service that looked healthy from every angle available to it: writes committed,
// projection events completed, projection heads advanced, no errors logged. And no vector document,
// no graph node and no full-text row was ever written by an ordinary write. Semantic search only
// ever returned anything because a repair/rebuild had been run by hand, which reaches nodes through
// a different query.
//
// A unit test could not have caught this, and neither could an end-to-end test that ran a rebuild
// first. It needs the real column types and a real write.
//
// Opt-in: set KG_TEST_POSTGRES_DSN to a database with the full migration set applied. See
// tests/pgmigrate for the container recipe.
package pglookup_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"kg-service/internal/platform/postgres"
	"kg-service/internal/write"
)

const (
	testDomainID = "pglookup_domain"
	testTenantID = "11111111-1111-1111-1111-111111111111"
	testAppID    = "11111111-1111-4111-8111-111111111111"
	testScope    = "bas:kg:pglookup-doc"
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

func seed(t *testing.T, repo *postgres.Repository, db *sql.DB) (nodeIDs []string, relID string) {
	t.Helper()
	ctx := context.Background()
	cleanup(t, db)
	t.Cleanup(func() { cleanup(t, db) })

	if _, err := db.ExecContext(ctx, `
		INSERT INTO domains (id, name, owner_tenant_id, status, version, visibility)
		VALUES ($1, 'pglookup domain', $2, 'active', 1, 'private')
		ON CONFLICT (id) DO NOTHING
	`, testDomainID, testTenantID); err != nil {
		t.Fatalf("seed domain: %v", err)
	}

	now := time.Now().UTC()
	for i, ref := range []string{"bas/pglookup-doc/n/A", "bas/pglookup-doc/n/B"} {
		node := write.NodeRecord{
			ID:            fmt.Sprintf("00000000-0000-4000-8000-00000000000%d", i+1),
			NodeType:      "FEATURE",
			DomainID:      testDomainID,
			OwnerTenantID: testTenantID,
			OwnerAppID:    testAppID,
			ACLVisibleTo:  []string{testTenantID + ":" + testAppID},
			Visibility:    "private",
			Properties:    map[string]any{"_kg_graph_scope": testScope, "summary": "pglookup"},
			DomainVersion: 1,
			ExternalRef:   ref,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		event := write.OutboxEvent{
			ID:            fmt.Sprintf("00000000-0000-4000-8000-0000000000a%d", i+1),
			AggregateType: "kg_node",
			AggregateID:   node.ID,
			EventType:     "NODE_UPSERTED",
			Payload:       map[string]any{"node_id": node.ID},
			Status:        "PENDING",
			CreatedAt:     now,
		}
		if err := repo.CreateNodeWithOutbox(ctx, node, event); err != nil {
			t.Fatalf("seed node %s: %v", ref, err)
		}
		nodeIDs = append(nodeIDs, node.ID)
	}

	relID = "00000000-0000-4000-8000-0000000000b1"
	rel := write.RelationshipRecord{
		ID:            relID,
		RelType:       "CONTAINS",
		FromNodeID:    nodeIDs[0],
		ToNodeID:      nodeIDs[1],
		DomainID:      testDomainID,
		OwnerTenantID: testTenantID,
		OwnerAppID:    testAppID,
		DomainVersion: 1,
		Properties:    map[string]any{"_kg_graph_scope": testScope},
		ExternalRef:   "bas/pglookup-doc/e/A-B",
		CreatedAt:     now,
	}
	relEvent := write.OutboxEvent{
		ID:            "00000000-0000-4000-8000-0000000000c1",
		AggregateType: "kg_relationship",
		AggregateID:   relID,
		EventType:     "RELATIONSHIP_UPSERTED",
		Payload:       map[string]any{"relationship_id": relID},
		Status:        "PENDING",
		CreatedAt:     now,
	}
	if err := repo.CreateNodeBundle(ctx, write.NodeRecord{
		ID:            "00000000-0000-4000-8000-0000000000d1",
		NodeType:      "FEATURE",
		DomainID:      testDomainID,
		OwnerTenantID: testTenantID,
		OwnerAppID:    testAppID,
		ACLVisibleTo:  []string{testTenantID + ":" + testAppID},
		Visibility:    "private",
		Properties:    map[string]any{"_kg_graph_scope": testScope},
		DomainVersion: 1,
		ExternalRef:   "bas/pglookup-doc/n/C",
		CreatedAt:     now,
		UpdatedAt:     now,
	}, []write.RelationshipRecord{rel}, relEvent); err != nil {
		t.Fatalf("seed relationship: %v", err)
	}

	return nodeIDs, rel.ID
}

// TestGetNodesByIDsFindsNodesThatExist is the direct regression. Before the ::uuid[] fix this
// returned an empty map for ids that were sitting in the table, and the caller could not tell the
// difference between "these were deleted" and "this query cannot run".
func TestGetNodesByIDsFindsNodesThatExist(t *testing.T) {
	repo, db := newRepo(t)
	nodeIDs, _ := seed(t, repo, db)

	found := repo.GetNodesByIDs(nodeIDs)
	if len(found) != len(nodeIDs) {
		t.Fatalf("GetNodesByIDs() returned %d of %d nodes; an empty result here is how a broken "+
			"lookup masqueraded as an empty table", len(found), len(nodeIDs))
	}
	for _, id := range nodeIDs {
		node, ok := found[id]
		if !ok {
			t.Fatalf("node %s missing from the result", id)
		}
		if node.ID != id {
			t.Fatalf("node keyed under %s carries id %s", id, node.ID)
		}
		if node.DomainID != testDomainID {
			t.Fatalf("node %s has domain %q, want %q", id, node.DomainID, testDomainID)
		}
	}
}

func TestGetRelationshipsByIDsFindsRelationshipsThatExist(t *testing.T) {
	repo, db := newRepo(t)
	_, relID := seed(t, repo, db)

	found := repo.GetRelationshipsByIDs([]string{relID})
	if len(found) != 1 {
		t.Fatalf("GetRelationshipsByIDs() returned %d relationships, want 1", len(found))
	}
	if _, ok := found[relID]; !ok {
		t.Fatalf("relationship %s missing from the result", relID)
	}
}

// TestGetNodesByIDsIgnoresIDsThatAreNotThere keeps the fix honest in the other direction: the point
// is to find rows that exist, not to start returning rows for ids that do not.
func TestGetNodesByIDsIgnoresIDsThatAreNotThere(t *testing.T) {
	repo, db := newRepo(t)
	nodeIDs, _ := seed(t, repo, db)

	absent := "00000000-0000-4000-8000-0000000000ff"
	found := repo.GetNodesByIDs(append([]string{absent}, nodeIDs...))
	if len(found) != len(nodeIDs) {
		t.Fatalf("GetNodesByIDs() returned %d rows, want %d", len(found), len(nodeIDs))
	}
	if _, ok := found[absent]; ok {
		t.Fatalf("id %s does not exist but came back anyway", absent)
	}
}

func cleanup(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, stmt := range []string{
		`DELETE FROM kg_outbox_events WHERE aggregate_id IN (SELECT id FROM kg_nodes WHERE domain_id = $1) OR aggregate_id IN (SELECT id FROM kg_relationships WHERE domain_id = $1)`,
		`DELETE FROM kg_relationships WHERE domain_id = $1`,
		`DELETE FROM kg_nodes WHERE domain_id = $1`,
	} {
		if _, err := db.ExecContext(ctx, stmt, testDomainID); err != nil {
			t.Fatalf("cleanup %q: %v", stmt, err)
		}
	}
}
