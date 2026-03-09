package data

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"kgs-platform/internal/observability"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.opentelemetry.io/otel/attribute"
	"kgs-platform/internal/biz"
)

// The Graph Usecase relies on these methods to execute planned strings

func (r *graphRepo) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.execute_query", attribute.String("neo4j.mode", "read"))
	defer span.End()

	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(traceCtx, cypher, params)
		if err != nil {
			return nil, err
		}

		var rows []map[string]any
		for res.Next(traceCtx) {
			rows = append(rows, res.Record().AsMap())
		}

		if err = res.Err(); err != nil {
			return nil, err
		}

		return map[string]any{"data": rows}, nil
	})

	if err != nil {
		observability.RecordSpanError(span, err)
		r.log.Errorf("Failed to execute read query: %v\nCypher: %s", err, cypher)
		return nil, err
	}

	return result.(map[string]any), nil
}

func (r *graphRepo) GetFullGraph(ctx context.Context, appID, tenantID string, limit, offset int) (*biz.FullGraphResult, error) {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.get_full_graph",
		attribute.String("app_id", appID),
		attribute.String("tenant_id", tenantID),
		attribute.Int("node_limit", limit),
		attribute.Int("node_offset", offset),
	)
	defer span.End()

	if limit <= 0 {
		limit = biz.MaxAllowedNodes
	}
	if limit > biz.MaxAllowedNodes {
		limit = biz.MaxAllowedNodes
	}
	if offset < 0 {
		offset = 0
	}

	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	out := &biz.FullGraphResult{
		Nodes: make([]biz.NodeResult, 0),
		Edges: make([]biz.EdgeResult, 0),
	}

	totalNodes, err := r.countByQuery(traceCtx, session, buildCountNodesQuery(), appID, tenantID)
	if err != nil {
		observability.RecordSpanError(span, err)
		return nil, fmt.Errorf("count nodes: %w", err)
	}
	out.TotalNodes = totalNodes

	totalEdges, err := r.countByQuery(traceCtx, session, buildCountEdgesQuery(), appID, tenantID)
	if err != nil {
		observability.RecordSpanError(span, err)
		return nil, fmt.Errorf("count edges: %w", err)
	}
	out.TotalEdges = totalEdges

	nodes, err := r.fetchFullGraphNodes(traceCtx, session, appID, tenantID, limit, offset)
	if err != nil {
		observability.RecordSpanError(span, err)
		return nil, fmt.Errorf("fetch nodes: %w", err)
	}
	out.Nodes = nodes

	if len(nodes) == 0 {
		return out, nil
	}

	nodeIDs := make([]string, 0, len(nodes))
	for _, node := range nodes {
		nodeIDs = append(nodeIDs, node.ID)
	}

	edges, err := r.fetchFullGraphEdges(traceCtx, session, appID, tenantID, nodeIDs)
	if err != nil {
		observability.RecordSpanError(span, err)
		return nil, fmt.Errorf("fetch edges: %w", err)
	}
	out.Edges = edges
	return out, nil
}

func (r *graphRepo) countByQuery(ctx context.Context, session neo4j.SessionWithContext, query, appID, tenantID string) (int, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
		})
		if err != nil {
			return nil, err
		}
		if !res.Next(ctx) {
			return int64(0), res.Err()
		}
		value, ok := res.Record().Get("total")
		if !ok {
			return nil, fmt.Errorf("missing total in count query result")
		}
		total, ok := value.(int64)
		if !ok {
			return nil, fmt.Errorf("invalid total type %T", value)
		}
		return total, nil
	})
	if err != nil {
		return 0, err
	}
	total, ok := result.(int64)
	if !ok {
		return 0, fmt.Errorf("invalid count result type %T", result)
	}
	return int(total), nil
}

func (r *graphRepo) fetchFullGraphNodes(ctx context.Context, session neo4j.SessionWithContext, appID, tenantID string, limit, offset int) ([]biz.NodeResult, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, buildGetFullGraphNodesQuery(), map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
			"limit":     limit,
			"offset":    offset,
		})
		if err != nil {
			return nil, err
		}

		nodes := make([]biz.NodeResult, 0, limit)
		for res.Next(ctx) {
			record := res.Record()
			rawNode, ok := record.Get("n")
			if !ok {
				return nil, fmt.Errorf("missing node field in get full graph query")
			}
			node, ok := rawNode.(neo4j.Node)
			if !ok {
				return nil, fmt.Errorf("invalid node type %T in get full graph query", rawNode)
			}

			props := cloneMap(node.Props)
			nodeID := fmt.Sprint(props["id"])
			if nodeID == "" || nodeID == "<nil>" {
				nodeID = fmt.Sprint(node.Id)
				props["id"] = nodeID
			}

			nodes = append(nodes, biz.NodeResult{
				ID:         nodeID,
				Labels:     append([]string(nil), node.Labels...),
				Properties: props,
			})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nodes, nil
	})
	if err != nil {
		return nil, err
	}
	nodes, ok := result.([]biz.NodeResult)
	if !ok {
		return nil, fmt.Errorf("invalid nodes result type %T", result)
	}
	return nodes, nil
}

func (r *graphRepo) fetchFullGraphEdges(ctx context.Context, session neo4j.SessionWithContext, appID, tenantID string, nodeIDs []string) ([]biz.EdgeResult, error) {
	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, buildGetFullGraphEdgesQuery(), map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
			"node_ids":  nodeIDs,
		})
		if err != nil {
			return nil, err
		}

		edges := make([]biz.EdgeResult, 0)
		for res.Next(ctx) {
			record := res.Record()
			rawRel, ok := record.Get("r")
			if !ok {
				return nil, fmt.Errorf("missing relation field in get full graph query")
			}
			rel, ok := rawRel.(neo4j.Relationship)
			if !ok {
				return nil, fmt.Errorf("invalid relation type %T in get full graph query", rawRel)
			}

			relType := fmt.Sprint(record.AsMap()["rel_type"])
			sourceID := fmt.Sprint(record.AsMap()["source_id"])
			targetID := fmt.Sprint(record.AsMap()["target_id"])
			props := cloneMap(rel.Props)
			edgeID := fmt.Sprint(props["id"])
			if edgeID == "" || edgeID == "<nil>" {
				edgeID = fmt.Sprint(rel.Id)
				props["id"] = edgeID
			}

			edges = append(edges, biz.EdgeResult{
				ID:           edgeID,
				RelationType: relType,
				SourceNodeID: sourceID,
				TargetNodeID: targetID,
				Properties:   props,
			})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return edges, nil
	})
	if err != nil {
		return nil, err
	}
	edges, ok := result.([]biz.EdgeResult)
	if !ok {
		return nil, fmt.Errorf("invalid edges result type %T", result)
	}
	return edges, nil
}

func buildCountNodesQuery() string {
	return `
		MATCH (n:Entity {app_id: $app_id, tenant_id: $tenant_id})
		RETURN count(n) AS total
	`
}

func buildCountEdgesQuery() string {
	return `
		MATCH (:Entity {app_id: $app_id, tenant_id: $tenant_id})-[r]->(:Entity {app_id: $app_id, tenant_id: $tenant_id})
		RETURN count(r) AS total
	`
}

func buildGetFullGraphNodesQuery() string {
	return `
		MATCH (n:Entity {app_id: $app_id, tenant_id: $tenant_id})
		RETURN n
		ORDER BY n.id
		SKIP $offset
		LIMIT $limit
	`
}

func buildGetFullGraphEdgesQuery() string {
	return `
		MATCH (a:Entity {app_id: $app_id, tenant_id: $tenant_id})-[r]->(b:Entity {app_id: $app_id, tenant_id: $tenant_id})
		WHERE a.id IN $node_ids AND b.id IN $node_ids
		RETURN r, type(r) AS rel_type, a.id AS source_id, b.id AS target_id
		ORDER BY r.id
	`
}

func (r *graphRepo) GetPageRank(ctx context.Context, namespace string) (map[string]float64, error) {
	cacheKey := "kgs:gds:pagerank:" + namespace
	cached, err := r.data.rc.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var parsed map[string]float64
		if json.Unmarshal([]byte(cached), &parsed) == nil {
			return parsed, nil
		}
	}

	graphName := "kgs-graph-" + namespace
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.gds.pagerank")
	defer span.End()
	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	out := map[string]float64{}
	_, err = session.ExecuteRead(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			CALL gds.pageRank.stream($graph_name)
			YIELD nodeId, score
			RETURN gds.util.asNode(nodeId).id AS id, score
		`
		res, err := tx.Run(traceCtx, query, map[string]any{"graph_name": graphName})
		if err != nil {
			return nil, err
		}
		for res.Next(traceCtx) {
			row := res.Record().AsMap()
			id := fmt.Sprint(row["id"])
			switch score := row["score"].(type) {
			case float64:
				out[id] = score
			case int64:
				out[id] = float64(score)
			}
		}
		return nil, res.Err()
	})
	if err != nil {
		observability.RecordSpanError(span, err)
		return nil, err
	}

	if buf, err := json.Marshal(out); err == nil {
		_ = r.data.rc.Set(ctx, cacheKey, string(buf), 15*time.Minute).Err()
	}
	return out, nil
}

func (r *graphRepo) GetDegreeCentrality(ctx context.Context, namespace string) (map[string]float64, error) {
	graphName := "kgs-graph-" + namespace
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.gds.degree")
	defer span.End()
	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	out := map[string]float64{}
	_, err := session.ExecuteRead(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			CALL gds.degree.stream($graph_name)
			YIELD nodeId, score
			RETURN gds.util.asNode(nodeId).id AS id, score
		`
		res, err := tx.Run(traceCtx, query, map[string]any{"graph_name": graphName})
		if err != nil {
			return nil, err
		}
		for res.Next(traceCtx) {
			row := res.Record().AsMap()
			id := fmt.Sprint(row["id"])
			switch score := row["score"].(type) {
			case float64:
				out[id] = score
			case int64:
				out[id] = float64(score)
			}
		}
		return nil, res.Err()
	})
	if err != nil {
		observability.RecordSpanError(span, err)
		return nil, err
	}
	return out, nil
}

var _ biz.GraphRepo = (*graphRepo)(nil)
