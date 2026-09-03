package domain

import "time"

type SearchResult struct {
	DocID         string
	SessionID     string
	ObsType       string
	Title         string
	Narrative     string
	Facts         []string
	Concepts      []string
	CombinedScore float64
	BM25Score     float64
	VectorScore   float64
}

type ContextBlock struct {
	Type    string // "memory" | "summary" | "observation"
	Content string
	Tokens  int
	Recency float64
	Source  string
}

type Summary struct {
	Narrative string
}

type Observation struct {
    ObsType   string
    Facts     []string
    Concepts  []string

	Title     string
	Narrative string
}

type AgentMemory struct {
	ID        string
	Type      string
	Title     string
	Content   string
	UpdatedAt time.Time
}
