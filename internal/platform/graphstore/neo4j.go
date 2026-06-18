package graphstore

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	neo4j "github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type CypherRunner interface {
	Run(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error)
}

type Neo4jGraphAdapter struct {
	Runner   CypherRunner
	Driver   neo4j.Driver
	fallback GraphAdapter
	Endpoint string
	mu       sync.Mutex
	initErr  error
}

func NewNeo4jGraphAdapter(cfg CypherConfig) *Neo4jGraphAdapter {
	adapter := &Neo4jGraphAdapter{
		fallback: NewInMemoryGraphAdapter(),
		Endpoint: cfg.Endpoint,
	}
	if strings.TrimSpace(cfg.Endpoint) != "" {
		driver, err := neo4j.NewDriver(cfg.Endpoint, neo4j.NoAuth())
		if err == nil {
			adapter.Driver = driver
		} else {
			adapter.initErr = err
		}
	}
	return adapter
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
	if a == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.UpsertNode(ctx, node)
	}
	_, err := a.runCypher(ctx, `
MERGE (n:`+sanitizeLabel(node.NodeType)+` {id: $id})
SET n.domain_id = $domain_id,
    n.owner_tenant_id = $owner_tenant_id,
    n.owner_app_id = $owner_app_id,
    n.visibility = $visibility,
    n.status_value = $status_value,
    n.is_deleted = $is_deleted,
    n._kg_sync_version = $sync_version,
    n.properties = $properties
`, map[string]any{
		"id":              node.ID,
		"domain_id":       node.DomainID,
		"owner_tenant_id": node.OwnerTenantID,
		"owner_app_id":    node.OwnerAppID,
		"visibility":      node.Visibility,
		"status_value":    node.StatusValue,
		"is_deleted":      node.IsDeleted,
		"sync_version":    node.SyncVersion,
		"properties":      clone(node.Properties),
	})
	return err
}

func (a *Neo4jGraphAdapter) DeleteNode(ctx context.Context, nodeID string) error {
	if a == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.DeleteNode(ctx, nodeID)
	}
	_, err := a.runCypher(ctx, `MATCH (n {id: $id}) DETACH DELETE n`, map[string]any{"id": nodeID})
	return err
}

func (a *Neo4jGraphAdapter) UpsertRelationship(ctx context.Context, rel GraphRelationship) error {
	if a == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.UpsertRelationship(ctx, rel)
	}
	_, err := a.runCypher(ctx, `
MATCH (from {id: $from_id})
MATCH (to {id: $to_id})
MERGE (from)-[r:`+sanitizeLabel(rel.RelType)+` {id: $id}]->(to)
SET r.domain_id = $domain_id,
    r._kg_sync_version = $sync_version,
    r.properties = $properties
`, map[string]any{
		"id":           rel.ID,
		"from_id":      rel.FromNodeID,
		"to_id":        rel.ToNodeID,
		"domain_id":    rel.DomainID,
		"sync_version": rel.SyncVersion,
		"properties":   clone(rel.Properties),
	})
	return err
}

func (a *Neo4jGraphAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	if a == nil {
		return fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.DeleteRelationship(ctx, relID)
	}
	_, err := a.runCypher(ctx, `MATCH ()-[r {id: $id}]-() DELETE r`, map[string]any{"id": relID})
	return err
}

func (a *Neo4jGraphAdapter) ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error) {
	if a == nil {
		return nil, fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.ExecuteQuery(ctx, query, params)
	}
	cypher, built := graphQueryToCypher(query, params)
	return a.runCypher(ctx, cypher, built)
}

func (a *Neo4jGraphAdapter) ListNodes(ctx context.Context) ([]GraphNode, error) {
	if a == nil {
		return nil, fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.ListNodes(ctx)
	}
	rows, err := a.runCypher(ctx, `
MATCH (n)
RETURN
    n.id AS id,
    labels(n)[0] AS node_type,
    n.domain_id AS domain_id,
    n.owner_tenant_id AS owner_tenant_id,
    coalesce(n.owner_app_id, '') AS owner_app_id,
    coalesce(n.visibility, '') AS visibility,
    coalesce(n.status_value, '') AS status_value,
    coalesce(n.is_deleted, false) AS is_deleted,
    coalesce(n._kg_sync_version, 0) AS sync_version,
    coalesce(n.properties, {}) AS properties,
    coalesce(n.created_at, datetime()) AS created_at,
    coalesce(n.updated_at, datetime()) AS updated_at
ORDER BY created_at, id
`, nil)
	if err != nil {
		return nil, err
	}
	return scanGraphNodes(rows)
}

func (a *Neo4jGraphAdapter) ListRelationships(ctx context.Context) ([]GraphRelationship, error) {
	if a == nil {
		return nil, fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return nil, err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.ListRelationships(ctx)
	}
	rows, err := a.runCypher(ctx, `
MATCH ()-[r]->()
RETURN
    r.id AS id,
    type(r) AS rel_type,
    startNode(r).id AS from_node_id,
    endNode(r).id AS to_node_id,
    r.domain_id AS domain_id,
    coalesce(r._kg_sync_version, 0) AS sync_version,
    coalesce(r.properties, {}) AS properties
ORDER BY id
`, nil)
	if err != nil {
		return nil, err
	}
	return scanGraphRelationships(rows)
}

func (a *Neo4jGraphAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	if a == nil {
		return 0, fmt.Errorf("neo4j graph adapter is not configured")
	}
	if err := a.ensureReady(); err != nil {
		return 0, err
	}
	if a.Runner == nil && a.Driver == nil {
		return a.fallback.ReadSyncVersion(ctx, entityID)
	}
	rows, err := a.runCypher(ctx, `
MATCH (n {id: $id})
RETURN coalesce(n._kg_sync_version, 0) AS sync_version
UNION ALL
MATCH ()-[r {id: $id}]-()
RETURN coalesce(r._kg_sync_version, 0) AS sync_version
LIMIT 1
`, map[string]any{"id": entityID})
	if err != nil {
		return 0, err
	}
	for _, row := range rows {
		if version, ok := readInt64(row["sync_version"]); ok {
			return version, nil
		}
	}
	return 0, nil
}

func (a *Neo4jGraphAdapter) runCypher(ctx context.Context, cypher string, params map[string]any) ([]map[string]any, error) {
	if a.Runner != nil {
		return a.Runner.Run(ctx, cypher, params)
	}
	if a.Driver == nil {
		return nil, fmt.Errorf("neo4j graph adapter is not configured")
	}
	session := a.Driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() {
		_ = session.Close(ctx)
	}()
	result, err := session.Run(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	records, err := result.Collect(ctx)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]any, 0, len(records))
	for _, record := range records {
		rows = append(rows, record.AsMap())
	}
	return rows, nil
}

func (a *Neo4jGraphAdapter) ensureReady() error {
	if a.initErr != nil {
		return a.initErr
	}
	return nil
}

func scanGraphNodes(rows []map[string]any) ([]GraphNode, error) {
	result := make([]GraphNode, 0, len(rows))
	for _, row := range rows {
		node := GraphNode{
			ID:            fmt.Sprint(row["id"]),
			NodeType:      fmt.Sprint(row["node_type"]),
			DomainID:      fmt.Sprint(row["domain_id"]),
			OwnerTenantID: fmt.Sprint(row["owner_tenant_id"]),
			OwnerAppID:    fmt.Sprint(row["owner_app_id"]),
			Visibility:    fmt.Sprint(row["visibility"]),
			StatusValue:   fmt.Sprint(row["status_value"]),
		}
		if v, ok := readBool(row["is_deleted"]); ok {
			node.IsDeleted = v
		}
		if v, ok := readInt64(row["sync_version"]); ok {
			node.SyncVersion = v
		}
		if raw, ok := row["properties"].(map[string]any); ok {
			node.Properties = clone(raw)
		} else if rawBytes, ok := row["properties"].([]byte); ok && len(rawBytes) > 0 {
			_ = json.Unmarshal(rawBytes, &node.Properties)
		}
		if createdAt, ok := readTime(row["created_at"]); ok {
			node.CreatedAt = createdAt
		}
		if updatedAt, ok := readTime(row["updated_at"]); ok {
			node.UpdatedAt = updatedAt
		}
		if node.Properties == nil {
			node.Properties = map[string]any{}
		}
		result = append(result, node)
	}
	return result, nil
}

func scanGraphRelationships(rows []map[string]any) ([]GraphRelationship, error) {
	result := make([]GraphRelationship, 0, len(rows))
	for _, row := range rows {
		rel := GraphRelationship{
			ID:         fmt.Sprint(row["id"]),
			RelType:    fmt.Sprint(row["rel_type"]),
			FromNodeID: fmt.Sprint(row["from_node_id"]),
			ToNodeID:   fmt.Sprint(row["to_node_id"]),
			DomainID:   fmt.Sprint(row["domain_id"]),
		}
		if v, ok := readInt64(row["sync_version"]); ok {
			rel.SyncVersion = v
		}
		if raw, ok := row["properties"].(map[string]any); ok {
			rel.Properties = clone(raw)
		} else if rawBytes, ok := row["properties"].([]byte); ok && len(rawBytes) > 0 {
			_ = json.Unmarshal(rawBytes, &rel.Properties)
		}
		if rel.Properties == nil {
			rel.Properties = map[string]any{}
		}
		result = append(result, rel)
	}
	return result, nil
}

func readInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case int:
		return int64(v), true
	case float64:
		return int64(v), true
	case fmt.Stringer:
		var parsed int64
		if _, err := fmt.Sscan(v.String(), &parsed); err == nil {
			return parsed, true
		}
	case string:
		var parsed int64
		if _, err := fmt.Sscan(v, &parsed); err == nil {
			return parsed, true
		}
	}
	return 0, false
}

func readBool(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(v) {
		case "true":
			return true, true
		case "false":
			return false, true
		}
	}
	return false, false
}

func readTime(value any) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case string:
		parsed, err := time.Parse(time.RFC3339Nano, v)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
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
