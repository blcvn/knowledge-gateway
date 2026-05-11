package engine_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/engine"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/ingestion"
	usecase "github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/usecase/engine"
)

// Mock implementations for testing would go here
type mockLLM struct{ calls int }
func (m *mockLLM) ExtractTopics(ctx context.Context, content string) ([]engine.TopicEntry, error) { m.calls++; return nil, nil }
func (m *mockLLM) GenerateGist(ctx context.Context, content string) (*engine.EventGist, error) { m.calls++; return &engine.EventGist{}, nil }
func (m *mockLLM) MergeTraits(ctx context.Context, existing map[string]any, newContent string) (map[string]any, error) { m.calls++; return nil, nil }

type mockProfile struct{}
func (m *mockProfile) FindByUser(ctx context.Context, tenantID, userID uuid.UUID) (*engine.Profile, error) { return &engine.Profile{}, nil }
func (m *mockProfile) Upsert(ctx context.Context, profile *engine.Profile) error { return nil }

type mockGist struct{}
func (m *mockGist) Create(ctx context.Context, gist *engine.EventGist) error { return nil }
func (m *mockGist) FindByUser(ctx context.Context, t, u uuid.UUID, limit int) ([]engine.EventGist, error) { return nil, nil }

type mockPub struct{}
func (m *mockPub) PublishBlobIngested(ctx context.Context, t, u uuid.UUID) error { return nil }
func (m *mockPub) PublishFlushCompleted(ctx context.Context, t, u uuid.UUID) error { return nil }

func TestYOLOMerge_EnforcesExactLLMCalls(t *testing.T) {
	llm := &mockLLM{}
	svc := usecase.NewService(llm, &mockProfile{}, &mockGist{}, &mockPub{})

	blobs := []ingestion.Blob{{Content: "hello world"}}
	res, err := svc.YOLOMerge(context.Background(), uuid.New(), uuid.New(), blobs)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if llm.calls != 3 {
		t.Fatalf("expected exactly 3 LLM calls, got %d", llm.calls)
	}
	if res.LLMCalls != 3 {
		t.Fatalf("expected result LLMCalls to be 3, got %d", res.LLMCalls)
	}
}
