package biz

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// EntityType defines the schema and constraints for a specific node label in Neo4j.
type EntityType struct {
	ID          uint           `gorm:"primaryKey"`
	AppID       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_tenant_entity"`
	TenantID    string         `gorm:"type:varchar(50);not null;default:'default';uniqueIndex:idx_app_tenant_entity"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_tenant_entity"` // e.g. "Customer", "Transaction"
	Description string         `gorm:"type:text"`
	Schema      datatypes.JSON `gorm:"type:jsonb;not null"` // JSON Schema definition for properties
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (EntityType) TableName() string {
	return "kgs_entity_types"
}

// RelationType defines the schema and constraints for a specific edge type in Neo4j.
type RelationType struct {
	ID          uint           `gorm:"primaryKey"`
	AppID       string         `gorm:"type:varchar(50);not null;uniqueIndex:idx_app_tenant_relation"`
	TenantID    string         `gorm:"type:varchar(50);not null;default:'default';uniqueIndex:idx_app_tenant_relation"`
	Name        string         `gorm:"type:varchar(100);not null;uniqueIndex:idx_app_tenant_relation"` // e.g. "PURCHASED", "TRANSFER_TO"
	Description string         `gorm:"type:text"`
	Properties  datatypes.JSON `gorm:"type:jsonb"` // JSON Schema for edge properties (optional)
	SourceTypes datatypes.JSON `gorm:"type:jsonb"` // List of valid source EntityType names
	TargetTypes datatypes.JSON `gorm:"type:jsonb"` // List of valid target EntityType names
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (RelationType) TableName() string {
	return "kgs_relation_types"
}
