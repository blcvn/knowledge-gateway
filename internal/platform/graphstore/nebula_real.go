//go:build nebula

package graphstore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"

	nebula_ng "github.com/vesoft-inc/nebula-go/v5"
	nebtypes "github.com/vesoft-inc/nebula-go/v5/pkg/types"
)

type nebulaRealAdapter struct {
	client nebtypes.Client
	graph  string
}

func newNebulaDelegate(cfg CypherConfig) GraphAdapter {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return NewInMemoryGraphAdapter()
	}
	client, err := nebula_ng.NewNebulaClient(nebulaAddress(cfg.Endpoint), "", "")
	if err != nil {
		return &nebulaRealAdapter{graph: strings.TrimSpace(cfg.Database)}
	}
	return &nebulaRealAdapter{client: client, graph: strings.TrimSpace(cfg.Database)}
}

func (a *nebulaRealAdapter) UpsertNode(ctx context.Context, node GraphNode) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("nebula graph adapter is not configured")
	}
	props := clone(node.Properties)
	props["id"] = node.ID
	props["_kg_sync_version"] = node.SyncVersion
	raw, err := json.Marshal(props)
	if err != nil {
		return err
	}
	stmt := a.withGraph(fmt.Sprintf("INSERT NODE %s (%s)", sanitizeLabel(node.NodeType), string(raw)))
	_, err = a.client.ExecuteContext(ctx, stmt)
	return err
}

func (a *nebulaRealAdapter) DeleteNode(ctx context.Context, nodeID string) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("nebula graph adapter is not configured")
	}
	stmt := a.withGraph(fmt.Sprintf("DELETE VERTEX %s", strconvQuoteNebula(nodeID)))
	_, err := a.client.ExecuteContext(ctx, stmt)
	return err
}

func (a *nebulaRealAdapter) UpsertRelationship(ctx context.Context, rel GraphRelationship) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("nebula graph adapter is not configured")
	}
	props := clone(rel.Properties)
	props["_kg_sync_version"] = rel.SyncVersion
	raw, err := json.Marshal(props)
	if err != nil {
		return err
	}
	stmt := a.withGraph(fmt.Sprintf(
		"INSERT EDGE %s (%s)-[%s]->(%s)",
		sanitizeLabel(rel.RelType),
		strconvQuoteNebula(rel.FromNodeID),
		string(raw),
		strconvQuoteNebula(rel.ToNodeID),
	))
	_, err = a.client.ExecuteContext(ctx, stmt)
	return err
}

func (a *nebulaRealAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	if a == nil || a.client == nil {
		return fmt.Errorf("nebula graph adapter is not configured")
	}
	stmt := a.withGraph(fmt.Sprintf("DELETE EDGE %s", strconvQuoteNebula(relID)))
	_, err := a.client.ExecuteContext(ctx, stmt)
	return err
}

func (a *nebulaRealAdapter) ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error) {
	if a == nil || a.client == nil {
		return nil, fmt.Errorf("nebula graph adapter is not configured")
	}
	cypher, built := graphQueryToCypher(query, params)
	return a.executeRows(ctx, a.withGraph(substituteNebulaParams(cypher, built)))
}

func (a *nebulaRealAdapter) ListNodes(ctx context.Context) ([]GraphNode, error) {
	rows, err := a.executeRows(ctx, a.withGraph(`
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
    coalesce(n.properties, {}) AS properties
`))
	if err != nil {
		return nil, err
	}
	return scanGraphNodes(rows)
}

func (a *nebulaRealAdapter) ListRelationships(ctx context.Context) ([]GraphRelationship, error) {
	rows, err := a.executeRows(ctx, a.withGraph(`
MATCH ()-[r]->()
RETURN
    r.id AS id,
    type(r) AS rel_type,
    startNode(r).id AS from_node_id,
    endNode(r).id AS to_node_id,
    r.domain_id AS domain_id,
    coalesce(r._kg_sync_version, 0) AS sync_version,
    coalesce(r.properties, {}) AS properties
`))
	if err != nil {
		return nil, err
	}
	return scanGraphRelationships(rows)
}

func (a *nebulaRealAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	rows, err := a.executeRows(ctx, a.withGraph(fmt.Sprintf(`
MATCH (n {id: %s})
RETURN coalesce(n._kg_sync_version, 0) AS sync_version
LIMIT 1
`, strconvQuoteNebula(entityID))))
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

func (a *nebulaRealAdapter) executeRows(ctx context.Context, stmt string) ([]map[string]any, error) {
	result, err := a.client.ExecuteContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	cols := append([]string(nil), result.Columns()...)
	rows := make([]map[string]any, 0, result.RowSize())
	for result.HasNext() {
		row, err := result.Next()
		if err != nil {
			return nil, err
		}
		values := row.Values()
		rowMap := make(map[string]any, len(cols))
		for i, col := range cols {
			if i < len(values) {
				rowMap[col] = nebulaValueToAny(values[i])
			}
		}
		rows = append(rows, rowMap)
	}
	return rows, nil
}

func (a *nebulaRealAdapter) withGraph(stmt string) string {
	graph := strings.TrimSpace(a.graph)
	if graph == "" {
		return stmt
	}
	return "USE " + graph + " " + stmt
}

func nebulaAddress(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return ""
	}
	if strings.Contains(endpoint, "://") {
		if u, err := url.Parse(endpoint); err == nil && u.Host != "" {
			return u.Host
		}
	}
	return endpoint
}

func strconvQuoteNebula(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}

func nebulaValueToAny(v nebtypes.Value) any {
	switch v.GetType() {
	case nebtypes.ValueTypeBool:
		val, _ := v.AsBool()
		return bool(val)
	case nebtypes.ValueTypeInt8:
		val, _ := v.AsInt8()
		return int64(val)
	case nebtypes.ValueTypeInt16:
		val, _ := v.AsInt16()
		return int64(val)
	case nebtypes.ValueTypeInt32:
		val, _ := v.AsInt32()
		return int64(val)
	case nebtypes.ValueTypeInt64:
		val, _ := v.AsInt64()
		return int64(val)
	case nebtypes.ValueTypeUInt8:
		val, _ := v.AsUInt8()
		return uint64(val)
	case nebtypes.ValueTypeUInt16:
		val, _ := v.AsUInt16()
		return uint64(val)
	case nebtypes.ValueTypeUInt32:
		val, _ := v.AsUInt32()
		return uint64(val)
	case nebtypes.ValueTypeUInt64:
		val, _ := v.AsUInt64()
		return uint64(val)
	case nebtypes.ValueTypeFloat:
		val, _ := v.AsFloat()
		return float64(val)
	case nebtypes.ValueTypeDouble:
		val, _ := v.AsDouble()
		return float64(val)
	case nebtypes.ValueTypeString:
		val, _ := v.AsString()
		return string(val)
	default:
		return v.String()
	}
}

func substituteNebulaParams(stmt string, params map[string]any) string {
	if len(params) == 0 {
		return stmt
	}
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return len(keys[i]) > len(keys[j])
	})
	replaced := stmt
	for _, key := range keys {
		replaced = strings.ReplaceAll(replaced, "$"+key, nebulaLiteral(params[key]))
	}
	return replaced
}

func nebulaLiteral(v any) string {
	switch val := v.(type) {
	case nil:
		return "NULL"
	case string:
		return strconvQuoteNebula(val)
	case fmt.Stringer:
		return strconvQuoteNebula(val.String())
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case int8:
		return fmt.Sprintf("%d", val)
	case int16:
		return fmt.Sprintf("%d", val)
	case int32:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case uint:
		return fmt.Sprintf("%d", val)
	case uint8:
		return fmt.Sprintf("%d", val)
	case uint16:
		return fmt.Sprintf("%d", val)
	case uint32:
		return fmt.Sprintf("%d", val)
	case uint64:
		return fmt.Sprintf("%d", val)
	case float32:
		return fmt.Sprintf("%g", val)
	case float64:
		return fmt.Sprintf("%g", val)
	case []string:
		items := make([]string, 0, len(val))
		for _, item := range val {
			items = append(items, strconvQuoteNebula(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	case []any:
		items := make([]string, 0, len(val))
		for _, item := range val {
			items = append(items, nebulaLiteral(item))
		}
		return "[" + strings.Join(items, ", ") + "]"
	default:
		return strconvQuoteNebula(fmt.Sprint(val))
	}
}
