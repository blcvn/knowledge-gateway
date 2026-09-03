package neo4j

import (
	"context"
	"fmt"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

type searchRepo struct{ driver *Neo4jDriver }

// NodeFulltextSearch — BM25 fulltext search using Neo4j fulltext index
func (r *searchRepo) NodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, labels []string) ([]*graph.EntityNode, error) {
	cypher := `
		CALL db.index.fulltext.queryNodes("entity_fulltext", $query)
		YIELD node, score
		WHERE node.group_id IN $group_ids
	`
	params := map[string]any{"query": query, "group_ids": groupIDs, "limit": limit}

	if len(labels) > 0 {
		cypher += " AND any(l IN node.labels WHERE l IN $labels)"
		params["labels"] = labels
	}
	cypher += " RETURN node ORDER BY score DESC LIMIT $limit"

	records, err := r.driver.ExecuteQuery(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EntityNode, 0, len(records))
	for _, rec := range records {
		n, _ := mapRecordToEntityNode(rec)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// NodeSimilaritySearch — vector cosine similarity using Neo4j vector index
func (r *searchRepo) NodeSimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.EntityNode, error) {
	cypher := `
		CALL db.index.vector.queryNodes("entity_name_embedding", $limit, $vector)
		YIELD node, score
		WHERE node.group_id IN $group_ids AND score >= $min_score
		RETURN node
		ORDER BY score DESC
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"vector": vector, "group_ids": groupIDs,
		"limit": limit * 2, "min_score": minScore,
	})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EntityNode, 0, len(records))
	for _, rec := range records {
		n, _ := mapRecordToEntityNode(rec)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	if len(nodes) > limit {
		nodes = nodes[:limit]
	}
	return nodes, nil
}

// NodeBFSSearch — breadth-first traversal from origin nodes up to maxDepth hops
func (r *searchRepo) NodeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityNode, error) {
	cypher := fmt.Sprintf(`
		MATCH path = (origin:Entity)-[:RELATES_TO*1..%d]-(n:Entity)
		WHERE origin.uuid IN $origin_uuids
		  AND n.group_id IN $group_ids
		  AND n.invalid_at IS NULL
		  AND NOT n.uuid IN $origin_uuids
		WITH DISTINCT n, min(length(path)) as distance
		ORDER BY distance ASC
		LIMIT $limit
		RETURN n
	`, maxDepth)

	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"origin_uuids": originUUIDs,
		"group_ids":    groupIDs,
		"limit":        limit,
	})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EntityNode, 0, len(records))
	for _, rec := range records {
		n, _ := mapRecordToEntityNode(rec)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// EdgeFulltextSearch — BM25 fulltext search on edge facts
func (r *searchRepo) EdgeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int, filters port.EdgeSearchFilters) ([]*graph.EntityEdge, error) {
	cypher := `
		CALL db.index.fulltext.queryRelationships("entity_edge_fulltext", $query)
		YIELD relationship as e, score
		WHERE e.group_id IN $group_ids
	`
	params := map[string]any{"query": query, "group_ids": groupIDs, "limit": limit}
	cypher += buildTemporalFilter("e", filters, params)
	cypher += " RETURN e, startNode(e).uuid as src_uuid, endNode(e).uuid as tgt_uuid ORDER BY score DESC LIMIT $limit"

	records, err := r.driver.ExecuteQuery(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	return mapRecordsToEntityEdges(records), nil
}

// EdgeSimilaritySearch — vector cosine similarity on edge fact embeddings
func (r *searchRepo) EdgeSimilaritySearch(ctx context.Context, req port.EdgeSimilarityReq) ([]*graph.EntityEdge, error) {
	cypher := `
		CALL db.index.vector.queryRelationships("entity_edge_fact_embedding", $limit, $vector)
		YIELD relationship as e, score
		WHERE e.group_id IN $group_ids AND score >= $min_score
	`
	params := map[string]any{
		"vector": req.Vector, "group_ids": req.GroupIDs,
		"limit": req.Limit * 2, "min_score": req.MinScore,
	}
	if req.SourceUUID != "" {
		cypher += " AND startNode(e).uuid = $src_uuid"
		params["src_uuid"] = req.SourceUUID
	}
	cypher += buildTemporalFilter("e", req.Filters, params)
	cypher += " RETURN e, startNode(e).uuid as src_uuid, endNode(e).uuid as tgt_uuid ORDER BY score DESC LIMIT $limit"

	records, err := r.driver.ExecuteQuery(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	edges := mapRecordsToEntityEdges(records)
	if len(edges) > req.Limit {
		edges = edges[:req.Limit]
	}
	return edges, nil
}

// EdgeBFSSearch — BFS traversal returning edges (not nodes)
func (r *searchRepo) EdgeBFSSearch(ctx context.Context, originUUIDs []string, maxDepth int, groupIDs []string, limit int) ([]*graph.EntityEdge, error) {
	cypher := fmt.Sprintf(`
		MATCH (origin:Entity)-[e:RELATES_TO*1..%d]-(:Entity)
		WHERE origin.uuid IN $origin_uuids
		  AND e.group_id IN $group_ids
		  AND e.invalid_at IS NULL
		UNWIND e as rel
		WITH DISTINCT rel
		LIMIT $limit
		RETURN rel as e, startNode(rel).uuid as src_uuid, endNode(rel).uuid as tgt_uuid
	`, maxDepth)

	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"origin_uuids": originUUIDs, "group_ids": groupIDs, "limit": limit,
	})
	if err != nil {
		return nil, err
	}
	return mapRecordsToEntityEdges(records), nil
}

// EpisodeFulltextSearch — fulltext search on episode content
func (r *searchRepo) EpisodeFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.EpisodicNode, error) {
	cypher := `
		CALL db.index.fulltext.queryNodes("episode_fulltext", $query)
		YIELD node, score
		WHERE node.group_id IN $group_ids
		RETURN node ORDER BY score DESC LIMIT $limit
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"query": query, "group_ids": groupIDs, "limit": limit,
	})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EpisodicNode, 0)
	for _, rec := range records {
		n, _ := mapRecordToEpisodicNode(rec)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// CommunityFulltextSearch — fulltext search on community names/summaries
func (r *searchRepo) CommunityFulltextSearch(ctx context.Context, query string, groupIDs []string, limit int) ([]*graph.CommunityNode, error) {
	cypher := `
		CALL db.index.fulltext.queryNodes("community_fulltext", $query)
		YIELD node, score
		WHERE node.group_id IN $group_ids
		RETURN node ORDER BY score DESC LIMIT $limit
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"query": query, "group_ids": groupIDs, "limit": limit,
	})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.CommunityNode, 0)
	for _, rec := range records {
		n, _ := mapRecordToCommunityNode(rec)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (r *searchRepo) CommunitySimilaritySearch(ctx context.Context, vector []float32, groupIDs []string, limit int, minScore float64) ([]*graph.CommunityNode, error) {
	cypher := `
		CALL db.index.vector.queryNodes("community_name_embedding", $limit, $vector)
		YIELD node, score
		WHERE node.group_id IN $group_ids AND score >= $min_score
		RETURN node ORDER BY score DESC
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"vector": vector, "group_ids": groupIDs, "limit": limit, "min_score": minScore,
	})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.CommunityNode, 0)
	for _, rec := range records {
		n, _ := mapRecordToCommunityNode(rec)
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

// NodeDistanceReranker — returns hop distance from center node using shortestPath
func (r *searchRepo) NodeDistanceReranker(ctx context.Context, nodeUUIDs []string, centerUUID string) (map[string]float64, error) {
	cypher := `
		MATCH (center:Entity {uuid: $center_uuid})
		UNWIND $node_uuids AS targetUUID
		MATCH (target:Entity {uuid: targetUUID})
		MATCH path = shortestPath((center)-[:RELATES_TO*1..5]-(target))
		RETURN targetUUID, length(path) as distance
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"center_uuid": centerUUID, "node_uuids": nodeUUIDs,
	})
	if err != nil {
		return nil, err
	}

	scores := make(map[string]float64, len(records))
	for _, rec := range records {
		if len(rec.Values) < 2 {
			continue
		}
		uuid, _ := rec.Values[0].(string)
		dist, _ := rec.Values[1].(int64)
		// Score inversely proportional to distance
		scores[uuid] = 1.0 / (float64(dist) + 1.0)
	}
	return scores, nil
}

// EpisodeMentionsReranker — returns count of episodes mentioning each node
func (r *searchRepo) EpisodeMentionsReranker(ctx context.Context, nodeUUIDs []string) (map[string]int, error) {
	cypher := `
		MATCH (ep:Episodic)-[:MENTIONS]->(n:Entity)
		WHERE n.uuid IN $node_uuids
		RETURN n.uuid as node_uuid, count(ep) as mention_count
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"node_uuids": nodeUUIDs})
	if err != nil {
		return nil, err
	}

	counts := make(map[string]int, len(records))
	for _, rec := range records {
		if len(rec.Values) < 2 {
			continue
		}
		uuid, _ := rec.Values[0].(string)
		count, _ := rec.Values[1].(int64)
		counts[uuid] = int(count)
	}
	return counts, nil
}

// buildTemporalFilter adds WHERE clauses for temporal filters
func buildTemporalFilter(alias string, f port.EdgeSearchFilters, params map[string]any) string {
	clause := ""
	if f.ValidAt != nil {
		clause += fmt.Sprintf(" AND (%s.valid_at IS NULL OR %s.valid_at <= $valid_at)", alias, alias)
		clause += fmt.Sprintf(" AND (%s.invalid_at IS NULL OR %s.invalid_at > $valid_at)", alias, alias)
		params["valid_at"] = f.ValidAt
	}
	if f.CreatedAtStart != nil {
		clause += fmt.Sprintf(" AND %s.created_at >= $created_at_start", alias)
		params["created_at_start"] = f.CreatedAtStart
	}
	if f.CreatedAtEnd != nil {
		clause += fmt.Sprintf(" AND %s.created_at <= $created_at_end", alias)
		params["created_at_end"] = f.CreatedAtEnd
	}
	return clause
}

func mapRecordsToEntityEdges(records []port.Record) []*graph.EntityEdge {
	edges := make([]*graph.EntityEdge, 0, len(records))
	for _, rec := range records {
		e, _ := mapRecordToEntityEdge(rec)
		if e != nil {
			edges = append(edges, e)
		}
	}
	return edges
}
