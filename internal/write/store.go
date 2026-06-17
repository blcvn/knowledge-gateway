package write

import (
	"errors"
	"sync"
)

var ErrDuplicateExternalRef = errors.New("duplicate external_ref")
var ErrNodeNotFound = errors.New("node not found")

type Store interface {
	CreateNodeWithOutbox(node NodeRecord, event OutboxEvent) error
	CreateNodeBundle(node NodeRecord, rels []RelationshipRecord, event OutboxEvent) error
	GetNodeByID(id string) (NodeRecord, bool)
	GetNodeByExternalRef(externalRef string) (NodeRecord, bool)
	ListNodes() []NodeRecord
	UpdateNodeWithOutbox(node NodeRecord, event OutboxEvent) error
	SoftDeleteNodeWithOutbox(node NodeRecord, event OutboxEvent) error
	CreateRelationshipWithOutbox(rel RelationshipRecord, event OutboxEvent) error
	GetRelationshipByID(id string) (RelationshipRecord, bool)
	ListRelationships() []RelationshipRecord
	ListOutboxEvents() []OutboxEvent
}

type MemoryStore struct {
	mu           sync.RWMutex
	nodes        map[string]NodeRecord
	externalRefs map[string]string
	rels         map[string]RelationshipRecord
	outbox       []OutboxEvent
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nodes:        map[string]NodeRecord{},
		externalRefs: map[string]string{},
		rels:         map[string]RelationshipRecord{},
		outbox:       []OutboxEvent{},
	}
}

func (s *MemoryStore) CreateNodeWithOutbox(node NodeRecord, event OutboxEvent) error {
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

func (s *MemoryStore) CreateNodeBundle(node NodeRecord, rels []RelationshipRecord, event OutboxEvent) error {
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

func (s *MemoryStore) UpdateNodeWithOutbox(node NodeRecord, event OutboxEvent) error {
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

func (s *MemoryStore) SoftDeleteNodeWithOutbox(node NodeRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.nodes[node.ID]; !ok {
		return ErrNodeNotFound
	}

	s.nodes[node.ID] = node
	s.outbox = append(s.outbox, event)
	return nil
}

func (s *MemoryStore) CreateRelationshipWithOutbox(rel RelationshipRecord, event OutboxEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rels[rel.ID] = rel
	s.outbox = append(s.outbox, event)
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
