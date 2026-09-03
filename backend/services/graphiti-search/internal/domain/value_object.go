package domain

import "time"

type SearchMethod string

const (
	MethodCosine SearchMethod = "cosine_similarity"
	MethodBM25   SearchMethod = "bm25"
	MethodBFS    SearchMethod = "breadth_first_search"
)

func (m SearchMethod) IsValid() bool {
	switch m {
	case MethodCosine, MethodBM25, MethodBFS:
		return true
	}
	return false
}

type RerankerType string

const (
	RerankerRRF             RerankerType = "rrf"
	RerankerMMR             RerankerType = "mmr"
	RerankerCrossEncoder    RerankerType = "cross_encoder"
	RerankerNodeDistance    RerankerType = "node_distance"
	RerankerEpisodeMentions RerankerType = "episode_mentions"
)

func (r RerankerType) IsValid() bool {
	switch r {
	case RerankerRRF, RerankerMMR, RerankerCrossEncoder, RerankerNodeDistance, RerankerEpisodeMentions:
		return true
	}
	return false
}

type ScoreWeight float64

type TemporalWindow struct {
	From time.Time
	To   time.Time
}

func (t *TemporalWindow) Validate() error {
	if t.To.Before(t.From) {
		return ErrInvalidQuery
	}
	return nil
}
