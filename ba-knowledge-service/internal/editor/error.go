package editor

import (
	"fmt"
	"strings"

	"github.com/blcvn/backend/services/ba-knowledge-service/internal/editor/validator"
)

// convertValidationErrors converts ValidationResult errors to AgentError format
func (a *ValidatorAgent) convertValidationErrors(validationErrors []validator.ValidationError) []AgentError {
	agentErrors := make([]AgentError, 0, len(validationErrors))

	for _, vErr := range validationErrors {
		agentErr := AgentError{
			Type:        a.mapValidationErrorType(vErr.Type),
			Message:     vErr.Message,
			SectionID:   vErr.SectionID,
			Recoverable: a.isRecoverableError(vErr),
			Suggestion:  a.generateSuggestion(vErr),
		}

		agentErrors = append(agentErrors, agentErr)
	}

	return agentErrors
}

// mapValidationErrorType maps validation error types to agent error types
func (a *ValidatorAgent) mapValidationErrorType(validationType string) string {
	mapping := map[string]string{
		// Structural errors
		"MISSING_SECTION":       "STRUCTURE_ERROR",
		"MISSING_METADATA":      "METADATA_ERROR",
		"INVALID_SECTION_ORDER": "STRUCTURE_ERROR",

		// Table errors
		"MISSING_TABLE":           "TABLE_ERROR",
		"INVALID_TABLE_HEADERS":   "TABLE_ERROR",
		"INSUFFICIENT_TABLE_ROWS": "TABLE_ERROR",
		"MALFORMED_TABLE":         "TABLE_ERROR",

		// Diagram errors
		"MISSING_DIAGRAM":        "DIAGRAM_ERROR",
		"INVALID_DIAGRAM_TYPE":   "DIAGRAM_ERROR",
		"FORBIDDEN_DIAGRAM_TYPE": "DIAGRAM_ERROR",
		"MALFORMED_DIAGRAM":      "DIAGRAM_ERROR",

		// Content errors
		"HTML_COMMENT_REMAINING":   "CONTENT_ERROR",
		"EMPTY_SECTION":            "CONTENT_ERROR",
		"MISSING_REQUIRED_CONTENT": "CONTENT_ERROR",

		// Semantic errors
		"SEMANTIC_ERROR":        "SEMANTIC_ERROR",
		"LOGICAL_INCONSISTENCY": "SEMANTIC_ERROR",
		"MISSING_INFO":          "SEMANTIC_ERROR",
		"CONTRADICTION":         "SEMANTIC_ERROR",
		"AMBIGUITY":             "SEMANTIC_ERROR",

		// Reference errors
		"INVALID_REFERENCE": "REFERENCE_ERROR",
		"BROKEN_LINK":       "REFERENCE_ERROR",
		"UNDEFINED_ENTITY":  "REFERENCE_ERROR",

		// Format errors
		"INVALID_FORMAT":   "FORMAT_ERROR",
		"INVALID_MARKDOWN": "FORMAT_ERROR",
	}

	if mappedType, exists := mapping[validationType]; exists {
		return mappedType
	}

	// Default to UNKNOWN_ERROR
	return "UNKNOWN_ERROR"
}

// isRecoverableError determines if an error can be auto-fixed
func (a *ValidatorAgent) isRecoverableError(vErr validator.ValidationError) bool {
	recoverableTypes := map[string]bool{
		// These can potentially be auto-fixed
		"MISSING_TABLE":          true,
		"INVALID_TABLE_HEADERS":  true,
		"HTML_COMMENT_REMAINING": true,
		"EMPTY_SECTION":          true,
		"MALFORMED_TABLE":        true,
		"MISSING_DIAGRAM":        true,
		"INVALID_DIAGRAM_TYPE":   true,

		// These are harder to auto-fix
		"MISSING_SECTION":        false,
		"LOGICAL_INCONSISTENCY":  false,
		"CONTRADICTION":          false,
		"UNDEFINED_ENTITY":       false,
		"FORBIDDEN_DIAGRAM_TYPE": false,
	}

	if recoverable, exists := recoverableTypes[vErr.Type]; exists {
		return recoverable
	}

	// Default to non-recoverable for unknown types
	return false
}

// generateSuggestion creates helpful suggestions for fixing errors
func (a *ValidatorAgent) generateSuggestion(vErr validator.ValidationError) string {
	// If validation error already has a suggestion, use it
	if vErr.Suggestion != "" {
		return vErr.Suggestion
	}

	// Generate suggestions based on error type
	switch vErr.Type {
	case "MISSING_SECTION":
		return a.generateMissingSectionSuggestion(vErr)

	case "MISSING_TABLE":
		return a.generateMissingTableSuggestion(vErr)

	case "INVALID_TABLE_HEADERS":
		return a.generateInvalidTableHeadersSuggestion(vErr)

	case "INSUFFICIENT_TABLE_ROWS":
		return a.generateInsufficientRowsSuggestion(vErr)

	case "MISSING_DIAGRAM":
		return a.generateMissingDiagramSuggestion(vErr)

	case "INVALID_DIAGRAM_TYPE":
		return a.generateInvalidDiagramTypeSuggestion(vErr)

	case "FORBIDDEN_DIAGRAM_TYPE":
		return "ASCII art diagrams are not allowed. Use Mermaid diagrams with ```mermaid code blocks."

	case "HTML_COMMENT_REMAINING":
		return "Replace HTML comment placeholders with actual content. Comments should only be used temporarily."

	case "EMPTY_SECTION":
		return fmt.Sprintf("Section '%s' is empty. Add appropriate content or remove the section.", vErr.SectionID)

	case "LOGICAL_INCONSISTENCY":
		return "Review the content for logical consistency. Ensure all information is coherent and non-contradictory."

	case "CONTRADICTION":
		return "Resolve contradictions in the document. Information should be consistent across all sections."

	case "AMBIGUITY":
		return "Clarify ambiguous statements. Use specific and precise language."

	case "UNDEFINED_ENTITY":
		return "Reference to undefined entity. Ensure all referenced actors, use cases, or entities are properly defined."

	case "INVALID_REFERENCE":
		return "Fix broken references. Ensure all cross-references point to valid sections or entities."

	case "MALFORMED_TABLE":
		return "Fix table structure. Ensure proper markdown table syntax with consistent column counts."

	case "MALFORMED_DIAGRAM":
		return "Fix diagram syntax. Ensure proper Mermaid diagram format."

	default:
		return "Please review and correct this error manually."
	}
}

// generateMissingSectionSuggestion creates suggestion for missing sections
func (a *ValidatorAgent) generateMissingSectionSuggestion(vErr validator.ValidationError) string {
	suggestions := map[string]string{
		"1":   "Add section '# 1 US → Use Case Mapping' with a table mapping user stories to use cases.",
		"2":   "Add section '# 2 Actor Definition' with subsections for human and system actors.",
		"2.1": "Add subsection '## 2.1 Human Actors' with a table listing all human actors.",
		"2.2": "Add subsection '## 2.2 System Actors' with a table listing all system actors.",
		"3":   "Add section '# 3 System Boundary' with a boundary diagram.",
		"3.1": "Add subsection '## 3.1 Diagram' with a Mermaid diagram showing system boundaries.",
		"4":   "Add section '# 4 Use Case Summary Table' with a comprehensive table of all use cases.",
		"5":   "Add section '# 5 Main Flow Sketch (High-Level)' with detailed use case flows.",
		"6":   "Add section '# 6 Integration Touchpoints' with a table of integration points.",
		"7":   "Add section '# 7 Data Entities Overview' with entity definitions and ER diagram.",
		"7.1": "Add subsection '## 7.1 Diagram:**' with a Mermaid ER diagram of data entities.",
		"8":   "Add section '# 8 Technical Notes & Concerns' with a table of technical concerns.",
	}

	if suggestion, exists := suggestions[vErr.SectionID]; exists {
		return suggestion
	}

	return fmt.Sprintf("Add the missing section: %s", vErr.Message)
}

// generateMissingTableSuggestion creates suggestion for missing tables
func (a *ValidatorAgent) generateMissingTableSuggestion(vErr validator.ValidationError) string {
	tableTemplates := map[string]string{
		"1": `| Use Case ID | Use Case Name | Related User Stories | Priority | Notes |
|-------------|---------------|---------------------|----------|-------|
| UC-001 | Example Use Case | US-001, US-002 | High | Example note |`,

		"2.1": `| Actor ID | Name | Description | Persona/Role |
|----------|------|-------------|--------------|
| A-001 | Example Actor | Example description | Example role |`,

		"2.2": `| Actor ID | Name | Description | Type |
|----------|------|-------------|------|
| SA-001 | Example System | Example description | External API |`,

		"4": `| UC ID | Name | Actor | Trigger | Preconditions | Postconditions | Priority |
|-------|------|-------|---------|---------------|----------------|----------|
| UC-001 | Example | Actor | Event | Precondition | Postcondition | High |`,

		"6": `| Use Case | Step | System | Purpose | Mode | Error Handling |
|----------|------|--------|---------|------|----------------|
| UC-001 | 1 | Example System | Purpose | Sync | Handle errors |`,

		"7": `| Entity | Description | States | Relationships |
|--------|-------------|--------|---------------|
| User | Example entity | Active, Inactive | Has many Orders |`,

		"8": `| ID | Concern | Affected UC | Mitigation | Owner |
|----|---------|-------------|------------|-------|
| TC-001 | Example concern | UC-001 | Mitigation strategy | Team |`,
	}

	if template, exists := tableTemplates[vErr.SectionID]; exists {
		return fmt.Sprintf("Add table in section %s with the following format:\n\n%s", vErr.SectionID, template)
	}

	return fmt.Sprintf("Add a properly formatted markdown table in section %s", vErr.SectionID)
}

// generateInvalidTableHeadersSuggestion creates suggestion for invalid headers
func (a *ValidatorAgent) generateInvalidTableHeadersSuggestion(vErr validator.ValidationError) string {
	expectedHeaders := map[string][]string{
		"1":   {"Use Case ID", "Use Case Name", "Related User Stories", "Priority", "Notes"},
		"2.1": {"Actor ID", "Name", "Description", "Persona/Role"},
		"2.2": {"Actor ID", "Name", "Description", "Type"},
		"3.1": {"System/Module", "Inside/Outside", "Responsibility", "Owner"},
		"4":   {"UC ID", "Name", "Actor", "Trigger", "Preconditions", "Postconditions", "Priority"},
		"6":   {"Use Case", "Step", "System", "Purpose", "Mode", "Error Handling"},
		"7":   {"Entity", "Description", "States", "Relationships"},
		"8":   {"ID", "Concern", "Affected UC", "Mitigation", "Owner"},
	}

	if headers, exists := expectedHeaders[vErr.SectionID]; exists {
		return fmt.Sprintf("Table headers should be: %s", strings.Join(headers, " | "))
	}

	return "Fix table headers to match the template requirements."
}

// generateInsufficientRowsSuggestion creates suggestion for insufficient rows
func (a *ValidatorAgent) generateInsufficientRowsSuggestion(vErr validator.ValidationError) string {
	return fmt.Sprintf("Add more rows to the table in section %s. Ensure all required data is included.", vErr.SectionID)
}

// generateMissingDiagramSuggestion creates suggestion for missing diagrams
func (a *ValidatorAgent) generateMissingDiagramSuggestion(vErr validator.ValidationError) string {
	diagramExamples := map[string]string{
		"3.1": `Add a Mermaid diagram showing system boundaries:

` + "```mermaid" + `
graph TD
    User[User] --> System[Your System]
    System --> Database[(Database)]
    System --> ExternalAPI[External API]
` + "```",

		"5": `Add a Mermaid sequence diagram for the use case flow:

` + "```mermaid" + `
sequenceDiagram
    participant User
    participant System
    participant Database
    
    User->>System: Action
    System->>Database: Query
    Database-->>System: Result
    System-->>User: Response
` + "```",

		"7.1": `Add a Mermaid ER diagram for data entities:

` + "```mermaid" + `
erDiagram
    USER ||--o{ ORDER : places
    ORDER ||--|{ LINE_ITEM : contains
    PRODUCT ||--o{ LINE_ITEM : "ordered in"
` + "```",
	}

	if example, exists := diagramExamples[vErr.SectionID]; exists {
		return example
	}

	return fmt.Sprintf("Add a Mermaid diagram in section %s using ```mermaid code block.", vErr.SectionID)
}

// generateInvalidDiagramTypeSuggestion creates suggestion for invalid diagram types
func (a *ValidatorAgent) generateInvalidDiagramTypeSuggestion(vErr validator.ValidationError) string {
	diagramTypes := map[string][]string{
		"3.1": {"graph TD", "graph LR", "flowchart"},
		"5":   {"sequenceDiagram"},
		"7.1": {"erDiagram"},
	}

	if types, exists := diagramTypes[vErr.SectionID]; exists {
		return fmt.Sprintf("Use one of the following Mermaid diagram types: %s", strings.Join(types, ", "))
	}

	return "Use the appropriate Mermaid diagram type for this section."
}

// convertValidationWarnings converts ValidationResult warnings to AgentWarning format
func (a *ValidatorAgent) convertValidationWarnings(validationWarnings []validator.ValidationWarning) []AgentWarning {
	agentWarnings := make([]AgentWarning, 0, len(validationWarnings))

	for _, vWarn := range validationWarnings {
		agentWarn := AgentWarning{
			Type:    a.mapValidationWarningType(vWarn),
			Message: vWarn.Message,
			Context: vWarn.Context,
		}

		agentWarnings = append(agentWarnings, agentWarn)
	}

	return agentWarnings
}

// mapValidationWarningType maps validation warnings to agent warning types
func (a *ValidatorAgent) mapValidationWarningType(vWarn validator.ValidationWarning) string {
	message := strings.ToLower(vWarn.Message)

	switch {
	case strings.Contains(message, "html comment"):
		return "PLACEHOLDER_WARNING"

	case strings.Contains(message, "mismatch"):
		return "COUNT_MISMATCH"

	case strings.Contains(message, "rows"):
		return "INSUFFICIENT_DATA"

	case strings.Contains(message, "diagram"):
		return "DIAGRAM_WARNING"

	case strings.Contains(message, "reference"):
		return "REFERENCE_WARNING"

	default:
		return "GENERAL_WARNING"
	}
}

// formatErrorSummary creates a human-readable summary of all errors
func (a *ValidatorAgent) formatErrorSummary(errors []AgentError) string {
	if len(errors) == 0 {
		return "No errors found."
	}

	// Group errors by type
	errorsByType := make(map[string][]AgentError)
	for _, err := range errors {
		errorsByType[err.Type] = append(errorsByType[err.Type], err)
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Found %d validation error(s):\n\n", len(errors)))

	// Sort error types for consistent output
	types := []string{
		"STRUCTURE_ERROR",
		"METADATA_ERROR",
		"TABLE_ERROR",
		"DIAGRAM_ERROR",
		"CONTENT_ERROR",
		"SEMANTIC_ERROR",
		"REFERENCE_ERROR",
		"FORMAT_ERROR",
		"UNKNOWN_ERROR",
	}

	for _, errType := range types {
		if errs, exists := errorsByType[errType]; exists {
			summary.WriteString(fmt.Sprintf("## %s (%d)\n", errType, len(errs)))
			for i, err := range errs {
				summary.WriteString(fmt.Sprintf("%d. %s", i+1, err.Message))
				if err.SectionID != "" {
					summary.WriteString(fmt.Sprintf(" [Section: %s]", err.SectionID))
				}
				summary.WriteString("\n")
				if err.Suggestion != "" {
					summary.WriteString(fmt.Sprintf("   💡 Suggestion: %s\n", err.Suggestion))
				}
			}
			summary.WriteString("\n")
		}
	}

	return summary.String()
}

// getRecoverableErrors filters and returns only recoverable errors
func (a *ValidatorAgent) getRecoverableErrors(errors []AgentError) []AgentError {
	recoverable := make([]AgentError, 0)

	for _, err := range errors {
		if err.Recoverable {
			recoverable = append(recoverable, err)
		}
	}

	return recoverable
}

// getCriticalErrors filters and returns only critical (non-recoverable) errors
func (a *ValidatorAgent) getCriticalErrors(errors []AgentError) []AgentError {
	critical := make([]AgentError, 0)

	for _, err := range errors {
		if !err.Recoverable {
			critical = append(critical, err)
		}
	}

	return critical
}
