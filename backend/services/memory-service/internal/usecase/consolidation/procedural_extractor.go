package consolidation

import (
	"context"
	"strings"
	"time"
)

// Step is a single procedural step extracted from observations.
type Step struct {
	Order       int
	Description string
	ToolsUsed   []string
}

// ProceduralMemory is a reusable workflow extracted from a session.
type ProceduralMemory struct {
	ID          string
	TenantID    string
	Project     string
	Title       string
	Description string
	Steps       []Step
	Tags        []string
	SessionID   string
	CreatedAt   time.Time
}

// ProceduralExtractorUseCase extracts reusable procedural memories from session summaries.
type ProceduralExtractorUseCase struct{}

func NewProceduralExtractorUseCase() *ProceduralExtractorUseCase {
	return &ProceduralExtractorUseCase{}
}

// ExtractRequest is the input for procedural extraction.
type ExtractRequest struct {
	TenantID  string
	SessionID string
	Project   string
	Narrative string
	Steps     []string
	Tags      []string
}

// ExtractResult contains any procedural memories found.
type ExtractResult struct {
	Found    bool
	Memories []ProceduralMemory
}

// Execute analyzes a session narrative to extract repeatable procedural workflows.
func (uc *ProceduralExtractorUseCase) Execute(_ context.Context, req ExtractRequest) (*ExtractResult, error) {
	if len(req.Steps) == 0 {
		return &ExtractResult{Found: false}, nil
	}

	steps := make([]Step, 0, len(req.Steps))
	for i, s := range req.Steps {
		steps = append(steps, Step{
			Order:       i + 1,
			Description: s,
			ToolsUsed:   extractTools(s),
		})
	}

	title := "Workflow: " + summarizeTitle(req.Narrative)
	mem := ProceduralMemory{
		TenantID:    req.TenantID,
		Project:     req.Project,
		Title:       title,
		Description: req.Narrative,
		Steps:       steps,
		Tags:        req.Tags,
		SessionID:   req.SessionID,
		CreatedAt:   time.Now(),
	}

	return &ExtractResult{Found: true, Memories: []ProceduralMemory{mem}}, nil
}

func extractTools(step string) []string {
	var tools []string
	keywords := []string{"read_file", "write_file", "run_command", "search", "grep", "git"}
	for _, kw := range keywords {
		if strings.Contains(strings.ToLower(step), kw) {
			tools = append(tools, kw)
		}
	}
	return tools
}

func summarizeTitle(narrative string) string {
	words := strings.Fields(narrative)
	if len(words) > 5 {
		return strings.Join(words[:5], " ") + "..."
	}
	return narrative
}
