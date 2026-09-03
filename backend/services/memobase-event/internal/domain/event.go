// Package domain defines the event domain for memobase-event service.
// SOL-MB-004: Event Timeline & Semantic Search (CR-MB-004)
package domain

import "time"

// Event represents a user interaction event stored in the timeline.
type Event struct {
	ID        string
	UserID    string
	ProjectID string
	EventData EventData
	Embedding []float32 // pre-computed by memobase-engine
	CreatedAt time.Time
	UpdatedAt time.Time
}

// EventData is the JSONB payload stored in user_events.event_data.
type EventData struct {
	EventTip    string      `json:"event_tip"`
	EventTags   []EventTag  `json:"event_tags"`
	ProfileDelta interface{} `json:"profile_delta,omitempty"`
}

// EventTag represents a key-value tag attached to an event.
type EventTag struct {
	Tag   string `json:"tag"`
	Value string `json:"value"`
}

// EventGist is a fine-grained sub-unit of an event (one bullet line).
type EventGist struct {
	ID        string
	EventID   string
	UserID    string
	ProjectID string
	GistData  GistData
	Embedding []float32 // per-gist embedding
	CreatedAt time.Time
}

// GistData is the payload stored for each event gist.
type GistData struct {
	GistContent string `json:"gist_content"`
}

// SearchResult represents a single semantic search hit.
type SearchResult struct {
	Event      Event
	Similarity float64
}

// GistSearchResult represents a single gist-level search hit.
type GistSearchResult struct {
	Gist       EventGist
	Similarity float64
}

// ErrEmbeddingDisabled is returned when semantic search is called but embedding is disabled.
var ErrEmbeddingDisabled = &domainError{"embedding is disabled; enable MEMOBASE_ENABLE_EVENT_EMBEDDING"}

type domainError struct{ msg string }

func (e *domainError) Error() string { return e.msg }
