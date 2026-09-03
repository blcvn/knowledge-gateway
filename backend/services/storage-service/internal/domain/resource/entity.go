// Package resource defines domain entities for resource ingestion.
//
// Part of storage-service (MERGE-P1-T4: absorbs ov-resource)
package resource

import "time"

// Resource represents an ingested external resource (file, URL, etc.).
type Resource struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenant_id"`
	URI       string    `json:"uri"`   // file:// | http:// | s3://
	Type      string    `json:"type"`  // "document" | "image" | "code" | "web"
	Status    string    `json:"status"` // "pending" | "processing" | "indexed" | "failed"
	EmbedPath string    `json:"embed_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// IngestJob is a request to ingest a resource.
type IngestJob struct {
	ResourceID string       `json:"resource_id"`
	URI        string       `json:"uri"`
	TenantID   string       `json:"tenant_id"`
	Options    IngestOptions `json:"options"`
	CreatedAt  time.Time    `json:"created_at"`
}

// IngestOptions controls how a resource is processed.
type IngestOptions struct {
	ChunkSize  int    `json:"chunk_size"`   // token chunk size for embedding
	Overlap    int    `json:"overlap"`      // overlap between chunks
	Language   string `json:"language"`     // hint for code syntax
	ExtractPDF bool   `json:"extract_pdf"`  // extract text from PDF
}
