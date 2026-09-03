package prompt

import (
	"fmt"

	"vnp-memory/pkg/graph"
)

// PromptContext contains the inputs needed to build a user prompt message
type PromptContext struct {
	Chunks        []string
	PrevEpisodes  []string                       // recent episode content for context window
	ExistingNodes []string                       // for dedupe: candidate entity names
	EntityTypes   map[string]graph.EntityTypeSchema
	EdgeTypes     map[string]graph.EdgeTypeSchema
	ReferenceTime string                         // ISO8601 for temporal context
	Language      string                         // e.g. "vi", "en" (empty = auto-detect)
	Source        string                         // episode source type
}

// PromptTemplate defines a complete LLM prompt configuration
type PromptTemplate struct {
	Name         string
	SystemPrompt string
	BuildUser    func(ctx PromptContext) string
	Schema       interface{} // JSON schema for structured output
}

// PromptRegistry stores all available prompt templates
type PromptRegistry struct {
	templates map[string]PromptTemplate
}

// NewPromptRegistry creates a registry pre-loaded with all 6 graphiti templates
func NewPromptRegistry() *PromptRegistry {
	reg := &PromptRegistry{templates: make(map[string]PromptTemplate)}
	reg.Register(extractNodesPrompt())
	reg.Register(extractEdgesPrompt())
	reg.Register(dedupeNodesPrompt())
	reg.Register(dedupeEdgesPrompt())
	reg.Register(summarizeNodesPrompt())
	reg.Register(summarizeSagasPrompt())
	return reg
}

func (r *PromptRegistry) Register(t PromptTemplate) {
	r.templates[t.Name] = t
}

func (r *PromptRegistry) Get(name string) (PromptTemplate, error) {
	t, ok := r.templates[name]
	if !ok {
		return PromptTemplate{}, fmt.Errorf("prompt template %q not found", name)
	}
	return t, nil
}

func (r *PromptRegistry) MustGet(name string) PromptTemplate {
	t, err := r.Get(name)
	if err != nil {
		panic(err)
	}
	return t
}

// Sanitize removes control characters that can cause LLM API errors
func Sanitize(text string) string {
	result := make([]rune, 0, len(text))
	for _, r := range text {
		if r < 32 && r != '\n' && r != '\t' && r != '\r' {
			continue
		}
		result = append(result, r)
	}
	return string(result)
}
