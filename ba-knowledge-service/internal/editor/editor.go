package editor

import (
	"context"
	"fmt"
	"time"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/domain"
	"github.com/blcvn/backend/services/ba-knowledge-service/internal/editor/validator"
)

// ValidatorAgent orchestrates document validation and knowledge graph updates
type ValidatorAgent struct {
	templateLoader *validator.TemplateLoader
	validator      *validator.DocumentValidator
	llmClient      domain.LLMService
	config         *AgentConfig
}

// NewValidatorAgent creates a new validator agent
func NewValidatorAgent(
	templateLoader *validator.TemplateLoader,
	llmClient domain.LLMService,
	config *AgentConfig,
) *ValidatorAgent {
	return &ValidatorAgent{
		templateLoader: templateLoader,
		llmClient:      llmClient,
		config:         config,
	}
}

type AgentConfig struct {
	EnableKGUpdate    bool
	EnableAutoFix     bool
	MaxRetries        int
	ValidationTimeout time.Duration
	KGUpdateTimeout   time.Duration
}

// ValidationContext holds context for the validation process
type ValidationContext struct {
	Document   *domain.Document
	NewContent string
	Template   *validator.Template
}

// AgentResult represents the final result of agent processing
type AgentResult struct {
	Success          bool
	ValidationResult *validator.ValidationResult
	Errors           []AgentError
	Warnings         []AgentWarning
	Metadata         map[string]interface{}
}

type AgentError struct {
	Type        string
	Message     string
	SectionID   string
	Recoverable bool
	Suggestion  string
}

type AgentWarning struct {
	Type    string
	Message string
	Context string
}

// Execute runs the complete validation and update pipeline
func (a *ValidatorAgent) Execute(ctx context.Context, doc *domain.Document, newContent string) (*AgentResult, error) {
	result := &AgentResult{
		Metadata: make(map[string]interface{}),
	}

	// ========================================
	// PHASE 1: PREPARATION & VALIDATION
	// ========================================

	// Step 1.1: Determine template type based on document tier
	template, err := a.determineTemplate(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to determine template: %w", err)
	}

	// Step 1.2: Create validation context
	validationCtx := &ValidationContext{
		Document:   doc,
		NewContent: newContent,
		Template:   template,
	}

	// Step 1.3: Validate document structure
	validationResult, err := a.validateDocument(ctx, validationCtx)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	result.ValidationResult = validationResult

	// Step 1.4: Check if validation passed
	if !validationResult.IsValid {
		result.Success = false
		result.Errors = a.convertValidationErrors(validationResult.Errors)

		// Auto-fix logic removed for migration simplicity - re-implement if needed using ChatAgent

		if !result.Success {
			return result, nil // Return with validation errors
		}
	}
	result.Success = true
	return result, nil
}

// determineTemplate selects the appropriate template based on document tier
func (a *ValidatorAgent) determineTemplate(doc *domain.Document) (*validator.Template, error) {
	var templateType string

	switch doc.Tier {
	case domain.TierURDIndex:
		templateType = "urd_index"
	case domain.TierURDOutline:
		templateType = "urd_outline"
	case domain.TierURDFull:
		templateType = "urd_full"
	default:
		return nil, fmt.Errorf("unknown document tier: %s", doc.Tier)
	}

	template, err := a.templateLoader.GetTemplate(templateType)
	if err != nil {
		return nil, fmt.Errorf("template not found for type %s: %w", templateType, err)
	}

	return template, nil
}

// validateDocument performs structural validation
func (a *ValidatorAgent) validateDocument(ctx context.Context, valCtx *ValidationContext) (*validator.ValidationResult, error) {
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, a.config.ValidationTimeout)
	defer cancel()

	// Initialize validator
	vali := validator.NewDocumentValidator(valCtx.NewContent, valCtx.Template)

	// Run validation
	result := vali.Validate()

	// Enhance with semantic validation
	// semanticErrors := a.performSemanticValidation(ctx, valCtx)
	// result.Errors = append(result.Errors, semanticErrors...)

	// Update IsValid flag
	result.IsValid = len(result.Errors) == 0

	return &result, nil
}
