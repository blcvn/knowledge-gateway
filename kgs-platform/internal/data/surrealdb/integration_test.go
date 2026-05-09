package surrealdb_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	sdb "github.com/blcvn/knowledge-gateway/kgs-platform/internal/data/surrealdb"

	"github.com/go-kratos/kratos/v2/log"
)

// SurrealDB integration test suite.
// Requires a running SurrealDB instance.
// Set SURREALDB_URL (default: ws://localhost:8000) to point to a test instance.
// Run: docker run --rm -p 8000:8000 surrealdb/surrealdb:v2 start --user root --pass test --log info memory

const (
	testNamespace = "kgs_test"
	testDatabase  = "integration"
	testUser      = "root"
	testPassword  = "test"
)

func surrealDBURL() string {
	url := os.Getenv("SURREALDB_URL")
	if url == "" {
		url = "ws://localhost:8000"
	}
	return url
}

func setupClient(t *testing.T) (*sdb.Client, func()) {
	t.Helper()
	logger := log.NewStdLogger(os.Stdout)

	client, cleanup, err := sdb.NewClient(surrealDBURL(), testNamespace, testDatabase, testUser, testPassword, logger)
	if err != nil {
		t.Skipf("SurrealDB not available at %s: %v", surrealDBURL(), err)
	}

	// Init schema
	if err := sdb.InitSchema(context.Background(), client, 1536, logger); err != nil {
		cleanup()
		t.Fatalf("schema init failed: %v", err)
	}

	return client, cleanup
}

// ── Test 1: RegistryRepo CRUD ─────────────────────────────────

func TestRegistryRepo(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	logger := log.NewStdLogger(os.Stdout)

	repo := sdb.NewSurrealRegistryRepo(client, logger)
	ctx := context.Background()

	// CreateApp
	appID := fmt.Sprintf("test-app-%d", time.Now().UnixNano())
	err := repo.CreateApp(ctx, &biz.App{
		AppID:       appID,
		AppName:     "Test App",
		Description: "Integration test",
		Owner:       "test",
		Status:      "ACTIVE",
	})
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}

	// GetApp
	app, err := repo.GetApp(ctx, appID)
	if err != nil {
		t.Fatalf("GetApp: %v", err)
	}
	if app.AppName != "Test App" {
		t.Errorf("expected app_name='Test App', got %q", app.AppName)
	}

	// ListApps
	apps, err := repo.ListApps(ctx)
	if err != nil {
		t.Fatalf("ListApps: %v", err)
	}
	found := false
	for _, a := range apps {
		if a.AppID == appID {
			found = true
		}
	}
	if !found {
		t.Errorf("app %s not found in ListApps (got %d apps)", appID, len(apps))
	}

	t.Logf("✅ RegistryRepo: CreateApp, GetApp, ListApps OK")
}

// ── Test 2: GraphWriteRepo with EnqueueOutbox = no-op ────────

func TestGraphWriteRepo(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	logger := log.NewStdLogger(os.Stdout)

	repo := sdb.NewSurrealGraphWriteRepo(client, logger)
	ctx := context.Background()

	entityID := fmt.Sprintf("entity-%d", time.Now().UnixNano())
	appID := "test-app"
	tenantID := "default"

	// UpsertEntity
	op, err := repo.UpsertEntity(ctx, biz.WriteEntity{
		EntityID:   entityID,
		AppID:      appID,
		TenantID:   tenantID,
		EntityType: "Customer",
		Name:       "Test Customer",
		Version:    1,
	})
	if err != nil {
		t.Fatalf("UpsertEntity: %v", err)
	}
	t.Logf("UpsertEntity op=%s", op)

	// EnqueueOutbox — should be NO-OP
	err = repo.EnqueueOutbox(ctx, biz.OutboxRecord{})
	if err != nil {
		t.Fatalf("EnqueueOutbox should be no-op, got: %v", err)
	}

	// UpsertEdge (in implicit transaction)
	edgeID := fmt.Sprintf("edge-%d", time.Now().UnixNano())
	_, err = repo.UpsertEdge(ctx, biz.WriteEdge{
		EdgeID:       edgeID,
		AppID:        appID,
		TenantID:     tenantID,
		FromEntityID: entityID,
		ToEntityID:   entityID, // self-reference for simplicity
		RelationType: "RELATES_TO",
	})
	if err != nil {
		t.Fatalf("UpsertEdge: %v", err)
	}

	t.Logf("✅ GraphWriteRepo: UpsertEntity, EnqueueOutbox(noop), UpsertEdge OK")
}

// ── Test 3: GraphRepo — Node/Edge CRUD + GetFullGraph ────────

func TestGraphRepo(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	logger := log.NewStdLogger(os.Stdout)

	repo := sdb.NewSurrealGraphRepo(client, logger)
	ctx := context.Background()

	appID := "test-graph"
	tenantID := "default"

	// CreateNode
	node1, err := repo.CreateNode(ctx, appID, tenantID, "Person", map[string]any{
		"name": "Alice",
	})
	if err != nil {
		t.Fatalf("CreateNode: %v", err)
	}
	nodeID1 := fmt.Sprint(node1["id"])
	t.Logf("Created node1: %s", nodeID1)

	// GetNode
	got, err := repo.GetNode(ctx, appID, tenantID, nodeID1)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got["label"] != "Person" {
		t.Errorf("expected label=Person, got %v", got["label"])
	}

	// CreateNode #2
	node2, err := repo.CreateNode(ctx, appID, tenantID, "Company", map[string]any{
		"name": "AcmeCorp",
	})
	if err != nil {
		t.Fatalf("CreateNode #2: %v", err)
	}
	nodeID2 := fmt.Sprint(node2["id"])

	// CreateEdge
	edge, err := repo.CreateEdge(ctx, appID, tenantID, "WORKS_AT", nodeID1, nodeID2, map[string]any{})
	if err != nil {
		t.Fatalf("CreateEdge: %v", err)
	}
	t.Logf("Created edge: %v", edge["id"])

	// GetFullGraph
	full, err := repo.GetFullGraph(ctx, appID, tenantID, 100, 0)
	if err != nil {
		t.Fatalf("GetFullGraph: %v", err)
	}
	if full.TotalNodes < 2 {
		t.Errorf("expected at least 2 nodes, got %d", full.TotalNodes)
	}
	if full.TotalEdges < 1 {
		t.Errorf("expected at least 1 edge, got %d", full.TotalEdges)
	}

	// DeleteNode (cascade edges)
	_, err = repo.DeleteNode(ctx, appID, tenantID, nodeID1)
	if err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}

	// Verify node is soft-deleted
	_, err = repo.GetNode(ctx, appID, tenantID, nodeID1)
	if err == nil {
		t.Error("expected node to be deleted, but GetNode succeeded")
	}

	t.Logf("✅ GraphRepo: CreateNode×2, GetNode, CreateEdge, GetFullGraph, DeleteNode(cascade) OK")
}

// ── Test 4: LockManager ──────────────────────────────────────

func TestLockManager(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	logger := log.NewStdLogger(os.Stdout)

	lockMgr := sdb.NewSurrealLockManager(client, logger)
	ctx := context.Background()

	ns := "test-ns"
	nodeID := "lock-node-1"

	// Acquire
	token, err := lockMgr.AcquireNodeLock(ctx, ns, nodeID, 10*time.Second)
	if err != nil {
		t.Fatalf("AcquireNodeLock: %v", err)
	}
	t.Logf("Acquired lock token=%s", token)

	// Concurrent acquire should fail
	_, err = lockMgr.AcquireNodeLock(ctx, ns, nodeID, 10*time.Second)
	if err == nil {
		t.Error("expected concurrent lock to fail, but it succeeded")
	}

	// Release
	err = lockMgr.Release(ctx, token)
	if err != nil {
		t.Fatalf("Release: %v", err)
	}

	// Re-acquire after release
	token2, err := lockMgr.AcquireNodeLock(ctx, ns, nodeID, 10*time.Second)
	if err != nil {
		t.Fatalf("Re-acquire after release: %v", err)
	}
	_ = lockMgr.Release(ctx, token2)

	t.Logf("✅ LockManager: Acquire, Contention, Release, Re-acquire OK")
}

// ── Test 5: OverlayStore ─────────────────────────────────────

func TestOverlayStore(t *testing.T) {
	client, cleanup := setupClient(t)
	defer cleanup()
	logger := log.NewStdLogger(os.Stdout)

	store := sdb.NewSurrealOverlayStore(client, logger)
	ctx := context.Background()

	overlayID := fmt.Sprintf("overlay-%d", time.Now().UnixNano())
	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	// Save
	err := store.SaveOverlay(ctx, overlayID, "test-ns", nil, nil, nil, nil, 1*time.Hour)
	if err != nil {
		t.Fatalf("SaveOverlay: %v", err)
	}

	// Get
	ov, err := store.GetOverlay(ctx, overlayID)
	if err != nil {
		t.Fatalf("GetOverlay: %v", err)
	}
	if ov["overlay_id"] != overlayID {
		t.Errorf("expected overlay_id=%s, got %v", overlayID, ov["overlay_id"])
	}

	// BindSession
	err = store.BindSession(ctx, sessionID, overlayID, 1*time.Hour)
	if err != nil {
		t.Fatalf("BindSession: %v", err)
	}

	// FindBySession
	foundID, err := store.FindBySession(ctx, sessionID)
	if err != nil {
		t.Fatalf("FindBySession: %v", err)
	}
	if foundID != overlayID {
		t.Errorf("expected overlay_id=%s from session, got %s", overlayID, foundID)
	}

	// UnbindSession + Delete
	_ = store.UnbindSession(ctx, sessionID)
	_ = store.DeleteOverlay(ctx, overlayID)

	t.Logf("✅ OverlayStore: Save, Get, BindSession, FindBySession, Delete OK")
}

// ── Test 6: QueryTranslator ──────────────────────────────────

func TestQueryTranslator(t *testing.T) {
	params := map[string]any{
		"app_id":    "test",
		"tenant_id": "default",
		"node_id":   "node-1",
	}

	tests := []struct {
		name   string
		cypher string
		ok     bool
	}{
		{
			name:   "context query",
			cypher: `MATCH (n {app_id: $app_id, tenant_id: $tenant_id})-[r]-(m) WHERE n.id = $node_id RETURN n, r, m`,
			ok:     true,
		},
		{
			name:   "impact query depth 3",
			cypher: `MATCH p=(n)-[*1..3]->(m) WHERE n.app_id = $app_id AND n.tenant_id = $tenant_id RETURN p`,
			ok:     true,
		},
		{
			name:   "coverage query depth 5",
			cypher: `MATCH p=(n)<-[*1..5]-(m) WHERE n.app_id = $app_id RETURN p`,
			ok:     true,
		},
		{
			name:   "subgraph query",
			cypher: `MATCH (n)-[r]->(m) WHERE n.id IN $node_ids AND n.app_id = $app_id RETURN n, r, m`,
			ok:     true,
		},
		{
			name:   "unsupported query",
			cypher: `CALL db.stats()`,
			ok:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			surql, err := sdb.TranslateCypherToSurrealQL(tt.cypher, params)
			if tt.ok {
				if err != nil {
					t.Errorf("expected OK, got error: %v", err)
				}
				if surql == "" {
					t.Error("expected non-empty SurrealQL")
				}
				t.Logf("Cypher → SurrealQL:\n  IN:  %s\n  OUT: %s", tt.cypher[:min(80, len(tt.cypher))], surql[:min(80, len(surql))])
			} else {
				if err == nil {
					t.Error("expected error for unsupported query, got nil")
				}
			}
		})
	}

	t.Logf("✅ QueryTranslator: 5 patterns tested")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
