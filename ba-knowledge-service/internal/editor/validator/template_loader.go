package validator

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// TemplateConfig represents the YAML structure
type TemplateConfig struct {
	Name        string                 `yaml:"name"`
	Type        string                 `yaml:"type"`
	Version     string                 `yaml:"version"`
	Description string                 `yaml:"description"`
	Metadata    []MetadataConfig       `yaml:"metadata"`
	Sections    []SectionConfig        `yaml:"sections"`
	Rules       []ValidationRuleConfig `yaml:"validation_rules"`
}

type MetadataConfig struct {
	Name     string `yaml:"name"`
	Pattern  string `yaml:"pattern"`
	Required bool   `yaml:"required"`
}

type SectionConfig struct {
	ID            string            `yaml:"id"`
	Title         string            `yaml:"title"`
	Level         int               `yaml:"level"`
	Type          string            `yaml:"type"`
	Required      bool              `yaml:"required"`
	Pattern       string            `yaml:"pattern"`
	Repeatable    bool              `yaml:"repeatable"`
	RepeatPattern string            `yaml:"repeat_pattern"`
	Table         *TableConfig      `yaml:"table"`
	Diagram       *DiagramConfig    `yaml:"diagram"`
	FollowedBy    *FollowedByConfig `yaml:"followed_by"`
	Subsections   []SectionConfig   `yaml:"subsections"`
	Instructions  string            `yaml:"instructions"`
}

type TableConfig struct {
	Headers      []string `yaml:"headers"`
	MinRows      int      `yaml:"min_rows"`
	AllowComment bool     `yaml:"allow_comment"`
}

type DiagramConfig struct {
	Type            string   `yaml:"type"`
	AllowedTypes    []string `yaml:"allowed_types"`
	ForbiddenTypes  []string `yaml:"forbidden_types"`
	RequiredPattern string   `yaml:"required_pattern"`
	Instructions    string   `yaml:"instructions"`
}

type FollowedByConfig struct {
	Type    string   `yaml:"type"`
	Headers []string `yaml:"headers"`
}

type ValidationRuleConfig struct {
	Name     string `yaml:"name"`
	Pattern  string `yaml:"pattern"`
	Severity string `yaml:"severity"`
	Message  string `yaml:"message"`
}

// Template represents compiled template
type Template struct {
	Config   *TemplateConfig
	Metadata []*MetadataField
	Sections []*Section
	Rules    []*ValidationRule
}

type MetadataField struct {
	Name     string
	Pattern  *regexp.Regexp
	Required bool
}

type Section struct {
	ID            string
	Title         string
	Level         int
	Type          SectionType
	Required      bool
	Pattern       *regexp.Regexp
	Repeatable    bool
	RepeatPattern *regexp.Regexp
	Table         *TableDefinition
	Diagram       *DiagramDefinition
	FollowedBy    *FollowedByDefinition
	Subsections   []*Section
	Instructions  string
}

type SectionType string

const (
	TableSection     SectionType = "table"
	DiagramSection   SectionType = "diagram"
	TextSection      SectionType = "text"
	ListSection      SectionType = "list"
	IterativeSection SectionType = "iterative"
	SectionGroup     SectionType = "section"
)

type TableDefinition struct {
	Headers      []string
	MinRows      int
	AllowComment bool
}

type DiagramDefinition struct {
	Type            string
	AllowedTypes    []string
	ForbiddenTypes  []string
	RequiredPattern *regexp.Regexp
	Instructions    string
}

type FollowedByDefinition struct {
	Type    string
	Headers []string
}

type ValidationRule struct {
	Name     string
	Pattern  *regexp.Regexp
	Severity string
	Message  string
}

// TemplateLoader loads templates from files
type TemplateLoader struct {
	templatesDir string
	templates    map[string]*Template
}

// NewTemplateLoader creates a new template loader
func NewTemplateLoader(templatesDir string) *TemplateLoader {
	return &TemplateLoader{
		templatesDir: templatesDir,
		templates:    make(map[string]*Template),
	}
}

// LoadTemplate loads a specific template file
func (tl *TemplateLoader) LoadTemplate(filename string) (*Template, error) {
	filepath := filepath.Join(tl.templatesDir, filename)

	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	var config TemplateConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse template YAML: %w", err)
	}

	template, err := tl.compileTemplate(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to compile template: %w", err)
	}

	tl.templates[config.Type] = template
	return template, nil
}

// LoadAllTemplates loads all templates from directory
func (tl *TemplateLoader) LoadAllTemplates() error {
	files, err := ioutil.ReadDir(tl.templatesDir)
	if err != nil {
		return fmt.Errorf("failed to read templates directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		ext := filepath.Ext(file.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		if _, err := tl.LoadTemplate(file.Name()); err != nil {
			return fmt.Errorf("failed to load template %s: %w", file.Name(), err)
		}
	}

	return nil
}

// GetTemplate retrieves a loaded template by type
func (tl *TemplateLoader) GetTemplate(templateType string) (*Template, error) {
	template, exists := tl.templates[templateType]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateType)
	}
	return template, nil
}

// ListTemplates returns all loaded template types
func (tl *TemplateLoader) ListTemplates() []string {
	types := make([]string, 0, len(tl.templates))
	for t := range tl.templates {
		types = append(types, t)
	}
	return types
}

// compileTemplate compiles config to template with regex patterns
func (tl *TemplateLoader) compileTemplate(config *TemplateConfig) (*Template, error) {
	template := &Template{
		Config:   config,
		Metadata: make([]*MetadataField, 0),
		Sections: make([]*Section, 0),
		Rules:    make([]*ValidationRule, 0),
	}

	// Compile metadata
	for _, m := range config.Metadata {
		pattern, err := regexp.Compile(m.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid metadata pattern for %s: %w", m.Name, err)
		}

		template.Metadata = append(template.Metadata, &MetadataField{
			Name:     m.Name,
			Pattern:  pattern,
			Required: m.Required,
		})
	}

	// Compile sections
	for _, s := range config.Sections {
		section, err := tl.compileSection(&s)
		if err != nil {
			return nil, fmt.Errorf("failed to compile section %s: %w", s.ID, err)
		}
		template.Sections = append(template.Sections, section)
	}

	// Compile validation rules
	for _, r := range config.Rules {
		pattern, err := regexp.Compile(r.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid rule pattern for %s: %w", r.Name, err)
		}

		template.Rules = append(template.Rules, &ValidationRule{
			Name:     r.Name,
			Pattern:  pattern,
			Severity: r.Severity,
			Message:  r.Message,
		})
	}

	return template, nil
}

// compileSection compiles section config recursively
func (tl *TemplateLoader) compileSection(config *SectionConfig) (*Section, error) {
	section := &Section{
		ID:           config.ID,
		Title:        config.Title,
		Level:        config.Level,
		Type:         SectionType(config.Type),
		Required:     config.Required,
		Repeatable:   config.Repeatable,
		Instructions: config.Instructions,
	}

	// Compile pattern
	if config.Pattern != "" {
		pattern, err := regexp.Compile(config.Pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid section pattern: %w", err)
		}
		section.Pattern = pattern
	}

	// Compile repeat pattern
	if config.RepeatPattern != "" {
		pattern, err := regexp.Compile(config.RepeatPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid repeat pattern: %w", err)
		}
		section.RepeatPattern = pattern
	}

	// Compile table definition
	if config.Table != nil {
		section.Table = &TableDefinition{
			Headers:      config.Table.Headers,
			MinRows:      config.Table.MinRows,
			AllowComment: config.Table.AllowComment,
		}
	}

	// Compile diagram definition
	if config.Diagram != nil {
		diagramDef := &DiagramDefinition{
			Type:           config.Diagram.Type,
			AllowedTypes:   config.Diagram.AllowedTypes,
			ForbiddenTypes: config.Diagram.ForbiddenTypes,
			Instructions:   config.Diagram.Instructions,
		}

		if config.Diagram.RequiredPattern != "" {
			pattern, err := regexp.Compile(config.Diagram.RequiredPattern)
			if err != nil {
				return nil, fmt.Errorf("invalid diagram pattern: %w", err)
			}
			diagramDef.RequiredPattern = pattern
		}

		section.Diagram = diagramDef
	}

	// Compile followed_by definition
	if config.FollowedBy != nil {
		section.FollowedBy = &FollowedByDefinition{
			Type:    config.FollowedBy.Type,
			Headers: config.FollowedBy.Headers,
		}
	}

	// Compile subsections recursively
	for _, sub := range config.Subsections {
		subsection, err := tl.compileSection(&sub)
		if err != nil {
			return nil, fmt.Errorf("failed to compile subsection %s: %w", sub.ID, err)
		}
		section.Subsections = append(section.Subsections, subsection)
	}

	return section, nil
}
