package workers

import (
	"context"
	"testing"
	"time"

	"kg-service/internal/platform/session"
	"kg-service/internal/write"
)

type scopedNodeStore struct {
	tenantID string
	appID    string
	node     write.NodeRecord
}

func (s *scopedNodeStore) ListOutboxEvents() []write.OutboxEvent { return nil }
func (s *scopedNodeStore) ClaimOutboxBatch(ctx context.Context, pageSize int) ([]write.OutboxEvent, error) {
	return nil, nil
}
func (s *scopedNodeStore) CreateNodesBulkWithOutbox(ctx context.Context, nodes []write.NodeRecord, events []write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) CreateNodeWithOutbox(ctx context.Context, node write.NodeRecord, event write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) CreateNodeBundle(ctx context.Context, node write.NodeRecord, rels []write.RelationshipRecord, event write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) CreateRelationshipsBulkWithOutbox(ctx context.Context, rels []write.RelationshipRecord, events []write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) UpdateNodeWithOutbox(ctx context.Context, node write.NodeRecord, event write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) SoftDeleteNodeWithOutbox(ctx context.Context, node write.NodeRecord, event write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) SoftDeleteNodesByExternalRefPrefixWithOutbox(ctx context.Context, prefix string, deletedAt time.Time) ([]write.NodeRecord, error) {
	return nil, nil
}
func (s *scopedNodeStore) CreateRelationshipWithOutbox(ctx context.Context, rel write.RelationshipRecord, event write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) GetOutboxEventByID(id string) (write.OutboxEvent, bool) {
	return write.OutboxEvent{}, false
}
func (s *scopedNodeStore) GetNodeByID(id string) (write.NodeRecord, bool) {
	if s.tenantID == "" || s.appID == "" || id != s.node.ID {
		return write.NodeRecord{}, false
	}
	return s.node, true
}
func (s *scopedNodeStore) GetNodesByIDs(ids []string) map[string]write.NodeRecord {
	out := map[string]write.NodeRecord{}
	for _, id := range ids {
		if node, ok := s.GetNodeByID(id); ok {
			out[id] = node
		}
	}
	return out
}
func (s *scopedNodeStore) GetNodeByExternalRef(externalRef string) (write.NodeRecord, bool) {
	return write.NodeRecord{}, false
}
func (s *scopedNodeStore) GetNodesByExternalRefs(externalRefs []string) map[string]write.NodeRecord {
	return map[string]write.NodeRecord{}
}
func (s *scopedNodeStore) GetRelationshipByID(id string) (write.RelationshipRecord, bool) {
	return write.RelationshipRecord{}, false
}
func (s *scopedNodeStore) GetRelationshipsByIDs(ids []string) map[string]write.RelationshipRecord {
	return map[string]write.RelationshipRecord{}
}
func (s *scopedNodeStore) SoftDeleteRelationshipsWithOutbox(ctx context.Context, relationshipIDs []string, deletedAt time.Time) ([]write.RelationshipRecord, error) {
	return nil, nil
}
func (s *scopedNodeStore) CreateOutboxEvents(ctx context.Context, events []write.OutboxEvent) error {
	return nil
}
func (s *scopedNodeStore) ListNodesBatch(afterID string, limit int) []write.NodeRecord { return nil }
func (s *scopedNodeStore) ListNodes() []write.NodeRecord                               { return nil }
func (s *scopedNodeStore) ListRelationshipsBatch(afterID string, limit int) []write.RelationshipRecord {
	return nil
}
func (s *scopedNodeStore) ListRelationships() []write.RelationshipRecord { return nil }
func (s *scopedNodeStore) UpdateOutboxStatus(ctx context.Context, eventID, status string, retryCount int, processedAt *time.Time) error {
	return nil
}
func (s *scopedNodeStore) UpsertProjectionVersion(ctx context.Context, record write.ProjectionVersionRecord) error {
	return nil
}
func (s *scopedNodeStore) GetProjectionVersion(entityID, entityKind string) (write.ProjectionVersionRecord, bool) {
	return write.ProjectionVersionRecord{}, false
}
func (s *scopedNodeStore) ListProjectionVersionsBatch(afterEntityKind, afterEntityID string, limit int) []write.ProjectionVersionRecord {
	return nil
}
func (s *scopedNodeStore) ListProjectionVersions() []write.ProjectionVersionRecord { return nil }
func (s *scopedNodeStore) GetGraphVersionEntities(versionID string) []write.GraphVersionEntityRecord {
	return nil
}
func (s *scopedNodeStore) ListPendingGraphVersionsBefore(cutoff time.Time) []write.GraphVersionRecord {
	return nil
}
func (s *scopedNodeStore) ArchiveGraphVersions(ctx context.Context, keepCount int, olderThan time.Time) ([]string, error) {
	return nil, nil
}
func (s *scopedNodeStore) GetGraphIdentityByID(ctx context.Context, identifierID string) (write.GraphIdentityRecord, bool) {
	return write.GraphIdentityRecord{}, false
}
func (s *scopedNodeStore) AbandonGraphVersion(ctx context.Context, versionID string) error {
	return nil
}
func (s *scopedNodeStore) CleanupExpiredSyncSession(ctx context.Context, versionID string) error {
	return nil
}
func (s *scopedNodeStore) ReleaseScopeLease(ctx context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error {
	return nil
}
func (s *scopedNodeStore) UpsertGraphProjectionHead(ctx context.Context, record write.GraphProjectionHeadRecord) error {
	return nil
}
func (s *scopedNodeStore) GetGraphProjectionHead(identifierID, backendKind, backendName string) (write.GraphProjectionHeadRecord, bool) {
	return write.GraphProjectionHeadRecord{}, false
}

type identitySessionManager struct {
	store *scopedNodeStore
}

func (m identitySessionManager) Within(ctx context.Context, identity session.WriteIdentity, fn func(session.SessionScope) error) (session.SessionScope, error) {
	scope := session.SessionScope{
		Identity: identity,
	}
	if m.store != nil {
		m.store.tenantID = identity.TenantID
		m.store.appID = identity.AppID
	}
	return scope, fn(scope)
}

func TestRuntimeFetchNodeForEventUsesEventIdentity(t *testing.T) {
	store := &scopedNodeStore{
		node: write.NodeRecord{
			ID:            "node-1",
			NodeType:      "Doc",
			DomainID:      "d1",
			OwnerTenantID: "tenant-1",
			OwnerAppID:    "app-1",
		},
	}
	runtime := NewRuntime(store, &noopOntology{}, nil, WithSessionManager(identitySessionManager{store: store}))

	node, ok := runtime.fetchNodeForEvent(context.Background(), write.OutboxEvent{
		ID: "evt-1",
		Payload: map[string]any{
			"node_id":         "node-1",
			"owner_tenant_id": "tenant-1",
			"owner_app_id":    "app-1",
		},
	}, "node-1")
	if !ok {
		t.Fatal("fetchNodeForEvent() = not found, want found")
	}
	if node.ID != "node-1" {
		t.Fatalf("node id = %q, want node-1", node.ID)
	}
}
