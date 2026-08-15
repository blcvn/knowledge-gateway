package write

import (
	"context"
	"strings"
	"testing"

	"kg-service/internal/access"
)

// scopeFixture builds a partitioned graph in one scope: a product slice plus two feature slices,
// each with a node and an edge. This is the shape a client that partitions its graph produces, and
// the shape every scope-delete assertion below is measured against.
type scopeFixture struct {
	svc        *Service
	store      *MemoryStore
	actor      access.Identity
	graphScope string
	sessionID  string
	nodeIDs    map[string]string // label -> node id
	relIDs     map[string]string // label -> relationship id
}

func newScopeFixture(t *testing.T) *scopeFixture {
	t.Helper()
	svc, store := newTestService(t)
	actor := adminActor()
	f := &scopeFixture{
		svc:        svc,
		store:      store,
		actor:      actor,
		graphScope: "bas:kg:doc-scope",
		nodeIDs:    map[string]string{},
		relIDs:     map[string]string{},
	}

	type nodeSpec struct {
		label      string
		nodeType   string
		level      string
		featureRef string
	}
	specs := []nodeSpec{
		{"product-a", "HopDongMau", "product", ""},
		{"product-b", "KhoanMau", "product", ""},
		{"f1-a", "KhoanMau", "feature", "F-001"},
		{"f1-b", "KhoanMau", "feature", "F-001"},
		{"f2-a", "KhoanMau", "feature", "F-002"},
	}
	for _, spec := range specs {
		props := map[string]any{
			"ten":             "Node " + spec.label,
			"_kg_graph_scope": f.graphScope,
			"kg_level":        spec.level,
			"feature_ref":     spec.featureRef,
			"document_id":     "doc-scope",
		}
		if spec.nodeType == "HopDongMau" {
			props["bridge_dinh_kem_ids"] = []any{"appendix-1"}
		}
		created, err := svc.CreateNode(actor, NodeCreateRequest{
			DomainID:    "noi_bo_hop_dong",
			NodeType:    spec.nodeType,
			Properties:  props,
			ExternalRef: "bas/doc-scope/n/" + spec.label,
		})
		if err != nil {
			t.Fatalf("CreateNode(%s) error = %v", spec.label, err)
		}
		f.nodeIDs[spec.label] = created.NodeID
	}

	type relSpec struct {
		label      string
		from, to   string
		level      string
		featureRef string
	}
	relSpecs := []relSpec{
		{"product-edge", "product-a", "product-b", "product", ""},
		{"f1-edge", "product-a", "f1-a", "feature", "F-001"},
		{"f2-edge", "product-a", "f2-a", "feature", "F-002"},
	}
	for _, spec := range relSpecs {
		created, err := svc.CreateRelationship(actor, RelationshipCreateRequest{
			RelType:    "THAM_CHIEU",
			FromNodeID: f.nodeIDs[spec.from],
			ToNodeID:   f.nodeIDs[spec.to],
			DomainID:   "noi_bo_hop_dong",
			Properties: map[string]any{
				"_kg_graph_scope": f.graphScope,
				"kg_level":        spec.level,
				"feature_ref":     spec.featureRef,
			},
			ExternalRef: "bas/doc-scope/e/" + spec.label,
		})
		if err != nil {
			t.Fatalf("CreateRelationship(%s) error = %v", spec.label, err)
		}
		f.relIDs[spec.label] = created.RelationshipID
	}

	session, err := svc.OpenSyncSession(context.Background(), actor, OpenSyncSessionRequest{
		DomainID:   "noi_bo_hop_dong",
		GraphScope: f.graphScope,
	})
	if err != nil {
		t.Fatalf("OpenSyncSession() error = %v", err)
	}
	f.sessionID = session.GraphVersionID
	return f
}

func (f *scopeFixture) liveNode(label string) bool {
	node, ok := f.store.GetNodeByID(f.nodeIDs[label])
	return ok && !node.IsDeleted
}

func (f *scopeFixture) liveRelationship(label string) bool {
	rel, ok := f.store.GetRelationshipByID(f.relIDs[label])
	return ok && !rel.IsDeleted
}

// TestDeleteByScopeFeatureLeavesProductAndSiblings is the central guarantee for a partitioned
// client: rewriting one feature must not touch the shared level or any other feature.
func TestDeleteByScopeFeatureLeavesProductAndSiblings(t *testing.T) {
	f := newScopeFixture(t)

	resp, err := f.svc.DeleteByScopeWithVersion(context.Background(), f.actor, ScopeDeleteRequest{
		ScopeFilter: ScopeFilter{
			DomainID:   "noi_bo_hop_dong",
			GraphScope: f.graphScope,
			Levels:     []ScopeLevel{{Level: "feature", FeatureRef: "F-001"}},
		},
		GraphVersionID: f.sessionID,
	})
	if err != nil {
		t.Fatalf("DeleteByScopeWithVersion() error = %v", err)
	}

	if len(resp.NodeIDs) != 2 {
		t.Fatalf("deleted nodes = %d, want 2 (f1-a, f1-b)", len(resp.NodeIDs))
	}
	if len(resp.RelationshipIDs) != 1 {
		t.Fatalf("deleted relationships = %d, want 1 (f1-edge)", len(resp.RelationshipIDs))
	}
	for _, label := range []string{"product-a", "product-b", "f2-a"} {
		if !f.liveNode(label) {
			t.Errorf("node %s was deleted but is out of scope", label)
		}
	}
	for _, label := range []string{"f1-a", "f1-b"} {
		if f.liveNode(label) {
			t.Errorf("node %s should have been deleted", label)
		}
	}
	if !f.liveRelationship("product-edge") || !f.liveRelationship("f2-edge") {
		t.Error("out-of-scope relationships were deleted")
	}
	if f.liveRelationship("f1-edge") {
		t.Error("f1-edge should have been deleted")
	}
}

// TestDeleteByScopeProductLeavesFeatures covers the other direction: rewriting the shared level
// must not cascade into the feature slices by itself.
func TestDeleteByScopeProductLeavesFeatures(t *testing.T) {
	f := newScopeFixture(t)

	if _, err := f.svc.DeleteByScopeWithVersion(context.Background(), f.actor, ScopeDeleteRequest{
		ScopeFilter: ScopeFilter{
			DomainID:   "noi_bo_hop_dong",
			GraphScope: f.graphScope,
			Levels:     []ScopeLevel{{Level: "product"}},
		},
		GraphVersionID: f.sessionID,
	}); err != nil {
		t.Fatalf("DeleteByScopeWithVersion() error = %v", err)
	}

	for _, label := range []string{"f1-a", "f1-b", "f2-a"} {
		if !f.liveNode(label) {
			t.Errorf("feature node %s was deleted by a product-scope delete", label)
		}
	}
	if f.liveNode("product-a") || f.liveNode("product-b") {
		t.Error("product nodes should have been deleted")
	}
}

// TestDeleteByScopeEmptyLevelsRemovesWholeScope pins the meaning of an empty level list: the whole
// graph, not "nothing".
func TestDeleteByScopeEmptyLevelsRemovesWholeScope(t *testing.T) {
	f := newScopeFixture(t)

	resp, err := f.svc.DeleteByScopeWithVersion(context.Background(), f.actor, ScopeDeleteRequest{
		ScopeFilter: ScopeFilter{
			DomainID:   "noi_bo_hop_dong",
			GraphScope: f.graphScope,
		},
		GraphVersionID: f.sessionID,
	})
	if err != nil {
		t.Fatalf("DeleteByScopeWithVersion() error = %v", err)
	}
	if len(resp.NodeIDs) != 5 {
		t.Fatalf("deleted nodes = %d, want 5", len(resp.NodeIDs))
	}
	for label := range f.nodeIDs {
		if f.liveNode(label) {
			t.Errorf("node %s survived a whole-scope delete", label)
		}
	}
}

// TestDeleteByScopeRejectsForeignScope keeps a session from mutating a graph it holds no lease on.
func TestDeleteByScopeRejectsForeignScope(t *testing.T) {
	f := newScopeFixture(t)

	_, err := f.svc.DeleteByScopeWithVersion(context.Background(), f.actor, ScopeDeleteRequest{
		ScopeFilter: ScopeFilter{
			DomainID:   "noi_bo_hop_dong",
			GraphScope: "bas:kg:some-other-doc",
		},
		GraphVersionID: f.sessionID,
	})
	if err == nil {
		t.Fatal("DeleteByScopeWithVersion() error = nil, want scope mismatch")
	}
	if !strings.Contains(err.Error(), "graph scope mismatch") {
		t.Fatalf("error = %v, want graph scope mismatch", err)
	}
	for label := range f.nodeIDs {
		if !f.liveNode(label) {
			t.Errorf("node %s was deleted despite the request being rejected", label)
		}
	}
}

// TestDeleteByScopeRequiresGraphVersion refuses an unversioned scope-wide delete: without a version
// there would be no manifest of what was removed.
func TestDeleteByScopeRequiresGraphVersion(t *testing.T) {
	f := newScopeFixture(t)

	_, err := f.svc.DeleteByScopeWithVersion(context.Background(), f.actor, ScopeDeleteRequest{
		ScopeFilter: ScopeFilter{
			DomainID:   "noi_bo_hop_dong",
			GraphScope: f.graphScope,
		},
	})
	if err == nil {
		t.Fatal("DeleteByScopeWithVersion() error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), "graph_version_id is required") {
		t.Fatalf("error = %v, want graph_version_id required", err)
	}
}

// TestDeleteByScopeRecordsVersionEntities proves the delete is auditable: every removed row is
// recorded against the open version as a DELETE.
func TestDeleteByScopeRecordsVersionEntities(t *testing.T) {
	f := newScopeFixture(t)

	resp, err := f.svc.DeleteByScopeWithVersion(context.Background(), f.actor, ScopeDeleteRequest{
		ScopeFilter: ScopeFilter{
			DomainID:   "noi_bo_hop_dong",
			GraphScope: f.graphScope,
			Levels:     []ScopeLevel{{Level: "feature", FeatureRef: "F-001"}},
		},
		GraphVersionID: f.sessionID,
	})
	if err != nil {
		t.Fatalf("DeleteByScopeWithVersion() error = %v", err)
	}

	entities := f.store.GetGraphVersionEntities(f.sessionID)
	recorded := map[string]string{}
	for _, entity := range entities {
		recorded[entity.EntityID] = entity.ChangeKind
	}
	for _, id := range append(append([]string{}, resp.NodeIDs...), resp.RelationshipIDs...) {
		if recorded[id] != "DELETE" {
			t.Errorf("entity %s recorded as %q, want DELETE", id, recorded[id])
		}
	}
}

// TestDeleteRelationshipsByExternalRefRemovesNamedEdges covers the delta delete a scoped snapshot
// performs after upserting the desired content.
func TestDeleteRelationshipsByExternalRefRemovesNamedEdges(t *testing.T) {
	f := newScopeFixture(t)

	resp, err := f.svc.DeleteRelationshipsByExternalRefWithVersion(context.Background(), f.actor, RelationshipDeleteByExternalRefRequest{
		ExternalRefs:   []string{"bas/doc-scope/e/f1-edge"},
		GraphVersionID: f.sessionID,
	})
	if err != nil {
		t.Fatalf("DeleteRelationshipsByExternalRefWithVersion() error = %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("count = %d, want 1", resp.Count)
	}
	if f.liveRelationship("f1-edge") {
		t.Error("f1-edge should have been deleted")
	}
	if !f.liveRelationship("product-edge") || !f.liveRelationship("f2-edge") {
		t.Error("unnamed relationships were deleted")
	}
}

// TestDeleteRelationshipsByExternalRefIgnoresUnknownRefs keeps the call idempotent: asking for an
// edge that is already gone is a satisfied request, not an error.
func TestDeleteRelationshipsByExternalRefIgnoresUnknownRefs(t *testing.T) {
	f := newScopeFixture(t)

	resp, err := f.svc.DeleteRelationshipsByExternalRefWithVersion(context.Background(), f.actor, RelationshipDeleteByExternalRefRequest{
		ExternalRefs:   []string{"bas/doc-scope/e/f1-edge", "bas/doc-scope/e/never-existed"},
		GraphVersionID: f.sessionID,
	})
	if err != nil {
		t.Fatalf("DeleteRelationshipsByExternalRefWithVersion() error = %v", err)
	}
	if resp.Count != 1 {
		t.Fatalf("count = %d, want 1 (the unknown ref must be ignored, not fatal)", resp.Count)
	}
	if len(resp.ExternalRefs) != 1 || resp.ExternalRefs[0] != "bas/doc-scope/e/f1-edge" {
		t.Fatalf("external_refs = %v, want only the ref that resolved", resp.ExternalRefs)
	}
}

// TestDeleteRelationshipsByExternalRefRequiresRefs rejects an empty request rather than silently
// succeeding, which would hide a caller bug that computed an empty delta by mistake.
func TestDeleteRelationshipsByExternalRefRequiresRefs(t *testing.T) {
	f := newScopeFixture(t)

	_, err := f.svc.DeleteRelationshipsByExternalRefWithVersion(context.Background(), f.actor, RelationshipDeleteByExternalRefRequest{
		ExternalRefs:   []string{"  ", ""},
		GraphVersionID: f.sessionID,
	})
	if err == nil {
		t.Fatal("error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), "external_refs is required") {
		t.Fatalf("error = %v, want external_refs required", err)
	}
}

// TestScopeFilterMatches locks the level-matching rule itself, which both stores and the SQL
// builder rely on.
func TestScopeFilterMatches(t *testing.T) {
	product := map[string]any{"kg_level": "product"}
	feature1 := map[string]any{"kg_level": "feature", "feature_ref": "F-001"}
	feature2 := map[string]any{"kg_level": "feature", "feature_ref": "F-002"}

	cases := []struct {
		name       string
		filter     ScopeFilter
		properties map[string]any
		want       bool
	}{
		{"empty levels matches product", ScopeFilter{}, product, true},
		{"empty levels matches feature", ScopeFilter{}, feature1, true},
		{"product filter matches product", ScopeFilter{Levels: []ScopeLevel{{Level: "product"}}}, product, true},
		{"product filter rejects feature", ScopeFilter{Levels: []ScopeLevel{{Level: "product"}}}, feature1, false},
		{"feature filter matches its ref", ScopeFilter{Levels: []ScopeLevel{{Level: "feature", FeatureRef: "F-001"}}}, feature1, true},
		{"feature filter rejects other ref", ScopeFilter{Levels: []ScopeLevel{{Level: "feature", FeatureRef: "F-001"}}}, feature2, false},
		{"feature filter without ref matches any feature", ScopeFilter{Levels: []ScopeLevel{{Level: "feature"}}}, feature2, true},
		{"union matches product", ScopeFilter{Levels: []ScopeLevel{{Level: "product"}, {Level: "feature", FeatureRef: "F-001"}}}, product, true},
		{"union matches the named feature", ScopeFilter{Levels: []ScopeLevel{{Level: "product"}, {Level: "feature", FeatureRef: "F-001"}}}, feature1, true},
		{"union rejects the other feature", ScopeFilter{Levels: []ScopeLevel{{Level: "product"}, {Level: "feature", FeatureRef: "F-001"}}}, feature2, false},
		{"missing level property matches nothing specific", ScopeFilter{Levels: []ScopeLevel{{Level: "product"}}}, map[string]any{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.filter.Matches(tc.properties); got != tc.want {
				t.Fatalf("Matches() = %v, want %v", got, tc.want)
			}
		})
	}
}
