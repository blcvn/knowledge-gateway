package validator

import (
	"bufio"
	"fmt"
	"strings"
)

// DocumentValidator validates documents against templates
type DocumentValidator struct {
	content  string
	lines    []string
	template *Template
	errors   []ValidationError
	warnings []ValidationWarning
}

type ValidationError struct {
	Type       string
	Message    string
	Line       int
	SectionID  string
	Severity   string
	Suggestion string
}

type ValidationWarning struct {
	Message string
	Line    int
	Context string
}

type ValidationResult struct {
	IsValid         bool
	Errors          []ValidationError
	Warnings        []ValidationWarning
	Statistics      DocumentStatistics
	MissingSections []string
	FoundSections   []string
}

type DocumentStatistics struct {
	TotalLines    int
	TotalSections int
	TotalTables   int
	TotalDiagrams int
	UseCaseCount  int
}

// NewDocumentValidator creates a new validator
func NewDocumentValidator(content string, template *Template) *DocumentValidator {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	
	return &DocumentValidator{
		content:  content,
		lines:    lines,
		template: template,
		errors:   []ValidationError{},
		warnings: []ValidationWarning{},
	}
}

// Validate performs full validation
func (dv *DocumentValidator) Validate() ValidationResult {
	result := ValidationResult{
		MissingSections: []string{},
		FoundSections:   []string{},
	}
	
	// Validate metadata
	dv.validateMetadata()
	
	// Validate sections
	dv.validateSections(dv.template.Sections)
	
	// Apply global validation rules
	dv.applyValidationRules()
	
	// Calculate statistics
	result.Statistics = dv.calculateStatistics()
	
	// Compile results
	result.IsValid = len(dv.errors) == 0
	result.Errors = dv.errors
	result.Warnings = dv.warnings
	
	return result
}

// validateMetadata validates document metadata
func (dv *DocumentValidator) validateMetadata() {
	for _, meta := range dv.template.Metadata {
		found := false
		for _, line := range dv.lines {
			if meta.Pattern.MatchString(line) {
				found = true
				break
			}
		}
		
		if !found && meta.Required {
			dv.errors = append(dv.errors, ValidationError{
				Type:     "MISSING_METADATA",
				Message:  fmt.Sprintf("Missing required metadata: %s", meta.Name),
				Severity: "ERROR",
			})
		}
	}
}

// validateSections validates all sections recursively
func (dv *DocumentValidator) validateSections(sections []*Section) {
	for _, section := range sections {
		dv.validateSection(section)
	}
}

// validateSection validates a single section
func (dv *DocumentValidator) validateSection(section *Section) {
	found := false
	lineNum := 0
	
	for i, line := range dv.lines {
		if section.Pattern != nil && section.Pattern.MatchString(line) {
			found = true
			lineNum = i + 1
			break
		}
	}
	
	if !found && section.Required {
		dv.errors = append(dv.errors, ValidationError{
			Type:      "MISSING_SECTION",
			Message:   fmt.Sprintf("Missing required section: %s (Level %d)", section.Title, section.Level),
			SectionID: section.ID,
			Severity:  "ERROR",
			Suggestion: fmt.Sprintf("Add section: %s", section.Title),
		})
		return
	}
	
	if found {
		// Validate section content based on type
		switch section.Type {
		case TableSection:
			dv.validateTable(section, lineNum)
		case DiagramSection:
			dv.validateDiagram(section, lineNum)
		case IterativeSection:
			dv.validateIterativeSection(section, lineNum)
		}
		
		// Validate subsections
		if len(section.Subsections) > 0 {
			dv.validateSections(section.Subsections)
		}
	}
}

// validateTable validates table structure
func (dv *DocumentValidator) validateTable(section *Section, startLine int) {
	if section.Table == nil {
		return
	}
	
	// Find table header
	headerLine := -1
	for i := startLine; i < len(dv.lines) && i < startLine+20; i++ {
		line := dv.lines[i]
		if strings.Contains(line, "|") && !strings.HasPrefix(strings.TrimSpace(line), "<!--") {
			headerLine = i
			break
		}
	}
	
	if headerLine == -1 {
		dv.errors = append(dv.errors, ValidationError{
			Type:      "MISSING_TABLE",
			Message:   fmt.Sprintf("Table not found for section: %s", section.Title),
			Line:      startLine,
			SectionID: section.ID,
			Severity:  "ERROR",
		})
		return
	}
	
	// Validate table headers
	headerCells := dv.parseTableRow(dv.lines[headerLine])
	expectedHeaders := section.Table.Headers
	
	if len(headerCells) != len(expectedHeaders) {
		dv.errors = append(dv.errors, ValidationError{
			Type:      "INVALID_TABLE_HEADERS",
			Message:   fmt.Sprintf("Table header mismatch in %s: expected %d columns, found %d", section.Title, len(expectedHeaders), len(headerCells)),
			Line:      headerLine + 1,
			SectionID: section.ID,
			Severity:  "ERROR",
		})
	}
	
	// Count data rows
	rowCount := 0
	for i := headerLine + 2; i < len(dv.lines); i++ {
		line := strings.TrimSpace(dv.lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			break
		}
		if strings.HasPrefix(line, "|") && !strings.HasPrefix(line, "|---") {
			if !section.Table.AllowComment || !strings.Contains(line, "<!--") {
				rowCount++
			}
		}
	}
	
	if rowCount < section.Table.MinRows {
		dv.warnings = append(dv.warnings, ValidationWarning{
			Message: fmt.Sprintf("Table %s has %d rows, expected at least %d", section.Title, rowCount, section.Table.MinRows),
			Line:    headerLine + 1,
		})
	}
}

// validateDiagram validates diagram structure
func (dv *DocumentValidator) validateDiagram(section *Section, startLine int) {
	// if section.Diagram == nil {
	// 	return
	// }
	
	// // Find diagram code block
	// diagramFound := false
	// diagramType := ""
	
	// for i := startLine; i < len(dv.lines) && i < startLine+50; i++ {
	// 	line := strings.TrimSpace(dv.lines[i])
		
	// 	if section.Diagram.RequiredPattern != nil && section.Diagram.RequiredPattern.MatchString(line) {
	// 		diagramFound = true
	// 		// Extract diagram type
	// 		if strings.Contains(line, "mermaid") {
	// 			// Check next line for diagram type
	// 			if i+1 < len(dv.lines) {
	// 				nextLine := strings.TrimSpace(dv.lines[i+1])
	// 				for _, allowedType := range section.Diagram.AllowedTypes {
	// 					if strings.HasPrefix(nextLine, allowedType) {
	// 						diagramType = allowedType
	// 						break
	// 					}
	// 				}
	// 			}
	// 		}
	// 		break
	// 	}
	// }
	
	// if !diagramFound {
	// 	dv.errors = append(dv.errors, ValidationError{
	// 		Type:      "MISSING_DIAGRAM",
	// 		Message:   fmt.Sprintf("Diagram not found for section: %s", section.Title),
	// 		Line:      startLine,
	// 		SectionID: section.ID,
	// 		Severity:  "ERROR",
	// 		Suggestion: section.Diagram.Instructions,
	// 	})
	// 	return
	// }
	
	// // Validate diagram type if specified
	// if len(section.Diagram.AllowedTypes) > 0 && diagramType == "" {
	// 	dv.warnings = append(dv.warnings, ValidationWarning{
	// 		Message: fmt.Sprintf("Could not verify diagram type for %s. Expected one of: %v", section.Title, section.Diagram.AllowedTypes),
	// 		Line:    startLine,
	// 	})
	// }
	
	// // Check for forbidden diagram types
	// for i := startLine; i < len(dv.lines) && i < startLine+50; i++ {
	// 	line := dv.lines[i]
	// 	for _, forbidden := range section.Diagram.ForbiddenTypes {
	// 		if forbidden == "ascii" && dv.containsASCIIArt(line) {
	// 			dv.errors = append(dv.errors, ValidationError{
	// 				Type:      "FORBIDDEN_DIAGRAM_TYPE",
	// 				Message:   "ASCII art diagrams are not allowed. Use Mermaid diagrams instead.",
	// 				Line:      i + 1,
	// 				SectionID: section.ID,
	// 				Severity:  "ERROR",
	// 			})
	// 			break
	// 		}
	// 	}
	// }
	
	// // Validate followed_by element (e.g., table after diagram)
	// if section.FollowedBy != nil {
	// 	dv.validateFollowedBy(section, startLine)
	// }
}

// validateIterativeSection validates repeating sections (e.g., use cases)
func (dv *DocumentValidator) validateIterativeSection(section *Section, startLine int) {
	if !section.Repeatable || section.RepeatPattern == nil {
		return
	}
	
	count := 0
	for i := startLine; i < len(dv.lines); i++ {
		line := dv.lines[i]
		
		// Stop at next same-level or higher section
		if strings.HasPrefix(line, strings.Repeat("#", section.Level)+" ") && i > startLine {
			break
		}
		
		if section.RepeatPattern.MatchString(line) {
			count++
			
			// Validate subsections for each occurrence
			if len(section.Subsections) > 0 {
				for _, sub := range section.Subsections {
					dv.validateSection(sub)
				}
			}
		}
	}
	
	if count == 0 {
		dv.warnings = append(dv.warnings, ValidationWarning{
			Message: fmt.Sprintf("No repeating items found in iterative section: %s", section.Title),
			Line:    startLine,
		})
	}
}

// validateFollowedBy validates elements that must follow a section
func (dv *DocumentValidator) validateFollowedBy(section *Section, startLine int) {
	if section.FollowedBy == nil {
		return
	}
	
	// Find the element that should follow
	switch section.FollowedBy.Type {
	case "table":
		// Look for table after diagram
		tableFound := false
		for i := startLine; i < len(dv.lines) && i < startLine+100; i++ {
			line := dv.lines[i]
			if strings.Contains(line, "|") && !strings.Contains(line, "```") {
				tableFound = true
				
				// Validate headers
				headerCells := dv.parseTableRow(line)
				expectedHeaders := section.FollowedBy.Headers
				
				if len(headerCells) != len(expectedHeaders) {
					dv.errors = append(dv.errors, ValidationError{
						Type:      "INVALID_FOLLOWED_BY_TABLE",
						Message:   fmt.Sprintf("Table after %s has incorrect headers", section.Title),
						Line:      i + 1,
						SectionID: section.ID,
						Severity:  "ERROR",
					})
				}
				break
			}
		}
		
		if !tableFound {
			dv.errors = append(dv.errors, ValidationError{
				Type:      "MISSING_FOLLOWED_BY_ELEMENT",
				Message:   fmt.Sprintf("Table should follow section: %s", section.Title),
				Line:      startLine,
				SectionID: section.ID,
				Severity:  "ERROR",
			})
		}
	}
}

// applyValidationRules applies global validation rules
func (dv *DocumentValidator) applyValidationRules() {
	for _, rule := range dv.template.Rules {
		for i, line := range dv.lines {
			if rule.Pattern.MatchString(line) {
				if rule.Severity == "error" {
					dv.errors = append(dv.errors, ValidationError{
						Type:     rule.Name,
						Message:  rule.Message,
						Line:     i + 1,
						Severity: "ERROR",
					})
				} else {
					dv.warnings = append(dv.warnings, ValidationWarning{
						Message: rule.Message,
						Line:    i + 1,
						Context: line,
					})
				}
			}
		}
	}
}

// Helper methods

func (dv *DocumentValidator) parseTableRow(line string) []string {
	cells := strings.Split(line, "|")
	result := []string{}
	for _, cell := range cells {
		trimmed := strings.TrimSpace(cell)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func (dv *DocumentValidator) containsASCIIArt(line string) bool {
	// Simple check for ASCII art patterns
	asciiPatterns := []string{"+--", "|--", "-->", "<--", "+-+", "|_|"}
	for _, pattern := range asciiPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func (dv *DocumentValidator) calculateStatistics() DocumentStatistics {
	stats := DocumentStatistics{
		TotalLines: len(dv.lines),
	}
	
	for _, line := range dv.lines {
		trimmed := strings.TrimSpace(line)
		
		// Count sections
		if strings.HasPrefix(trimmed, "#") {
			stats.TotalSections++
		}
		
		// Count tables
		if strings.HasPrefix(trimmed, "|") && strings.Contains(trimmed, "---") {
			stats.TotalTables++
		}
		
		// Count diagrams
		if strings.Contains(trimmed, "```mermaid") {
			stats.TotalDiagrams++
		}
		
		// Count use cases
		if strings.Contains(trimmed, "UC-") && strings.HasPrefix(trimmed, "##") {
			stats.UseCaseCount++
		}
	}
	
	return stats
}