package search

import "time"

// SearchQuery represents the core domain entity for a hybrid search operation.
type SearchQuery struct {
\tID             string            `json:"id"`
\tText           string            `json:"text"`
\tSpaceID        string            `json:"space_id"`
\tContainerTags  []string          `json:"container_tags,omitempty"`
\tMetadataFilter map[string]string `json:"metadata_filter,omitempty"`
\tLimit          int               `json:"limit"`
\tIncludeContext bool              `json:"include_context"`
\tCreatedAt      time.Time         `json:"created_at"`
}

// SearchResult represents a returned document/chunk from the search engine.
type SearchResult struct {
\tDocumentID string                 `json:"document_id"`
\tChunkID    string                 `json:"chunk_id,omitempty"`
\tContent    string                 `json:"content"`
\tScore      float64                `json:"score"`
\tMetadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Valid checks if the SearchQuery contains the minimum required fields.
func (q *SearchQuery) Valid() bool {
\tif q.Text == "" {
\t\treturn false
\t}
\tif q.Limit <= 0 {
\t\tq.Limit = 10 // Set default limit
\t}
\treturn true
}

// HybridSearchCriteria wraps both vector and BM25 specifics.
type HybridSearchCriteria struct {
\tQueryVector []float32
\tRawText     string
\tAlpha       float64 // Weight between vector and text search, usually 0.5
}
