package search

import (
	"context"
	"errors"
	"sort"
	"time"

	"kgs-platform/internal/observability"
)

const (
	defaultTopK  = 10
	defaultAlpha = 0.6
	defaultBeta  = 0.2
	// Allow large retrieval windows for whole-graph hydration flows (e.g. ba-agent GetByDocumentID).
	maxSearchTopK = 10000
)

type SearchEngine interface {
	HybridSearch(ctx context.Context, namespace, query string, opts Options) ([]Result, error)
}

type Options struct {
	TopK            int
	Alpha           float64
	Beta            float64
	EntityTypes     []string
	Domains         []string
	MinConfidence   float64
	ProvenanceTypes []string
}

type Result struct {
	ID            string
	Label         string
	Properties    map[string]any
	SemanticScore float64
	TextScore     float64
	Centrality    float64
	Score         float64
}

type VectorRetriever interface {
	Search(ctx context.Context, namespace, query string, topK int) ([]Result, error)
}

type TextRetriever interface {
	Search(ctx context.Context, namespace, query string, topK int) ([]Result, error)
}

type CentralityScorer interface {
	Scores(ctx context.Context, namespace string, nodeIDs []string) (map[string]float64, error)
}

type Engine struct {
	vector     VectorRetriever
	text       TextRetriever
	centrality CentralityScorer
}

func NewEngine(vector VectorRetriever, text TextRetriever, centrality CentralityScorer) *Engine {
	return &Engine{
		vector:     vector,
		text:       text,
		centrality: centrality,
	}
}

func (e *Engine) HybridSearch(ctx context.Context, namespace, query string, opts Options) ([]Result, error) {
	started := time.Now()
	defer func() {
		observability.ObserveSearchDuration("hybrid", time.Since(started))
	}()
	if query == "" {
		return nil, errors.New("query is required")
	}
	opts = withDefaults(opts)

	var semanticResults []Result
	var textResults []Result
	var semanticErr error
	var textErr error

	if e.vector != nil {
		semanticResults, semanticErr = e.vector.Search(ctx, namespace, query, opts.TopK)
	}
	if e.text != nil {
		textResults, textErr = e.text.Search(ctx, namespace, query, opts.TopK)
	}
	if semanticErr != nil && textErr != nil {
		return nil, errors.New("both semantic and text search failed")
	}

	blended := Blend(semanticResults, textResults, opts.Alpha)
	if len(blended) == 0 {
		return []Result{}, nil
	}

	if e.centrality != nil {
		ids := make([]string, 0, len(blended))
		for _, item := range blended {
			ids = append(ids, item.ID)
		}
		if centralityMap, err := e.centrality.Scores(ctx, namespace, ids); err == nil {
			blended = RerankWithCentrality(blended, centralityMap, opts.Beta)
		}
	}

	filtered := ApplyFilters(blended, opts)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})
	if len(filtered) > opts.TopK {
		filtered = filtered[:opts.TopK]
	}
	return filtered, nil
}

func withDefaults(opts Options) Options {
	if opts.TopK <= 0 {
		opts.TopK = defaultTopK
	}
	if opts.TopK > maxSearchTopK {
		opts.TopK = maxSearchTopK
	}
	if opts.Alpha < 0 || opts.Alpha > 1 {
		opts.Alpha = defaultAlpha
	}
	if opts.Beta < 0 {
		opts.Beta = defaultBeta
	}
	return opts
}
