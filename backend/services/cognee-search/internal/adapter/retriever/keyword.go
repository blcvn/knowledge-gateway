package retriever

import (
	"context"

	"vnp-memory/services/cognee-search/internal/usecase"
)

// KeywordRetriever implements KEYWORD search using Neo4j fulltext index with NodeSet label filter.
type KeywordRetriever struct {
	// neo4jClient Neo4jClient — injected in production
}

// NewKeywordRetriever creates a KeywordRetriever.
func NewKeywordRetriever() *KeywordRetriever { return &KeywordRetriever{} }

// Strategy returns the search strategy this retriever handles.
func (r *KeywordRetriever) Strategy() usecase.SearchStrategy { return usecase.StrategyKeyword }

// Retrieve performs a fulltext keyword search with optional NodeSet label filter.
//
// Cypher (with NodeSets):
//
//	CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
//	WHERE all(tag IN $node_sets WHERE tag IN labels(node))
//	RETURN node, score ORDER BY score DESC LIMIT $top_k
//
// Without NodeSets: no WHERE clause added.
func (r *KeywordRetriever) Retrieve(ctx context.Context, req usecase.SearchRequest) ([]usecase.SearchResult, error) {
	params := map[string]any{
		"query": req.Query,
		"top_k": req.TopK,
	}

	// Base fulltext query
	cypher := `CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score`

	if len(req.NodeSets) > 0 {
		// [NEW] AND filter by NodeSet labels — all specified tags must be labels of the node
		params["node_sets"] = req.NodeSets
		cypher += ` WHERE all(tag IN $node_sets WHERE tag IN labels(node))`
	}

	cypher += ` RETURN node, score ORDER BY score DESC LIMIT $top_k`

	// Production implementation:
	// return r.neo4jClient.QueryNodes(ctx, cypher, params)

	_ = cypher
	_ = params
	return []usecase.SearchResult{}, nil
}
