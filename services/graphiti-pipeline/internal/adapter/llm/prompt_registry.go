package llm

import (
	"bytes"
	"text/template"
)

type PromptRegistry struct {
	templates map[string]*template.Template
}

func NewPromptRegistry() *PromptRegistry {
	return &PromptRegistry{
		templates: make(map[string]*template.Template),
	}
}

func (r *PromptRegistry) LoadTemplate(name string, tmplString string) error {
	tmpl, err := template.New(name).Parse(tmplString)
	if err != nil {
		return err
	}
	r.templates[name] = tmpl
	return nil
}

func (r *PromptRegistry) Render(name string, data interface{}) (string, error) {
	tmpl, ok := r.templates[name]
	if !ok {
		return "", nil // Or error "template not found"
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
