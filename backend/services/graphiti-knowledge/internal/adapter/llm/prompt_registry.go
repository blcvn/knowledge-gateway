package llm

import (
	"bytes"
	"errors"
	"text/template"
	"vnp-memory/services/graphiti-knowledge/domain"
	"vnp-memory/services/graphiti-knowledge/usecase/port"
)

var ErrPromptNotFound = errors.New("prompt template not found")

type promptRegistry struct {
	templates map[string]domain.PromptTemplate
	compiled  map[string]*template.Template
}

func NewPromptRegistry() port.PromptRegistry {
	r := &promptRegistry{
		templates: make(map[string]domain.PromptTemplate),
		compiled:  make(map[string]*template.Template),
	}
	r.registerDefaults()
	return r
}

func (r *promptRegistry) registerDefaults() {
	defaults := []domain.PromptTemplate{
		{ID: "extract_entities", Model: "gpt-4o", SystemPrompt: "Extract named entities from content", UserPrompt: "{{.Content}} {{.PreviousEpisodes}} {{.EntityTypes}}"},
		{ID: "resolve_entities", Model: "gpt-4o-mini", SystemPrompt: "Compare extracted vs existing for dedup", UserPrompt: "{{.Extracted}} {{.Existing}}"},
		{ID: "extract_edges", Model: "gpt-4o", SystemPrompt: "Extract fact triples with temporal info", UserPrompt: "{{.Content}} {{.Entities}} {{.PreviousEpisodes}}"},
		{ID: "resolve_edges", Model: "gpt-4o-mini", SystemPrompt: "Detect contradictions between edges", UserPrompt: "{{.NewEdge}} {{.ExistingEdges}}"},
		{ID: "summarize_community", Model: "gpt-4o-mini", SystemPrompt: "Generate community summary", UserPrompt: "{{.Members}} {{.Edges}}"},
		{ID: "classify_entity", Model: "gpt-4o-mini", SystemPrompt: "Classify entity type from context", UserPrompt: "{{.Entity}} {{.Context}}"},
		{ID: "expand_summary", Model: "gpt-4o-mini", SystemPrompt: "Expand entity summary with new info", UserPrompt: "{{.Entity}} {{.NewContext}}"},
	}

	for _, t := range defaults {
		r.templates[t.ID] = t
		tmpl, _ := template.New(t.ID).Parse(t.UserPrompt)
		r.compiled[t.ID] = tmpl
	}
}

func (r *promptRegistry) Render(templateID string, vars map[string]interface{}) (string, error) {
	tmpl, ok := r.compiled[templateID]
	if !ok {
		return "", ErrPromptNotFound
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (r *promptRegistry) GetModel(templateID string) string {
	if t, ok := r.templates[templateID]; ok {
		return t.Model
	}
	return ""
}

func (r *promptRegistry) List() []domain.PromptTemplate {
	list := make([]domain.PromptTemplate, 0, len(r.templates))
	for _, t := range r.templates {
		list = append(list, t)
	}
	return list
}
