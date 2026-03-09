package data

import (
	"context"
	"fmt"
	"time"

	"kgs-platform/internal/observability"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.opentelemetry.io/otel/attribute"
)

// CreateEdge creates a new namespaced relationship in Neo4j
func (r *graphRepo) CreateEdge(ctx context.Context, appID, tenantID string, relationType string, sourceNodeID string, targetNodeID string, properties map[string]any) (map[string]any, error) {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.create_edge", attribute.String("neo4j.relation_type", relationType))
	defer span.End()
	started := time.Now()
	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	cleanRelationType, err := sanitizeCypherIdentifier(relationType)
	if err != nil {
		return nil, err
	}
	props := cloneMap(properties)
	edgeID := ensureID(props)

	result, err := session.ExecuteWrite(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := buildCreateEdgeQuery(cleanRelationType)
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

		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("edge endpoints not found: source=%s target=%s relation=%s", sourceNodeID, targetNodeID, cleanRelationType)
	})

	if err != nil {
		observability.RecordSpanError(span, err)
		r.log.Errorf("Failed to create edge app_id=%s tenant_id=%s relation=%s source=%s target=%s err=%v", appID, tenantID, cleanRelationType, sourceNodeID, targetNodeID, err)
		return nil, err
	}

	edgeProps, ok := result.(map[string]any)
	if !ok || edgeProps == nil {
		return nil, fmt.Errorf("edge creation returned empty result: source=%s target=%s relation=%s", sourceNodeID, targetNodeID, cleanRelationType)
	}
	r.log.Infof("CreateEdge succeeded app_id=%s tenant_id=%s relation=%s source=%s target=%s edge_id=%v duration=%s",
		appID, tenantID, cleanRelationType, sourceNodeID, targetNodeID, edgeProps["id"], time.Since(started))
	return edgeProps, nil
}

func buildCreateEdgeQuery(cleanRelationType string) string {
	return fmt.Sprintf(`
		MATCH (a {app_id: $app_id, tenant_id: $tenant_id, id: $source_node_id})
		MATCH (b {app_id: $app_id, tenant_id: $tenant_id, id: $target_node_id})
		MERGE (a)-[rel:%s {app_id: $app_id, tenant_id: $tenant_id, id: $edge_id}]->(b)
		ON CREATE SET rel += $props, rel.created_at = datetime()
		ON MATCH SET rel += $props, rel.updated_at = datetime()
		RETURN rel
	`, cleanRelationType)
}
