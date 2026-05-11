package tests

import (
	"context"
	"testing"
	"time"

	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/usecase"
)

// MockSagaRepo implements usecase.SagaStateRepo
type MockSagaRepo struct {
	states map[string]*domain.SagaState
}

func NewMockSagaRepo() *MockSagaRepo {
	return &MockSagaRepo{states: make(map[string]*domain.SagaState)}
}

func (m *MockSagaRepo) Create(ctx context.Context, state *domain.SagaState) error {
	m.states[state.ID] = state
	return nil
}

func (m *MockSagaRepo) Get(ctx context.Context, id string) (*domain.SagaState, error) {
	return m.states[id], nil
}

func (m *MockSagaRepo) Update(ctx context.Context, state *domain.SagaState) error {
	m.states[state.ID] = state
	return nil
}

func (m *MockSagaRepo) GetStuckSagas(ctx context.Context, timeoutMinutes int, limit int) ([]*domain.SagaState, error) {
	return nil, nil
}

// MockEpisodeRepo
type MockEpisodeRepo struct{}
func (m *MockEpisodeRepo) Create(ctx context.Context, episode *domain.Episode) error { return nil }
func (m *MockEpisodeRepo) GetByHash(ctx context.Context, contentHash string) (*domain.Episode, error) { return nil, nil }

// MockEventPublisher
type MockEventPublisher struct {
	PublishedEvents []domain.DomainEvent
}
func (m *MockEventPublisher) Publish(ctx context.Context, event domain.DomainEvent) error {
	m.PublishedEvents = append(m.PublishedEvents, event)
	return nil
}

// MockKnowledgeClient
type MockKnowledgeClient struct{}
func (m *MockKnowledgeClient) ExtractEntities(ctx context.Context, episode domain.Episode) ([]map[string]interface{}, error) { return []map[string]interface{}{}, nil }
func (m *MockKnowledgeClient) ResolveEntities(ctx context.Context, groupID string, entities []map[string]interface{}) error { return nil }
func (m *MockKnowledgeClient) ExtractEdges(ctx context.Context, episode domain.Episode, entities []map[string]interface{}) ([]map[string]interface{}, error) { return []map[string]interface{}{}, nil }
func (m *MockKnowledgeClient) ResolveEdges(ctx context.Context, groupID string, edges []map[string]interface{}) error { return nil }
func (m *MockKnowledgeClient) UpdateCommunity(ctx context.Context, groupID string) error { return nil }

// MockStoreClient
type MockStoreClient struct{}
func (m *MockStoreClient) SaveBulk(ctx context.Context, groupID string, data map[string]interface{}) error { return nil }
func (m *MockStoreClient) RollbackBulk(ctx context.Context, groupID string, sagaID string) error { return nil }

func TestIngestEpisode_EndToEndSagaFlow(t *testing.T) {
	ctx := context.Background()

	mockSagaRepo := NewMockSagaRepo()
	mockEpisodeRepo := &MockEpisodeRepo{}
	mockPub := &MockEventPublisher{}
	mockKnow := &MockKnowledgeClient{}
	mockStore := &MockStoreClient{}

	orchestrator := usecase.NewSagaOrchestrator(mockKnow, mockStore, mockSagaRepo, mockPub)
	ingestUC := usecase.NewIngestEpisodeUseCase(mockEpisodeRepo, orchestrator)

	ep, err := ingestUC.Execute(ctx, "Test Episode", "group-1", "Hello World", domain.SourceText, time.Now())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if ep == nil || ep.SagaID == nil {
		t.Fatalf("expected episode with sagaID")
	}

	// Give the orchestrator goroutine a moment to finish (since executeSaga is async in this scaffold)
	time.Sleep(100 * time.Millisecond)

	state, _ := mockSagaRepo.Get(ctx, *ep.SagaID)
	if state.Status != domain.SagaStatusCompleted {
		t.Errorf("expected saga status to be completed, got %s", state.Status)
	}

	if len(mockPub.PublishedEvents) == 0 {
		t.Errorf("expected events to be published")
	}
}
