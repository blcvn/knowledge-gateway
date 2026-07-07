package graphstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SurrealConfig holds connection parameters for a SurrealDB instance.
// Endpoint is the full HTTP base URL, e.g. "http://surrealdb:8000".
type SurrealConfig struct {
	Endpoint  string
	Namespace string
	Database  string
	Username  string
	Password  string
}

// SurrealGraphAdapter implements GraphAdapter (and BatchGraphAdapter) using
// SurrealDB's HTTP REST API.  No official SDK is required — all queries are
// sent as raw SurrealQL to POST /sql.
//
// Table naming convention:
//   - Nodes     → kg_node_<NodeType>   (e.g. kg_node_Document)
//   - Relations → kg_rel_<RelType>     (e.g. kg_rel_LINKS)
//
// When Endpoint is empty the adapter silently delegates every call to an
// in-memory fallback so that unit tests and config-validation work without a
// running SurrealDB instance.
type SurrealGraphAdapter struct {
	cfg      SurrealConfig
	client   *http.Client
	fallback GraphAdapter
}

// NewSurrealGraphAdapter returns a SurrealGraphAdapter ready for use.
// If cfg.Endpoint is empty it behaves like InMemoryGraphAdapter.
func NewSurrealGraphAdapter(cfg SurrealConfig) *SurrealGraphAdapter {
	if cfg.Namespace == "" {
		cfg.Namespace = "kg"
	}
	if cfg.Database == "" {
		cfg.Database = "kg"
	}
	if cfg.Username == "" {
		cfg.Username = "root"
	}
	if cfg.Password == "" {
		cfg.Password = "root"
	}
	return &SurrealGraphAdapter{
		cfg:      cfg,
		client:   &http.Client{Timeout: 10 * time.Second},
		fallback: NewInMemoryGraphAdapter(),
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func surrealNodeTable(nodeType string) string {
	return "kg_node_" + sanitizeLabel(nodeType)
}

func surrealRelTable(relType string) string {
	return "kg_rel_" + sanitizeLabel(relType)
}

func surrealRecordID(table, id string) string {
	return fmt.Sprintf("%s:⟨%s⟩", table, id)
}

// runSQL sends one or more SurrealQL statements to POST /sql and returns all
// result rows from the first successful statement.
func (a *SurrealGraphAdapter) runSQL(ctx context.Context, query string) ([]map[string]any, error) {
	url := strings.TrimRight(a.cfg.Endpoint, "/") + "/sql"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBufferString(query))
	if err != nil {
		return nil, fmt.Errorf("surreal: build request: %w", err)
	}
	req.SetBasicAuth(a.cfg.Username, a.cfg.Password)
	req.Header.Set("Content-Type", "application/text")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Surreal-NS", a.cfg.Namespace)
	req.Header.Set("Surreal-DB", a.cfg.Database)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("surreal: http %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("surreal: read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("surreal: http %d: %s", resp.StatusCode, string(body))
	}

	// SurrealDB wraps each statement result in an array:
	// [{"status":"OK","result":[...]}]
	var envelope []struct {
		Status string           `json:"status"`
		Detail string           `json:"detail"`
		Result []map[string]any `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("surreal: decode envelope: %w", err)
	}
	for _, e := range envelope {
		if e.Status != "OK" {
			return nil, fmt.Errorf("surreal: query error: %s", e.Detail)
		}
		return e.Result, nil
	}
	return nil, nil
}

// runSQLNoResult executes a statement that returns no interesting rows.
func (a *SurrealGraphAdapter) runSQLNoResult(ctx context.Context, query string) error {
	_, err := a.runSQL(ctx, query)
	return err
}

// marshalSurrealProps converts a map to a SurrealQL-compatible JSON string
// for embedding in SET clauses.
func marshalSurrealProps(m map[string]any) string {
	if len(m) == 0 {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// ── GraphAdapter implementation ───────────────────────────────────────────────

func (a *SurrealGraphAdapter) UpsertNode(ctx context.Context, node GraphNode) error {
	if a.cfg.Endpoint == "" {
		return a.fallback.UpsertNode(ctx, node)
	}
	table := surrealNodeTable(node.NodeType)
	rid := surrealRecordID(table, node.ID)
	props := marshalSurrealProps(node.Properties)

	q := fmt.Sprintf(`
UPSERT %s SET
  id_str           = %q,
  node_type        = %q,
  domain_id        = %q,
  owner_tenant_id  = %q,
  owner_app_id     = %q,
  visibility       = %q,
  status_value     = %q,
  is_deleted       = %v,
  _kg_sync_version = %d,
  properties       = %s,
  acl_visible_to   = %s,
  created_at       = %q,
  updated_at       = %q;`,
		rid,
		node.ID,
		node.NodeType,
		node.DomainID,
		node.OwnerTenantID,
		node.OwnerAppID,
		node.Visibility,
		node.StatusValue,
		node.IsDeleted,
		node.SyncVersion,
		props,
		marshalStringSlice(node.ACLVisibleTo),
		node.CreatedAt.UTC().Format(time.RFC3339),
		node.UpdatedAt.UTC().Format(time.RFC3339),
	)
	if err := a.runSQLNoResult(ctx, q); err != nil {
		return fmt.Errorf("surreal UpsertNode id=%s type=%s: %w", node.ID, node.NodeType, err)
	}
	return nil
}

func (a *SurrealGraphAdapter) DeleteNode(ctx context.Context, nodeID string) error {
	if a.cfg.Endpoint == "" {
		return a.fallback.DeleteNode(ctx, nodeID)
	}
	// We don't know the node type here, so we need to find the record first.
	// Strategy: query all kg_node_* tables by id_str — or simply mark as deleted.
	// For simplicity, iterate known tables via a SELECT across all node tables.
	// SurrealDB supports cross-table via UNION-style or by querying each table.
	// Use a simpler approach: delete by id_str across node tables using a raw scan.
	q := fmt.Sprintf(`
LET $target_id = %q;
FOR $tb IN (SELECT VALUE name FROM db::tables() WHERE name STARTS WITH 'kg_node_') {
  DELETE type::table($tb) WHERE id_str = $target_id;
};`, nodeID)
	if err := a.runSQLNoResult(ctx, q); err != nil {
		return fmt.Errorf("surreal DeleteNode id=%s: %w", nodeID, err)
	}
	return nil
}

func (a *SurrealGraphAdapter) UpsertRelationship(ctx context.Context, rel GraphRelationship) error {
	if a.cfg.Endpoint == "" {
		return a.fallback.UpsertRelationship(ctx, rel)
	}
	// We need the actual record IDs for from/to nodes.
	// Lookup from_node and to_node by id_str across kg_node_* tables.
	props := marshalSurrealProps(rel.Properties)

	q := fmt.Sprintf(`
LET $from_id = %q;
LET $to_id   = %q;
LET $rel_id  = %q;

-- Find source and target records
LET $from_rec = (SELECT id FROM array::flatten(
  (SELECT VALUE (SELECT id FROM type::table(name) WHERE id_str = $from_id LIMIT 1)
   FROM db::tables() WHERE name STARTS WITH 'kg_node_')
) LIMIT 1)[0].id;

LET $to_rec = (SELECT id FROM array::flatten(
  (SELECT VALUE (SELECT id FROM type::table(name) WHERE id_str = $to_id LIMIT 1)
   FROM db::tables() WHERE name STARTS WITH 'kg_node_')
) LIMIT 1)[0].id;

IF $from_rec AND $to_rec THEN {
  LET $edge_table = %q;
  LET $edge_rid   = type::thing($edge_table, $rel_id);
  UPSERT $edge_rid SET
    id_str           = $rel_id,
    rel_type         = %q,
    domain_id        = %q,
    _kg_sync_version = %d,
    properties       = %s
  ;
  RELATE $from_rec -> $edge_rid -> $to_rec;
} END;`,
		rel.FromNodeID,
		rel.ToNodeID,
		rel.ID,
		surrealRelTable(rel.RelType),
		rel.RelType,
		rel.DomainID,
		rel.SyncVersion,
		props,
	)
	if err := a.runSQLNoResult(ctx, q); err != nil {
		return fmt.Errorf("surreal UpsertRelationship id=%s type=%s: %w", rel.ID, rel.RelType, err)
	}
	return nil
}

func (a *SurrealGraphAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	if a.cfg.Endpoint == "" {
		return a.fallback.DeleteRelationship(ctx, relID)
	}
	q := fmt.Sprintf(`
FOR $tb IN (SELECT VALUE name FROM db::tables() WHERE name STARTS WITH 'kg_rel_') {
  DELETE type::table($tb) WHERE id_str = %q;
};`, relID)
	if err := a.runSQLNoResult(ctx, q); err != nil {
		return fmt.Errorf("surreal DeleteRelationship id=%s: %w", relID, err)
	}
	return nil
}

// ── Batch operations ──────────────────────────────────────────────────────────

func (a *SurrealGraphAdapter) UpsertNodesBatch(ctx context.Context, nodes []GraphNode) error {
	for _, node := range nodes {
		if err := a.UpsertNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func (a *SurrealGraphAdapter) DeleteNodesBatch(ctx context.Context, nodes []GraphNode) error {
	for _, node := range nodes {
		if err := a.DeleteNode(ctx, node.ID); err != nil {
			return err
		}
	}
	return nil
}

func (a *SurrealGraphAdapter) UpsertRelationshipsBatch(ctx context.Context, rels []GraphRelationship) error {
	for _, rel := range rels {
		if err := a.UpsertRelationship(ctx, rel); err != nil {
			return err
		}
	}
	return nil
}

func (a *SurrealGraphAdapter) DeleteRelationshipsBatch(ctx context.Context, rels []GraphRelationship) error {
	for _, rel := range rels {
		if err := a.DeleteRelationship(ctx, rel.ID); err != nil {
			return err
		}
	}
	return nil
}

// ── ExecuteQuery ──────────────────────────────────────────────────────────────

// ExecuteQuery translates a GraphQuery into SurrealQL and executes it.
// Multi-hop traversal with up to 2 hops is handled natively; deeper
// traversals fall back to the in-memory engine populated from ListNodes /
// ListRelationships.
func (a *SurrealGraphAdapter) ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error) {
	if a.cfg.Endpoint == "" {
		return a.fallback.ExecuteQuery(ctx, query, params)
	}

	// For deep traversal or complex multi-hop, fall back to in-memory engine.
	if len(query.Hops) > 2 || query.Strategy == "deep_traversal" {
		return a.executeQueryViaMemory(ctx, query, params)
	}

	startTable := surrealNodeTable(query.StartNodeType)

	// Build WHERE clauses for the start node.
	var whereClauses []string
	for k, v := range query.StartMatch {
		val := resolveParam(v, params)
		whereClauses = append(whereClauses, fmt.Sprintf("properties.%s = %s", k, surrealLiteral(val)))
	}
	whereClauses = append(whereClauses, "is_deleted = false")

	whereStr := ""
	if len(whereClauses) > 0 {
		whereStr = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Build traversal chain.
	// SurrealDB arrow syntax: node ->(rel_table)->node
	traversal := buildSurrealTraversal(query.Hops)

	var selectFields string
	if len(query.ReturnFields) > 0 {
		selectFields = strings.Join(query.ReturnFields, ", ")
	} else {
		selectFields = "id_str AS id, node_type, domain_id, properties, _kg_sync_version"
	}

	// If there are hops, include traversal result as nested field.
	var q string
	if traversal == "" {
		q = fmt.Sprintf("SELECT %s FROM %s%s;", selectFields, startTable, whereStr)
	} else {
		q = fmt.Sprintf("SELECT %s, %s AS _hop_chain FROM %s%s FETCH _hop_chain;",
			selectFields, traversal, startTable, whereStr)
	}

	rows, err := a.runSQL(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("surreal ExecuteQuery: %w", err)
	}

	return normalizeSurrealRows(rows), nil
}

// executeQueryViaMemory loads all nodes+rels into the in-memory adapter and
// delegates. Used for deep/complex traversals not easily expressible in SurrealQL.
func (a *SurrealGraphAdapter) executeQueryViaMemory(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error) {
	nodes, err := a.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	rels, err := a.ListRelationships(ctx)
	if err != nil {
		return nil, err
	}
	mem := NewInMemoryGraphAdapter()
	for _, n := range nodes {
		_ = mem.UpsertNode(ctx, n)
	}
	for _, r := range rels {
		_ = mem.UpsertRelationship(ctx, r)
	}
	return mem.ExecuteQuery(ctx, query, params)
}

// buildSurrealTraversal builds the arrow-chain for graph hops.
// Example: one hop LINKS out → "->kg_rel_LINKS->*"
func buildSurrealTraversal(hops []GraphQueryHop) string {
	if len(hops) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, hop := range hops {
		relTable := surrealRelTable(hop.RelType)
		toTable := surrealNodeTable(hop.ToNodeType)
		switch strings.ToLower(hop.Direction) {
		case "in":
			sb.WriteString(fmt.Sprintf("<-%s<-%s", relTable, toTable))
		case "both":
			sb.WriteString(fmt.Sprintf("<->%s<->%s", relTable, toTable))
		default: // out
			sb.WriteString(fmt.Sprintf("->%s->%s", relTable, toTable))
		}
	}
	return sb.String()
}

// ── ListNodes / ListRelationships / ReadSyncVersion ──────────────────────────

func (a *SurrealGraphAdapter) ListNodes(ctx context.Context) ([]GraphNode, error) {
	if a.cfg.Endpoint == "" {
		return a.fallback.ListNodes(ctx)
	}
	q := `
SELECT id_str, node_type, domain_id, owner_tenant_id, owner_app_id,
       visibility, status_value, is_deleted, _kg_sync_version,
       properties, acl_visible_to, created_at, updated_at
FROM array::flatten(
  (SELECT VALUE (SELECT * FROM type::table(name))
   FROM db::tables() WHERE name STARTS WITH 'kg_node_')
);`
	rows, err := a.runSQL(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("surreal ListNodes: %w", err)
	}
	return parseSurrealNodes(rows), nil
}

func (a *SurrealGraphAdapter) ListRelationships(ctx context.Context) ([]GraphRelationship, error) {
	if a.cfg.Endpoint == "" {
		return a.fallback.ListRelationships(ctx)
	}
	q := `
SELECT id_str, rel_type, in.id_str AS from_node_id, out.id_str AS to_node_id,
       domain_id, _kg_sync_version, properties
FROM array::flatten(
  (SELECT VALUE (SELECT *, in.*, out.* FROM type::table(name))
   FROM db::tables() WHERE name STARTS WITH 'kg_rel_')
);`
	rows, err := a.runSQL(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("surreal ListRelationships: %w", err)
	}
	return parseSurrealRelationships(rows), nil
}

func (a *SurrealGraphAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	if a.cfg.Endpoint == "" {
		return a.fallback.ReadSyncVersion(ctx, entityID)
	}
	q := fmt.Sprintf(`
SELECT _kg_sync_version FROM array::flatten([
  (SELECT _kg_sync_version FROM array::flatten(
    (SELECT VALUE (SELECT _kg_sync_version FROM type::table(name) WHERE id_str = %q LIMIT 1)
     FROM db::tables() WHERE name STARTS WITH 'kg_node_')
  ) LIMIT 1),
  (SELECT _kg_sync_version FROM array::flatten(
    (SELECT VALUE (SELECT _kg_sync_version FROM type::table(name) WHERE id_str = %q LIMIT 1)
     FROM db::tables() WHERE name STARTS WITH 'kg_rel_')
  ) LIMIT 1)
]) LIMIT 1;`, entityID, entityID)

	rows, err := a.runSQL(ctx, q)
	if err != nil {
		return 0, fmt.Errorf("surreal ReadSyncVersion id=%s: %w", entityID, err)
	}
	for _, row := range rows {
		if v, ok := readInt64(row["_kg_sync_version"]); ok {
			return v, nil
		}
	}
	return 0, nil
}

// ── Parsing helpers ───────────────────────────────────────────────────────────

func parseSurrealNodes(rows []map[string]any) []GraphNode {
	result := make([]GraphNode, 0, len(rows))
	for _, row := range rows {
		node := GraphNode{
			ID:            fmt.Sprint(row["id_str"]),
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
		if v, ok := readInt64(row["_kg_sync_version"]); ok {
			node.SyncVersion = v
		}
		if props, ok := row["properties"].(map[string]any); ok {
			node.Properties = clone(props)
		} else {
			node.Properties = map[string]any{}
		}
		if acl, ok := row["acl_visible_to"].([]any); ok {
			for _, item := range acl {
				node.ACLVisibleTo = append(node.ACLVisibleTo, fmt.Sprint(item))
			}
		}
		if ts, ok := readTime(row["created_at"]); ok {
			node.CreatedAt = ts
		}
		if ts, ok := readTime(row["updated_at"]); ok {
			node.UpdatedAt = ts
		}
		result = append(result, node)
	}
	return result
}

func parseSurrealRelationships(rows []map[string]any) []GraphRelationship {
	result := make([]GraphRelationship, 0, len(rows))
	for _, row := range rows {
		rel := GraphRelationship{
			ID:         fmt.Sprint(row["id_str"]),
			RelType:    fmt.Sprint(row["rel_type"]),
			FromNodeID: fmt.Sprint(row["from_node_id"]),
			ToNodeID:   fmt.Sprint(row["to_node_id"]),
			DomainID:   fmt.Sprint(row["domain_id"]),
		}
		if v, ok := readInt64(row["_kg_sync_version"]); ok {
			rel.SyncVersion = v
		}
		if props, ok := row["properties"].(map[string]any); ok {
			rel.Properties = clone(props)
		} else {
			rel.Properties = map[string]any{}
		}
		result = append(result, rel)
	}
	return result
}

func normalizeSurrealRows(rows []map[string]any) []map[string]any {
	// Rename id_str → id for callers that expect the "id" key.
	for _, row := range rows {
		if v, ok := row["id_str"]; ok {
			row["id"] = v
			delete(row, "id_str")
		}
	}
	return rows
}

// ── SurrealQL literal helpers ─────────────────────────────────────────────────

// surrealLiteral converts a Go value to a SurrealQL literal string.
func surrealLiteral(v any) string {
	if v == nil {
		return "NONE"
	}
	switch val := v.(type) {
	case string:
		return fmt.Sprintf("%q", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", val)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return "NONE"
		}
		return string(b)
	}
}

// resolveParam resolves a "$param" reference from params map.
func resolveParam(v any, params map[string]any) any {
	s, ok := v.(string)
	if !ok || !strings.HasPrefix(s, "$") {
		return v
	}
	key := strings.TrimPrefix(s, "$")
	if resolved, exists := params[key]; exists {
		return resolved
	}
	return v
}

// marshalStringSlice serialises a []string to a SurrealQL array literal.
func marshalStringSlice(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(ss))
	for _, s := range ss {
		parts = append(parts, fmt.Sprintf("%q", s))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

// ExportBuildSurrealTraversal exposes buildSurrealTraversal for white-box unit
// testing from the _test package.  Not for production use.
var ExportBuildSurrealTraversal = buildSurrealTraversal
