package neo4j

import (
	"context"
	"fmt"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

// --- SearchRepository ---

// CosineSimilaritySearch returns top-K nodes ordered by embedding similarity.
func (d *Driver) CosineSimilaritySearch(ctx context.Context, groupID string, embedding domain.EmbeddingVector, limit int) ([]domain.SearchResult, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	// Neo4j 5.x vector index query
	cypher := `
		CALL db.index.vector.queryNodes('entity_name_embedding', $limit, $embedding)
		YIELD node, score
		WHERE node.group_id = $group_id
		RETURN node.uuid AS uuid, node.name AS name, node.summary AS summary,
		       'Entity' AS node_label, node.group_id AS group_id, score
		ORDER BY score DESC
	`
	result, err := session.Run(ctx, cypher, map[string]any{
		"group_id":  groupID,
		"embedding": toFloat64Slice(embedding),
		"limit":     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: cosine search: %w", err)
	}

	return d.collectSearchResults(ctx, result)
}

// FulltextSearch returns BM25-ranked results.
func (d *Driver) FulltextSearch(ctx context.Context, groupID, query string, limit int) ([]domain.SearchResult, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		CALL db.index.fulltext.queryNodes('entity_name_fulltext', $query)
		YIELD node, score
		WHERE node.group_id = $group_id
		RETURN node.uuid AS uuid, node.name AS name, node.summary AS summary,
		       'Entity' AS node_label, node.group_id AS group_id, score
		ORDER BY score DESC
		LIMIT $limit
	`
	result, err := session.Run(ctx, cypher, map[string]any{
		"group_id": groupID,
		"query":    query,
		"limit":    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: fulltext search: %w", err)
	}

	return d.collectSearchResults(ctx, result)
}

// BFSSearch traverses the graph breadth-first from a start node.
func (d *Driver) BFSSearch(ctx context.Context, startNodeID string, depth, limit int) ([]domain.SearchResult, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH path = (start:Entity {uuid: $start_node_id})-[:RELATES_TO*1..` + fmt.Sprintf("%d", depth) + `]-(neighbor:Entity)
		WITH neighbor, length(path) AS distance
		RETURN DISTINCT neighbor.uuid AS uuid, neighbor.name AS name,
		       neighbor.summary AS summary, 'Entity' AS node_label,
		       neighbor.group_id AS group_id, 1.0/(1.0+distance) AS score,
		       distance
		ORDER BY distance ASC
		LIMIT $limit
	`
	result, err := session.Run(ctx, cypher, map[string]any{
		"start_node_id": startNodeID,
		"limit":         limit,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: bfs search: %w", err)
	}

	return d.collectSearchResults(ctx, result)
}

// --- CommunityRepository ---

// GetCommunity retrieves a CommunityNode by UUID.
func (d *Driver) GetCommunity(ctx context.Context, groupID, uuid string) (*domain.CommunityNode, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `MATCH (n:Community {uuid: $uuid, group_id: $group_id}) RETURN n`
	result, err := session.Run(ctx, cypher, map[string]any{
		"uuid":     uuid,
		"group_id": groupID,
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: get community: %w", err)
	}
	if result.Next(ctx) {
		return d.recordToCommunityNode(result.Record())
	}
	return nil, domain.ErrNodeNotFound
}

// ListCommunities returns all communities for a group.
func (d *Driver) ListCommunities(ctx context.Context, groupID string) ([]domain.CommunityNode, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `MATCH (n:Community {group_id: $group_id}) RETURN n ORDER BY n.name`
	result, err := session.Run(ctx, cypher, map[string]any{"group_id": groupID})
	if err != nil {
		return nil, fmt.Errorf("neo4j: list communities: %w", err)
	}

	var communities []domain.CommunityNode
	for result.Next(ctx) {
		c, err := d.recordToCommunityNode(result.Record())
		if err != nil {
			d.logger.Warn("skip community", "error", err)
			continue
		}
		communities = append(communities, *c)
	}
	return communities, nil
}

// DeleteCommunity removes a community and its HAS_MEMBER edges.
func (d *Driver) DeleteCommunity(ctx context.Context, groupID, uuid string) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `MATCH (n:Community {uuid: $uuid, group_id: $group_id}) DETACH DELETE n`
	_, err := session.Run(ctx, cypher, map[string]any{
		"uuid":     uuid,
		"group_id": groupID,
	})
	if err != nil {
		return fmt.Errorf("neo4j: delete community: %w", err)
	}
	return nil
}

// collectSearchResults maps Neo4j result rows to domain.SearchResult slices.
func (d *Driver) collectSearchResults(ctx context.Context, result neo4j.ResultWithContext) ([]domain.SearchResult, error) {
	var results []domain.SearchResult
	for result.Next(ctx) {
		record := result.Record()
		sr := domain.SearchResult{
			UUID:      getRecordString(record, "uuid"),
			NodeLabel: getRecordString(record, "node_label"),
			Name:      getRecordString(record, "name"),
			Summary:   getRecordString(record, "summary"),
			Fact:      getRecordString(record, "fact"),
			GroupID:   getRecordString(record, "group_id"),
		}
		if score, ok := record.Get("score"); ok {
			if f, ok := score.(float64); ok {
				sr.Score = f
			}
		}
		if dist, ok := record.Get("distance"); ok {
			if d, ok := dist.(int64); ok {
				sr.Distance = int(d)
			}
		}
		results = append(results, sr)
	}
	return results, nil
}

// recordToCommunityNode maps a Neo4j record to a domain.CommunityNode.
func (d *Driver) recordToCommunityNode(record *neo4j.Record) (*domain.CommunityNode, error) {
	nodeVal, ok := record.Get("n")
	if !ok {
		return nil, fmt.Errorf("neo4j: missing 'n' in record")
	}
	dbNode, ok := nodeVal.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("neo4j: unexpected type: %T", nodeVal)
	}
	props := dbNode.Props
	c := &domain.CommunityNode{
		UUID:    getStringProp(props, "uuid"),
		Name:    getStringProp(props, "name"),
		GroupID: getStringProp(props, "group_id"),
		Summary: getStringProp(props, "summary"),
	}
	if level, ok := props["level"].(int64); ok {
		c.Level = int(level)
	}
	return c, nil
}

// getRecordString safely extracts a string value from a Neo4j record.
func getRecordString(record *neo4j.Record, key string) string {
	v, ok := record.Get(key)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// toFloat64Slice converts a float32 EmbeddingVector to float64 for Neo4j.
func toFloat64Slice(v domain.EmbeddingVector) []float64 {
	out := make([]float64, len(v))
	for i, f := range v {
		out[i] = float64(f)
	}
	return out
}
