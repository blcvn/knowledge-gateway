package saga

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"graphiti-pipeline/internal/domain/ingestion"
	"graphiti-pipeline/internal/usecase/port"
)

type MockSagaRepo struct {}
func (m *MockSagaRepo) Save(ctx context.Context, saga ingestion.Saga) error { return nil }
func (m *MockSagaRepo) UpdateState(ctx context.Context, id string, state ingestion.SagaState, step ingestion.PipelineStep) error { return nil }

type MockPublisher struct {
	Failed bool
	Success bool
}
func (m *MockPublisher) PublishEpisodeIngested(ctx context.Context, event ingestion.EpisodeIngested) error { m.Success = true; return nil }
func (m *MockPublisher) PublishEpisodeFailed(ctx context.Context, event ingestion.EpisodeFailed) error { m.Failed = true; return nil }

type MockStore struct {}
func (m *MockStore) SaveBulk(ctx context.Context, req port.SaveBulkRequest) error { return nil }
func (m *MockStore) RollbackBulk(ctx context.Context, id string) error { return nil }

type MockLock struct {}
func (m *MockLock) Acquire(ctx context.Context, groupID ingestion.GroupID) (func(), error) { return func() {}, nil }

type MockKnowledge struct {
	FailOnExtract bool
}
func (m *MockKnowledge) ExtractEntities(ctx context.Context, ep ingestion.Episode) error { 
	if m.FailOnExtract {
		return errors.New("simulated error")
	}
	return nil 
}
func (m *MockKnowledge) ResolveEntities(ctx context.Context, ep ingestion.Episode) error { return nil }
func (m *MockKnowledge) ExtractEdges(ctx context.Context, ep ingestion.Episode) error { return nil }
func (m *MockKnowledge) ResolveEdges(ctx context.Context, ep ingestion.Episode) error { return nil }
func (m *MockKnowledge) GenerateEmbeddings(ctx context.Context, ep ingestion.Episode) error { return nil }
func (m *MockKnowledge) UpdateCommunity(ctx context.Context, ep ingestion.Episode) error { return nil }

func TestSagaOrchestrator_Execute_Success(t *testing.T) {
	pub := &MockPublisher{}
	orc := NewSagaOrchestrator(
		&MockSagaRepo{}, pub, &MockStore{}, &MockLock{}, &MockKnowledge{},
		zap.NewNop(), otel.Tracer("test"),
	)
	err := orc.Execute(context.Background(), ingestion.Episode{ID: "ep-1", GroupID: "group-1"})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !pub.Success {
		t.Error("expected PublishEpisodeIngested to be called")
	}
}

func TestSagaOrchestrator_Execute_Failure(t *testing.T) {
	pub := &MockPublisher{}
	orc := NewSagaOrchestrator(
		&MockSagaRepo{}, pub, &MockStore{}, &MockLock{}, &MockKnowledge{FailOnExtract: true},
		zap.NewNop(), otel.Tracer("test"),
	)
	err := orc.Execute(context.Background(), ingestion.Episode{ID: "ep-1", GroupID: "group-1"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !pub.Failed {
		t.Error("expected PublishEpisodeFailed to be called")
	}
}
