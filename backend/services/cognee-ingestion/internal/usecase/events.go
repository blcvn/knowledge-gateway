package usecase

// DataIngestedEvent is published to NATS after data is ingested.
// Downstream cognee-cognify service consumes this to trigger the knowledge graph pipeline.
type DataIngestedEvent struct {
	DatasetID string   `json:"dataset_id"`
	TenantID  string   `json:"tenant_id"`
	EntryIDs  []string `json:"entry_ids"`
	NodeSets  []string `json:"node_sets"` // [NEW] CR-002 — propagate to cognify for multi-label assignment
}
