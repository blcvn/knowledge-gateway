package domain

import (
	"time"
)

type DocumentType string

const (
	DocTypePRD        DocumentType = "PRD"
	DocTypeURDIndex   DocumentType = "URD_INDEX"
	DocTypeURDOutline DocumentType = "URD_OUTLINE"
	DocTypeURDFull    DocumentType = "URD_FULL"
)

type DocumentStatus string

const (
	DocStatusDraft    DocumentStatus = "DRAFT"
	DocStatusInReview DocumentStatus = "IN_REVIEW"
	DocStatusApproved DocumentStatus = "APPROVED"
	DocStatusArchived DocumentStatus = "ARCHIVED"
)

// Document represents a knowledge artifact in Postgres
type Document struct {
	ID          string         `gorm:"primaryKey;type:uuid"`
	ProjectID   string         `gorm:"type:uuid;not null;index"`
	DocType     DocumentType   `gorm:"type:varchar(20);not null"`
	Title       string         `gorm:"type:varchar(255)"`
	Version     string         `gorm:"type:varchar(20);default:'1.0.0'"`
	Status      DocumentStatus `gorm:"type:varchar(20);default:'DRAFT'"`
	S3Key       string         `gorm:"type:varchar(500)"`
	GraphNodeID string         `gorm:"type:varchar(100)"` // Reference to Neo4j node ID
	CreatedBy   string         `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DocumentLineage tracks dependencies between documents
type DocumentLineage struct {
	ID               uint   `gorm:"primaryKey"`
	SourceDocID      string `gorm:"type:uuid;not null;index"`
	TargetDocID      string `gorm:"type:uuid;not null;index"`
	RelationshipType string `gorm:"type:varchar(50)"` // GENERATES, EXPANDS_TO, DERIVED_FROM
	CreatedAt        time.Time
}

// ExternalSource maps internal documents to external systems (Confluence, Jira)
type ExternalSource struct {
	ID            uint   `gorm:"primaryKey"`
	SourceType    string `gorm:"type:varchar(20)"` // CONFLUENCE, JIRA
	ExternalID    string `gorm:"type:varchar(255);index"`
	InternalDocID string `gorm:"type:uuid;not null;index"`
	SyncStatus    string `gorm:"type:varchar(20)"` // SYNCED, OUTDATED, CONFLICT
	LastSyncAt    time.Time
}
