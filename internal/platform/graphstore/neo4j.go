package graphstore

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

type CypherRunner interface {
	Run(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error)
}

type Neo4jGraphAdapter struct {
	Runner CypherRunner
}

func NewNeo4jGraphAdapter(runner CypherRunner) *Neo4jGraphAdapter {
	return &Neo4jGraphAdapter{Runner: runner}
}

func graphQueryToCypher(query GraphQuery, params map[string]any) (string, map[string]any) {
	if params == nil {
		params = map[string]any{}
	}
	strategy := query.Strategy
	if strategy == "" {
		strategy = "default"
	}
	maxDepth := query.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 5
	}
	if strategy == "deep_traversal" {
		maxDepth = maxInt(maxDepth, 10)
	}
	built := map[string]any{}
	parts := []string{fmt.Sprintf("MATCH (n0:%s)", query.StartNodeType)}
	if len(query.StartMatch) > 0 {
		parts = append(parts, "WHERE "+renderMatch("n0", query.StartMatch, built))
	}
	current := "n0"
	for i, hop := range query.Hops {
		next := fmt.Sprintf("n%d", i+1)
		rel := fmt.Sprintf("r%d", i+1)
		dir := strings.ToLower(hop.Direction)
		switch dir {
		case "in":
			parts = append(parts, fmt.Sprintf("MATCH (%s)<-[%s:%s]-(%s:%s)", current, rel, hop.RelType, next, hop.ToNodeType))
		case "both":
			parts = append(parts, fmt.Sprintf("MATCH (%s)-[%s:%s]-(%s:%s)", current, rel, hop.RelType, next, hop.ToNodeType))
		default:
			parts = append(parts, fmt.Sprintf("MATCH (%s)-[%s:%s]->(%s:%s)", current, rel, hop.RelType, next, hop.ToNodeType))
		}
		if len(hop.Filter) > 0 {
			parts = append(parts, "WHERE "+renderMatch(next, hop.Filter, built))
		}
		current = next
	}
	fields := append([]string(nil), query.ReturnFields...)
	if len(fields) == 0 {
		fields = []string{"id"}
	}
	parts = append(parts, "RETURN "+strings.Join(fields, ", "))
	if maxDepth > 0 {
		parts = append(parts, fmt.Sprintf("/* max_depth:%d strategy:%s */", maxDepth, strategy))
	}
	for k, v := range params {
		built[k] = v
	}
	return strings.Join(parts, " "), built
}

func renderMatch(prefix string, match map[string]any, params map[string]any) string {
	keys := make([]string, 0, len(match))
	for key := range match {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	clauses := make([]string, 0, len(keys))
	for _, key := range keys {
		paramName := prefix + "_" + key
		params[paramName] = match[key]
		clauses = append(clauses, fmt.Sprintf("%s.%s = $%s", prefix, key, paramName))
	}
	return strings.Join(clauses, " AND ")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (a *Neo4jGraphAdapter) UpsertNode(ctx context.Context, node GraphNode) error {
	if a == nil || a.Runner == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	_, err := a.Runner.Run(ctx, `
MERGE (n:`+sanitizeLabel(node.NodeType)+` {id: $id})
SET n.domain_id = $domain_id,
    n.owner_tenant_id = $owner_tenant_id,
    n.owner_app_id = $owner_app_id,
    n.visibility = $visibility,
    n.status_value = $status_value,
    n.is_deleted = $is_deleted,
    n.properties = $properties
`, map[string]any{
		"id":              node.ID,
		"domain_id":       node.DomainID,
		"owner_tenant_id": node.OwnerTenantID,
		"owner_app_id":    node.OwnerAppID,
		"visibility":      node.Visibility,
		"status_value":    node.StatusValue,
		"is_deleted":      node.IsDeleted,
		"properties":      clone(node.Properties),
	})
	return err
}

func (a *Neo4jGraphAdapter) DeleteNode(ctx context.Context, nodeID string) error {
	if a == nil || a.Runner == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	_, err := a.Runner.Run(ctx, `MATCH (n {id: $id}) SET n.is_deleted = true`, map[string]any{"id": nodeID})
	return err
}

func (a *Neo4jGraphAdapter) UpsertRelationship(ctx context.Context, rel GraphRelationship) error {
	if a == nil || a.Runner == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	_, err := a.Runner.Run(ctx, `
MATCH (from {id: $from_id})
MATCH (to {id: $to_id})
MERGE (from)-[r:`+sanitizeLabel(rel.RelType)+` {id: $id}]->(to)
SET r.domain_id = $domain_id,
    r.properties = $properties
`, map[string]any{
		"id":         rel.ID,
		"from_id":    rel.FromNodeID,
		"to_id":      rel.ToNodeID,
		"domain_id":  rel.DomainID,
		"properties": clone(rel.Properties),
	})
	return err
}

func (a *Neo4jGraphAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	if a == nil || a.Runner == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	_, err := a.Runner.Run(ctx, `MATCH ()-[r {id: $id}]-() DELETE r`, map[string]any{"id": relID})
	return err
}

func (a *Neo4jGraphAdapter) ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error) {
	if a == nil || a.Runner == nil {
		return nil, fmt.Errorf("neo4j graph adapter is not configured")
	}
	cypher, built := graphQueryToCypher(query, params)
	return a.Runner.Run(ctx, cypher, built)
}

func (a *Neo4jGraphAdapter) ListNodes(context.Context) ([]GraphNode, error) { return nil, nil }
func (a *Neo4jGraphAdapter) ListRelationships(context.Context) ([]GraphRelationship, error) {
	return nil, nil
}

func sanitizeLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "Node"
	}
	var b strings.Builder
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
