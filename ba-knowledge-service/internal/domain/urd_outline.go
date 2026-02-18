package v32

import "time"

// ActorType represents the type of actor
type ActorType string

const (
	ActorTypeHuman          ActorType = "human"
	ActorTypeSystem         ActorType = "system"
	ActorTypeExternalSystem ActorType = "external_system"
)

// Actor represents an actor in the URD outline
type Actor struct {
	ID              string    `json:"id"` // Pattern: ACT-001
	Name            string    `json:"name"`
	Role            string    `json:"role"`
	Type            ActorType `json:"type"`
	Description     string    `json:"description"`
	RelatedPersonas []string  `json:"related_personas"` // Persona IDs
	Capabilities    []string  `json:"capabilities"`
}

// UseCasePriority represents use case priority
type UseCasePriority string

const (
	UseCasePriorityMustHave   UseCasePriority = "must_have"
	UseCasePriorityShouldHave UseCasePriority = "should_have"
	UseCasePriorityCouldHave  UseCasePriority = "could_have"
)

// UseCase represents a use case in the URD outline
type UseCase struct {
	ID                 string          `json:"id"` // Pattern: UC-001
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	PrimaryActor       string          `json:"primary_actor"`    // Actor ID
	SecondaryActors    []string        `json:"secondary_actors"` // Actor IDs
	Preconditions      []string        `json:"preconditions"`
	Postconditions     []string        `json:"postconditions"`
	MainFlow           []string        `json:"main_flow"`
	AlternativeFlows   []string        `json:"alternative_flows"`
	ExceptionFlows     []string        `json:"exception_flows"`
	RelatedUserStories []string        `json:"related_user_stories"` // UserStory IDs
	BusinessRules      []string        `json:"business_rules"`       // BusinessRule IDs
	Priority           UseCasePriority `json:"priority"`
}

// OutlineIntegration represents an integration in the URD outline
type OutlineIntegration struct {
	IntegrationID  string   `json:"integration_id"`    // Integration ID from PRD
	UsedInUseCases []string `json:"used_in_use_cases"` // UseCase IDs
	DataExchanged  []string `json:"data_exchanged"`
	ErrorHandling  string   `json:"error_handling"`
}

// EntitySource represents where the entity was derived from
type EntitySource string

const (
	EntitySourceFromPRD  EntitySource = "from_prd"
	EntitySourceDerived  EntitySource = "derived"
	EntitySourceInferred EntitySource = "inferred"
)

// OutlineEntity represents an entity in the URD outline
type OutlineEntity struct {
	EntityID           string       `json:"entity_id"` // Pattern: ENT-001
	Name               string       `json:"name"`
	Description        string       `json:"description,omitempty"` // Added for compatibility with Full URD
	Source             EntitySource `json:"source"`
	UsedInUseCases     []string     `json:"used_in_use_cases"`    // UseCase IDs
	Attributes         []string     `json:"attributes"`           // High-level list
	RelatedPRDEntities []string     `json:"related_prd_entities"` // Entity IDs from PRD
}

// TechnicalNoteCategory represents the category of technical note
type TechnicalNoteCategory string

const (
	TechnicalNoteCategoryMissingInfo         TechnicalNoteCategory = "missing_info"
	TechnicalNoteCategoryClarificationNeeded TechnicalNoteCategory = "clarification_needed"
	TechnicalNoteCategoryAssumption          TechnicalNoteCategory = "assumption"
	TechnicalNoteCategoryConstraint          TechnicalNoteCategory = "constraint"
)

// TechnicalNoteSeverity represents the severity of a technical note
type TechnicalNoteSeverity string

const (
	TechnicalNoteSeverityBlocker TechnicalNoteSeverity = "blocker"
	TechnicalNoteSeverityHigh    TechnicalNoteSeverity = "high"
	TechnicalNoteSeverityMedium  TechnicalNoteSeverity = "medium"
	TechnicalNoteSeverityLow     TechnicalNoteSeverity = "low"
)

// TechnicalNote represents a technical note or concern
type TechnicalNote struct {
	ID               string                `json:"id"`
	Category         TechnicalNoteCategory `json:"category"`
	Note             string                `json:"note"`
	AffectedUseCases []string              `json:"affected_use_cases"` // UseCase IDs
	Severity         TechnicalNoteSeverity `json:"severity"`
	Concern          string                `json:"concern"`     // Added for index gen compatibility
	Mitigation       string                `json:"mitigation"`  // Added for index gen compatibility
	Description      string                `json:"description"` // Added for index gen compatibility
}

// OpenQuestion represents an open question that needs clarification
type OpenQuestion struct {
	ID               string   `json:"id"` // Pattern: Q-001
	Question         string   `json:"question"`
	Context          string   `json:"context"`
	AffectedUseCases []string `json:"affected_use_cases"` // UseCase IDs
	ProposedAnswer   string   `json:"proposed_answer,omitempty"`
	DecisionMaker    string   `json:"decision_maker"` // Role who should answer
}

// CoverageMetrics represents coverage metrics for the outline
type CoverageMetrics struct {
	TotalUserStories          int      `json:"total_user_stories"`
	MappedUserStories         int      `json:"mapped_user_stories"`
	CoveragePercentage        float64  `json:"coverage_percentage"`
	UnmappedUserStoryIDs      []string `json:"unmapped_user_story_ids"`
	TotalFeatures             int      `json:"total_features"`
	MappedFeatures            int      `json:"mapped_features"`
	FeatureCoveragePercentage float64  `json:"feature_coverage_percentage"`
}

// URDOutline represents the complete URD outline for a module
type URDOutline struct {
	ModuleName        string               `json:"module_name"`
	ModuleDescription string               `json:"module_description"`
	GeneratedAt       time.Time            `json:"generated_at"`
	GeneratedBy       string               `json:"generated_by"`
	Confidence        float64              `json:"confidence"`
	Actors            []Actor              `json:"actors"`
	UseCases          []UseCase            `json:"use_cases"`
	Integrations      []OutlineIntegration `json:"integrations"`
	Entities          []OutlineEntity      `json:"entities"`
	TechnicalNotes    []TechnicalNote      `json:"technical_notes"`
	OpenQuestions     []OpenQuestion       `json:"open_questions"`
	Coverage          CoverageMetrics      `json:"coverage"`
}

// ValidationError represents a validation error
type ValidationError struct {
	RuleID        string   `json:"rule_id"`
	Message       string   `json:"message"`
	AffectedItems []string `json:"affected_items"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	RuleID        string   `json:"rule_id"`
	Message       string   `json:"message"`
	AffectedItems []string `json:"affected_items"`
	Suggestion    string   `json:"suggestion"`
}

// ValidationReport represents the validation report for an outline
type ValidationReport struct {
	IsValid              bool                `json:"is_valid"`
	Confidence           float64             `json:"confidence"`
	Errors               []ValidationError   `json:"errors"`
	Warnings             []ValidationWarning `json:"warnings"`
	CoverageSummary      CoverageMetrics     `json:"coverage_summary"`
	RegenerationAttempts int                 `json:"regeneration_attempts"`
	Timestamp            time.Time           `json:"timestamp"`
}

// GenerationConfig represents configuration for outline generation
type GenerationConfig struct {
	MinConfidenceThreshold   float64 `json:"min_confidence_threshold"`
	MaxRegenerationAttempts  int     `json:"max_regeneration_attempts"`
	EnableAutoFix            bool    `json:"enable_auto_fix"`
	EnableParallelProcessing bool    `json:"enable_parallel_processing"`
}

// DefaultGenerationConfig returns default generation configuration
func DefaultGenerationConfig() GenerationConfig {
	return GenerationConfig{
		MinConfidenceThreshold:   0.70,
		MaxRegenerationAttempts:  3,
		EnableAutoFix:            true,
		EnableParallelProcessing: true,
	}
}
