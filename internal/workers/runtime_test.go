package workers

import (
	"context"
	"errors"
	"os"
	"sort"
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
	"kg-service/internal/platform/vector"
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

func TestRuntimeProjectsNodesToRealMemgraph(t *testing.T) {
	endpoint := strings.TrimSpace(os.Getenv("KG_MEMGRAPH_ENDPOINT"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("KG_GRAPH_ENDPOINT"))
	}
	if endpoint == "" {
		t.Skip("set KG_MEMGRAPH_ENDPOINT or KG_GRAPH_ENDPOINT to run Memgraph runtime integration")
	}

	fixture := newWorkerFixture(t)
	store, ontologySvc, cache := fixture.store, fixture.ontologySvc, fixture.cache
	runtime := NewRuntime(store, ontologySvc, &cache)
	adapter := graphstore.NewMemgraphGraphAdapter(graphstore.CypherConfig{
		Endpoint: endpoint,
		Database: os.Getenv("KG_GRAPH_DATABASE"),
	})
	runtime.SetGraphAdapter(adapter)

	t.Cleanup(func() {
		_ = adapter.DeleteRelationship(context.Background(), fixture.relationshipID)
		_ = adapter.DeleteNode(context.Background(), fixture.parentID)
		_ = adapter.DeleteNode(context.Background(), fixture.childID)
	})

	report := runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected processed events")
	}

	nodes, err := adapter.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if !hasGraphNodeID(nodes, fixture.parentID) {
		t.Fatalf("Memgraph missing parent node %s after PollOnce()", fixture.parentID)
	}
	if !hasGraphNodeID(nodes, fixture.childID) {
		t.Fatalf("Memgraph missing child node %s after PollOnce()", fixture.childID)
	}

	rels, err := adapter.ListRelationships(context.Background())
	if err != nil {
		t.Fatalf("ListRelationships() error = %v", err)
	}
	if !hasGraphRelationshipID(rels, fixture.relationshipID) {
		t.Fatalf("Memgraph missing relationship %s after PollOnce()", fixture.relationshipID)
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

func TestRuntimeAdvancesGraphHeadFromGraphVersionEvent(t *testing.T) {
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
				DomainVersion: 7,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
		outboxEvents: []write.OutboxEvent{
			{
				ID:            "evt-graph-1",
				AggregateType: "kg_graph_version",
				AggregateID:   "graphver-1",
				EventType:     "GRAPH_VERSION_SEALED",
				CreatedAt:     now,
				Payload: map[string]any{
					"graph_identifier_id":  "graph-1",
					"graph_version_id":     "graphver-1",
					"graph_version_number": int64(1),
					"reference_id":         "ref-1",
					"entity_ids":           []string{"node-1"},
				},
			},
		},
		graphVersionEntities: map[string][]write.GraphVersionEntityRecord{
			"graphver-1": {
				{VersionID: "graphver-1", EntityKind: "node", EntityID: "node-1", ChangeKind: "UPSERT"},
			},
		},
	}
	runtime := NewRuntime(store, &noopOntology{}, nil)

	report := runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected graph-version event to be processed")
	}
	head, ok := store.GetGraphProjectionHead("graph-1", "graph", "")
	if !ok {
		t.Fatal("graph projection head missing")
	}
	if head.AppliedVersionNumber != 1 || head.AppliedVersionID != "graphver-1" {
		t.Fatalf("graph head = %#v, want version 1", head)
	}
}

func TestRuntimeAdvancesGraphHeadForEmptyGraphVersion(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		outboxEvents: []write.OutboxEvent{
			{
				ID:            "evt-graph-empty",
				AggregateType: "kg_graph_version",
				AggregateID:   "graphver-empty",
				EventType:     "GRAPH_VERSION_SEALED",
				CreatedAt:     now,
				Payload: map[string]any{
					"graph_identifier_id":  "graph-empty",
					"graph_version_id":     "graphver-empty",
					"graph_version_number": int64(2),
					"reference_id":         "ref-empty",
					"entity_ids":           []string{},
				},
			},
		},
		graphVersionEntities: map[string][]write.GraphVersionEntityRecord{
			"graphver-empty": []write.GraphVersionEntityRecord{},
		},
	}
	runtime := NewRuntime(store, &noopOntology{}, nil)

	report := runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected empty graph-version event to be processed")
	}
	graphHead, ok := store.GetGraphProjectionHead("graph-empty", "graph", "")
	if !ok {
		t.Fatal("graph projection head missing")
	}
	if graphHead.AppliedVersionNumber != 2 || graphHead.AppliedVersionID != "graphver-empty" {
		t.Fatalf("graph head = %#v, want version 2", graphHead)
	}
	vectorHead, ok := store.GetGraphProjectionHead("graph-empty", "vector", "")
	if !ok {
		t.Fatal("vector projection head missing")
	}
	if vectorHead.AppliedVersionNumber != 2 || vectorHead.AppliedVersionID != "graphver-empty" {
		t.Fatalf("vector head = %#v, want version 2", vectorHead)
	}
}

func TestRuntimeAdvancesProjectionHeadsForNodeEvent(t *testing.T) {
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
				DomainVersion: 3,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
		outboxEvents: []write.OutboxEvent{
			{
				ID:            "evt-node-1",
				AggregateType: "kg_node",
				AggregateID:   "node-1",
				EventType:     "NODE_UPSERTED",
				CreatedAt:     now,
				Payload: map[string]any{
					"node_id":              "node-1",
					"graph_identifier_id":  "graph-1",
					"graph_version_id":     "graphver-3",
					"graph_version_number": int64(3),
				},
			},
		},
	}
	runtime := NewRuntime(store, &noopOntology{}, nil)

	report := runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected node event to be processed")
	}
	graphHead, ok := store.GetGraphProjectionHead("graph-1", "graph", "")
	if !ok {
		t.Fatal("graph projection head missing")
	}
	if graphHead.AppliedVersionNumber != 3 || graphHead.AppliedVersionID != "graphver-3" {
		t.Fatalf("graph head = %#v, want version 3", graphHead)
	}
	vectorHead, ok := store.GetGraphProjectionHead("graph-1", "vector", "")
	if !ok {
		t.Fatal("vector projection head missing")
	}
	if vectorHead.AppliedVersionNumber != 3 || vectorHead.AppliedVersionID != "graphver-3" {
		t.Fatalf("vector head = %#v, want version 3", vectorHead)
	}
}

func TestRuntimeHandlesGraphVersionSealedWithBatchEmbedding(t *testing.T) {
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
				DomainVersion: 7,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
			{
				ID:            "node-2",
				NodeType:      "Doc",
				DomainID:      "d1",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				ACLVisibleTo:  []string{"tenant-a:app-a"},
				Properties:    map[string]any{"title": "beta"},
				DomainVersion: 3,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		},
		outboxEvents: []write.OutboxEvent{
			{
				ID:            "evt-graph-batch",
				AggregateType: "kg_graph_version",
				AggregateID:   "graphver-batch",
				EventType:     "GRAPH_VERSION_SEALED",
				CreatedAt:     now,
				Payload: map[string]any{
					"graph_identifier_id":  "graph-batch",
					"graph_version_id":     "graphver-batch",
					"graph_version_number": int64(9),
					"reference_id":         "ref-batch",
					"entity_ids":           []string{"node-1", "node-2"},
				},
			},
		},
		graphVersionEntities: map[string][]write.GraphVersionEntityRecord{
			"graphver-batch": {
				{VersionID: "graphver-batch", EntityKind: "node", EntityID: "node-1", ChangeKind: "UPSERT"},
				{VersionID: "graphver-batch", EntityKind: "node", EntityID: "node-2", ChangeKind: "UPSERT"},
			},
		},
	}
	runtime := NewRuntime(store, &noopOntology{}, nil)
	provider := &batchCountingProvider{dimensions: 4, modelID: "batch-model"}
	runtime.SetEmbeddingRouter(vector.DirectRouter{Provider: provider})

	report := runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected graph-version event to be processed")
	}
	if provider.batchCalls == 0 {
		t.Fatal("expected EmbedBatch to be used")
	}
	if provider.singleCalls != 0 {
		t.Fatalf("single embed calls = %d, want 0", provider.singleCalls)
	}
	if len(runtime.Graph().Nodes) != 2 {
		t.Fatalf("graph nodes = %d, want 2", len(runtime.Graph().Nodes))
	}
	if len(runtime.Vector().Documents) != 2 {
		t.Fatalf("vector docs = %d, want 2", len(runtime.Vector().Documents))
	}
	head, ok := store.GetGraphProjectionHead("graph-batch", "graph", "")
	if !ok {
		t.Fatal("graph projection head missing")
	}
	if head.AppliedVersionNumber != 9 || head.AppliedVersionID != "graphver-batch" {
		t.Fatalf("graph head = %#v, want version 9", head)
	}
	vectorHead, ok := store.GetGraphProjectionHead("graph-batch", "vector", "")
	if !ok {
		t.Fatal("vector projection head missing")
	}
	if vectorHead.AppliedVersionNumber != 9 || vectorHead.AppliedVersionID != "graphver-batch" {
		t.Fatalf("vector head = %#v, want version 9", vectorHead)
	}
}

func TestRuntimeCleansExpiredSyncSessions(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		graphVersions: []write.GraphVersionRecord{
			{
				VersionID:     "graphver-old",
				IdentifierID:  "graph-old",
				VersionNumber: 4,
				VersionStatus: "PENDING_ENTITIES",
				CreatedAt:     now.Add(-3 * time.Hour),
			},
		},
		graphIdentities: map[string]write.GraphIdentityRecord{
			"graph-old": {
				IdentifierID:  "graph-old",
				OwnerTenantID: "tenant-a",
				OwnerAppID:    "app-a",
				GraphScope:    "project:test",
			},
		},
		scopeLeases: map[string]struct{}{
			"tenant-a:app-a:project:test": {},
		},
	}
	runtime := NewRuntime(store, &noopOntology{}, nil)

	cleaned := runtime.cleanupExpiredSyncSessions(context.Background())
	if cleaned != 1 {
		t.Fatalf("cleaned = %d, want 1", cleaned)
	}
	if got := store.graphVersions[0].VersionStatus; got != "ABANDONED" {
		t.Fatalf("version status = %s, want ABANDONED", got)
	}
	if len(store.scopeLeases) != 0 {
		t.Fatalf("scope leases = %#v, want empty", store.scopeLeases)
	}
}

func TestRuntimeCleanupExpiredSyncSessionsSkipsFailedCleanup(t *testing.T) {
	store := &cleanupErrorStore{
		versionFixtureStore: versionFixtureStore{
			graphVersions: []write.GraphVersionRecord{
				{
					VersionID:     "graphver-old",
					IdentifierID:  "graph-old",
					VersionNumber: 4,
					VersionStatus: "PENDING_ENTITIES",
					CreatedAt:     time.Now().UTC().Add(-3 * time.Hour),
				},
			},
		},
		err: errors.New("cleanup failed"),
	}
	runtime := NewRuntime(store, &noopOntology{}, nil)

	cleaned := runtime.cleanupExpiredSyncSessions(context.Background())
	if cleaned != 0 {
		t.Fatalf("cleaned = %d, want 0 when cleanup fails", cleaned)
	}
	if got := store.graphVersions[0].VersionStatus; got != "PENDING_ENTITIES" {
		t.Fatalf("version status = %s, want PENDING_ENTITIES", got)
	}
}

func TestRuntimeEndToEndSyncSessionParity(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	if report := runtime.PollOnce(); report.Processed == 0 {
		t.Fatal("expected initial fixture events to be processed")
	}

	graphScope := "domain:test-domain:tenant:" + fixture.actor.TenantID + ":app:" + fixture.actor.AppID
	sessionResp, err := fixture.writeSvc.OpenSyncSession(context.Background(), fixture.actor, write.OpenSyncSessionRequest{
		DomainID:   "test-domain",
		GraphScope: graphScope,
	})
	if err != nil {
		t.Fatalf("OpenSyncSession() error = %v", err)
	}

	created, err := fixture.writeSvc.CreateNodesBulkWithContext(context.Background(), fixture.actor, write.NodeBulkCreateRequest{
		GraphVersionID: sessionResp.GraphVersionID,
		Nodes: []write.NodeCreateRequest{
			{DomainID: "test-domain", NodeType: "Parent", ExternalRef: "sync-a", Properties: map[string]any{"ten": "A"}},
			{DomainID: "test-domain", NodeType: "Parent", ExternalRef: "sync-b", Properties: map[string]any{"ten": "B"}},
			{DomainID: "test-domain", NodeType: "Parent", ExternalRef: "sync-c", Properties: map[string]any{"ten": "C"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateNodesBulkWithContext() error = %v", err)
	}
	if len(created.Succeeded) != 3 {
		t.Fatalf("created = %d, want 3", len(created.Succeeded))
	}
	if _, err := fixture.writeSvc.UpdateNodeWithContext(context.Background(), fixture.actor, created.Succeeded[1].NodeID, write.NodeUpdateRequest{
		Properties:     map[string]any{"ten": "B-Updated"},
		GraphVersionID: sessionResp.GraphVersionID,
	}); err != nil {
		t.Fatalf("UpdateNodeWithContext() error = %v", err)
	}
	if _, err := fixture.writeSvc.DeleteNodeWithVersion(context.Background(), fixture.actor, created.Succeeded[2].NodeID, sessionResp.GraphVersionID); err != nil {
		t.Fatalf("DeleteNodeWithVersion() error = %v", err)
	}
	if err := fixture.writeSvc.CommitSyncSession(context.Background(), fixture.actor, sessionResp.SessionID); err != nil {
		t.Fatalf("CommitSyncSession() error = %v", err)
	}

	report := runtime.PollOnce()
	if report.Processed != 1 {
		t.Fatalf("processed = %d, want 1", report.Processed)
	}

	var syncEvents []write.OutboxEvent
	for _, event := range fixture.store.ListOutboxEvents() {
		if event.Payload == nil {
			continue
		}
		if event.Payload["graph_version_id"] == sessionResp.GraphVersionID {
			syncEvents = append(syncEvents, event)
		}
	}
	if len(syncEvents) != 1 {
		t.Fatalf("sync events = %d, want 1", len(syncEvents))
	}
	if syncEvents[0].Status != string(EventDone) {
		t.Fatalf("sync event status = %s, want %s", syncEvents[0].Status, EventDone)
	}
	if _, ok := runtime.Graph().Nodes[created.Succeeded[0].NodeID]; !ok {
		t.Fatal("projected node sync-a missing")
	}
	if _, ok := runtime.Graph().Nodes[created.Succeeded[1].NodeID]; !ok {
		t.Fatal("projected node sync-b missing")
	}
	if _, ok := runtime.Graph().Nodes[created.Succeeded[2].NodeID]; ok {
		t.Fatal("deleted node sync-c still projected")
	}
	if got := runtime.Graph().Nodes[created.Succeeded[1].NodeID].Properties["ten"]; got != "B-Updated" {
		t.Fatalf("projected updated node property = %v, want B-Updated", got)
	}
	if _, ok := runtime.Vector().Documents[created.Succeeded[2].NodeID]; ok {
		t.Fatal("deleted node sync-c still present in vector store")
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

func TestRuntimeSkipsStaleEventsForMissingEntities(t *testing.T) {
	// NODE_UPSERTED for a node that no longer exists in the store is a stale
	// outbox event (entity was deleted after the event was written). The worker
	// should mark it DONE on the first attempt rather than retrying forever.
	runtime := NewRuntime(&failingStore{
		events: []write.OutboxEvent{
			{ID: "evt-1", EventType: "NODE_UPSERTED", Payload: map[string]any{"node_id": "missing"}},
		},
	}, &noopOntology{}, nil)

	runtime.PollOnce()
	env, ok := runtime.EventEnvelope("evt-1")
	if !ok {
		t.Fatal("event envelope missing")
	}
	if env.Status != EventDone {
		t.Fatalf("status = %s, want done", env.Status)
	}
	events := runtime.store.ListOutboxEvents()
	if len(events) != 1 || events[0].Status != string(EventDone) {
		t.Fatalf("persisted events = %#v, want done", events)
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
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now, Payload: map[string]any{"graph_identifier_id": "graph-1", "graph_version_id": "v5", "graph_version_number": int64(5)}},
		},
		graphHeads: map[string]write.GraphProjectionHeadRecord{
			"graph-1:graph:": {IdentifierID: "graph-1", BackendKind: "graph", BackendName: "", AppliedVersionID: "v5", AppliedVersionNumber: 5},
		},
	}
	runtime := &Runtime{store: store, lagToleranceWindow: 30 * time.Second, maxRetries: 3}

	status, ok := runtime.EntitySyncStatus("node-1", "kg_node")
	if !ok {
		t.Fatal("EntitySyncStatus() = not found, want status")
	}
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
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now.Add(-10 * time.Second), Payload: map[string]any{"graph_identifier_id": "graph-1", "graph_version_id": "v4", "graph_version_number": int64(4)}},
		},
		graphHeads: map[string]write.GraphProjectionHeadRecord{
			"graph-1:graph:": {IdentifierID: "graph-1", BackendKind: "graph", BackendName: "", AppliedVersionID: "v4", AppliedVersionNumber: 4},
		},
	}
	runtime := &Runtime{store: store, lagToleranceWindow: 30 * time.Second, maxRetries: 3}

	status, ok := runtime.EntitySyncStatus("node-1", "kg_node")
	if !ok {
		t.Fatal("EntitySyncStatus() = not found, want status")
	}
	if status.GraphLagClass != SyncLagClassInFlight {
		t.Fatalf("status = %#v, want in-flight", status)
	}
}

func TestEntitySyncStatus_UsesLiveGraphBackendVersion(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", LastGraphSyncedAt: now, GraphVersion: 5, VectorVersion: 5, LastVectorSyncedAt: now},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now, Payload: map[string]any{"graph_identifier_id": "graph-1", "graph_version_id": "v5", "graph_version_number": int64(5)}},
		},
		graphHeads: map[string]write.GraphProjectionHeadRecord{
			"graph-1:graph:": {IdentifierID: "graph-1", BackendKind: "graph", BackendName: "", AppliedVersionID: "v5", AppliedVersionNumber: 5},
		},
	}
	runtime := &Runtime{
		store:              store,
		graphAdapter:       staleVersionGraphAdapter{},
		lagToleranceWindow: 30 * time.Second,
		maxRetries:         3,
	}

	status, ok := runtime.EntitySyncStatus("node-1", "kg_node")
	if !ok {
		t.Fatal("EntitySyncStatus() = not found, want status")
	}
	if status.GraphVersion != 41 {
		t.Fatalf("graph version = %d, want live backend version 41 from adapter stub", status.GraphVersion)
	}
	if status.GraphLagClass == SyncLagClassSynced {
		t.Fatalf("status = %#v, want graph lag to reflect backend drift", status)
	}
}

func TestEntitySyncStatus_IgnoresZeroLiveGraphBackendVersion(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", LastGraphSyncedAt: now, GraphVersion: 5, VectorVersion: 5, LastVectorSyncedAt: now},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now, Payload: map[string]any{"graph_identifier_id": "graph-1", "graph_version_id": "v5", "graph_version_number": int64(5)}},
		},
		graphHeads: map[string]write.GraphProjectionHeadRecord{
			"graph-1:graph:": {IdentifierID: "graph-1", BackendKind: "graph", BackendName: "", AppliedVersionID: "v5", AppliedVersionNumber: 5},
		},
	}
	runtime := &Runtime{
		store:              store,
		graphAdapter:       zeroVersionGraphAdapter{},
		lagToleranceWindow: 30 * time.Second,
		maxRetries:         3,
	}

	status, ok := runtime.EntitySyncStatus("node-1", "kg_node")
	if !ok {
		t.Fatal("EntitySyncStatus() = not found, want status")
	}
	if status.GraphVersion != 5 {
		t.Fatalf("graph version = %d, want stored version 5 when live backend reports zero", status.GraphVersion)
	}
	if status.GraphLagClass == SyncLagClassSynced {
		t.Fatalf("status = %#v, want graph lag to remain non-synced when the graph node is not queryable", status)
	}
}

func TestEntitySyncStatus_UsesLiveGraphBackendVersionButNodeIsMissing(t *testing.T) {
	now := time.Now().UTC()
	store := &versionFixtureStore{
		projectionVersions: []write.ProjectionVersionRecord{
			{EntityID: "node-1", EntityKind: "kg_node", SourceVersion: 5, SourceEventID: "evt-1", LastGraphSyncedAt: now, GraphVersion: 5, VectorVersion: 5, LastVectorSyncedAt: now},
		},
		outboxEvents: []write.OutboxEvent{
			{ID: "evt-1", AggregateType: "kg_node", AggregateID: "node-1", EventType: "NODE_UPSERTED", CreatedAt: now, Payload: map[string]any{"graph_identifier_id": "graph-1", "graph_version_id": "v5", "graph_version_number": int64(5)}},
		},
		graphHeads: map[string]write.GraphProjectionHeadRecord{
			"graph-1:graph:": {IdentifierID: "graph-1", BackendKind: "graph", BackendName: "", AppliedVersionID: "v5", AppliedVersionNumber: 5},
		},
	}
	runtime := &Runtime{
		store:              store,
		graphAdapter:       zeroVersionGraphAdapter{},
		lagToleranceWindow: 30 * time.Second,
		maxRetries:         3,
	}

	status, ok := runtime.EntitySyncStatus("node-1", "kg_node")
	if !ok {
		t.Fatal("EntitySyncStatus() = not found, want status")
	}
	if status.GraphProjectionReady {
		t.Fatalf("status = %#v, want projection to remain unreadable", status)
	}
	if status.GraphLagClass == SyncLagClassSynced {
		t.Fatalf("status = %#v, want graph lag to remain non-synced when the graph node is missing", status)
	}
}

func TestRuntimeReportsGraphProjectionWriteFailure(t *testing.T) {
	fixture := newWorkerFixture(t)
	runtime := NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	runtime.graphAdapter = failingGraphAdapter{}

	report := runtime.PollOnce()
	if report.Failed == 0 {
		t.Fatal("expected projection failure when graph adapter rejects writes")
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

type cleanupErrorStore struct {
	versionFixtureStore
	err error
}

func (s *cleanupErrorStore) CleanupExpiredSyncSession(ctx context.Context, versionID string) error {
	return s.err
}

func (f *failingStore) ListOutboxEvents() []write.OutboxEvent { return f.events }
func (f *failingStore) ClaimOutboxBatch(ctx context.Context, pageSize int) ([]write.OutboxEvent, error) {
	out := make([]write.OutboxEvent, 0, len(f.events))
	for i := range f.events {
		if f.events[i].Status != "" && f.events[i].Status != string(EventPending) && f.events[i].Status != string(EventFailed) {
			continue
		}
		f.events[i].Status = string(EventProcessing)
		out = append(out, f.events[i])
		if pageSize > 0 && len(out) >= pageSize {
			break
		}
	}
	return out, nil
}
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
func (f *failingStore) GetNodesByIDs(ids []string) map[string]write.NodeRecord {
	return map[string]write.NodeRecord{}
}
func (f *failingStore) GetRelationshipByID(id string) (write.RelationshipRecord, bool) {
	return write.RelationshipRecord{}, false
}
func (f *failingStore) GetRelationshipsByIDs(ids []string) map[string]write.RelationshipRecord {
	return map[string]write.RelationshipRecord{}
}
func (f *failingStore) SoftDeleteRelationshipsWithOutbox(ctx context.Context, relationshipIDs []string, deletedAt time.Time) ([]write.RelationshipRecord, error) {
	return nil, nil
}
func (f *failingStore) ListNodesBatch(afterID string, limit int) []write.NodeRecord { return nil }
func (f *failingStore) ListNodes() []write.NodeRecord                               { return nil }
func (f *failingStore) ListRelationshipsBatch(afterID string, limit int) []write.RelationshipRecord {
	return nil
}
func (f *failingStore) ListRelationships() []write.RelationshipRecord { return nil }
func (f *failingStore) GetProjectionVersion(entityID, entityKind string) (write.ProjectionVersionRecord, bool) {
	return write.ProjectionVersionRecord{}, false
}
func (f *failingStore) UpsertProjectionVersion(ctx context.Context, record write.ProjectionVersionRecord) error {
	return nil
}
func (f *failingStore) ListProjectionVersionsBatch(afterEntityKind, afterEntityID string, limit int) []write.ProjectionVersionRecord {
	return nil
}
func (f *failingStore) ListProjectionVersions() []write.ProjectionVersionRecord { return nil }
func (f *failingStore) GetGraphVersionEntities(versionID string) []write.GraphVersionEntityRecord {
	return nil
}
func (f *failingStore) ListPendingGraphVersionsBefore(cutoff time.Time) []write.GraphVersionRecord {
	return nil
}
func (f *failingStore) GetGraphIdentityByID(ctx context.Context, identifierID string) (write.GraphIdentityRecord, bool) {
	return write.GraphIdentityRecord{}, false
}
func (f *failingStore) AbandonGraphVersion(ctx context.Context, versionID string) error {
	return nil
}
func (f *failingStore) CleanupExpiredSyncSession(ctx context.Context, versionID string) error {
	return nil
}
func (f *failingStore) ReleaseScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error {
	return nil
}
func (f *failingStore) UpsertGraphProjectionHead(ctx context.Context, record write.GraphProjectionHeadRecord) error {
	return nil
}
func (f *failingStore) GetGraphProjectionHead(identifierID, backendKind, backendName string) (write.GraphProjectionHeadRecord, bool) {
	return write.GraphProjectionHeadRecord{}, false
}
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
	nodes                []write.NodeRecord
	rels                 []write.RelationshipRecord
	projectionVersions   []write.ProjectionVersionRecord
	outboxEvents         []write.OutboxEvent
	graphVersionEntities map[string][]write.GraphVersionEntityRecord
	graphVersions        []write.GraphVersionRecord
	graphIdentities      map[string]write.GraphIdentityRecord
	scopeLeases          map[string]struct{}
	graphHeads           map[string]write.GraphProjectionHeadRecord
	releasedLeases       []string
}

func (s *versionFixtureStore) ListOutboxEvents() []write.OutboxEvent {
	return append([]write.OutboxEvent(nil), s.outboxEvents...)
}
func (s *versionFixtureStore) ClaimOutboxBatch(ctx context.Context, pageSize int) ([]write.OutboxEvent, error) {
	out := make([]write.OutboxEvent, 0, len(s.outboxEvents))
	for i := range s.outboxEvents {
		if s.outboxEvents[i].Status != "" && s.outboxEvents[i].Status != string(EventPending) && s.outboxEvents[i].Status != string(EventFailed) {
			continue
		}
		s.outboxEvents[i].Status = string(EventProcessing)
		out = append(out, s.outboxEvents[i])
		if pageSize > 0 && len(out) >= pageSize {
			break
		}
	}
	return out, nil
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
func (s *versionFixtureStore) GetNodesByIDs(ids []string) map[string]write.NodeRecord {
	out := map[string]write.NodeRecord{}
	for _, id := range ids {
		if node, ok := s.GetNodeByID(id); ok {
			out[id] = node
		}
	}
	return out
}
func (s *versionFixtureStore) GetRelationshipByID(id string) (write.RelationshipRecord, bool) {
	for _, rel := range s.rels {
		if rel.ID == id {
			return rel, true
		}
	}
	return write.RelationshipRecord{}, false
}
func (s *versionFixtureStore) GetRelationshipsByIDs(ids []string) map[string]write.RelationshipRecord {
	out := map[string]write.RelationshipRecord{}
	for _, id := range ids {
		if rel, ok := s.GetRelationshipByID(id); ok {
			out[id] = rel
		}
	}
	return out
}
func (s *versionFixtureStore) SoftDeleteRelationshipsWithOutbox(ctx context.Context, relationshipIDs []string, deletedAt time.Time) ([]write.RelationshipRecord, error) {
	deleted := make([]write.RelationshipRecord, 0, len(relationshipIDs))
	for i := range s.rels {
		for _, id := range relationshipIDs {
			if s.rels[i].ID != id {
				continue
			}
			s.rels[i].IsDeleted = true
			deleted = append(deleted, s.rels[i])
			break
		}
	}
	return deleted, nil
}
func (s *versionFixtureStore) ListNodesBatch(afterID string, limit int) []write.NodeRecord {
	nodes := append([]write.NodeRecord(nil), s.nodes...)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	out := make([]write.NodeRecord, 0, len(nodes))
	for _, node := range nodes {
		if afterID != "" && node.ID <= afterID {
			continue
		}
		out = append(out, node)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
func (s *versionFixtureStore) ListNodes() []write.NodeRecord {
	return append([]write.NodeRecord(nil), s.nodes...)
}
func (s *versionFixtureStore) ListRelationshipsBatch(afterID string, limit int) []write.RelationshipRecord {
	rels := append([]write.RelationshipRecord(nil), s.rels...)
	sort.Slice(rels, func(i, j int) bool {
		return rels[i].ID < rels[j].ID
	})
	out := make([]write.RelationshipRecord, 0, len(rels))
	for _, rel := range rels {
		if afterID != "" && rel.ID <= afterID {
			continue
		}
		out = append(out, rel)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
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
func (s *versionFixtureStore) UpsertProjectionVersion(ctx context.Context, record write.ProjectionVersionRecord) error {
	s.projectionVersions = append(s.projectionVersions, record)
	return nil
}
func (s *versionFixtureStore) ListProjectionVersions() []write.ProjectionVersionRecord {
	return append([]write.ProjectionVersionRecord(nil), s.projectionVersions...)
}
func (s *versionFixtureStore) GetGraphVersionEntities(versionID string) []write.GraphVersionEntityRecord {
	return append([]write.GraphVersionEntityRecord(nil), s.graphVersionEntities[versionID]...)
}
func (s *versionFixtureStore) ListPendingGraphVersionsBefore(cutoff time.Time) []write.GraphVersionRecord {
	result := make([]write.GraphVersionRecord, 0)
	for _, version := range s.graphVersions {
		if !strings.EqualFold(version.VersionStatus, "PENDING_ENTITIES") {
			continue
		}
		if !cutoff.IsZero() && !version.CreatedAt.Before(cutoff) {
			continue
		}
		result = append(result, version)
	}
	return result
}
func (s *versionFixtureStore) GetGraphIdentityByID(ctx context.Context, identifierID string) (write.GraphIdentityRecord, bool) {
	if s.graphIdentities == nil {
		return write.GraphIdentityRecord{}, false
	}
	record, ok := s.graphIdentities[identifierID]
	return record, ok
}
func (s *versionFixtureStore) AbandonGraphVersion(ctx context.Context, versionID string) error {
	for i := range s.graphVersions {
		if s.graphVersions[i].VersionID == versionID {
			s.graphVersions[i].VersionStatus = "ABANDONED"
		}
	}
	return nil
}
func (s *versionFixtureStore) CleanupExpiredSyncSession(ctx context.Context, versionID string) error {
	for i := range s.graphVersions {
		if s.graphVersions[i].VersionID == versionID {
			s.graphVersions[i].VersionStatus = "ABANDONED"
			if s.scopeLeases != nil {
				if identity, ok := s.graphIdentities[s.graphVersions[i].IdentifierID]; ok {
					key := identity.OwnerTenantID + ":" + identity.OwnerAppID + ":" + identity.GraphScope
					delete(s.scopeLeases, key)
				}
			}
		}
	}
	return nil
}
func (s *versionFixtureStore) ReleaseScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error {
	s.releasedLeases = append(s.releasedLeases, ownerTenantID+":"+ownerAppID+":"+graphScope+":"+versionID)
	return nil
}
func (s *versionFixtureStore) UpsertGraphProjectionHead(ctx context.Context, record write.GraphProjectionHeadRecord) error {
	if s.graphHeads == nil {
		s.graphHeads = map[string]write.GraphProjectionHeadRecord{}
	}
	key := record.IdentifierID + ":" + record.BackendKind + ":" + record.BackendName
	s.graphHeads[key] = record
	return nil
}
func (s *versionFixtureStore) GetGraphProjectionHead(identifierID, backendKind, backendName string) (write.GraphProjectionHeadRecord, bool) {
	if s.graphHeads == nil {
		return write.GraphProjectionHeadRecord{}, false
	}
	key := identifierID + ":" + backendKind + ":" + backendName
	record, ok := s.graphHeads[key]
	return record, ok
}
func (s *versionFixtureStore) ListProjectionVersionsBatch(afterEntityKind, afterEntityID string, limit int) []write.ProjectionVersionRecord {
	versions := append([]write.ProjectionVersionRecord(nil), s.projectionVersions...)
	sort.Slice(versions, func(i, j int) bool {
		if versions[i].EntityKind == versions[j].EntityKind {
			return versions[i].EntityID < versions[j].EntityID
		}
		return versions[i].EntityKind < versions[j].EntityKind
	})
	out := make([]write.ProjectionVersionRecord, 0, len(versions))
	for _, version := range versions {
		if afterEntityKind != "" {
			if version.EntityKind < afterEntityKind {
				continue
			}
			if version.EntityKind == afterEntityKind && version.EntityID <= afterEntityID {
				continue
			}
		}
		out = append(out, version)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
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

type zeroVersionGraphAdapter struct{}

func (zeroVersionGraphAdapter) UpsertNode(ctx context.Context, node graphstore.GraphNode) error {
	return nil
}
func (zeroVersionGraphAdapter) DeleteNode(ctx context.Context, nodeID string) error {
	return nil
}
func (zeroVersionGraphAdapter) UpsertRelationship(ctx context.Context, rel graphstore.GraphRelationship) error {
	return nil
}
func (zeroVersionGraphAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	return nil
}
func (zeroVersionGraphAdapter) ExecuteQuery(ctx context.Context, query graphstore.GraphQuery, params map[string]any) ([]map[string]any, error) {
	return nil, nil
}
func (zeroVersionGraphAdapter) ListNodes(ctx context.Context) ([]graphstore.GraphNode, error) {
	return nil, nil
}
func (zeroVersionGraphAdapter) ListRelationships(ctx context.Context) ([]graphstore.GraphRelationship, error) {
	return nil, nil
}
func (zeroVersionGraphAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	return 0, nil
}

type failingGraphAdapter struct{}

func (failingGraphAdapter) UpsertNode(ctx context.Context, node graphstore.GraphNode) error {
	return errors.New("graph upsert failed")
}
func (failingGraphAdapter) DeleteNode(ctx context.Context, nodeID string) error {
	return errors.New("graph delete failed")
}
func (failingGraphAdapter) UpsertRelationship(ctx context.Context, rel graphstore.GraphRelationship) error {
	return errors.New("graph relationship upsert failed")
}
func (failingGraphAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	return errors.New("graph relationship delete failed")
}
func (failingGraphAdapter) ExecuteQuery(ctx context.Context, query graphstore.GraphQuery, params map[string]any) ([]map[string]any, error) {
	return nil, errors.New("graph query failed")
}
func (failingGraphAdapter) ListNodes(ctx context.Context) ([]graphstore.GraphNode, error) {
	return nil, errors.New("graph list nodes failed")
}
func (failingGraphAdapter) ListRelationships(ctx context.Context) ([]graphstore.GraphRelationship, error) {
	return nil, errors.New("graph list relationships failed")
}
func (failingGraphAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	return 0, nil
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
	store          *write.MemoryStore
	ontologySvc    *ontology.Service
	writeSvc       *write.Service
	cache          rediscache.Client
	actor          access.Identity
	parentID       string
	childID        string
	relationshipID string
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
	rel, err := writeSvc.CreateRelationship(actor, write.RelationshipCreateRequest{
		RelType:    "PARENT_OF",
		FromNodeID: parent.NodeID,
		ToNodeID:   child.NodeID,
		DomainID:   "test-domain",
		Properties: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}

	return workerFixture{
		store:          store,
		ontologySvc:    ontologySvc,
		writeSvc:       writeSvc,
		cache:          cache,
		actor:          actor,
		parentID:       parent.NodeID,
		childID:        child.NodeID,
		relationshipID: rel.RelationshipID,
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

func hasGraphNodeID(nodes []graphstore.GraphNode, want string) bool {
	for _, node := range nodes {
		if node.ID == want {
			return true
		}
	}
	return false
}

func hasGraphRelationshipID(rels []graphstore.GraphRelationship, want string) bool {
	for _, rel := range rels {
		if rel.ID == want {
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

type batchCountingProvider struct {
	dimensions  int
	modelID     string
	singleCalls int
	batchCalls  int
}

func (p *batchCountingProvider) Embed(_ context.Context, _ string) ([]float64, error) {
	p.singleCalls++
	return []float64{1, 2, 3, 4}, nil
}

func (p *batchCountingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	p.batchCalls++
	result := make([][]float64, 0, len(texts))
	for range texts {
		vec, err := p.Embed(ctx, "")
		if err != nil {
			return nil, err
		}
		result = append(result, vec)
	}
	p.singleCalls -= len(texts)
	return result, nil
}

func (p *batchCountingProvider) Dimensions() int { return p.dimensions }
func (p *batchCountingProvider) ModelID() string { return p.modelID }

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
