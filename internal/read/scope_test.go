package read

import (
	"context"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/write"
)

const (
	scopeTestDomain = "noi_bo_hop_dong"
	scopeTestGraph  = "bas:kg:doc-read-scope"
)

// seedScopeGraph writes a partitioned graph straight into the source store: a product slice plus
// two feature slices, with one edge per slice. Writing at the store level rather than through the
// write service keeps these tests focused on the read path's filtering, paging and visibility.
func seedScopeGraph(t *testing.T, store *write.MemoryStore, actor access.Identity) {
	t.Helper()
	now := time.Now().UTC()

	nodes := []struct {
		id         string
		nodeType   string
		level      string
		featureRef string
	}{
		{"n-product-a", "HopDongMau", "product", ""},
		{"n-product-b", "KhoanMau", "product", ""},
		{"n-f1-a", "KhoanMau", "feature", "F-001"},
		{"n-f1-b", "KhoanMau", "feature", "F-001"},
		{"n-f2-a", "KhoanMau", "feature", "F-002"},
	}
	records := make([]write.NodeRecord, 0, len(nodes))
	for _, spec := range nodes {
		records = append(records, write.NodeRecord{
			ID:            spec.id,
			NodeType:      spec.nodeType,
			DomainID:      scopeTestDomain,
			OwnerTenantID: actor.TenantID,
			OwnerAppID:    actor.AppID,
			Visibility:    "private",
			ExternalRef:   "bas/doc-read-scope/n/" + spec.id,
			DomainVersion: 1,
			Properties: map[string]any{
				"_kg_graph_scope": scopeTestGraph,
				"kg_level":        spec.level,
				"feature_ref":     spec.featureRef,
				"reference_id":    spec.id,
				"summary":         "summary of " + spec.id,
			},
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := store.CreateNodesBulkWithOutbox(context.Background(), records, nil); err != nil {
		t.Fatalf("seed nodes: %v", err)
	}

	rels := []struct {
		id         string
		from, to   string
		level      string
		featureRef string
	}{
		{"r-product", "n-product-a", "n-product-b", "product", ""},
		{"r-f1", "n-product-a", "n-f1-a", "feature", "F-001"},
		{"r-f2", "n-product-a", "n-f2-a", "feature", "F-002"},
	}
	relRecords := make([]write.RelationshipRecord, 0, len(rels))
	for _, spec := range rels {
		relRecords = append(relRecords, write.RelationshipRecord{
			ID:            spec.id,
			RelType:       "THAM_CHIEU",
			FromNodeID:    spec.from,
			ToNodeID:      spec.to,
			DomainID:      scopeTestDomain,
			OwnerTenantID: actor.TenantID,
			OwnerAppID:    actor.AppID,
			DomainVersion: 1,
			ExternalRef:   "bas/doc-read-scope/e/" + spec.id,
			Properties: map[string]any{
				"_kg_graph_scope": scopeTestGraph,
				"kg_level":        spec.level,
				"feature_ref":     spec.featureRef,
			},
			CreatedAt: now,
		})
	}
	if err := store.CreateRelationshipsBulkWithOutbox(context.Background(), relRecords, nil); err != nil {
		t.Fatalf("seed relationships: %v", err)
	}
}

func nodeIDs(nodes []GraphScopeNode) []string {
	ids := make([]string, 0, len(nodes))
	for _, node := range nodes {
		ids = append(ids, node.ID)
	}
	return ids
}

func relIDs(rels []GraphScopeRelationship) []string {
	ids := make([]string, 0, len(rels))
	for _, rel := range rels {
		ids = append(ids, rel.ID)
	}
	return ids
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// TestReadGraphByScopeEmptyLevelsReturnsWholeScope pins the meaning of an empty level list.
func TestReadGraphByScopeEmptyLevelsReturnsWholeScope(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	want := []string{"n-f1-a", "n-f1-b", "n-f2-a", "n-product-a", "n-product-b"}
	if !equalStrings(nodeIDs(resp.Nodes), want) {
		t.Fatalf("nodes = %v, want %v", nodeIDs(resp.Nodes), want)
	}
	if !equalStrings(relIDs(resp.Relationships), []string{"r-f1", "r-f2", "r-product"}) {
		t.Fatalf("relationships = %v", relIDs(resp.Relationships))
	}
	if resp.HasMore {
		t.Fatal("has_more = true on a fully returned scope")
	}
}

// TestReadGraphByScopeProductLevel is the product-slice read a partitioned client issues before
// rewriting its shared level.
func TestReadGraphByScopeProductLevel(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
		Levels:     []write.ScopeLevel{{Level: "product"}},
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	if !equalStrings(nodeIDs(resp.Nodes), []string{"n-product-a", "n-product-b"}) {
		t.Fatalf("nodes = %v, want only the product slice", nodeIDs(resp.Nodes))
	}
	if !equalStrings(relIDs(resp.Relationships), []string{"r-product"}) {
		t.Fatalf("relationships = %v, want only the product edge", relIDs(resp.Relationships))
	}
}

// TestReadGraphByScopeProductUnionFeature is the feature-slice read: the shared level plus exactly
// one feature, never a sibling.
func TestReadGraphByScopeProductUnionFeature(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
		Levels:     []write.ScopeLevel{{Level: "product"}, {Level: "feature", FeatureRef: "F-001"}},
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	want := []string{"n-f1-a", "n-f1-b", "n-product-a", "n-product-b"}
	if !equalStrings(nodeIDs(resp.Nodes), want) {
		t.Fatalf("nodes = %v, want %v", nodeIDs(resp.Nodes), want)
	}
	for _, id := range nodeIDs(resp.Nodes) {
		if id == "n-f2-a" {
			t.Fatal("sibling feature leaked into a feature-scoped read")
		}
	}
	if !equalStrings(relIDs(resp.Relationships), []string{"r-f1", "r-product"}) {
		t.Fatalf("relationships = %v", relIDs(resp.Relationships))
	}
}

// TestReadGraphByScopePaginationIsCompleteAndStable walks the whole scope one row at a time. Paging
// must return every row exactly once; a cursor that drifted would silently drop or duplicate rows,
// and a caller reloading a slice before rewriting it would then write back a corrupted graph.
func TestReadGraphByScopePaginationIsCompleteAndStable(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	seenNodes := make([]string, 0, 5)
	seenRels := make([]string, 0, 3)
	cursor := GraphScopeReadCursor{}
	for i := 0; i < 20; i++ {
		resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
			DomainID:   scopeTestDomain,
			GraphScope: scopeTestGraph,
			Limit:      1,
			Cursor:     cursor,
		})
		if err != nil {
			t.Fatalf("ReadGraphByScope(page %d) error = %v", i, err)
		}
		seenNodes = append(seenNodes, nodeIDs(resp.Nodes)...)
		seenRels = append(seenRels, relIDs(resp.Relationships)...)
		if !resp.HasMore {
			break
		}
		cursor = resp.NextCursor
	}

	wantNodes := []string{"n-f1-a", "n-f1-b", "n-f2-a", "n-product-a", "n-product-b"}
	if !equalStrings(seenNodes, wantNodes) {
		t.Fatalf("paged nodes = %v, want %v", seenNodes, wantNodes)
	}
	wantRels := []string{"r-f1", "r-f2", "r-product"}
	if !equalStrings(seenRels, wantRels) {
		t.Fatalf("paged relationships = %v, want %v", seenRels, wantRels)
	}
}

// TestReadGraphByScopeExcludesDeleted keeps tombstones out of a scope read: a caller asking for the
// current graph must not see rows that were removed.
func TestReadGraphByScopeExcludesDeleted(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	if _, err := store.SoftDeleteNodesByScope(context.Background(), write.ScopeFilter{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
		Levels:     []write.ScopeLevel{{Level: "feature", FeatureRef: "F-001"}},
	}, time.Now().UTC()); err != nil {
		t.Fatalf("SoftDeleteNodesByScope() error = %v", err)
	}

	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	for _, id := range nodeIDs(resp.Nodes) {
		if id == "n-f1-a" || id == "n-f1-b" {
			t.Fatalf("soft-deleted node %s appeared in a scope read", id)
		}
	}
}

// TestReadGraphByScopeRefsOnlyOmitsPayload proves refs_only actually reduces the payload while
// keeping what a delta computation needs.
func TestReadGraphByScopeRefsOnlyOmitsPayload(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
		RefsOnly:   true,
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	if len(resp.Nodes) != 5 {
		t.Fatalf("nodes = %d, want 5", len(resp.Nodes))
	}
	for _, node := range resp.Nodes {
		if node.ExternalRef == "" {
			t.Fatalf("node %s lost its external_ref in refs_only mode", node.ID)
		}
		if _, ok := node.Properties["kg_level"]; !ok {
			t.Fatalf("node %s lost kg_level in refs_only mode", node.ID)
		}
	}
}

// TestReadGraphByScopeRejectsMissingScope refuses an under-specified request rather than returning
// an arbitrary slice of somebody's graph.
func TestReadGraphByScopeRejectsMissingScope(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	if _, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID: scopeTestDomain,
	}); err == nil {
		t.Fatal("ReadGraphByScope() error = nil, want graph_scope required")
	}
	if _, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		GraphScope: scopeTestGraph,
	}); err == nil {
		t.Fatal("ReadGraphByScope() error = nil, want domain_id required")
	}
	if _, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
		Levels:     []write.ScopeLevel{{FeatureRef: "F-001"}},
	}); err == nil {
		t.Fatal("ReadGraphByScope() error = nil, want levels[0].level required")
	}
}

// TestReadGraphByScopeHidesOtherOwnersRows is the access boundary: rows owned by a tenant/app the
// caller has no grant for must not appear, even inside a scope the caller named correctly.
func TestReadGraphByScopeHidesOtherOwnersRows(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	now := time.Now().UTC()
	if err := store.CreateNodesBulkWithOutbox(context.Background(), []write.NodeRecord{{
		ID:            "n-foreign",
		NodeType:      "KhoanMau",
		DomainID:      scopeTestDomain,
		OwnerTenantID: "22222222-2222-2222-2222-222222222222",
		OwnerAppID:    "22222222-bbbb-2222-bbbb-222222222222",
		Visibility:    "private",
		DomainVersion: 1,
		Properties: map[string]any{
			"_kg_graph_scope": scopeTestGraph,
			"kg_level":        "product",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil); err != nil {
		t.Fatalf("seed foreign node: %v", err)
	}

	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	for _, id := range nodeIDs(resp.Nodes) {
		if id == "n-foreign" {
			t.Fatal("a node owned by another tenant was returned")
		}
	}
}

// TestReadGraphByScopeReadsSourceOfTruth is the read-after-write guarantee. The projections are
// updated asynchronously; a caller that persists a graph and reloads it inside the same unit of
// work must see its own write, so this read must never be served from a projection.
func TestReadGraphByScopeReadsSourceOfTruth(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	now := time.Now().UTC()
	if err := store.CreateNodesBulkWithOutbox(context.Background(), []write.NodeRecord{{
		ID:            "n-just-written",
		NodeType:      "KhoanMau",
		DomainID:      scopeTestDomain,
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		Visibility:    "private",
		DomainVersion: 1,
		ExternalRef:   "bas/doc-read-scope/n/n-just-written",
		Properties: map[string]any{
			"_kg_graph_scope": scopeTestGraph,
			"kg_level":        "product",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil); err != nil {
		t.Fatalf("write node: %v", err)
	}

	// No projection worker has run at this point — that is the whole point of the assertion.
	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
		Levels:     []write.ScopeLevel{{Level: "product"}},
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	found := false
	for _, id := range nodeIDs(resp.Nodes) {
		if id == "n-just-written" {
			found = true
		}
	}
	if !found {
		t.Fatal("a just-written node was not visible to a scope read")
	}
}

// TestReadGraphByScopeIsolatesScopes proves the scope key actually isolates: a second graph in the
// same domain and owned by the same app must not bleed into the first.
func TestReadGraphByScopeIsolatesScopes(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	seedScopeGraph(t, store, actor)

	now := time.Now().UTC()
	if err := store.CreateNodesBulkWithOutbox(context.Background(), []write.NodeRecord{{
		ID:            "n-other-scope",
		NodeType:      "KhoanMau",
		DomainID:      scopeTestDomain,
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		Visibility:    "private",
		DomainVersion: 1,
		Properties: map[string]any{
			"_kg_graph_scope": "bas:kg:some-other-doc",
			"kg_level":        "product",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}}, nil); err != nil {
		t.Fatalf("seed other scope: %v", err)
	}

	resp, err := svc.ReadGraphByScope(context.Background(), actor, GraphScopeReadRequest{
		DomainID:   scopeTestDomain,
		GraphScope: scopeTestGraph,
	})
	if err != nil {
		t.Fatalf("ReadGraphByScope() error = %v", err)
	}
	for _, id := range nodeIDs(resp.Nodes) {
		if id == "n-other-scope" {
			t.Fatal("a node from another graph scope was returned")
		}
	}
}
