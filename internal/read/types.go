package read

import "time"

type TemplateListItem struct {
	TemplateName string            `json:"template_name"`
	Status       string            `json:"status"`
	ParamSchema  []ParamSchemaItem `json:"param_schema"`
}

type ParamSchemaItem struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required,omitempty"`
}

type NodeResponse struct {
	ID            string         `json:"id"`
	NodeType      string         `json:"node_type"`
	DomainID      string         `json:"domain_id"`
	OwnerTenantID string         `json:"owner_tenant_id"`
	OwnerAppID    string         `json:"owner_app_id"`
	Visibility    string         `json:"visibility"`
	Properties    map[string]any `json:"properties"`
	Relationships []string       `json:"relationships"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type TemplateExecutionRequest struct {
	Params map[string]any `json:"params"`
}

type TemplateExecutionResponse struct {
	Results     []map[string]any `json:"results"`
	QueryTimeMs int              `json:"query_time_ms"`
}
