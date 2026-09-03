package surrealdb

import (
	"context"
	"fmt"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// surrealGraphRepo implements biz.GraphRepo using SurrealDB graph features (RELATE).
type surrealGraphRepo struct {
	client *Client
	log    *log.Helper
}

func NewSurrealGraphRepo(client *Client, logger log.Logger) biz.GraphRepo {
	return &surrealGraphRepo{
		client: client,
		log:    log.NewHelper(logger),
	}
}

func (r *surrealGraphRepo) CreateNode(ctx context.Context, appID, tenantID, label string, properties map[string]any) (map[string]any, error) {
	nodeID, _ := properties["id"].(string)
	if nodeID == "" {
		nodeID = uuid.NewString()
		properties["id"] = nodeID
	}

	sql := `CREATE type::thing('kg_entities', $entity_id) SET
		entity_id = $entity_id,
		app_id = $app_id,
		tenant_id = $tenant_id,
		entity_type = $label,
		name = $name,
		properties = $properties,
		confidence = 1.0,
		version = 1,
		is_deleted = false,
		created_at = time::now(),
		updated_at = time::now()`

	name, _ := properties["name"].(string)
	if name == "" {
		name = nodeID
	}

	_, err := r.client.Query(ctx, sql, map[string]any{
		"entity_id":  nodeID,
		"app_id":     appID,
		"tenant_id":  tenantID,
		"label":      label,
		"name":       name,
		"properties": properties,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] CreateNode failed app_id=%s node_id=%s err=%v", appID, nodeID, err)
		return nil, err
	}

	result := cloneMap(properties)
	result["id"] = nodeID
	result["label"] = label
	result["entity_type"] = label
	return result, nil
}

func (r *surrealGraphRepo) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	sql := `SELECT * FROM kg_entities
		WHERE entity_id = $entity_id AND app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
		LIMIT 1`
	result, err := r.client.Query(ctx, sql, map[string]any{
		"entity_id": nodeID,
		"app_id":    appID,
		"tenant_id": tenantID,
	})
	if err != nil {
		return nil, err
	}
	nodes, err := unmarshalSlice[map[string]any](result)
	if err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}
	return normalizeEntityMap(nodes[0]), nil
}

func (r *surrealGraphRepo) CreateEdge(ctx context.Context, appID, tenantID, relationType, sourceNodeID, targetNodeID string, properties map[string]any) (map[string]any, error) {
	edgeID, _ := properties["id"].(string)
	if edgeID == "" {
		edgeID = uuid.NewString()
	}

	sql := `CREATE type::thing('kg_edges', $edge_id) SET
		edge_id = $edge_id,
		app_id = $app_id,
		tenant_id = $tenant_id,
		from_entity_id = $source,
		to_entity_id = $target,
		relation_type = $relation_type,
		properties = $properties,
		confidence = 1.0,
		is_deleted = false,
		created_at = time::now(),
		updated_at = time::now()`

	_, err := r.client.Query(ctx, sql, map[string]any{
		"edge_id":       edgeID,
		"app_id":        appID,
		"tenant_id":     tenantID,
		"source":        sourceNodeID,
		"target":        targetNodeID,
		"relation_type": relationType,
		"properties":    properties,
	})
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] CreateEdge failed edge_id=%s err=%v", edgeID, err)
		return nil, err
	}

	result := cloneMap(properties)
	result["id"] = edgeID
	result["relation_type"] = relationType
	result["source_node_id"] = sourceNodeID
	result["target_node_id"] = targetNodeID
	return result, nil
}

func (r *surrealGraphRepo) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	// Translate Cypher → SurrealQL via QueryTranslator
	surql, err := TranslateCypherToSurrealQL(cypher, params)
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] ExecuteQuery translate failed cypher=%q err=%v", truncate(cypher, 100), err)
		return nil, fmt.Errorf("cypher translation: %w", err)
	}

	result, err := r.client.Query(ctx, surql, params)
	if err != nil {
		r.log.Errorf("[KGS][SurrealDB] ExecuteQuery failed surql=%q err=%v", truncate(surql, 100), err)
		return nil, err
	}

	rows, err := unmarshalSlice[map[string]any](result)
	if err != nil {
		return nil, err
	}
	return map[string]any{"data": rows}, nil
}

func (r *surrealGraphRepo) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error) {
	// Get nodes
	nodeSQL := `SELECT * FROM kg_entities
		WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
		ORDER BY created_at DESC LIMIT $limit START $offset`
	nodeResult, err := r.client.Query(ctx, nodeSQL, map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"limit":     limit,
		"offset":    offset,
	})
	if err != nil {
		return nil, err
	}
	rawNodes, _ := unmarshalSlice[map[string]any](nodeResult)

	// Get edges
	edgeSQL := `SELECT * FROM kg_edges
		WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false
		ORDER BY created_at DESC LIMIT $limit START $offset`
	edgeResult, err := r.client.Query(ctx, edgeSQL, map[string]any{
		"app_id":    appID,
		"tenant_id": tenantID,
		"limit":     limit,
		"offset":    offset,
	})
	if err != nil {
		return nil, err
	}
	rawEdges, _ := unmarshalSlice[map[string]any](edgeResult)

	// Count totals
	countSQL := `SELECT count() AS total FROM kg_entities WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false GROUP ALL`
	countResult, _ := r.client.Query(ctx, countSQL, map[string]any{"app_id": appID, "tenant_id": tenantID})
	totalNodes := extractCount(countResult)

	countEdgeSQL := `SELECT count() AS total FROM kg_edges WHERE app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false GROUP ALL`
	countEdgeResult, _ := r.client.Query(ctx, countEdgeSQL, map[string]any{"app_id": appID, "tenant_id": tenantID})
	totalEdges := extractCount(countEdgeResult)

	// Map results
	nodes := make([]biz.NodeResult, 0, len(rawNodes))
	for _, raw := range rawNodes {
		nodes = append(nodes, biz.NodeResult{
			ID:         fmt.Sprint(raw["entity_id"]),
			Labels:     []string{fmt.Sprint(raw["entity_type"])},
			Properties: raw,
		})
	}

	edges := make([]biz.EdgeResult, 0, len(rawEdges))
	for _, raw := range rawEdges {
		edges = append(edges, biz.EdgeResult{
			ID:           fmt.Sprint(raw["edge_id"]),
			RelationType: fmt.Sprint(raw["relation_type"]),
			SourceNodeID: fmt.Sprint(raw["from_entity_id"]),
			TargetNodeID: fmt.Sprint(raw["to_entity_id"]),
			Properties:   raw,
		})
	}

	return &biz.FullGraphResult{
		Nodes:      nodes,
		Edges:      edges,
		TotalNodes: totalNodes,
		TotalEdges: totalEdges,
	}, nil
}

func (r *surrealGraphRepo) DeleteNode(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
	// Delete related edges first
	edgeSQL := `UPDATE kg_edges SET is_deleted = true, deleted_at = time::now()
		WHERE (from_entity_id = $node_id OR to_entity_id = $node_id)
		AND app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false`
	_, _ = r.client.Query(ctx, edgeSQL, map[string]any{
		"node_id":   nodeID,
		"app_id":    appID,
		"tenant_id": tenantID,
	})

	// Delete the node
	nodeSQL := `UPDATE kg_entities SET is_deleted = true, deleted_at = time::now()
		WHERE entity_id = $entity_id AND app_id = $app_id AND tenant_id = $tenant_id`
	_, err := r.client.Query(ctx, nodeSQL, map[string]any{
		"entity_id": nodeID,
		"app_id":    appID,
		"tenant_id": tenantID,
	})
	if err != nil {
		return 0, err
	}

	return 0, nil // edge count not tracked in simple delete
}

func (r *surrealGraphRepo) DeleteEdge(ctx context.Context, appID, tenantID, edgeID string) error {
	sql := `UPDATE kg_edges SET is_deleted = true, deleted_at = time::now()
		WHERE edge_id = $edge_id AND app_id = $app_id AND tenant_id = $tenant_id`
	_, err := r.client.Query(ctx, sql, map[string]any{
		"edge_id":   edgeID,
		"app_id":    appID,
		"tenant_id": tenantID,
	})
	return err
}

func (r *surrealGraphRepo) BatchDeleteNodes(ctx context.Context, appID, tenantID string, nodeIDs []string) (int, int, error) {
	if len(nodeIDs) == 0 {
		return 0, 0, nil
	}

	// Delete edges connected to any of the nodes
	edgeSQL := `UPDATE kg_edges SET is_deleted = true, deleted_at = time::now()
		WHERE (from_entity_id IN $node_ids OR to_entity_id IN $node_ids)
		AND app_id = $app_id AND tenant_id = $tenant_id AND is_deleted = false`
	_, _ = r.client.Query(ctx, edgeSQL, map[string]any{
		"node_ids":  nodeIDs,
		"app_id":    appID,
		"tenant_id": tenantID,
	})

	// Delete nodes
	nodeSQL := `UPDATE kg_entities SET is_deleted = true, deleted_at = time::now()
		WHERE entity_id IN $node_ids AND app_id = $app_id AND tenant_id = $tenant_id`
	_, err := r.client.Query(ctx, nodeSQL, map[string]any{
		"node_ids":  nodeIDs,
		"app_id":    appID,
		"tenant_id": tenantID,
	})
	if err != nil {
		return 0, 0, err
	}

	return len(nodeIDs), 0, nil
}

// ── Helpers ───────────────────────────────────────────────────

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func extractCount(raw any) int {
	rows, err := unmarshalSlice[map[string]any](raw)
	if err != nil || len(rows) == 0 {
		return 0
	}
	if total, ok := rows[0]["total"]; ok {
		switch v := total.(type) {
		case float64:
			return int(v)
		case int:
			return v
		}
	}
	return 0
}

var _ biz.GraphRepo = (*surrealGraphRepo)(nil)
