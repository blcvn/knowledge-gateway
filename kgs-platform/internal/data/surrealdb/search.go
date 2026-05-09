package surrealdb

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

// ── Vector Retriever (replaces Qdrant) ────────────────────────

// surrealVectorRetriever implements search.VectorRetriever using SurrealDB vector indexes.
type surrealVectorRetriever struct {
	client *Client
	log    *log.Helper
}

func NewSurrealVectorRetriever(client *Client, logger log.Logger) *surrealVectorRetriever {
	return &surrealVectorRetriever{
		client: client,
		log:    log.NewHelper(logger),
	}
}

// SearchResult represents a single search result with score.
type SearchResult struct {
	EntityID   string         `json:"entity_id"`
	Name       string         `json:"name"`
	EntityType string         `json:"entity_type"`
	Score      float64        `json:"score"`
	Properties map[string]any `json:"properties,omitempty"`
}

// Search performs vector similarity search using SurrealDB's native vector index.
func (r *surrealVectorRetriever) Search(ctx context.Context, appID, tenantID string, embedding []float32, topK int) ([]SearchResult, error) {
	sql := `SELECT entity_id, name, entity_type, properties,
		vector::similarity::cosine(embedding, $vec) AS score
		FROM kg_entities
		WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false AND embedding IS NOT NONE
		ORDER BY score DESC LIMIT $topk`

	result, err := r.client.Query(ctx, sql, map[string]any{
		"vec":       embedding,
		"app_id":    appID,
		"tenant_id": tenantID,
		"topk":      topK,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] VectorSearch failed app=%s err=%v", appID, err)
		return nil, err
	}

	items, err := unmarshalSlice[SearchResult](result)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ── Text Retriever (replaces PG full-text) ────────────────────

// surrealTextRetriever implements search.TextRetriever using SurrealDB full-text search.
type surrealTextRetriever struct {
	client *Client
	log    *log.Helper
}

func NewSurrealTextRetriever(client *Client, logger log.Logger) *surrealTextRetriever {
	return &surrealTextRetriever{
		client: client,
		log:    log.NewHelper(logger),
	}
}

// Search performs full-text search using SurrealDB's BM25 analyzer.
func (r *surrealTextRetriever) Search(ctx context.Context, appID, tenantID, query string, topK int) ([]SearchResult, error) {
	sql := `SELECT entity_id, name, entity_type, properties, search::score(1) AS score
		FROM kg_entities
		WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
		AND name @1@ $query
		ORDER BY score DESC LIMIT $topk`

	result, err := r.client.Query(ctx, sql, map[string]any{
		"query":     query,
		"app_id":    appID,
		"tenant_id": tenantID,
		"topk":      topK,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] TextSearch failed app=%s query=%q err=%v", appID, query, err)
		return nil, err
	}

	items, err := unmarshalSlice[SearchResult](result)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// ── Centrality Scorer (replaces Neo4j PageRank) ───────────────

// surrealCentralityScorer uses degree centrality as an approximation.
// Option A: count(incoming + outgoing edges) — fast, no pre-computation needed.
type surrealCentralityScorer struct {
	client *Client
	log    *log.Helper
}

func NewSurrealCentralityScorer(client *Client, logger log.Logger) *surrealCentralityScorer {
	return &surrealCentralityScorer{
		client: client,
		log:    log.NewHelper(logger),
	}
}

// Scores returns degree-centrality scores for the given node IDs.
func (r *surrealCentralityScorer) Scores(ctx context.Context, appID, tenantID string, nodeIDs []string) (map[string]float64, error) {
	if len(nodeIDs) == 0 {
		return map[string]float64{}, nil
	}

	scores := make(map[string]float64, len(nodeIDs))

	sql := `SELECT
		entity_id,
		(SELECT count() FROM kg_edges WHERE from_entity_id = $parent.entity_id AND is_deleted = false GROUP ALL).count AS out_degree,
		(SELECT count() FROM kg_edges WHERE to_entity_id = $parent.entity_id AND is_deleted = false GROUP ALL).count AS in_degree
		FROM kg_entities
		WHERE entity_id IN $node_ids AND app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false`

	result, err := r.client.Query(ctx, sql, map[string]any{
		"node_ids":  nodeIDs,
		"app_id":    appID,
		"tenant_id": tenantID,
	})
	if err != nil {
		r.log.Warnf("[KGS][SurrealDB] CentralityScorer failed, returning uniform scores err=%v", err)
		for _, id := range nodeIDs {
			scores[id] = 1.0
		}
		return scores, nil
	}

	rows, _ := unmarshalSlice[map[string]any](result)
	maxDegree := 1.0

	type nodeScore struct {
		id     string
		degree float64
	}
	var ns []nodeScore

	for _, row := range rows {
		id := fmt.Sprint(row["entity_id"])
		outDeg, _ := row["out_degree"].(float64)
		inDeg, _ := row["in_degree"].(float64)
		degree := outDeg + inDeg
		if degree > maxDegree {
			maxDegree = degree
		}
		ns = append(ns, nodeScore{id: id, degree: degree})
	}

	// Normalize to [0, 1]
	for _, n := range ns {
		scores[n.id] = n.degree / maxDegree
	}

	// Fill missing IDs with 0
	for _, id := range nodeIDs {
		if _, ok := scores[id]; !ok {
			scores[id] = 0.0
		}
	}

	return scores, nil
}
