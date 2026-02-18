package editor

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/domain"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/editor/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLLM is a mock implementation of domain.LLMService
type MockLLM struct {
	mock.Mock
}

func (m *MockLLM) Chat(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	args := m.Called(ctx, systemPrompt, userPrompt)
	return args.String(0), args.Error(1)
}

// setupTestEnvironment creates a temporary directory with template files
func setupTestEnvironment(t *testing.T) (string, func()) {
	tmpDir, err := ioutil.TempDir("", "editor_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	// Create a dummy urd_index.yaml template
	templateContent := `
name: "URD Index"
type: "urd_index"
version: "1.0"
description: "Test Template"
metadata:
  - name: "Module"
    pattern: "Module: .+"
    required: true

sections:
  - id: "section-1"
    title: "Section 1"
    level: 1
    type: "text"
    required: true
    pattern: "^# Section 1$"
`
	err = ioutil.WriteFile(filepath.Join(tmpDir, "urd_index.yaml"), []byte(templateContent), 0644)
	if err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

func TestValidatorAgent_Execute(t *testing.T) {
	templatesDir, cleanup := setupTestEnvironment(t)
	defer cleanup()

	templateLoader := validator.NewTemplateLoader(templatesDir)
	err := templateLoader.LoadAllTemplates()
	assert.NoError(t, err)

	mockLLM := new(MockLLM)

	config := &AgentConfig{
		EnableKGUpdate:    false,
		EnableAutoFix:     false,
		ValidationTimeout: 5 * time.Second,
	}

	agent := NewValidatorAgent(templateLoader, mockLLM, config) // Uses factory

	t.Run("Success Validation", func(t *testing.T) {
		doc := &domain.Document{
			Tier: domain.TierURDIndex,
		}
		newContent := `
Module: Test Module
# Section 1
Content here.
`
		ctx := context.Background()
		result, err := agent.Execute(ctx, doc, newContent)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.ValidationResult)
		assert.Empty(t, result.Errors)
	})

	t.Run("Validation Failure - Missing Metadata", func(t *testing.T) {
		doc := &domain.Document{
			Tier: domain.TierURDIndex,
		}
		// Missing metadata "Module: ..."
		newContent := `
# Section 1
Content here.
`
		ctx := context.Background()
		result, err := agent.Execute(ctx, doc, newContent)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Errors)

		found := false
		for _, e := range result.Errors {
			if e.Type == "METADATA_ERROR" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected METADATA_ERROR")
	})

	t.Run("Validation Failure - Missing Section", func(t *testing.T) {
		doc := &domain.Document{
			Tier: domain.TierURDIndex,
		}
		// Missing Section 1
		newContent := `
Module: Test Module
# Section 2
Wrong section.
`
		ctx := context.Background()
		result, err := agent.Execute(ctx, doc, newContent)

		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.NotEmpty(t, result.Errors)

		found := false
		for _, e := range result.Errors {
			if e.Type == "STRUCTURE_ERROR" && e.SectionID == "section-1" {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected STRUCTURE_ERROR for section-1")
	})
}
