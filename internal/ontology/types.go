package ontology

import "time"

type Domain struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description,omitempty"`
	OwnerTenantID  string    `json:"owner_tenant_id"`
	ParentDomainID string    `json:"parent_domain_id,omitempty"`
	Status         string    `json:"status"`
	Version        int       `json:"version"`
	Visibility     string    `json:"visibility"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type OntologyVersion struct {
	DomainID       string         `json:"domain_id"`
	Version        int            `json:"version"`
	BreakingChange bool           `json:"breaking_change"`
	Changes        map[string]any `json:"changes,omitempty"`
	PublishedAt    time.Time      `json:"published_at"`
}

type NodeTypeSchema struct {
	ID              string           `json:"id"`
	DomainID        string           `json:"domain_id"`
	NodeTypeName    string           `json:"node_type_name"`
	GraphLabel      string           `json:"graph_label"`
	RequiredProps   []PropertySchema `json:"required_props"`
	OptionalProps   []PropertySchema `json:"optional_props"`
	ValidationRules []string         `json:"validation_rules"`
	Version         int              `json:"version"`
	CreatedAt       time.Time        `json:"created_at"`
}

type RelTypeSchema struct {
	ID            string           `json:"id"`
	DomainID      string           `json:"domain_id"`
	RelTypeName   string           `json:"rel_type_name"`
	FromNodeType  string           `json:"from_node_type"`
	ToNodeType    string           `json:"to_node_type"`
	SameDomain    bool             `json:"same_domain"`
	RequiredProps []PropertySchema `json:"required_props"`
	OptionalProps []PropertySchema `json:"optional_props"`
}

type CrossDomainRelRule struct {
	ID                string   `json:"id"`
	RelTypeName       string   `json:"rel_type_name"`
	FromDomainID      string   `json:"from_domain_id"`
	ToDomainID        string   `json:"to_domain_id"`
	FromNodeTypes     []string `json:"from_node_types"`
	ToNodeTypes       []string `json:"to_node_types"`
	Required          bool     `json:"required"`
	ExceptionTypes    []string `json:"exception_types,omitempty"`
	BridgePropertyKey string   `json:"bridge_property_key"`
}

type PropertySchema struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type QueryTemplate struct {
	ID           string            `json:"id"`
	DomainID     string            `json:"domain_id"`
	TemplateName string            `json:"template_name"`
	PatternSpec  map[string]any    `json:"pattern_spec"`
	ParamSchema  []ParameterSchema `json:"param_schema"`
	ReturnFields []string          `json:"return_fields"`
	Description  string            `json:"description,omitempty"`
	Status       string            `json:"status"`
	Version      int               `json:"version"`
	CreatedAt    time.Time         `json:"created_at"`
}

type ParameterSchema struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

type StatusFieldConfig struct {
	DomainID            string         `json:"domain_id"`
	StatusFieldName     string         `json:"status_field_name"`
	ValidStatusValues   []string       `json:"valid_status_values"`
	WarningStatusValues []string       `json:"warning_status_values,omitempty"`
	CascadeRules        []CascadeRule  `json:"cascade_rules,omitempty"`
	AuthorityFieldName  string         `json:"authority_field_name,omitempty"`
	AuthorityValuesMap  map[string]int `json:"authority_values_map,omitempty"`
}

type CascadeRule struct {
	FromNodeType string `json:"from_node_type"`
	ViaRel       string `json:"via_rel"`
	ToNodeType   string `json:"to_node_type"`
}

type DomainCreateRequest struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Visibility  string `json:"visibility"`
}

type NodeTypeCreateRequest struct {
	NodeTypeName    string           `json:"node_type_name"`
	GraphLabel      string           `json:"graph_label"`
	RequiredProps   []PropertySchema `json:"required_props"`
	OptionalProps   []PropertySchema `json:"optional_props"`
	ValidationRules []string         `json:"validation_rules"`
}

type RelTypeCreateRequest struct {
	RelTypeName   string           `json:"rel_type_name"`
	FromNodeType  string           `json:"from_node_type"`
	ToNodeType    string           `json:"to_node_type"`
	SameDomain    bool             `json:"same_domain"`
	RequiredProps []PropertySchema `json:"required_props"`
	OptionalProps []PropertySchema `json:"optional_props"`
}

type QueryTemplateCreateRequest struct {
	TemplateName string            `json:"template_name"`
	PatternSpec  map[string]any    `json:"pattern_spec"`
	ParamSchema  []ParameterSchema `json:"param_schema"`
	ReturnFields []string          `json:"return_fields"`
	Description  string            `json:"description"`
}

type StatusFieldConfigRequest struct {
	StatusFieldName     string         `json:"status_field_name"`
	ValidStatusValues   []string       `json:"valid_status_values"`
	WarningStatusValues []string       `json:"warning_status_values"`
	CascadeRules        []CascadeRule  `json:"cascade_rules"`
	AuthorityFieldName  string         `json:"authority_field_name"`
	AuthorityValuesMap  map[string]int `json:"authority_values_map"`
}

type EffectiveOntologyResponse struct {
	Domains []DomainSummary `json:"domains"`
}

type DomainSummary struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
}

type DomainDetailsResponse struct {
	Domain            Domain             `json:"domain"`
	NodeTypes         []NodeTypeSchema   `json:"node_types"`
	RelTypes          []RelTypeSchema    `json:"rel_types"`
	QueryTemplates    []QueryTemplate    `json:"query_templates"`
	StatusFieldConfig *StatusFieldConfig `json:"status_field_config,omitempty"`
}
