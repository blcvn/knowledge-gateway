package data

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// CreateEdge creates a new namespaced relationship in Neo4j
func (r *graphRepo) CreateEdge(ctx context.Context, appID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// DYNAMIC CYPHER - safe because parameterization
		query := `
			MATCH (a {app_id: $app_id, id: $source_node_id})
			MATCH (b {app_id: $app_id, id: $target_node_id})
			CREATE (a)-[rel:` + relationType + ` {app_id: $app_id}]->(b)
			SET rel += $props
			RETURN rel
		`
		params := map[string]interface{}{
			"app_id":         appID,
			"source_node_id": sourceNodeID,
			"target_node_id": targetNodeID,
			"props":          properties,
		}

		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		if res.Next(ctx) {
			edge := res.Record().Values[0].(neo4j.Relationship)
			return edge.Props, nil
		}

		return nil, res.Err()
	})

	if err != nil {
		r.log.Errorf("Failed to create edge: %v", err)
		return nil, err
	}

	return result.(map[string]any), nil
}
