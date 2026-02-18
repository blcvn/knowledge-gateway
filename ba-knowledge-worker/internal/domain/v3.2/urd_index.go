package v32

import "time"

// URDIndex represents the complete Index document (Section 1-8)
// Aligned with URD Index Template from docs/v2/URD_template_index.md
type URDIndex struct {
	ID          string
	ProjectID   string
	ModuleName  string
	Version     int
	CreatedDate time.Time

	// Section 1: US → Use Case Mapping
	USToUCMapping []USToUCMap

	// Section 2: Actor Definition
	HumanActors  []Actor
	SystemActors []Actor

	// Section 3: System Boundary
	SystemBoundary SystemBoundary

	// Section 4: Use Case Summary Table
	UseCaseSummaryTable []UseCaseSummary

	// Section 5: Main Flow Sketch (High-Level)
	FlowSketches []FlowSketch

	// Section 6: Integration Touchpoints
	IntegrationTouchpoints []IntegrationTouchpoint

	// Section 7: Data Entities Overview
	DataEntities []DataEntity

	// Section 8: Technical Notes & Concerns
	TechnicalNotes []TechnicalNote

	ApprovedAt *time.Time
}

// Section 1: US → UC Mapping
type USToUCMap struct {
	UserStoryID string
	UseCaseIDs  []string // 1 US có thể map với nhiều UC
	MappingNote string   // Ghi chú lý do mapping
}

// Note: Actor type is defined in urd_outline.go and shared across tiers

// Section 3: System Boundary
type SystemBoundary struct {
	InScope    []string // Danh sách tính năng/chức năng trong phạm vi
	OutScope   []string // Danh sách tính năng/chức năng ngoài phạm vi
	DiagramURL string   // Link đến diagram (PlantUML/Mermaid)
}

// Section 4: Use Case Summary Table
type UseCaseSummary struct {
	ID              string
	Name            string
	PrimaryActor    string
	Trigger         string
	ExpectedOutcome string
	Priority        string // "critical", "high", "medium", "low"
}

// Section 5: Main Flow Sketch (High-Level)
type FlowSketch struct {
	UseCaseID string
	Steps     []FlowSketchStep
}

type FlowSketchStep struct {
	StepNumber  int
	Description string // High-level description only, NO details
}

// Section 6: Integration Touchpoints
type IntegrationTouchpoint struct {
	ID          string
	Name        string
	Type        string // "external_api", "database", "message_queue", "third_party"
	Direction   string // "inbound", "outbound", "bidirectional"
	Description string
	AffectedUCs []string // Use Cases sử dụng integration này
}

// Section 7: Data Entities Overview
type DataEntity struct {
	ID            string
	Name          string
	Description   string
	Type          string // "table", "document", "cache", "file"
	KeyAttributes []string
	DiagramURL    string // Link đến ERD hoặc data model diagram
}

// Note: TechnicalNote type is defined in urd_outline.go and shared across tiers
