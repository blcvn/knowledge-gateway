package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"vnp-memory/services/graphiti-knowledge/internal/domain"
	"vnp-memory/services/graphiti-knowledge/internal/usecase"
	"vnp-memory/services/graphiti-knowledge/internal/usecase/dto"
	"vnp-memory/services/graphiti-knowledge/tests"
)

func TestExtractEntitiesUseCase_Execute_Success(t *testing.T) {
	// Arrange
	mockLLM := new(tests.MockLLMClient)
	mockRegistry := new(tests.MockPromptRegistry)
	mockParser := new(tests.MockParser)

	uc := usecase.NewExtractEntitiesUseCase(mockLLM, mockRegistry, mockParser)

	req := dto.ExtractEntitiesRequest{
		Content: "Elon Musk founded SpaceX.",
	}

	mockRegistry.On("Render", "extract_entities", mock.Anything).Return("Prompt: Elon Musk founded SpaceX.", nil)
	mockRegistry.On("GetModel", "extract_entities").Return("gpt-4o")

	mockLLM.On("Complete", mock.Anything, "Prompt: Elon Musk founded SpaceX.", "gpt-4o").
		Return("```json\n{\"entities\": [{\"name\": \"Elon Musk\", \"label\": \"Person\"}]}\n```", domain.TokenUsage{TotalTokens: 100}, nil)

	mockParser.On("ParseJSON", mock.Anything).Return(`{"entities": [{"name": "Elon Musk", "label": "Person"}]}`)

	// Act
	entities, usage, err := uc.Execute(context.Background(), req)

	// Assert
	assert.NoError(t, err)
	assert.Equal(t, 100, usage.TotalTokens)
	assert.Len(t, entities, 1)
	assert.Equal(t, "Elon Musk", entities[0].Name)
	assert.Equal(t, "Person", entities[0].Label)

	mockLLM.AssertExpectations(t)
	mockRegistry.AssertExpectations(t)
	mockParser.AssertExpectations(t)
}
