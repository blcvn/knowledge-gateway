package model

import (
	"time"

	"github.com/google/uuid"
)

// === File System (ov-fs origin) ===

// File represents an encrypted file or directory stored in VikingFS.
type File struct {
	ID         uuid.UUID `json:"id" db:"id"`
	AccountID  string    `json:"account_id" db:"account_id"`
	Path       string    `json:"path" db:"path"`
	Content    []byte    `json:"-" db:"content"` // Encrypted OVE1 envelope
	IsDir      bool      `json:"is_dir" db:"is_dir"`
	L0Abstract string    `json:"l0_abstract" db:"l0_abstract"` // ~100 token summary
	L1Abstract string    `json:"l1_abstract" db:"l1_abstract"` // ~2K token overview
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// RelationType defines the nature of the link between two files.
type RelationType string

const (
	RelReferences    RelationType = "references"
	RelExtractedFrom RelationType = "extracted_from"
	RelSummarizes    RelationType = "summarizes"
)

// FileRelation maps cross-file references and extraction lineages.
type FileRelation struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	SourceFileID uuid.UUID    `json:"source_file_id" db:"source_file_id"`
	TargetFileID uuid.UUID    `json:"target_file_id" db:"target_file_id"`
	RelationType RelationType `json:"relation_type" db:"relation_type"`
}

// === Cryptography (ov-crypto origin) ===

type ProviderType string

const (
	ProviderLocal  ProviderType = "local"
	ProviderVault  ProviderType = "vault"
	ProviderAWSKMS ProviderType = "aws_kms"
)

// AccountKey tracks the tenant's master encryption key versions.
type AccountKey struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	AccountID    string       `json:"account_id" db:"account_id"`
	KeyVersion   int          `json:"key_version" db:"key_version"`
	ProviderType ProviderType `json:"provider_type" db:"provider_type"`
}

// APIKeyHash stores the Argon2id hash of a tenant's API keys.
type APIKeyHash struct {
	ID        uuid.UUID `json:"id" db:"id"`
	AccountID string    `json:"account_id" db:"account_id"`
	KeyHash   []byte    `json:"-" db:"key_hash"`
	Role      string    `json:"role" db:"role"` // root / admin / user / agent
}

// === Resource Ingestion (ov-resource origin) ===

type ParserType string

const (
	ParserTreeSitter ParserType = "treesitter"
	ParserMarkdown   ParserType = "markdown"
	ParserDocument   ParserType = "document"
)

type ResourceStatus string

const (
	StatusPending    ResourceStatus = "pending"
	StatusProcessing ResourceStatus = "processing"
	StatusCompleted  ResourceStatus = "completed"
)

// Resource tracks an ingestion job mapping external files to VikingFS.
type Resource struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	SourcePath string         `json:"source_path" db:"source_path"`
	TargetPath string         `json:"target_path" db:"target_path"`
	ParserType ParserType     `json:"parser_type" db:"parser_type"`
	Status     ResourceStatus `json:"status" db:"status"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
}

type WatchStatus string

const (
	WatchActive WatchStatus = "active"
	WatchPaused WatchStatus = "paused"
)

// WatchTask configures a background poller for a source directory.
type WatchTask struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	SourcePath     string      `json:"source_path" db:"source_path"`
	PollIntervalMs int64       `json:"poll_interval_ms" db:"poll_interval_ms"`
	Status         WatchStatus `json:"status" db:"status"`
}
