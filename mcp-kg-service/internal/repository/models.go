package repository

import "time"

// RequirementNodeModel mirrors the requirement_nodes table used by KG tooling.
type RequirementNodeModel struct {
	ID          string `gorm:"primaryKey;column:id"`
	DocumentID  string `gorm:"column:document_id;index"`
	ReferenceID string `gorm:"column:reference_id;index"`
	Type        string `gorm:"column:type;index"`
	Summary     string `gorm:"column:summary"`
	Description string `gorm:"column:description"`
	SourceID    string `gorm:"column:source_id"`
	Metadata    []byte `gorm:"column:metadata;type:jsonb"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (RequirementNodeModel) TableName() string { return "requirement_nodes" }

// DependencyEdgeModel mirrors the dependency_edges table used by KG tooling.
type DependencyEdgeModel struct {
	ID         string `gorm:"primaryKey;column:id"`
	DocumentID string `gorm:"column:document_id;index"`
	SourceID   string `gorm:"column:source_id;index"`
	TargetID   string `gorm:"column:target_id;index"`
	Type       string `gorm:"column:type;index"`
	Reason     string `gorm:"column:reason"`
	CreatedAt  time.Time
}

func (DependencyEdgeModel) TableName() string { return "dependency_edges" }

// DocumentArtifactModel stores rendered markdown artifacts associated with one graph document.
type DocumentArtifactModel struct {
	DocumentID  string `gorm:"primaryKey;column:document_id"`
	DocKind     string `gorm:"primaryKey;column:doc_kind"`
	Content     string `gorm:"column:content"`
	ContentType string `gorm:"column:content_type"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (DocumentArtifactModel) TableName() string { return "document_artifacts" }
