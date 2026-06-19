package write

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrDuplicateExternalRef = errors.New("duplicate external_ref")
var ErrNodeNotFound = errors.New("node not found")

type MemoryStore struct {
	mu                 sync.RWMutex
	nodes              map[string]NodeRecord
	externalRefs       map[string]string
	rels               map[string]RelationshipRecord
	outbox             []OutboxEvent
	projectionVersions map[string]map[string]ProjectionVersionRecord
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nodes:              map[string]NodeRecord{},
		externalRefs:       map[string]string{},
		rels:               map[string]RelationshipRecord{},
		outbox:             []OutboxEvent{},
		projectionVersions: map[string]map[string]ProjectionVersionRecord{},
	}
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

func (s *MemoryStore) GetNodeByID(id string) (NodeRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[id]
	return node, ok
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

func (s *MemoryStore) ListNodes() []NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]NodeRecord, 0, len(s.nodes))
	for _, node := range s.nodes {
		result = append(result, node)
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

func (s *MemoryStore) CreateRelationshipWithOutbox(_ context.Context, rel RelationshipRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rels[rel.ID] = rel
	s.outbox = append(s.outbox, event)
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

func (s *MemoryStore) ListRelationships() []RelationshipRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RelationshipRecord, 0, len(s.rels))
	for _, rel := range s.rels {
		result = append(result, rel)
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
