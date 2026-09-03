// Package cognee defines domain entities for the Cognee adapter.
//
// Absorbed from: cognee-ingestion, cognee-cognify, cognee-search, cognee-pipeline
// (MERGE-P2-T2) — kg-service acts as HTTP adapter to Cognee Python service.
//
// Extensions:
//   - CR-COGNEE-001: MemifyJob — non-destructive graph enrichment
//   - CR-COGNEE-002: NodeSets — tag-based memory partitioning on DataItem + SearchRequest
//   - CR-COGNEE-003: DataPoint — schema-defined atomic knowledge unit
package cognee

import "time"

// Dataset represents a Cognee dataset tracked in kg-service.
type Dataset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TenantID  string    `json:"tenant_id"`
	Status    string    `json:"status"` // "empty"|"uploading"|"ready"|"cognifying"|"indexed"
	DataCount int       `json:"data_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ContentType defines the supported types of ingested data.
type ContentType string

const (
	ContentTypeText       ContentType = "text"
	ContentTypePDF        ContentType = "pdf"
	ContentTypeURL        ContentType = "url"
	ContentTypeJSON       ContentType = "json"
	ContentTypePDFLayout  ContentType = "PDF_LAYOUT" // [CR-COGNEE-004] Advanced Layout-aware PDF
	ContentTypeTabularFK  ContentType = "TABULAR_FK" // [CR-COGNEE-004] Tabular Data with FK edges
)

// DataItem represents a piece of data to upload to a dataset.
// CR-COGNEE-002: NodeSets enables tag-based memory partitioning.
// CR-COGNEE-004: Supports advanced extraction configs.
type DataItem struct {
	DatasetID   string         `json:"dataset_id"`
	ContentType ContentType    `json:"content_type"`
	Content     []byte         `json:"content,omitempty"`
	URI         string         `json:"uri,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	NodeSets    []string       `json:"node_sets,omitempty"` // [CR-COGNEE-002]
	Config      map[string]any `json:"config,omitempty"`    // [CR-COGNEE-004] e.g. {"pdf_mode": "LAYOUT_AWARE"}
}

// PipelineTemplate represents a predefined custom pipeline template.
type PipelineTemplate string

const (
	TemplateStandard  PipelineTemplate = "STANDARD"
	TemplateEmbedOnly PipelineTemplate = "EMBED_ONLY" // [CR-COGNEE-006]
	TemplateFastIndex PipelineTemplate = "FAST_INDEX" // [CR-COGNEE-006]
	TemplateTemporal  PipelineTemplate = "TEMPORAL"   // [CR-COGNEE-006]
	TemplateGraphOnly PipelineTemplate = "GRAPH_ONLY" // [CR-COGNEE-006]
)

// PipelineConfig overrides the default cognify pipeline behavior.
// CR-COGNEE-006: Custom Pipelines
type PipelineConfig struct {
	Template     PipelineTemplate `json:"template,omitempty"`
	Steps        []string         `json:"steps,omitempty"`
	ChunkSize    int              `json:"chunk_size,omitempty"`
	CustomPrompt string           `json:"custom_prompt,omitempty"`
	TemporalMode bool             `json:"temporal_mode,omitempty"`
}

// CognifyJob tracks an async cognification pipeline run.
type CognifyJob struct {
	JobID     string     `json:"job_id"`
	DatasetID string     `json:"dataset_id"`
	Status    string     `json:"status"` // "pending"|"running"|"completed"|"failed"
	Progress  float64    `json:"progress"`
	StartedAt time.Time  `json:"started_at"`
	DoneAt    *time.Time `json:"done_at,omitempty"`
}

// MemifyJob tracks an async non-destructive graph enrichment run.
// CR-COGNEE-001: Memify enriches the knowledge graph without rebuilding.
type MemifyJob struct {
	PipelineRunID string     `json:"pipeline_run_id"`
	DatasetID     string     `json:"dataset_id"`
	TenantID      string     `json:"tenant_id"`
	Status        string     `json:"status"` // "QUEUED"|"RUNNING"|"COMPLETED"|"FAILED"
	DeriveFacts   bool       `json:"derive_facts"`
	EmbedTriplets bool       `json:"embed_triplets"`
	BatchSize     int        `json:"batch_size"`
	NewNodes      int        `json:"new_nodes,omitempty"`
	NewEdges      int        `json:"new_edges,omitempty"`
	StartedAt     time.Time  `json:"started_at"`
	DoneAt        *time.Time `json:"done_at,omitempty"`
}

// MemifyConfig holds optional configuration for a Memify run.
// CR-COGNEE-001
type MemifyConfig struct {
	DeriveFacts   bool `json:"derive_facts"`    // default true
	EmbedTriplets bool `json:"embed_triplets"` // default true
	BatchSize     int  `json:"batch_size"`     // default 50
}

// SearchRequest encapsulates search parameters including optional NodeSet filter.
// CR-COGNEE-002: NodeSets narrows search scope to tagged partitions.
// CR-COGNEE-005: Feedback Loop - save interactions or submit feedback.
type SearchRequest struct {
	Query           string   `json:"query"`
	SearchType      string   `json:"search_type"` // "semantic"|"graph"|"hybrid"|"GRAPH_COMPLETION"|"FEEDBACK"
	DatasetID       string   `json:"dataset_id,omitempty"`
	NodeSets        []string `json:"node_sets,omitempty"` // [CR-COGNEE-002] filter by node tags
	TopK            int      `json:"top_k,omitempty"`
	SaveInteraction bool     `json:"save_interaction,omitempty"` // [CR-COGNEE-005] log this query
	FeedbackFor     *string  `json:"feedback_for,omitempty"`     // [CR-COGNEE-005] ID of interaction to rate
	FeedbackScore   *float64 `json:"feedback_score,omitempty"`   // [CR-COGNEE-005] -1.0 to 1.0 rating
}

// SearchResult from Cognee Python service.
type SearchResult struct {
	Content  string         `json:"content"`
	Score    float64        `json:"score"`
	Source   string         `json:"source"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// DatasetResponse from Cognee API.
type DatasetResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DataPoint is an atomic knowledge unit with stable identity and versioning.
// CR-COGNEE-003: Bypasses LLM entity extraction by mapping schema directly to graph.
type DataPoint struct {
	ID          string             `json:"id"`           // UUID string (stable identity)
	Type        string             `json:"type"`         // Schema type: "Paper", "User", "Product", ...
	Fields      map[string]any     `json:"fields"`       // All field values
	IndexFields []string           `json:"index_fields"` // Only embed these fields
	Relations   []DataPointRelation `json:"relations,omitempty"`
}

// DataPointRelation describes an explicit edge between DataPoints.
// CR-COGNEE-003
type DataPointRelation struct {
	TargetID string  `json:"target_id"`
	Label    string  `json:"label"`  // Edge label: "authored_by", "belongs_to", ...
	Weight   float64 `json:"weight"` // Edge weight
}

// AddDataPointsRequest is the request to ingest structured DataPoints.
// CR-COGNEE-003: Supports NodeSet tagging (CR-COGNEE-002).
type AddDataPointsRequest struct {
	DatasetID  string      `json:"dataset_id"`
	TenantID   string      `json:"tenant_id"`
	DataPoints []DataPoint `json:"data_points"`
	NodeSets   []string    `json:"node_sets,omitempty"` // [CR-COGNEE-002]
}

// AddDataPointsResponse summarises the result of a DataPoint ingestion.
// CR-COGNEE-003
type AddDataPointsResponse struct {
	Accepted int      `json:"accepted"`
	IDs      []string `json:"ids"`
}
