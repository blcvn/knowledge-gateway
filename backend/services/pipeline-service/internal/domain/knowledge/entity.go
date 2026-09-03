// Package knowledge defines domain entities from ba-knowledge-service/worker.
//
// Absorbed from: ba-knowledge-service, ba-knowledge-worker
// (MERGE-P3-T1)
package knowledge

import "time"

// PRD is a Product Requirements Document to be indexed.
type PRD struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
	Status    string    `json:"status"` // "draft"|"indexed"|"failed"
	TenantID  string    `json:"tenant_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Outline is the generated structure from a PRD.
type Outline struct {
	ID       string           `json:"id"`
	PRDID    string           `json:"prd_id"`
	Sections []OutlineSection `json:"sections"`
	Status   string           `json:"status"` // "pending"|"ready"|"failed"
}

// OutlineSection is a recursive section within an Outline.
type OutlineSection struct {
	Title    string           `json:"title"`
	Level    int              `json:"level"`
	Content  string           `json:"content,omitempty"`
	Children []OutlineSection `json:"children,omitempty"`
}

// IndexJob is a task to index or outline a PRD.
type IndexJob struct {
	Type   string `json:"type"`   // "index_prd"|"gen_outline"
	PRDID  string `json:"prd_id"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Task type constants (from ba-knowledge-worker).
const (
	TaskTypeIndexPRD   = "index_prd"
	TaskTypeGenOutline = "gen_outline"
	// Cross-engine tasks
	TaskTypeGraphitiIngest = "graphiti.ingest"
	TaskTypeCogneeCognify  = "cognee.cognify"
	TaskTypeMemobaseFlush  = "memobase.flush"
)
