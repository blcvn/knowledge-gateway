package dto

import "github.com/google/uuid"

// TriggerCognifyRequest is the input for starting a cognification pipeline.
type TriggerCognifyRequest struct {
	TenantID      string    `json:"tenant_id"`
	DatasetID     uuid.UUID `json:"dataset_id"`
	EntryIDs      []string  `json:"entry_ids,omitempty"`
	NodeSets      []string  `json:"node_sets,omitempty"`
	ChunkSize     int       `json:"chunk_size,omitempty"`
	ChunkOverlap  int       `json:"chunk_overlap,omitempty"`
	SkipDedup     bool      `json:"skip_dedup,omitempty"`
	SkipSummarize bool      `json:"skip_summarize,omitempty"`
	OntologyID    string    `json:"ontology_id,omitempty"`
}

// TriggerCognifyReq is an alias for backward compat.
type TriggerCognifyReq = TriggerCognifyRequest
