package workers

import (
	"context"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/session"
	"kg-service/internal/platform/vectorstore"
	"kg-service/internal/write"
)

type recordingSessionManager struct{}

func (recordingSessionManager) Within(ctx context.Context, identity session.WriteIdentity, fn func(session.SessionScope) error) (session.SessionScope, error) {
	scope := session.SessionScope{
		Identity: identity,
		Statements: []string{
			"BEGIN",
			"SET LOCAL app.tenant_id = '" + identity.TenantID + "'",
			"SET LOCAL app.app_id = '" + identity.AppID + "'",
			"COMMIT",
		},
		Transactional: true,
	}
	return scope, fn(scope)
}

func TestRuntimeProjectsNodeRelationshipAndCascade(t *testing.T) {
	fixture := newWorkerFixture(t)
	store, ontologySvc, cache := fixture.store, fixture.ontologySvc, fixture.cache
	runtime := NewRuntime(store, ontologySvc, &cache)

	report := runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected processed events")
	}
	if _, ok := runtime.Graph().Nodes[fixture.parentID]; !ok {
		t.Fatal("projected graph node missing")
	}
	if _, ok := runtime.Vector().Documents[fixture.childID]; !ok {
		t.Fatal("projected vector doc missing")
	}
	if got := runtime.Graph().Nodes[fixture.childID].StatusValue; got != "con_hieu_luc" {
		t.Fatalf("child status = %q, want con_hieu_luc", got)
	}
}

func TestRuntimeSyncsFullTextSearchIndex(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	recorder := &recordingFTSAdapter{}
	runtime.ftsAdapter = recorder

	runtime.PollOnce()
	if recorder.indexCount == 0 {
		t.Fatal("expected FTS index calls during projection")
	}
	if recorder.deleteCount != 0 {
		t.Fatalf("delete_count = %d, want 0", recorder.deleteCount)
	}

	if _, err := fixture.writeSvc.DeleteNode(fixture.actor, fixture.childID); err != nil {
		t.Fatalf("DeleteNode() error = %v", err)
	}
	runtime.PollOnce()
	if recorder.deleteCount == 0 {
		t.Fatal("expected FTS delete call during node deletion")
	}
}

func TestRuntimePollOnceIsIdempotentForSeenEvents(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)

	first := runtime.PollOnce()
	second := runtime.PollOnce()
	if first.Processed == 0 {
		t.Fatal("first poll expected processed events")
	}
	if second.Processed != 0 || second.Failed != 0 || second.DeadLetter != 0 {
		t.Fatalf("second poll = %+v, want zero work for seen events", second)
	}
	events := fixture.store.ListOutboxEvents()
	for _, event := range events {
		if event.Status != string(EventDone) {
			t.Fatalf("event %s status = %s, want done", event.ID, event.Status)
		}
		if event.ProcessedAt == nil {
			t.Fatalf("event %s missing processed_at", event.ID)
		}
	}

	restarted := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	restartedReport := restarted.PollOnce()
	if restartedReport.Processed != 0 || restartedReport.Failed != 0 || restartedReport.DeadLetter != 0 {
		t.Fatalf("restarted poll = %+v, want zero work", restartedReport)
	}
}

func TestRuntimeGeneratesNonEmptyEmbeddingsFromSearchableContent(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	runtime.PollOnce()

	doc, ok := runtime.Vector().Documents[fixture.parentID]
	if !ok {
		t.Fatal("projected vector doc missing")
	}
	if len(doc.Embedding) == 0 {
		t.Fatal("embedding is empty")
	}
	allZero := true
	for _, value := range doc.Embedding {
		if value != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatalf("embedding = %#v, want non-zero values", doc.Embedding)
	}
}

func TestRuntimeRetriesAndDeadLettersTransientFailures(t *testing.T) {
	runtime := NewRuntime(&failingStore{
		events: []write.OutboxEvent{
			{ID: "evt-1", EventType: "NODE_UPSERTED", Payload: map[string]any{"node_id": "missing"}},
		},
	}, &noopOntology{}, nil)

	for i := 0; i < 3; i++ {
		runtime.PollOnce()
	}
	env, ok := runtime.EventEnvelope("evt-1")
	if !ok {
		t.Fatal("event envelope missing")
	}
	if env.Status != EventDeadLetter {
		t.Fatalf("status = %s, want dead letter", env.Status)
	}
	if env.RetryCount != 3 {
		t.Fatalf("retry_count = %d, want 3", env.RetryCount)
	}
	events := runtime.store.ListOutboxEvents()
	if len(events) != 1 || events[0].Status != string(EventDeadLetter) {
		t.Fatalf("persisted events = %#v, want dead letter", events)
	}
}

func TestRuntimeAccessGrantFanoutInvalidatesACLAndExpandsPayload(t *testing.T) {
	fixture := newWorkerFixture(t)
	store, ontologySvc, cache := fixture.store, fixture.ontologySvc, fixture.cache
	runtime := NewRuntime(store, ontologySvc, &cache)
	runtime.PollOnce()
	if err := cache.SetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", map[string]any{"cached": true}, time.Minute); err != nil {
		t.Fatalf("SetJSON() error = %v", err)
	}

	err := runtime.handleAccessGrantChanged(map[string]any{
		"grantor_tenant_id": "11111111-1111-1111-1111-111111111111",
		"grantee_tenant_id": "22222222-2222-2222-2222-222222222222",
		"grantee_app_id":    "22222222-aaaa-2222-aaaa-222222222222",
		"scope_type":        "domain",
		"scope_value":       "test-domain",
		"permission":        "read",
	})
	if err != nil {
		t.Fatalf("handleAccessGrantChanged() error = %v", err)
	}

	node := runtime.Graph().Nodes[fixture.parentID]
	if !containsString(node.ACLVisibleTo, "22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222") {
		t.Fatalf("ACLVisibleTo = %#v", node.ACLVisibleTo)
	}
	var cached any
	if ok, err := cache.GetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", &cached); err != nil || ok {
		t.Fatalf("expected cache invalidation, ok=%v err=%v", ok, err)
	}

	if err := cache.SetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", map[string]any{"cached": true}, time.Minute); err != nil {
		t.Fatalf("SetJSON() revoke setup error = %v", err)
	}
	if err := runtime.handleAccessGrantChanged(map[string]any{
		"grantor_tenant_id": "11111111-1111-1111-1111-111111111111",
		"grantee_tenant_id": "22222222-2222-2222-2222-222222222222",
		"grantee_app_id":    "22222222-aaaa-2222-aaaa-222222222222",
		"scope_type":        "domain",
		"scope_value":       "test-domain",
		"permission":        "read",
		"status":            "revoked",
	}); err != nil {
		t.Fatalf("handleAccessGrantChanged(revoked) error = %v", err)
	}
	if containsString(runtime.Graph().Nodes[fixture.parentID].ACLVisibleTo, "22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222") {
		t.Fatalf("parent ACLVisibleTo still contains revoked token: %#v", runtime.Graph().Nodes[fixture.parentID].ACLVisibleTo)
	}
	if containsString(runtime.Vector().Documents[fixture.childID].ACLVisibleTo, "22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222") {
		t.Fatalf("child vector ACLVisibleTo still contains revoked token: %#v", runtime.Vector().Documents[fixture.childID].ACLVisibleTo)
	}
	if ok, _ := cache.GetJSON("acl:22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222", &cached); ok {
		t.Fatal("expected cache invalidation after revoke")
	}
}

func TestRuntimeReconcileReportsHealthyState(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	runtime.PollOnce()

	if _, err := fixture.writeSvc.UpdateNode(fixture.actor, fixture.childID, write.NodeUpdateRequest{Properties: map[string]any{"tinh_trang": "con_hieu_luc"}}); err != nil {
		t.Fatalf("UpdateNode(child) error = %v", err)
	}
	runtime.PollOnce()

	report := runtime.Reconcile()
	t.Logf("report=%+v", report)
	if report.Overall != "pass" {
		t.Fatalf("overall = %q, want pass", report.Overall)
	}
	if report.GraphDriftCount != 0 {
		t.Fatalf("graph_drift_count = %d, want 0", report.GraphDriftCount)
	}
	if report.VectorDriftCount != 0 {
		t.Fatalf("vector_drift_count = %d, want 0", report.VectorDriftCount)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("issues = %#v, want none", report.Issues)
	}
}

func TestReconcile_InFlightLagNotDrift(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		nodes: []write.NodeRecord{
			{
				ID:            "node-1",
				NodeType:      "Doc",
				DomainID:      "d1",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				ACLVisibleTo:  []string{"tenant-a:app-a"},
				Properties:    map[string]any{"title": "alpha"},
				DomainVersion: 5,
			},
		},
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", SourceUpdatedAt: now.Add(-10 * time.Second)},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now.Add(-10 * time.Second), RetryCount: 0},
		},
	}
	runtime := &Runtime{
		store: store,
		graphAdapter: staleVersionGraphAdapter{
			nodes: []graphstore.GraphNode{{ID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, Properties: map[string]any{"title": "alpha"}, SyncVersion: 4}},
			rels:  []graphstore.GraphRelationship{},
		},
		vectorAdapter: staleVersionVectorAdapter{
			docs: []vectorstore.VectorDocument{{NodeID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, DomainProps: map[string]any{"title": "alpha"}, SyncVersion: 5}},
		},
		graph:              &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector:             &VectorStore{Documents: map[string]VectorDocument{}},
		lagToleranceWindow: 30 * time.Second,
		maxRetries:         3,
	}

	report := runtime.Reconcile()
	if report.Overall != "pass" {
		t.Fatalf("overall = %q, want pass", report.Overall)
	}
	if report.GraphDriftCount != 0 {
		t.Fatalf("graph_drift_count = %d, want 0", report.GraphDriftCount)
	}
	if report.GraphInFlightCount != 1 {
		t.Fatalf("graph_in_flight_count = %d, want 1", report.GraphInFlightCount)
	}
}

func TestReconcile_LaggingRaisesWarn(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		nodes: []write.NodeRecord{
			{
				ID:            "node-1",
				NodeType:      "Doc",
				DomainID:      "d1",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				ACLVisibleTo:  []string{"tenant-a:app-a"},
				Properties:    map[string]any{"title": "alpha"},
				DomainVersion: 5,
			},
		},
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", SourceUpdatedAt: now.Add(-1 * time.Minute)},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now.Add(-1 * time.Minute), RetryCount: 1},
		},
	}
	runtime := &Runtime{
		store: store,
		graphAdapter: staleVersionGraphAdapter{
			nodes: []graphstore.GraphNode{{ID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, Properties: map[string]any{"title": "alpha"}, SyncVersion: 4}},
			rels:  []graphstore.GraphRelationship{},
		},
		vectorAdapter: staleVersionVectorAdapter{
			docs: []vectorstore.VectorDocument{{NodeID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, DomainProps: map[string]any{"title": "alpha"}, SyncVersion: 5}},
		},
		graph:              &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector:             &VectorStore{Documents: map[string]VectorDocument{}},
		lagToleranceWindow: 30 * time.Second,
		maxRetries:         3,
	}

	report := runtime.Reconcile()
	if report.Overall != "warn" {
		t.Fatalf("overall = %q, want warn", report.Overall)
	}
	if report.GraphLaggingCount != 1 {
		t.Fatalf("graph_lagging_count = %d, want 1", report.GraphLaggingCount)
	}
	if len(report.Issues) == 0 || report.Issues[0].Kind != "graph_lag_lagging" {
		t.Fatalf("issues = %#v, want graph_lag_lagging", report.Issues)
	}
}

func TestReconcile_StuckRetryExhausted(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		nodes: []write.NodeRecord{
			{
				ID:            "node-1",
				NodeType:      "Doc",
				DomainID:      "d1",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				ACLVisibleTo:  []string{"tenant-a:app-a"},
				Properties:    map[string]any{"title": "alpha"},
				DomainVersion: 5,
			},
		},
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", SourceUpdatedAt: now.Add(-1 * time.Minute)},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now.Add(-1 * time.Minute), RetryCount: 3},
		},
	}
	runtime := &Runtime{
		store: store,
		graphAdapter: staleVersionGraphAdapter{
			nodes: []graphstore.GraphNode{{ID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, Properties: map[string]any{"title": "alpha"}, SyncVersion: 4}},
			rels:  []graphstore.GraphRelationship{},
		},
		vectorAdapter: staleVersionVectorAdapter{
			docs: []vectorstore.VectorDocument{{NodeID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, DomainProps: map[string]any{"title": "alpha"}, SyncVersion: 5}},
		},
		graph:              &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector:             &VectorStore{Documents: map[string]VectorDocument{}},
		lagToleranceWindow: 30 * time.Second,
		maxRetries:         3,
	}

	report := runtime.Reconcile()
	if report.Overall != "fail" {
		t.Fatalf("overall = %q, want fail", report.Overall)
	}
	if report.GraphDriftCount == 0 {
		t.Fatal("graph_drift_count = 0, want drift")
	}
	if !containsIssueKind(report.Issues, "graph_lag_stuck") {
		t.Fatalf("issues = %#v, want graph_lag_stuck", report.Issues)
	}
}

func TestReconcile_GraphSyncedVectorLagging(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		nodes: []write.NodeRecord{
			{
				ID:            "node-1",
				NodeType:      "Doc",
				DomainID:      "d1",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				ACLVisibleTo:  []string{"tenant-a:app-a"},
				Properties:    map[string]any{"title": "alpha"},
				DomainVersion: 5,
			},
		},
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", SourceUpdatedAt: now.Add(-1 * time.Minute), LastGraphSyncedAt: now.Add(-1 * time.Second), LastVectorSyncedAt: now.Add(-1 * time.Minute)},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now.Add(-1 * time.Minute), RetryCount: 1},
		},
	}
	runtime := &Runtime{
		store: store,
		graphAdapter: staleVersionGraphAdapter{
			nodes: []graphstore.GraphNode{{ID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, Properties: map[string]any{"title": "alpha"}, SyncVersion: 5}},
			rels:  []graphstore.GraphRelationship{},
		},
		vectorAdapter: staleVersionVectorAdapter{
			docs: []vectorstore.VectorDocument{{NodeID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, DomainProps: map[string]any{"title": "alpha"}, SyncVersion: 4}},
		},
		graph:              &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector:             &VectorStore{Documents: map[string]VectorDocument{}},
		lagToleranceWindow: 30 * time.Second,
		maxRetries:         3,
	}

	report := runtime.Reconcile()
	if report.GraphDriftCount != 0 {
		t.Fatalf("graph_drift_count = %d, want 0", report.GraphDriftCount)
	}
	if report.VectorLaggingCount != 1 {
		t.Fatalf("vector_lagging_count = %d, want 1", report.VectorLaggingCount)
	}
}

func TestEntitySyncStatus_Synced(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", LastGraphSyncedAt: now, LastVectorSyncedAt: now, GraphVersion: 5, VectorVersion: 5},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now},
		},
	}
	runtime := &Runtime{store: store, lagToleranceWindow: 30 * time.Second, maxRetries: 3}

	status := runtime.EntitySyncStatus("node-1", "kg_node")
	if status.GraphLagClass != SyncLagClassSynced || status.VectorLagClass != SyncLagClassSynced {
		t.Fatalf("status = %#v, want synced", status)
	}
}

func TestEntitySyncStatus_InFlight(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", LastGraphSyncedAt: now, GraphVersion: 4, VectorVersion: 5, LastVectorSyncedAt: now},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now.Add(-10 * time.Second)},
		},
	}
	runtime := &Runtime{store: store, lagToleranceWindow: 30 * time.Second, maxRetries: 3}

	status := runtime.EntitySyncStatus("node-1", "kg_node")
	if status.GraphLagClass != SyncLagClassInFlight {
		t.Fatalf("status = %#v, want in-flight", status)
	}
}
func TestRuntimeReconcileReportsReplicaDrift(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	runtime.PollOnce()

	parent := runtime.Graph().Nodes[fixture.parentID]
	parent.NodeType = "Mismatch"
	runtime.Graph().Nodes[fixture.parentID] = parent
	if err := runtime.vectorAdapter.Delete(context.Background(), fixture.childID); err != nil {
		t.Fatalf("vectorAdapter.Delete() error = %v", err)
	}
	if err := runtime.vectorAdapter.Upsert(context.Background(), vectorstore.VectorDocument{NodeID: "orphan-vector-node", NodeType: "Orphan", DomainID: "test-domain"}); err != nil {
		t.Fatalf("vectorAdapter.Upsert() error = %v", err)
	}
	runtime.Graph().Nodes["orphan-graph-node"] = GraphNode{ID: "orphan-graph-node", NodeType: "Orphan", DomainID: "test-domain"}
	for id := range runtime.Graph().Rels {
		delete(runtime.Graph().Rels, id)
		break
	}

	report := runtime.Reconcile()
	if report.Overall != "fail" {
		t.Fatalf("overall = %q, want fail", report.Overall)
	}
	if report.GraphDriftCount == 0 {
		t.Fatal("graph_drift_count = 0, want drift")
	}
	if report.VectorDriftCount == 0 {
		t.Fatal("vector_drift_count = 0, want drift")
	}
	for _, kind := range []string{"graph_mismatch", "orphan_graph_node", "orphan_vector_doc"} {
		if !containsIssueKind(report.Issues, kind) {
			t.Fatalf("missing issue kind %q in %#v", kind, report.Issues)
		}
	}
}

func TestRuntimeReconcileReportsStaleProjectionVersion(t *testing.T) {
	store := &versionFixtureStore{
		nodes: []write.NodeRecord{
			{
				ID:            "node-1",
				NodeType:      "Doc",
				DomainID:      "d1",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				ACLVisibleTo:  []string{"tenant-a:app-a"},
				Properties:    map[string]any{"title": "alpha"},
				DomainVersion: 42,
			},
		},
		rels: []write.RelationshipRecord{
			{
				ID:            "rel-1",
				RelType:       "LINKS",
				FromNodeID:    "node-1",
				ToNodeID:      "node-1",
				DomainID:      "d1",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				DomainVersion: 42,
			},
		},
		projectionVersions: []write.ProjectionVersionRecord{
			{
				EntityID:      "node-1",
				EntityKind:    "kg_node",
				SourceVersion: 42,
				SourceEventID: "evt-node-1",
			},
			{
				EntityID:      "orphan-ledger-node",
				EntityKind:    "kg_node",
				SourceVersion: 11,
				SourceEventID: "evt-orphan",
			},
		},
	}
	runtime := &Runtime{
		store: store,
		graphAdapter: staleVersionGraphAdapter{
			nodes: []graphstore.GraphNode{{ID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, Properties: map[string]any{"title": "alpha"}, SyncVersion: 41}},
			rels:  []graphstore.GraphRelationship{{ID: "rel-1", RelType: "LINKS", FromNodeID: "node-1", ToNodeID: "node-1", DomainID: "d1", SyncVersion: 41}},
		},
		vectorAdapter: staleVersionVectorAdapter{
			docs: []vectorstore.VectorDocument{{NodeID: "node-1", NodeType: "Doc", DomainID: "d1", OwnerTenantID: "tenant-a", OwnerAppID: "app-a", ACLVisibleTo: []string{"tenant-a:app-a"}, DomainProps: map[string]any{"title": "alpha"}, SyncVersion: 41}},
		},
		graph:  &GraphStore{Nodes: map[string]GraphNode{}, Rels: map[string]GraphRelationship{}},
		vector: &VectorStore{Documents: map[string]VectorDocument{}},
	}

	report := runtime.Reconcile()
	if report.Overall != "fail" {
		t.Fatalf("overall = %q, want fail", report.Overall)
	}
	if !containsIssueKind(report.Issues, "graph_lag_stuck") || !containsIssueKind(report.Issues, "vector_lag_stuck") {
		t.Fatalf("issues = %#v, want graph_lag_stuck and vector_lag_stuck", report.Issues)
	}
	if report.ProjectionVersionDriftCount == 0 {
		t.Fatal("projection_version_drift_count = 0, want ledger drift")
	}
	if !containsIssueKind(report.Issues, "orphan_projection_version") {
		t.Fatalf("issues = %#v, want orphan_projection_version", report.Issues)
	}
}

type noopOntology struct{}

func (noopOntology) GetStatusFieldConfig(domainID string) (*ontology.StatusFieldConfig, error) {
	return nil, nil
}

type failingStore struct {
	events []write.OutboxEvent
}

func (f *failingStore) ListOutboxEvents() []write.OutboxEvent { return f.events }
func (f *failingStore) GetOutboxEventByID(id string) (write.OutboxEvent, bool) {
	for _, event := range f.events {
		if event.ID == id {
			return event, true
		}
	}
	return write.OutboxEvent{}, false
}
func (f *failingStore) GetNodeByID(id string) (write.NodeRecord, bool) {
	return write.NodeRecord{}, false
}
func (f *failingStore) GetRelationshipByID(id string) (write.RelationshipRecord, bool) {
	return write.RelationshipRecord{}, false
}
func (f *failingStore) ListNodes() []write.NodeRecord                 { return nil }
func (f *failingStore) ListRelationships() []write.RelationshipRecord { return nil }
func (f *failingStore) GetProjectionVersion(entityID, entityKind string) (write.ProjectionVersionRecord, bool) {
	return write.ProjectionVersionRecord{}, false
}
func (f *failingStore) ListProjectionVersions() []write.ProjectionVersionRecord { return nil }
func (f *failingStore) UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error {
	for i := range f.events {
		if f.events[i].ID != eventID {
			continue
		}
		f.events[i].Status = status
		f.events[i].RetryCount = retryCount
		f.events[i].ProcessedAt = processedAt
		return nil
	}
	return nil
}

type versionFixtureStore struct {
	nodes              []write.NodeRecord
	rels               []write.RelationshipRecord
	projectionVersions []write.ProjectionVersionRecord
	outboxEvents       []write.OutboxEvent
}

func (s *versionFixtureStore) ListOutboxEvents() []write.OutboxEvent {
	return append([]write.OutboxEvent(nil), s.outboxEvents...)
}
func (s *versionFixtureStore) GetOutboxEventByID(id string) (write.OutboxEvent, bool) {
	for _, event := range s.outboxEvents {
		if event.ID == id {
			return event, true
		}
	}
	return write.OutboxEvent{}, false
}
func (s *versionFixtureStore) GetNodeByID(id string) (write.NodeRecord, bool) {
	for _, node := range s.nodes {
		if node.ID == id {
			return node, true
		}
	}
	return write.NodeRecord{}, false
}
func (s *versionFixtureStore) GetRelationshipByID(id string) (write.RelationshipRecord, bool) {
	for _, rel := range s.rels {
		if rel.ID == id {
			return rel, true
		}
	}
	return write.RelationshipRecord{}, false
}
func (s *versionFixtureStore) ListNodes() []write.NodeRecord {
	return append([]write.NodeRecord(nil), s.nodes...)
}
func (s *versionFixtureStore) ListRelationships() []write.RelationshipRecord {
	return append([]write.RelationshipRecord(nil), s.rels...)
}
func (s *versionFixtureStore) GetProjectionVersion(entityID, entityKind string) (write.ProjectionVersionRecord, bool) {
	for _, record := range s.projectionVersions {
		if record.EntityID == entityID && record.EntityKind == entityKind {
			return record, true
		}
	}
	return write.ProjectionVersionRecord{}, false
}
func (s *versionFixtureStore) ListProjectionVersions() []write.ProjectionVersionRecord {
	return append([]write.ProjectionVersionRecord(nil), s.projectionVersions...)
}
func (s *versionFixtureStore) UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error {
	return nil
}

type staleVersionGraphAdapter struct {
	nodes []graphstore.GraphNode
	rels  []graphstore.GraphRelationship
}

func (a staleVersionGraphAdapter) UpsertNode(ctx context.Context, node graphstore.GraphNode) error {
	return nil
}
func (a staleVersionGraphAdapter) DeleteNode(ctx context.Context, nodeID string) error { return nil }
func (a staleVersionGraphAdapter) UpsertRelationship(ctx context.Context, rel graphstore.GraphRelationship) error {
	return nil
}
func (a staleVersionGraphAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	return nil
}
func (a staleVersionGraphAdapter) ExecuteQuery(ctx context.Context, query graphstore.GraphQuery, params map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (a staleVersionGraphAdapter) ListNodes(ctx context.Context) ([]graphstore.GraphNode, error) {
	return append([]graphstore.GraphNode(nil), a.nodes...), nil
}
func (a staleVersionGraphAdapter) ListRelationships(ctx context.Context) ([]graphstore.GraphRelationship, error) {
	return append([]graphstore.GraphRelationship(nil), a.rels...), nil
}
func (a staleVersionGraphAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	return 41, nil
}

type staleVersionVectorAdapter struct {
	docs []vectorstore.VectorDocument
}

func (a staleVersionVectorAdapter) Upsert(ctx context.Context, doc vectorstore.VectorDocument) error {
	return nil
}
func (a staleVersionVectorAdapter) Delete(ctx context.Context, nodeID string) error { return nil }
func (a staleVersionVectorAdapter) ANN(ctx context.Context, query []float64, filter vectorstore.VectorFilter, opts vectorstore.ANNOptions) ([]vectorstore.VectorResult, error) {
	return nil, nil
}
func (a staleVersionVectorAdapter) Snapshot(ctx context.Context) ([]vectorstore.VectorDocument, error) {
	return append([]vectorstore.VectorDocument(nil), a.docs...), nil
}
func (a staleVersionVectorAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	return 41, nil
}

type workerFixture struct {
	store       *write.MemoryStore
	ontologySvc *ontology.Service
	writeSvc    *write.Service
	cache       rediscache.Client
	actor       access.Identity
	parentID    string
	childID     string
}

func newWorkerFixture(t *testing.T) workerFixture {
	t.Helper()
	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}
	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologySvc := ontology.NewService(ontologyStore, accessResolver)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	if _, err := ontologySvc.CreateDomain(actor, actor.TenantID, ontology.DomainCreateRequest{ID: "test-domain", Name: "Test Domain"}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "test-domain", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "Parent",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(parent) error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "test-domain", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "Child",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(child) error = %v", err)
	}
	if _, err := ontologySvc.CreateRelType(actor, actor.TenantID, "test-domain", ontology.RelTypeCreateRequest{
		RelTypeName:  "PARENT_OF",
		FromNodeType: "Parent",
		ToNodeType:   "Child",
		SameDomain:   true,
	}); err != nil && !containsText(err.Error(), "already exists") {
		t.Fatalf("CreateRelType() error = %v", err)
	}
	if _, err := ontologySvc.UpsertStatusFieldConfig(actor, actor.TenantID, "test-domain", ontology.StatusFieldConfigRequest{
		StatusFieldName:   "tinh_trang",
		ValidStatusValues: []string{"con_hieu_luc"},
		CascadeRules: []ontology.CascadeRule{
			{FromNodeType: "Parent", ViaRel: "PARENT_OF", ToNodeType: "Child"},
		},
	}); err != nil {
		t.Fatalf("UpsertStatusFieldConfig() error = %v", err)
	}

	store := write.NewMemoryStore()
	writeSvc := write.NewService(store, ontologySvc, accessResolver, &recordingSessionManager{}, nil)
	parent, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "test-domain",
		NodeType:   "Parent",
		Properties: map[string]any{"ten": "Cha", "tinh_trang": "con_hieu_luc"},
	})
	if err != nil {
		t.Fatalf("CreateNode(parent) error = %v", err)
	}
	child, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "test-domain",
		NodeType:   "Child",
		Properties: map[string]any{"ten": "Con", "tinh_trang": "khac"},
	})
	if err != nil {
		t.Fatalf("CreateNode(child) error = %v", err)
	}
	if _, err := writeSvc.CreateRelationship(actor, write.RelationshipCreateRequest{
		RelType:    "PARENT_OF",
		FromNodeID: parent.NodeID,
		ToNodeID:   child.NodeID,
		DomainID:   "test-domain",
		Properties: map[string]any{},
	}); err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}

	return workerFixture{
		store:       store,
		ontologySvc: ontologySvc,
		writeSvc:    writeSvc,
		cache:       cache,
		actor:       actor,
		parentID:    parent.NodeID,
		childID:     child.NodeID,
	}
}

func containsText(haystack, needle string) bool { return strings.Contains(haystack, needle) }

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsIssueKind(issues []ReconciliationIssue, want string) bool {
	for _, issue := range issues {
		if issue.Kind == want {
			return true
		}
	}
	return false
}

type recordingFTSAdapter struct {
	indexCount  int
	deleteCount int
	lastIndex   fts.FTSDocument
}

func (r *recordingFTSAdapter) Index(_ context.Context, doc fts.FTSDocument) error {
	r.indexCount++
	r.lastIndex = doc
	return nil
}

func (r *recordingFTSAdapter) Delete(_ context.Context, nodeID string) error {
	r.deleteCount++
	return nil
}

func (r *recordingFTSAdapter) Search(_ context.Context, query fts.FTSQuery, filter fts.FTSFilter, opts fts.SearchOptions) ([]fts.FTSResult, error) {
	return nil, nil
}
