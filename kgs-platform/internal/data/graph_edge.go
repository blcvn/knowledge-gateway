package data

import (
	"context"
	"fmt"

	"kgs-platform/internal/observability"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.opentelemetry.io/otel/attribute"
)

// CreateEdge creates a new namespaced relationship in Neo4j
func (r *graphRepo) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.create_edge", attribute.String("neo4j.relation_type", relationType))
	defer span.End()
	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	cleanRelationType, err := sanitizeCypherIdentifier(relationType)
	if err != nil {
		return nil, err
	}
	props := cloneMap(properties)
	edgeID := ensureID(props)

	result, err := session.ExecuteWrite(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf(`
			MATCH (a {app_id: $app_id, tenant_id: $tenant_id, id: $source_node_id})
			MATCH (b {app_id: $app_id, tenant_id: $tenant_id, id: $target_node_id})
			CREATE (a)-[rel:%s {app_id: $app_id, tenant_id: $tenant_id, id: $edge_id}]->(b)
			SET rel += $props
			RETURN rel
		`, cleanRelationType)
		params := map[string]interface{}{
			"app_id":         appID,
			"tenant_id":      tenantID,
			"source_node_id": sourceNodeID,
			"target_node_id": targetNodeID,
			"edge_id":        edgeID,
			"props":          props,
		}

		res, err := tx.Run(traceCtx, query, params)
		if err != nil {
			return nil, err
		}

		if res.Next(traceCtx) {
			edge := res.Record().Values[0].(neo4j.Relationship)
			return edge.Props, nil
		}

		return nil, res.Err()
	})

	if err != nil {
		observability.RecordSpanError(span, err)
		r.log.Errorf("Failed to create edge: %v", err)
		return nil, err
	}

	return result.(map[string]any), nil
}
