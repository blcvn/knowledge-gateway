package v32

import "time"

// URDFull represents the complete Full URD document (Tier 3)
// This is the most detailed tier with API specs, sequence diagrams, and comprehensive flows
type URDFull struct {
	ID          string
	ProjectID   string
	ModuleName  string
	Version     int
	CreatedDate time.Time

	// Metadata
	ModuleDescription string
	GeneratedAt       time.Time
	GeneratedBy       string

	// Core Components (inherited and expanded from Outline)
	Actors         []Actor
	UseCases       []FullUseCase
	Integrations   []OutlineIntegration
	Entities       []OutlineEntity
	TechnicalNotes []TechnicalNote
	OpenQuestions  []OpenQuestion

	// Full-tier specific
	APISpecifications  []APISpecification
	DataFlowDiagrams   []DataFlowDiagram
	SystemArchitecture string // Mermaid diagram or description

	ApprovedAt *time.Time
}

// FullUseCase represents a fully detailed use case with API specs and diagrams
type FullUseCase struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	PrimaryActor    string          `json:"primary_actor"`
	SecondaryActors []string        `json:"secondary_actors"`
	Preconditions   []string        `json:"preconditions"`
	Postconditions  []string        `json:"postconditions"`
	Priority        UseCasePriority `json:"priority"`

	// Detailed Flows
	MainFlow         []DetailedFlowStep `json:"main_flow"`
	AlternativeFlows []AlternativeFlow  `json:"alternative_flows"`
	ExceptionFlows   []ExceptionFlow    `json:"exception_flows"`

	// Full-tier specific
	APIEndpoints    []APIEndpoint `json:"api_endpoints"`
	SequenceDiagram string        `json:"sequence_diagram"` // Mermaid diagram
	DataEntities    []string      `json:"data_entities"`    // Entity IDs involved
	BusinessRules   []string      `json:"business_rules"`   // Business Rule IDs
	ValidationRules []string      `json:"validation_rules"` // Validation rules
	PerformanceReqs []string      `json:"performance_reqs"` // Performance requirements
	SecurityReqs    []string      `json:"security_reqs"`    // Security requirements
}

// DetailedFlowStep represents a detailed step in a use case flow
type DetailedFlowStep struct {
	StepNumber      int      `json:"step_number"`
	Actor           string   `json:"actor"`
	Action          string   `json:"action"`
	SystemResponse  string   `json:"system_response"`
	DataInvolved    []string `json:"data_involved"`      // Entity names
	BusinessRules   []string `json:"business_rules"`     // Rule IDs
	ValidationRules []string `json:"validation_rules"`   // Validation descriptions
	APICall         string   `json:"api_call,omitempty"` // API endpoint if applicable
}

// AlternativeFlow represents an alternative flow in a use case
type AlternativeFlow struct {
	ID        string             `json:"id"`
	Name      string             `json:"name"`
	Trigger   string             `json:"trigger"`
	Steps     []DetailedFlowStep `json:"steps"`
	RejoinsAt int                `json:"rejoins_at"` // Step number where it rejoins main flow (0 = ends)
}

// ExceptionFlow represents an exception flow in a use case
type ExceptionFlow struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Trigger    string             `json:"trigger"`
	ErrorType  string             `json:"error_type"`
	Steps      []DetailedFlowStep `json:"steps"`
	Resolution string             `json:"resolution"`
}

// APIEndpoint represents an API endpoint specification
type APIEndpoint struct {
	ID          string         `json:"id"`
	Method      string         `json:"method"` // GET, POST, PUT, DELETE, PATCH
	Path        string         `json:"path"`
	Description string         `json:"description"`
	Request     RequestSchema  `json:"request"`
	Response    ResponseSchema `json:"response"`
	UsedInSteps []int          `json:"used_in_steps"` // Step numbers
	UseCaseID   string         `json:"use_case_id"`
}

// RequestSchema represents API request schema
type RequestSchema struct {
	ContentType string                 `json:"content_type"`
	Schema      map[string]interface{} `json:"schema"`
	Example     string                 `json:"example"`
	Headers     map[string]string      `json:"headers,omitempty"`
	QueryParams map[string]string      `json:"query_params,omitempty"`
}

// ResponseSchema represents API response schema
type ResponseSchema struct {
	StatusCode  int                    `json:"status_code"`
	ContentType string                 `json:"content_type"`
	Schema      map[string]interface{} `json:"schema"`
	Example     string                 `json:"example"`
	Headers     map[string]string      `json:"headers,omitempty"`
}

// APISpecification represents a complete API specification for a module
type APISpecification struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	BaseURL     string        `json:"base_url"`
	Endpoints   []APIEndpoint `json:"endpoints"`
	AuthMethod  string        `json:"auth_method"`
	Description string        `json:"description"`
}

// DataFlowDiagram represents a data flow diagram
type DataFlowDiagram struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Diagram     string   `json:"diagram"` // Mermaid diagram
	UseCaseIDs  []string `json:"use_case_ids"`
}
