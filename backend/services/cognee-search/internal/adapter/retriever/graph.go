package retriever

import (
	"context"

	"vnp-memory/services/cognee-search/internal/usecase"
)

// GraphRetriever implements GRAPH_COMPLETION search using Neo4j Cypher with NodeSet label filtering.
type GraphRetriever struct {
	// neo4jClient Neo4jClient — injected in production
}

// NewGraphRetriever creates a GraphRetriever.
func NewGraphRetriever() *GraphRetriever { return &GraphRetriever{} }

// Strategy returns the search strategy this retriever handles.
func (r *GraphRetriever) Strategy() usecase.SearchStrategy { return usecase.StrategyGraphCompletion }

// Retrieve performs a graph traversal with NodeSet label predicate filtering.
//
// Cypher query (with NodeSets):
//
//	MATCH (n)-[r]->(m)
//	WHERE n.tenant_id = $tenant_id
//	  AND ($dataset_id IS NULL OR n.dataset_id = $dataset_id)
//	  AND all(tag IN $node_sets WHERE tag IN labels(n))
//	CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
//	WHERE node.id = n.id
//	RETURN n, r, m, score ORDER BY score DESC LIMIT $top_k
//
// Without NodeSets: same query without the label predicate.
func (r *GraphRetriever) Retrieve(ctx context.Context, req usecase.SearchRequest) ([]usecase.SearchResult, error) {
	params := map[string]any{
		"query":     req.Query,
		"tenant_id": req.TenantID,
		"top_k":     req.TopK,
	}
	if req.DatasetID != nil {
		params["dataset_id"] = req.DatasetID.String()
	}

	var cypher string
	if len(req.NodeSets) > 0 {
		// [NEW] Filter by ALL NodeSet labels using Cypher label predicates
		// `all(tag IN $node_sets WHERE tag IN labels(n))` — ALL tags must be present on node
		params["node_sets"] = req.NodeSets
		cypher = `
			MATCH (n)-[r]->(m)
			WHERE n.tenant_id = $tenant_id
			  AND ($dataset_id IS NULL OR n.dataset_id = $dataset_id)
			  AND all(tag IN $node_sets WHERE tag IN labels(n))
			CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
			WHERE node.id = n.id
			RETURN n, r, m, score
			ORDER BY score DESC
			LIMIT $top_k
		`
	} else {
		cypher = `
			MATCH (n)-[r]->(m)
			WHERE n.tenant_id = $tenant_id
			  AND ($dataset_id IS NULL OR n.dataset_id = $dataset_id)
			CALL db.index.fulltext.queryNodes('nodeTextIndex', $query) YIELD node, score
			WHERE node.id = n.id
			RETURN n, r, m, score
			ORDER BY score DESC
			LIMIT $top_k
		`
	}

	// Production implementation:
	// return r.neo4jClient.QueryNodesAndEdges(ctx, cypher, params)

	_ = cypher
	_ = params
	return []usecase.SearchResult{}, nil
}
