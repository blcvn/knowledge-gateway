package write

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"
)

var ErrDuplicateExternalRef = errors.New("duplicate external_ref")
var ErrNodeNotFound = errors.New("node not found")

type MemoryStore struct {
	mu                   sync.RWMutex
	nodes                map[string]NodeRecord
	externalRefs         map[string]string
	rels                 map[string]RelationshipRecord
	outbox               []OutboxEvent
	projectionVersions   map[string]map[string]ProjectionVersionRecord
	graphIdentities      map[string]GraphIdentityRecord
	graphVersions        map[string][]GraphVersionRecord
	graphVersionEntities map[string][]GraphVersionEntityRecord
	scopeLeases          map[string]ScopeLeaseRecord
	graphHeads           map[string]map[string]map[string]GraphProjectionHeadRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nodes:                map[string]NodeRecord{},
		externalRefs:         map[string]string{},
		rels:                 map[string]RelationshipRecord{},
		outbox:               []OutboxEvent{},
		projectionVersions:   map[string]map[string]ProjectionVersionRecord{},
		graphIdentities:      map[string]GraphIdentityRecord{},
		graphVersions:        map[string][]GraphVersionRecord{},
		graphVersionEntities: map[string][]GraphVersionEntityRecord{},
		scopeLeases:          map[string]ScopeLeaseRecord{},
		graphHeads:           map[string]map[string]map[string]GraphProjectionHeadRecord{},
	}
}

func (s *MemoryStore) CreateNodesBulkWithOutbox(_ context.Context, nodes []NodeRecord, events []OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, node := range nodes {
		if node.ExternalRef != "" {
			if existingID, exists := s.externalRefs[node.ExternalRef]; exists && existingID != node.ID {
				return ErrDuplicateExternalRef
			}
		}
		s.nodes[node.ID] = node
		if node.ExternalRef != "" {
			s.externalRefs[node.ExternalRef] = node.ID
		}
	}
	for _, event := range events {
		s.outbox = append(s.outbox, event)
	}
	return nil
}

func (s *MemoryStore) CreateNodeWithOutbox(_ context.Context, node NodeRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node.ExternalRef != "" {
		if _, exists := s.externalRefs[node.ExternalRef]; exists {
			return ErrDuplicateExternalRef
		}
	}

	s.nodes[node.ID] = node
	if node.ExternalRef != "" {
		s.externalRefs[node.ExternalRef] = node.ID
	}
	s.outbox = append(s.outbox, event)
	return nil
}

func (s *MemoryStore) CreateNodeBundle(_ context.Context, node NodeRecord, rels []RelationshipRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if node.ExternalRef != "" {
		if _, exists := s.externalRefs[node.ExternalRef]; exists {
			return ErrDuplicateExternalRef
		}
	}

	s.nodes[node.ID] = node
	if node.ExternalRef != "" {
		s.externalRefs[node.ExternalRef] = node.ID
	}
	for _, rel := range rels {
		s.rels[rel.ID] = rel
	}
	s.outbox = append(s.outbox, event)
	return nil
}

func (s *MemoryStore) CreateRelationshipsBulkWithOutbox(_ context.Context, rels []RelationshipRecord, events []OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rel := range rels {
		s.rels[rel.ID] = rel
	}
	for _, event := range events {
		s.outbox = append(s.outbox, event)
	}
	return nil
}

func (s *MemoryStore) GetNodeByID(id string) (NodeRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[id]
	return node, ok
}

func (s *MemoryStore) GetNodesByIDs(ids []string) map[string]NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]NodeRecord, len(ids))
	for _, id := range ids {
		if node, ok := s.nodes[id]; ok {
			result[id] = node
		}
	}
	return result
}

func (s *MemoryStore) GetNodeByExternalRef(externalRef string) (NodeRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.externalRefs[externalRef]
	if !ok {
		return NodeRecord{}, false
	}
	node, ok := s.nodes[id]
	return node, ok
}

func (s *MemoryStore) GetNodesByExternalRefs(externalRefs []string) map[string]NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]NodeRecord, len(externalRefs))
	for _, externalRef := range externalRefs {
		id, ok := s.externalRefs[externalRef]
		if !ok {
			continue
		}
		if node, ok := s.nodes[id]; ok {
			result[externalRef] = node
		}
	}
	return result
}

func (s *MemoryStore) ListNodes() []NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]NodeRecord, 0, len(s.nodes))
	for _, node := range s.nodes {
		result = append(result, node)
	}
	return result
}

func (s *MemoryStore) ListNodesBatch(afterID string, limit int) []NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]NodeRecord, 0, limit)
	nodes := make([]NodeRecord, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
	for _, node := range nodes {
		if afterID != "" && node.ID <= afterID {
			continue
		}
		result = append(result, node)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func (s *MemoryStore) UpdateNodeWithOutbox(_ context.Context, node NodeRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.nodes[node.ID]
	if !ok {
		return ErrNodeNotFound
	}

	if node.ExternalRef != existing.ExternalRef {
		if node.ExternalRef != "" {
			if existingID, exists := s.externalRefs[node.ExternalRef]; exists && existingID != node.ID {
				return ErrDuplicateExternalRef
			}
		}
		if existing.ExternalRef != "" {
			delete(s.externalRefs, existing.ExternalRef)
		}
		if node.ExternalRef != "" {
			s.externalRefs[node.ExternalRef] = node.ID
		}
	}

	s.nodes[node.ID] = node
	s.outbox = append(s.outbox, event)
	return nil
}

func (s *MemoryStore) UpdateNode(_ context.Context, node NodeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.nodes[node.ID]
	if !ok {
		return ErrNodeNotFound
	}
	if node.ExternalRef != existing.ExternalRef {
		if node.ExternalRef != "" {
			if existingID, exists := s.externalRefs[node.ExternalRef]; exists && existingID != node.ID {
				return ErrDuplicateExternalRef
			}
		}
		if existing.ExternalRef != "" {
			delete(s.externalRefs, existing.ExternalRef)
		}
		if node.ExternalRef != "" {
			s.externalRefs[node.ExternalRef] = node.ID
		}
	}
	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryStore) SoftDeleteNodeWithOutbox(_ context.Context, node NodeRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nodes[node.ID]; !ok {
		return ErrNodeNotFound
	}

	s.nodes[node.ID] = node
	s.outbox = append(s.outbox, event)
	return nil
}

func (s *MemoryStore) SoftDeleteNode(_ context.Context, node NodeRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.nodes[node.ID]; !ok {
		return ErrNodeNotFound
	}
	s.nodes[node.ID] = node
	return nil
}

func (s *MemoryStore) SoftDeleteNodesByExternalRefPrefix(_ context.Context, prefix string, deletedAt time.Time) ([]NodeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := make([]NodeRecord, 0)
	for id, node := range s.nodes {
		if node.IsDeleted || node.ExternalRef == "" || !strings.HasPrefix(node.ExternalRef, prefix) {
			continue
		}
		node.IsDeleted = true
		node.UpdatedAt = deletedAt
		s.nodes[id] = node
		deleted = append(deleted, node)
	}
	return deleted, nil
}

func (s *MemoryStore) SoftDeleteNodesByExternalRefPrefixWithOutbox(_ context.Context, prefix string, deletedAt time.Time) ([]NodeRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := make([]NodeRecord, 0)
	for id, node := range s.nodes {
		if node.IsDeleted || node.ExternalRef == "" || !strings.HasPrefix(node.ExternalRef, prefix) {
			continue
		}
		node.IsDeleted = true
		node.UpdatedAt = deletedAt
		s.nodes[id] = node
		deleted = append(deleted, node)
	}
	return deleted, nil
}

func (s *MemoryStore) CreateRelationshipWithOutbox(_ context.Context, rel RelationshipRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rels[rel.ID] = rel
	s.outbox = append(s.outbox, event)
	return nil
}

func (s *MemoryStore) SoftDeleteRelationshipsWithOutbox(_ context.Context, relationshipIDs []string, deletedAt time.Time) ([]RelationshipRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deleted := make([]RelationshipRecord, 0, len(relationshipIDs))
	for _, id := range relationshipIDs {
		rel, ok := s.rels[id]
		if !ok {
			continue
		}
		rel.IsDeleted = true
		s.rels[id] = rel
		deleted = append(deleted, rel)
	}
	_ = deletedAt
	return deleted, nil
}

func (s *MemoryStore) CreateOutboxEvents(_ context.Context, events []OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outbox = append(s.outbox, events...)
	return nil
}

func (s *MemoryStore) UpdateOutboxStatus(_ context.Context, eventID, status string, retryCount int, processedAt *time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.outbox {
		if s.outbox[i].ID != eventID {
			continue
		}
		s.outbox[i].Status = status
		s.outbox[i].RetryCount = retryCount
		s.outbox[i].ProcessedAt = processedAt
		return nil
	}
	return nil
}

func (s *MemoryStore) GetRelationshipByID(id string) (RelationshipRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rel, ok := s.rels[id]
	return rel, ok
}

func (s *MemoryStore) GetRelationshipsByIDs(ids []string) map[string]RelationshipRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]RelationshipRecord, len(ids))
	for _, id := range ids {
		if rel, ok := s.rels[id]; ok {
			result[id] = rel
		}
	}
	return result
}

func (s *MemoryStore) ListRelationships() []RelationshipRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RelationshipRecord, 0, len(s.rels))
	for _, rel := range s.rels {
		result = append(result, rel)
	}
	return result
}

func (s *MemoryStore) ListRelationshipsBatch(afterID string, limit int) []RelationshipRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]RelationshipRecord, 0, limit)
	rels := make([]RelationshipRecord, 0, len(s.rels))
	for _, rel := range s.rels {
		rels = append(rels, rel)
	}
	sort.Slice(rels, func(i, j int) bool {
		return rels[i].ID < rels[j].ID
	})
	for _, rel := range rels {
		if afterID != "" && rel.ID <= afterID {
			continue
		}
		result = append(result, rel)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func (s *MemoryStore) ListOutboxEvents() []OutboxEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]OutboxEvent, len(s.outbox))
	copy(result, s.outbox)
	return result
}

func (s *MemoryStore) ClaimOutboxBatch(_ context.Context, pageSize int) ([]OutboxEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	events := make([]OutboxEvent, 0, pageSize)
	for i := range s.outbox {
		if s.outbox[i].Status != "PENDING" {
			continue
		}
		s.outbox[i].Status = "PROCESSING"
		events = append(events, s.outbox[i])
		if pageSize > 0 && len(events) >= pageSize {
			break
		}
	}
	return events, nil
}

func (s *MemoryStore) GetOutboxEventByID(id string) (OutboxEvent, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, event := range s.outbox {
		if event.ID == id {
			return event, true
		}
	}
	return OutboxEvent{}, false
}

func (s *MemoryStore) UpsertProjectionVersion(_ context.Context, record ProjectionVersionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := record.EntityKind
	if s.projectionVersions[key] == nil {
		s.projectionVersions[key] = map[string]ProjectionVersionRecord{}
	}
	s.projectionVersions[key][record.EntityID] = record
	return nil
}

func graphIdentityKey(ownerTenantID, ownerAppID, graphScope string) string {
	return ownerTenantID + ":" + ownerAppID + ":" + graphScope
}

func (s *MemoryStore) SealGraphVersion(_ context.Context, request GraphVersionSealRequest) (GraphIdentityRecord, GraphVersionRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	key := graphIdentityKey(request.OwnerTenantID, request.OwnerAppID, request.GraphScope)
	identity, ok := s.graphIdentities[key]
	if !ok {
		identity = GraphIdentityRecord{
			IdentifierID:      newID("graph"),
			OwnerTenantID:     request.OwnerTenantID,
			OwnerAppID:        request.OwnerAppID,
			GraphScope:        request.GraphScope,
			HeadVersionNumber: 0,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		s.graphIdentities[key] = identity
	}
	identity.HeadVersionNumber++
	version := GraphVersionRecord{
		VersionID:       newID("graphver"),
		IdentifierID:    identity.IdentifierID,
		VersionNumber:   identity.HeadVersionNumber,
		ReferenceID:     request.ReferenceID,
		StorageClass:    fallback(request.StorageClass, "ONLINE"),
		VersionStatus:   fallback(request.VersionStatus, "SEALED"),
		ChangeSummary:   request.ChangeSummary,
		ManifestLocator: request.ManifestLocator,
		CreatedAt:       now,
		SealedAt:        now,
	}
	identity.HeadVersionID = version.VersionID
	identity.UpdatedAt = now
	s.graphIdentities[key] = identity
	s.graphVersions[identity.IdentifierID] = append(s.graphVersions[identity.IdentifierID], version)
	if len(request.Entities) > 0 {
		s.graphVersionEntities[version.VersionID] = append(s.graphVersionEntities[version.VersionID], request.Entities...)
	}
	return identity, version, nil
}

func (s *MemoryStore) FinalizeGraphVersion(_ context.Context, versionID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for identityKey, identity := range s.graphIdentities {
		versions := s.graphVersions[identity.IdentifierID]
		for idx := range versions {
			if versions[idx].VersionID != versionID {
				continue
			}
			if !strings.EqualFold(versions[idx].VersionStatus, "PENDING_ENTITIES") {
				return 0, nil
			}
			versions[idx].VersionStatus = "SEALED"
			if versions[idx].SealedAt.IsZero() {
				versions[idx].SealedAt = now
			}
			s.graphVersions[identity.IdentifierID] = versions
			identity.UpdatedAt = now
			if identity.HeadVersionID == versionID {
				identity.HeadVersionID = versionID
			}
			s.graphIdentities[identityKey] = identity
			return 1, nil
		}
	}
	return 0, nil
}

func (s *MemoryStore) AddGraphVersionEntities(_ context.Context, versionID string, entities []GraphVersionEntityRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(entities) == 0 {
		return nil
	}
	indexByKey := map[string]int{}
	for idx, entity := range s.graphVersionEntities[versionID] {
		indexByKey[entity.EntityKind+":"+entity.EntityID] = idx
	}
	for _, entity := range entities {
		key := entity.EntityKind + ":" + entity.EntityID
		if idx, ok := indexByKey[key]; ok {
			s.graphVersionEntities[versionID][idx] = entity
			continue
		}
		indexByKey[key] = len(s.graphVersionEntities[versionID])
		s.graphVersionEntities[versionID] = append(s.graphVersionEntities[versionID], entity)
	}
	return nil
}

func (s *MemoryStore) AbandonGraphVersion(_ context.Context, versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for identityKey, identity := range s.graphIdentities {
		versions := s.graphVersions[identity.IdentifierID]
		for idx := range versions {
			if versions[idx].VersionID != versionID {
				continue
			}
			versions[idx].VersionStatus = "ABANDONED"
			if versions[idx].SealedAt.IsZero() {
				versions[idx].SealedAt = now
			}
			s.graphVersions[identity.IdentifierID] = versions
			identity.UpdatedAt = now
			s.graphIdentities[identityKey] = identity
			return nil
		}
	}
	return nil
}

func (s *MemoryStore) CleanupExpiredSyncSession(_ context.Context, versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var identity GraphIdentityRecord
	found := false
	for identityKey, candidate := range s.graphIdentities {
		versions := s.graphVersions[candidate.IdentifierID]
		for idx := range versions {
			if versions[idx].VersionID != versionID {
				continue
			}
			versions[idx].VersionStatus = "ABANDONED"
			if versions[idx].SealedAt.IsZero() {
				versions[idx].SealedAt = time.Now().UTC()
			}
			s.graphVersions[candidate.IdentifierID] = versions
			identity = candidate
			found = true
			_ = identityKey
			break
		}
		if found {
			break
		}
	}
	if !found {
		return nil
	}
	key := scopeLeaseKey(identity.OwnerTenantID, identity.OwnerAppID, identity.GraphScope)
	if lease, ok := s.scopeLeases[key]; ok && lease.VersionID == versionID {
		delete(s.scopeLeases, key)
	}
	return nil
}

func (s *MemoryStore) GetGraphVersionByID(_ context.Context, versionID string) (GraphVersionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, versions := range s.graphVersions {
		for _, version := range versions {
			if version.VersionID == versionID {
				return version, true
			}
		}
	}
	return GraphVersionRecord{}, false
}

func (s *MemoryStore) ListPendingGraphVersionsBefore(cutoff time.Time) []GraphVersionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]GraphVersionRecord, 0)
	for _, versions := range s.graphVersions {
		for _, version := range versions {
			if !strings.EqualFold(version.VersionStatus, "PENDING_ENTITIES") {
				continue
			}
			if !cutoff.IsZero() && !version.CreatedAt.Before(cutoff) {
				continue
			}
			result = append(result, version)
		}
	}
	return result
}

func (s *MemoryStore) GetGraphIdentityByID(_ context.Context, identifierID string) (GraphIdentityRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, identity := range s.graphIdentities {
		if identity.IdentifierID == identifierID {
			return identity, true
		}
	}
	return GraphIdentityRecord{}, false
}

func scopeLeaseKey(ownerTenantID, ownerAppID, graphScope string) string {
	return graphIdentityKey(ownerTenantID, ownerAppID, graphScope)
}

func (s *MemoryStore) AcquireScopeLease(_ context.Context, ownerTenantID, ownerAppID, graphScope, versionID string, expiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scopeLeaseKey(ownerTenantID, ownerAppID, graphScope)
	now := time.Now().UTC()
	if existing, ok := s.scopeLeases[key]; ok {
		if existing.VersionID != versionID && existing.ExpiresAt.After(now) {
			return ErrScopeLocked
		}
		delete(s.scopeLeases, key)
	}
	s.scopeLeases[key] = ScopeLeaseRecord{
		OwnerTenantID: ownerTenantID,
		OwnerAppID:    ownerAppID,
		GraphScope:    graphScope,
		VersionID:     versionID,
		ExpiresAt:     expiresAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	return nil
}

func (s *MemoryStore) ReleaseScopeLease(_ context.Context, ownerTenantID, ownerAppID, graphScope, versionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := scopeLeaseKey(ownerTenantID, ownerAppID, graphScope)
	if existing, ok := s.scopeLeases[key]; ok && existing.VersionID == versionID {
		delete(s.scopeLeases, key)
	}
	return nil
}

func (s *MemoryStore) GetScopeLease(_ context.Context, ownerTenantID, ownerAppID, graphScope string) (ScopeLeaseRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := scopeLeaseKey(ownerTenantID, ownerAppID, graphScope)
	lease, ok := s.scopeLeases[key]
	return lease, ok
}

func (s *MemoryStore) GetGraphVersionEntities(versionID string) []GraphVersionEntityRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]GraphVersionEntityRecord(nil), s.graphVersionEntities[versionID]...)
}

func (s *MemoryStore) UpsertGraphProjectionHead(_ context.Context, record GraphProjectionHeadRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.graphHeads[record.IdentifierID] == nil {
		s.graphHeads[record.IdentifierID] = map[string]map[string]GraphProjectionHeadRecord{}
	}
	if s.graphHeads[record.IdentifierID][record.BackendKind] == nil {
		s.graphHeads[record.IdentifierID][record.BackendKind] = map[string]GraphProjectionHeadRecord{}
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now().UTC()
	}
	if existing, ok := s.graphHeads[record.IdentifierID][record.BackendKind][record.BackendName]; ok {
		if existing.AppliedVersionNumber > record.AppliedVersionNumber {
			return nil
		}
	}
	s.graphHeads[record.IdentifierID][record.BackendKind][record.BackendName] = record
	return nil
}

func (s *MemoryStore) GetGraphProjectionHead(identifierID, backendKind, backendName string) (GraphProjectionHeadRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	byKind, ok := s.graphHeads[identifierID]
	if !ok {
		return GraphProjectionHeadRecord{}, false
	}
	byName, ok := byKind[backendKind]
	if !ok {
		return GraphProjectionHeadRecord{}, false
	}
	record, ok := byName[backendName]
	return record, ok
}

func (s *MemoryStore) GetProjectionVersion(entityID, entityKind string) (ProjectionVersionRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if kindVersions, ok := s.projectionVersions[entityKind]; ok {
		if record, ok := kindVersions[entityID]; ok {
			return record, true
		}
	}
	return ProjectionVersionRecord{}, false
}

func (s *MemoryStore) ListProjectionVersions() []ProjectionVersionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]ProjectionVersionRecord, 0)
	for _, kindVersions := range s.projectionVersions {
		for _, record := range kindVersions {
			result = append(result, record)
		}
	}
	return result
}

func (s *MemoryStore) ListProjectionVersionsBatch(afterEntityKind, afterEntityID string, limit int) []ProjectionVersionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]ProjectionVersionRecord, 0)
	for _, kindVersions := range s.projectionVersions {
		for _, record := range kindVersions {
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].EntityKind == records[j].EntityKind {
			return records[i].EntityID < records[j].EntityID
		}
		return records[i].EntityKind < records[j].EntityKind
	})
	result := make([]ProjectionVersionRecord, 0, limit)
	for _, record := range records {
		if afterEntityKind != "" {
			if record.EntityKind < afterEntityKind || (record.EntityKind == afterEntityKind && record.EntityID <= afterEntityID) {
				continue
			}
		}
		result = append(result, record)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}
