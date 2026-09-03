package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

// Re-use MockEpisodeRepo from integration tests or create a simple one here
type MockSimpleEpisodeRepo struct {
	Count int
}

func (m *MockSimpleEpisodeRepo) Create(ctx context.Context, episode *domain.Episode) error {
	m.Count++
	return nil
}
func (m *MockSimpleEpisodeRepo) GetByHash(ctx context.Context, contentHash string) (*domain.Episode, error) {
	return nil, nil
}

// Mock Orchestrator interface replacement (SagaOrchestrator is a struct, but we'll use a mocked repo for its internals)
// Since SagaOrchestrator is a concrete type, we mock its dependencies.
func TestBulkIngest_Execute(t *testing.T) {
	ctx := context.Background()

	mockRepo := &MockSimpleEpisodeRepo{}
	
	// Quick mock components for the orchestrator
	mockPub := &mockPublisher{}
	mockKnow := &mockKnowledge{}
	mockStore := &mockStore{}
	mockSaga := &mockSagaRepo{}

	orch := NewSagaOrchestrator(mockKnow, mockStore, mockSaga, mockPub)
	bulkUC := NewBulkIngestUseCase(mockRepo, orch)

	inputs := []BulkIngestInput{
		{Name: "Ep 1", Body: "Body 1", Source: domain.SourceText, ReferenceTime: time.Now()},
		{Name: "Ep 2", Body: "Body 2", Source: domain.SourceText, ReferenceTime: time.Now()},
		{Name: "Ep 3", Body: "Body 3", Source: domain.SourceText, ReferenceTime: time.Now()},
	}

	eps, err := bulkUC.Execute(ctx, "test-group", inputs)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(eps) != 3 {
		t.Errorf("expected 3 episodes, got %d", len(eps))
	}

	if mockRepo.Count != 3 {
		t.Errorf("expected 3 creations, got %d", mockRepo.Count)
	}

	// Sagas should have been started
	if eps[0].SagaID == nil {
		t.Errorf("expected saga ID to be assigned")
	}
}

// Mock Implementations
type mockPublisher struct{}
func (m *mockPublisher) Publish(ctx context.Context, event domain.DomainEvent) error { return nil }

type mockKnowledge struct{}
func (m *mockKnowledge) ExtractEntities(ctx context.Context, episode domain.Episode) ([]map[string]interface{}, error) { return nil, nil }
func (m *mockKnowledge) ResolveEntities(ctx context.Context, groupID string, entities []map[string]interface{}) error { return nil }
func (m *mockKnowledge) ExtractEdges(ctx context.Context, episode domain.Episode, entities []map[string]interface{}) ([]map[string]interface{}, error) { return nil, nil }
func (m *mockKnowledge) ResolveEdges(ctx context.Context, groupID string, edges []map[string]interface{}) error { return nil }
func (m *mockKnowledge) UpdateCommunity(ctx context.Context, groupID string) error { return nil }

type mockStore struct{}
func (m *mockStore) SaveBulk(ctx context.Context, groupID string, data map[string]interface{}) error { return nil }
func (m *mockStore) RollbackBulk(ctx context.Context, groupID string, sagaID string) error { return nil }

type mockSagaRepo struct{}
func (m *mockSagaRepo) Create(ctx context.Context, state *domain.SagaState) error { return nil }
func (m *mockSagaRepo) Get(ctx context.Context, id string) (*domain.SagaState, error) { return nil, nil }
func (m *mockSagaRepo) Update(ctx context.Context, state *domain.SagaState) error { return nil }
func (m *mockSagaRepo) GetStuckSagas(ctx context.Context, timeoutMinutes int, limit int) ([]*domain.SagaState, error) { return nil, nil }
