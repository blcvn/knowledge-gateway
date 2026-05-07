package data

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// QueryNodesByProject returns all nodes belonging to a project_id.
// Nodes are stored with property {project_id: "xxx"} by doc_to_kg and ui-knowledge-service.
func (r *graphRepo) QueryNodesByProject(ctx context.Context, projectID string) ([]map[string]any, error) {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (n)
			WHERE n.project_id = $project_id
			RETURN labels(n) AS labels, properties(n) AS props
			LIMIT 2000
		`, map[string]any{"project_id": projectID})
		if err != nil {
			return nil, err
		}

		var nodes []map[string]any
		for res.Next(ctx) {
			rec := res.Record()
			labelsRaw, _ := rec.Get("labels")
			propsRaw, _ := rec.Get("props")

			labels := toStrSlice(labelsRaw)
			props := toAnyMap(propsRaw)
			if props == nil {
				props = map[string]any{}
			}

			// Pick primary label (first non-generic one)
			nodeLabel := pickPrimaryLabel(labels)
			nodeID, _ := props["id"].(string)
			if nodeID == "" {
				continue
			}

			nodes = append(nodes, map[string]any{
				"id":         nodeID,
				"label":      nodeLabel,
				"properties": props,
			})
		}
		return nodes, res.Err()
	})
	if err != nil {
		r.log.Errorf("QueryNodesByProject failed: %v", err)
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]map[string]any), nil
}

// QueryEdgesByProject returns all relationships between nodes in the same project_id.
func (r *graphRepo) QueryEdgesByProject(ctx context.Context, projectID string) ([]map[string]any, error) {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (src)-[r]->(dst)
			WHERE src.project_id = $project_id AND dst.project_id = $project_id
			RETURN src.id AS src_id, dst.id AS dst_id, type(r) AS rel_type, properties(r) AS rel_props
			LIMIT 5000
		`, map[string]any{"project_id": projectID})
		if err != nil {
			return nil, err
		}

		var edges []map[string]any
		i := 0
		for res.Next(ctx) {
			rec := res.Record()
			srcID, _ := rec.Get("src_id")
			dstID, _ := rec.Get("dst_id")
			relType, _ := rec.Get("rel_type")
			relPropsRaw, _ := rec.Get("rel_props")

			src := toString(srcID)
			dst := toString(dstID)
			rt := toString(relType)
			if src == "" || dst == "" {
				continue
			}

			edges = append(edges, map[string]any{
				"id":         edgeID(i),
				"from":       src,
				"to":         dst,
				"label":      rt,
				"properties": toAnyMap(relPropsRaw),
			})
			i++
		}
		return edges, res.Err()
	})
	if err != nil {
		r.log.Errorf("QueryEdgesByProject failed: %v", err)
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	return result.([]map[string]any), nil
}

// DeleteProjectGraph removes all nodes (and their relationships) for a given project_id.
// Uses DETACH DELETE which handles relationship cleanup automatically.
// Returns number of nodes deleted.
func (r *graphRepo) DeleteProjectGraph(ctx context.Context, projectID string) (int64, error) {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (n)
			WHERE n.project_id = $project_id
			DETACH DELETE n
		`, map[string]any{"project_id": projectID})
		if err != nil {
			return int64(0), err
		}
		// Consume result to get counters
		summary, err := res.Consume(ctx)
		if err != nil {
			return int64(0), err
		}
		return int64(summary.Counters().NodesDeleted()), nil
	})
	if err != nil {
		r.log.Errorf("DeleteProjectGraph failed: %v", err)
		return 0, err
	}
	return result.(int64), nil
}



func toStrSlice(v any) []string {
	if v == nil {
		return nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		out = append(out, toString(item))
	}
	return out
}

func toAnyMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// pickPrimaryLabel returns the first non-generic label from the Neo4j label list.
func pickPrimaryLabel(labels []string) string {
	skip := map[string]bool{"Node": true}
	for _, l := range labels {
		if !skip[l] {
			return l
		}
	}
	if len(labels) > 0 {
		return labels[0]
	}
	return "UNKNOWN"
}

func edgeID(i int) string {
	return "edge_" + itoa(i)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}
