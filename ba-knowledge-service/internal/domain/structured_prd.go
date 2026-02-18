package v32

// Note: This file is a direct port of prd_model.go from the reference implementation
// adapted for the v3.2 domain package in ba-agent-service.

// OutOfScopeItem represents a feature that is out of scope
type OutOfScopeItem struct {
	Feature      string `json:"feature"`
	Reason       string `json:"reason"`
	PlannedPhase string `json:"planned_phase,omitempty"`
}

// ScopeDefinition represents project scope (top-level, not nested)
type ScopeDefinition struct {
	InScope    []string         `json:"in_scope,omitempty"`
	OutOfScope []OutOfScopeItem `json:"out_of_scope,omitempty"`
}

// Scope represents the project scope (DEPRECATED - use ScopeDefinition)
type Scope struct {
	InScope    []string         `json:"in_scope,omitempty"`
	OutOfScope []OutOfScopeItem `json:"out_of_scope,omitempty"`
}

// PRDMetadata represents metadata about the PRD document
type PRDMetadata struct {
	ProductName string `json:"product_name" validate:"required"`
	Version     string `json:"version" validate:"required"`
	Author      string `json:"author,omitempty"`
	LastUpdated string `json:"last_updated,omitempty"`
	Status      string `json:"status" validate:"required,oneof=draft review approved archived"`
	PRDID       string `json:"prd_id"` // UUID generated during parsing
}

// GlossaryTerm represents a term in the glossary
type GlossaryTerm struct {
	ID           string   `json:"id" validate:"required,matches=^TERM-[0-9]{3}$"`
	Term         string   `json:"term" validate:"required"`
	Meaning      string   `json:"meaning" validate:"required"`
	English      string   `json:"english,omitempty"`
	RelatedTerms []string `json:"related_terms,omitempty"`
	Category     string   `json:"category" validate:"required,oneof=technical business domain"`
}

// Permission represents a permission in the matrix
type Permission struct {
	Action       string   `json:"action" validate:"required"`
	AllowedRoles []string `json:"allowed_roles" validate:"required,min=1"`
	Description  string   `json:"description,omitempty"`
}

// PermissionMatrix represents the RBAC permission matrix
type PermissionMatrix struct {
	Roles       []string     `json:"roles" validate:"required,min=1"`
	Permissions []Permission `json:"permissions" validate:"required,min=1"`
}

// UserTask represents a task in the user story map
type UserTask struct {
	TaskName    string   `json:"task_name" validate:"required"`
	UserStories []string `json:"user_stories" validate:"required,min=1,dive,matches=^US-[0-9]{3}$"`
	Priority    int      `json:"priority" validate:"required,min=1"`
}

// Activity represents an activity in the user story map
type Activity struct {
	ActivityName string     `json:"activity_name" validate:"required"`
	UserTasks    []UserTask `json:"user_tasks" validate:"required,min=1"`
}

// Release represents a release in the user story map
type Release struct {
	ReleaseName     string   `json:"release_name" validate:"required"`
	IncludedStories []string `json:"included_stories" validate:"required,min=1,dive,matches=^US-[0-9]{3}$"`
}

// UserStoryMap represents the user story mapping
type UserStoryMap struct {
	UserType         string     `json:"user_type,omitempty"`
	ActivityBackbone []Activity `json:"activity_backbone" validate:"required,min=1"`
	Releases         []Release  `json:"releases,omitempty"`
}

// AnalyticsRequirement represents an analytics/tracking requirement
type AnalyticsRequirement struct {
	MetricID           string            `json:"metric_id" validate:"required,matches=^METRIC-[0-9]{3}$"`
	MetricName         string            `json:"metric_name" validate:"required"`
	Event              string            `json:"event" validate:"required"`
	Properties         map[string]string `json:"properties,omitempty"`
	Tool               string            `json:"tool,omitempty"`
	ReportingFrequency string            `json:"reporting_frequency" validate:"required,oneof=realtime daily weekly monthly"`
	Dashboard          string            `json:"dashboard,omitempty"`
}

// SuccessMetric represents a success metric for the product
type SuccessMetric struct {
	MetricName        string `json:"metric_name" validate:"required"`
	Target            string `json:"target" validate:"required"`
	MeasurementMethod string `json:"measurement_method,omitempty"`
}

// ProductOverview represents the high-level product information
type ProductOverview struct {
	Name           string          `json:"name" validate:"required,min=1"`
	Description    string          `json:"description" validate:"required,min=10"`
	Vision         string          `json:"vision,omitempty"`
	TargetRelease  string          `json:"target_release,omitempty"`
	Objectives     []string        `json:"objectives" validate:"required,min=1"`
	SuccessMetrics []SuccessMetric `json:"success_metrics,omitempty"`
	// Deprecated fields kept for backward compatibility
	Scope       Scope    `json:"scope,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
}

// TechnicalLevel represents the technical expertise level
type TechnicalLevel string

const (
	TechnicalLevelNovice       TechnicalLevel = "novice"
	TechnicalLevelIntermediate TechnicalLevel = "intermediate"
	TechnicalLevelExpert       TechnicalLevel = "expert"
)

// Persona represents a user persona
type Persona struct {
	ID             string         `json:"id" validate:"required,matches=^P[0-9]{3}$"`
	Name           string         `json:"name" validate:"required"`
	Role           string         `json:"role" validate:"required"`
	Goals          []string       `json:"goals" validate:"required,min=1"`
	PainPoints     []string       `json:"pain_points,omitempty"`
	Behaviors      []string       `json:"behaviors,omitempty"`
	Motivations    []string       `json:"motivations,omitempty"`
	Barriers       []string       `json:"barriers,omitempty"`
	TechnicalLevel TechnicalLevel `json:"technical_level" validate:"required,oneof=novice intermediate expert"`
	UsageFrequency string         `json:"usage_frequency" validate:"omitempty,oneof=daily weekly monthly occasional"`
}

// FeaturePriority represents feature priority levels
type FeaturePriority string

const (
	FeaturePriorityCritical FeaturePriority = "critical"
	FeaturePriorityHigh     FeaturePriority = "high"
	FeaturePriorityMedium   FeaturePriority = "medium"
	FeaturePriorityLow      FeaturePriority = "low"
)

// Feature represents a product feature
type Feature struct {
	ID                 string   `json:"id" validate:"required,matches=^F[0-9]{3}$"`
	Name               string   `json:"name" validate:"required"`
	Description        string   `json:"description" validate:"required"`
	Priority           string   `json:"priority" validate:"required,oneof=P0 P1 P2 P3"`
	Status             string   `json:"status" validate:"omitempty,oneof=planned in_progress completed cancelled"`
	Category           string   `json:"category,omitempty"`
	Dependencies       []string `json:"dependencies,omitempty" validate:"dive,matches=^F[0-9]{3}$"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	TechnicalNotes     string   `json:"technical_notes,omitempty"`
	// Deprecated: Use string priority P0-P3 instead
	PriorityLegacy FeaturePriority `json:"priority_legacy,omitempty"`
}

// UserStoryPriority represents user story priority using MoSCoW method
type UserStoryPriority string

const (
	UserStoryPriorityMustHave   UserStoryPriority = "must_have"
	UserStoryPriorityShouldHave UserStoryPriority = "should_have"
	UserStoryPriorityCouldHave  UserStoryPriority = "could_have"
	UserStoryPriorityWontHave   UserStoryPriority = "wont_have"
)

// UserStory represents a user story
type UserStory struct {
	ID                 string   `json:"id" validate:"required,matches=^US-[0-9]{3}$"`
	FeatureID          string   `json:"feature_id" validate:"required,matches=^F[0-9]{3}$"`
	AsA                string   `json:"as_a" validate:"required"`
	IWant              string   `json:"i_want" validate:"required"`
	SoThat             string   `json:"so_that" validate:"required"`
	Priority           string   `json:"priority" validate:"required,oneof=P0 P1 P2 P3"`
	Size               string   `json:"size" validate:"omitempty,oneof=XS S M L XL"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	Dependencies       []string `json:"dependencies,omitempty" validate:"dive,matches=^US-[0-9]{3}$"`
	Status             string   `json:"status" validate:"omitempty,oneof=backlog ready in_progress done"`
	// Deprecated: Use string priority P0-P3 instead
	PriorityLegacy UserStoryPriority `json:"priority_legacy,omitempty"`
}

// BusinessRule represents a business rule
type BusinessRule struct {
	ID              string   `json:"id" validate:"required,matches=^BR-[0-9]{2}$"`
	Name            string   `json:"name" validate:"required"`
	Description     string   `json:"description" validate:"required"`
	RuleLogic       string   `json:"rule_logic,omitempty"`
	AppliesTo       []string `json:"applies_to" validate:"required,min=1"`
	ValidationLogic string   `json:"validation_logic,omitempty"`
	ErrorMessage    string   `json:"error_message,omitempty"`
	Severity        string   `json:"severity" validate:"omitempty,oneof=critical high medium low"`
	// Deprecated fields kept for backward compatibility
	Rule          string `json:"rule,omitempty"`
	ErrorHandling string `json:"error_handling,omitempty"`
}

// IntegrationType represents the type of integration
type IntegrationType string

const (
	IntegrationTypeRESTAPI       IntegrationType = "rest_api"
	IntegrationTypeGraphQL       IntegrationType = "graphql"
	IntegrationTypeWebhook       IntegrationType = "webhook"
	IntegrationTypeMessageQueue  IntegrationType = "message_queue"
	IntegrationTypeDatabase      IntegrationType = "database"
	IntegrationTypeThirdPartySDK IntegrationType = "third_party_sdk"
)

// DataFlow represents the direction of data flow
type DataFlow string

const (
	DataFlowInbound       DataFlow = "inbound"
	DataFlowOutbound      DataFlow = "outbound"
	DataFlowBidirectional DataFlow = "bidirectional"
)

// Integration represents an external integration
type Integration struct {
	ID             string          `json:"id" validate:"required,matches=^INT-[0-9]{3}$"`
	SystemName     string          `json:"system_name" validate:"required"`
	Type           IntegrationType `json:"type" validate:"required,oneof=rest_api graphql oauth webhook database third_party_sdk"`
	Purpose        string          `json:"purpose" validate:"required"`
	Direction      DataFlow        `json:"direction" validate:"required,oneof=inbound outbound bidirectional"`
	Status         string          `json:"status" validate:"omitempty,oneof=ready planned blocked"`
	Authentication string          `json:"authentication,omitempty"`
	Endpoints      []string        `json:"endpoints,omitempty"`
	DataFlowDesc   string          `json:"data_flow,omitempty"` // Description of what data flows
	ErrorScenarios []string        `json:"error_scenarios,omitempty"`
	// Deprecated fields kept for backward compatibility
	Name     string   `json:"name,omitempty"`
	DataFlow DataFlow `json:"data_flow_legacy,omitempty"`
}

// FlowStep represents a single step in a user flow
type FlowStep struct {
	StepNumber       int      `json:"step_number" validate:"required,min=1"`
	Screen           string   `json:"screen,omitempty"`
	Action           string   `json:"action" validate:"required"`
	Actor            string   `json:"actor" validate:"required"`
	Result           string   `json:"result,omitempty"`
	SystemBehavior   string   `json:"system_behavior,omitempty"`
	AlternativePaths []string `json:"alternative_paths,omitempty"`
	// Deprecated: Use Result and SystemBehavior instead
	SystemResponse string `json:"system_response,omitempty"`
}

// AlternativePath represents an alternative path in a user flow
type AlternativePath struct {
	Condition string     `json:"condition" validate:"required"`
	Steps     []FlowStep `json:"steps" validate:"required,min=1"`
	Outcome   string     `json:"outcome,omitempty"`
}

// UserFlow represents a complete user flow
type UserFlow struct {
	ID                 string            `json:"id" validate:"required,matches=^UF-[0-9]{3}$"`
	Name               string            `json:"name" validate:"required"`
	Description        string            `json:"description,omitempty"`
	Steps              []FlowStep        `json:"steps" validate:"required,min=1"`
	InvolvedPersonas   []string          `json:"involved_personas" validate:"required,dive,matches=^P[0-9]{3}$"`
	RelatedFeatures    []string          `json:"related_features,omitempty" validate:"dive,matches=^F[0-9]{3}$"`
	RelatedUserStories []string          `json:"related_user_stories,omitempty" validate:"dive,matches=^US-[0-9]{3}$"`
	EntryPoint         string            `json:"entry_point,omitempty"`
	ExitPoint          string            `json:"exit_point,omitempty"`
	AlternativePaths   []AlternativePath `json:"alternative_paths,omitempty"`
}

// EntityAttribute represents an attribute of an entity
type EntityAttribute struct {
	Name        string `json:"name" validate:"required"`
	Type        string `json:"type" validate:"required"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

// RelationshipType represents the type of relationship between entities
type RelationshipType string

const (
	RelationshipTypeOneToOne   RelationshipType = "one_to_one"
	RelationshipTypeOneToMany  RelationshipType = "one_to_many"
	RelationshipTypeManyToMany RelationshipType = "many_to_many"
)

// EntityRelationship represents a relationship between entities
type EntityRelationship struct {
	EntityID         string           `json:"entity_id" validate:"required,matches=^E[0-9]{3}$"`
	RelationshipType RelationshipType `json:"relationship_type" validate:"required,oneof=one_to_one one_to_many many_to_many"`
}

// Entity represents a data entity
type Entity struct {
	ID            string               `json:"id" validate:"required,matches=^E[0-9]{3}$"`
	Name          string               `json:"name" validate:"required"`
	Description   string               `json:"description,omitempty"`
	Attributes    []EntityAttribute    `json:"attributes" validate:"required,min=1"`
	Relationships []EntityRelationship `json:"relationships,omitempty"`
}

// StructuredPRD represents the complete structured PRD model
type StructuredPRD struct {
	Metadata              PRDMetadata            `json:"Metadata" validate:"required"`
	Glossary              []GlossaryTerm         `json:"Glossary,omitempty"`
	Personas              []Persona              `json:"Personas" validate:"required,min=1"`
	ProductOverview       ProductOverview        `json:"ProductOverview" validate:"required"`
	Features              []Feature              `json:"Features" validate:"required,min=1"`
	PermissionMatrix      PermissionMatrix       `json:"PermissionMatrix,omitempty"`
	Integrations          []Integration          `json:"Integrations,omitempty"`
	UserFlows             []UserFlow             `json:"UserFlows,omitempty"`
	UserStories           []UserStory            `json:"UserStories" validate:"required,min=1"`
	UserStoryMap          UserStoryMap           `json:"UserStoryMap,omitempty"`
	BusinessRules         []BusinessRule         `json:"BusinessRules,omitempty"`
	AnalyticsRequirements []AnalyticsRequirement `json:"AnalyticsRequirements,omitempty"`
	ScopeDefinition       ScopeDefinition        `json:"ScopeDefinition,omitempty"`
	// Deprecated fields kept for backward compatibility
	Entities []Entity `json:"entities,omitempty"`
}
