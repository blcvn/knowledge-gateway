package tests

import (
	"context"

	"github.com/stretchr/testify/mock"
	"vnp-memory/services/graphiti-knowledge/domain"
)

type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string, model string) (string, domain.TokenUsage, error) {
	args := m.Called(ctx, prompt, model)
	return args.String(0), args.Get(1).(domain.TokenUsage), args.Error(2)
}

type MockPromptRegistry struct {
	mock.Mock
}

func (m *MockPromptRegistry) Render(templateID string, vars map[string]interface{}) (string, error) {
	args := m.Called(templateID, vars)
	return args.String(0), args.Error(1)
}

func (m *MockPromptRegistry) GetModel(templateID string) string {
	args := m.Called(templateID)
	return args.String(0)
}

func (m *MockPromptRegistry) List() []domain.PromptTemplate {
	return nil
}

type MockParser struct {
	mock.Mock
}

func (m *MockParser) ParseJSON(input string) string {
	args := m.Called(input)
	return args.String(0)
}

type MockEmbedderClient struct {
	mock.Mock
}

func (m *MockEmbedderClient) Embed(ctx context.Context, text string, model string) (domain.EmbeddingVector, error) {
	args := m.Called(ctx, text, model)
	return args.Get(0).(domain.EmbeddingVector), args.Error(1)
}

func (m *MockEmbedderClient) EmbedBatch(ctx context.Context, texts []string, model string) ([]domain.EmbeddingVector, error) {
	args := m.Called(ctx, texts, model)
	return args.Get(0).([]domain.EmbeddingVector), args.Error(1)
}

type MockGraphReader struct {
	mock.Mock
}

func (m *MockGraphReader) FindSimilarEntities(ctx context.Context, embedding domain.EmbeddingVector, threshold float64) ([]domain.ExtractedEntity, error) {
	args := m.Called(ctx, embedding, threshold)
	return args.Get(0).([]domain.ExtractedEntity), args.Error(1)
}

func (m *MockGraphReader) GetEntityByName(ctx context.Context, name string, groupID string) (*domain.ExtractedEntity, error) {
	args := m.Called(ctx, name, groupID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ExtractedEntity), args.Error(1)
}

func (m *MockGraphReader) FindSimilarEdges(ctx context.Context, embedding domain.EmbeddingVector, threshold float64) ([]domain.ExtractedEdge, error) {
	args := m.Called(ctx, embedding, threshold)
	return args.Get(0).([]domain.ExtractedEdge), args.Error(1)
}
