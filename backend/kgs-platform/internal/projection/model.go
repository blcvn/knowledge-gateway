package projection

import (
	"encoding/json"
	"time"
)

type ViewDefinition struct {
	ID                 string    `json:"id"`
	AppID              string    `json:"app_id"`
	TenantID           string    `json:"tenant_id"`
	RoleName           string    `json:"role_name"`
	AllowedEntityTypes []string  `json:"allowed_entity_types"`
	AllowedFields      []string  `json:"allowed_fields"`
	PIIMaskFields      []string  `json:"pii_mask_fields"`
	CreatedAt          time.Time `json:"created_at"`
}

type ViewDefinitionRecord struct {
	ID                     string    `gorm:"primaryKey;size:64"`
	AppID                  string    `gorm:"index:idx_view_ns_role,priority:1;size:128"`
	TenantID               string    `gorm:"index:idx_view_ns_role,priority:2;size:128"`
	RoleName               string    `gorm:"index:idx_view_ns_role,priority:3;size:64"`
	AllowedEntityTypesJSON string    `gorm:"column:allowed_entity_types;type:text"`
	AllowedFieldsJSON      string    `gorm:"column:allowed_fields;type:text"`
	PIIMaskFieldsJSON      string    `gorm:"column:pii_mask_fields;type:text"`
	CreatedAt              time.Time `gorm:"autoCreateTime"`
}

func (ViewDefinitionRecord) TableName() string {
	return "kgs_view_definitions"
}

func (r ViewDefinitionRecord) toDomain() ViewDefinition {
	return ViewDefinition{
		ID:                 r.ID,
		AppID:              r.AppID,
		TenantID:           r.TenantID,
		RoleName:           r.RoleName,
		AllowedEntityTypes: decodeJSONStringArray(r.AllowedEntityTypesJSON),
		AllowedFields:      decodeJSONStringArray(r.AllowedFieldsJSON),
		PIIMaskFields:      decodeJSONStringArray(r.PIIMaskFieldsJSON),
		CreatedAt:          r.CreatedAt,
	}
}

func toRecord(view ViewDefinition) ViewDefinitionRecord {
	return ViewDefinitionRecord{
		ID:                     view.ID,
		AppID:                  view.AppID,
		TenantID:               view.TenantID,
		RoleName:               view.RoleName,
		AllowedEntityTypesJSON: encodeJSONStringArray(view.AllowedEntityTypes),
		AllowedFieldsJSON:      encodeJSONStringArray(view.AllowedFields),
		PIIMaskFieldsJSON:      encodeJSONStringArray(view.PIIMaskFields),
	}
}

func encodeJSONStringArray(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func decodeJSONStringArray(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return []string{}
	}
	return out
}
