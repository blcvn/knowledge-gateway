package data

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/observability"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.opentelemetry.io/otel/attribute"
)

type graphRepo struct {
	data *Data
	log  *log.Helper
}

// NewGraphRepo .
func NewGraphRepo(data *Data, logger log.Logger) biz.GraphRepo {
	repo := &graphRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
	if err := repo.ensureGlobalFullTextIndex(context.Background()); err != nil {
		repo.log.Warnf("failed to ensure global fulltext index: %v", err)
	}
	return repo
}

// CreateNode creates a new namespaced node in Neo4j
func (r *graphRepo) CreateNode(ctx context.Context, appID, tenantID string, label string, properties map[string]any) (map[string]any, error) {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.create_node", attribute.String("neo4j.label", label))
	defer span.End()
	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	cleanLabel, err := sanitizeCypherIdentifier(label)
	if err != nil {
		return nil, err
	}
	props := cloneMap(properties)
	nodeID := ensureID(props)
	props["_unique_key"] = buildNodeUniqueKey(appID, tenantID, nodeID)

	result, err := session.ExecuteWrite(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := buildCreateNodeQuery(cleanLabel)
		params := map[string]interface{}{
			"app_id":     appID,
			"tenant_id":  tenantID,
			"node_id":    nodeID,
			"unique_key": props["_unique_key"],
			"props":      props,
		}

		res, err := tx.Run(traceCtx, query, params)
		if err != nil {
			return nil, err
		}

		if res.Next(traceCtx) {
			node := res.Record().Values[0].(neo4j.Node)
			return enrichNodePropsWithLabels(node), nil
		}

		return nil, res.Err()
	})

	if err != nil {
		observability.RecordSpanError(span, err)
		r.log.Errorf("Failed to create node: %v", err)
		return nil, err
	}

	return result.(map[string]any), nil
}

func enrichNodePropsWithLabels(node neo4j.Node) map[string]any {
	props := cloneMap(node.Props)
	if len(node.Labels) == 0 {
		return props
	}

	labels := make([]string, 0, len(node.Labels))
	var primaryLabel string
	for _, label := range node.Labels {
		trimmed := strings.TrimSpace(label)
		if trimmed == "" {
			continue
		}
		labels = append(labels, trimmed)
		if primaryLabel == "" && !strings.EqualFold(trimmed, "Entity") {
			primaryLabel = trimmed
		}
	}

	if primaryLabel == "" && len(labels) > 0 {
		primaryLabel = labels[0]
	}
	if primaryLabel != "" {
		props["label"] = primaryLabel
		props["entity_type"] = primaryLabel
	}
	if len(labels) > 0 {
		props["labels"] = labels
	}
	return props
}

func buildCreateNodeQuery(cleanLabel string) string {
	return fmt.Sprintf(`
		MERGE (n:Entity:%s {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
		ON CREATE SET n += $props, n.created_at = datetime(), n._unique_key = $unique_key
		ON MATCH SET n += $props, n.updated_at = datetime(), n._unique_key = $unique_key
		RETURN n
	`, cleanLabel)
}

func buildNodeUniqueKey(appID, tenantID, nodeID string) string {
	return fmt.Sprintf("%s|%s|%s", strings.TrimSpace(appID), strings.TrimSpace(tenantID), strings.TrimSpace(nodeID))
}

func (r *graphRepo) GetNode(ctx context.Context, appID, tenantID, nodeID string) (map[string]any, error) {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.get_node", attribute.String("neo4j.node_id", nodeID))
	defer span.End()
	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
			RETURN n
			LIMIT 1
		`
		res, err := tx.Run(traceCtx, query, map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
			"node_id":   nodeID,
		})
		if err != nil {
			return nil, err
		}
		if res.Next(traceCtx) {
			node := res.Record().Values[0].(neo4j.Node)
			return enrichNodePropsWithLabels(node), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("node not found")
	})
	if err != nil {
		observability.RecordSpanError(span, err)
		return nil, err
	}
	return result.(map[string]any), nil
}

// DeleteNode removes a node and all incident edges (detach delete).
// It returns the number of edges removed by cascade delete.
func (r *graphRepo) DeleteNode(ctx context.Context, appID, tenantID, nodeID string) (int, error) {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.delete_node")
	defer span.End()

	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		countRes, err := tx.Run(traceCtx, buildDeleteNodeEdgeCountQuery(), map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
			"node_id":   nodeID,
		})
		if err != nil {
			return nil, err
		}
		if !countRes.Next(traceCtx) {
			if err := countRes.Err(); err != nil {
				return nil, err
			}
			return int64(0), nil
		}
		edgeCount, _ := countRes.Record().Get("edge_count")

		_, err = tx.Run(traceCtx, buildDeleteNodeQuery(), map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
			"node_id":   nodeID,
		})
		if err != nil {
			return nil, err
		}
		return toInt64(edgeCount), nil
	})
	if err != nil {
		observability.RecordSpanError(span, err)
		return 0, err
	}

	return int(toInt64(result)), nil
}

// BatchDeleteNodes removes multiple nodes (and their incident edges) in one transaction.
func (r *graphRepo) BatchDeleteNodes(ctx context.Context, appID, tenantID string, nodeIDs []string) (deleted, edgesRemoved int, err error) {
	if len(nodeIDs) == 0 {
		return 0, 0, nil
	}

	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.batch_delete_nodes")
	defer span.End()

	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		countRes, err := tx.Run(traceCtx, buildBatchDeleteNodesCountQuery(), map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
			"node_ids":  nodeIDs,
		})
		if err != nil {
			return nil, err
		}
		if !countRes.Next(traceCtx) {
			if err := countRes.Err(); err != nil {
				return nil, err
			}
			return []int64{0, 0}, nil
		}

		record := countRes.Record()
		deletedCount, _ := record.Get("deleted")
		edgeCount, _ := record.Get("edges_removed")

		_, err = tx.Run(traceCtx, buildBatchDeleteNodesQuery(), map[string]any{
			"app_id":    appID,
			"tenant_id": tenantID,
			"node_ids":  nodeIDs,
		})
		if err != nil {
			return nil, err
		}

		return []int64{toInt64(deletedCount), toInt64(edgeCount)}, nil
	})
	if err != nil {
		observability.RecordSpanError(span, err)
		return 0, 0, err
	}

	counts, _ := result.([]int64)
	if len(counts) != 2 {
		return 0, 0, nil
	}
	return int(counts[0]), int(counts[1]), nil
}

func ensureID(props map[string]any) string {
	if props == nil {
		return uuid.NewString()
	}
	if id, ok := props["id"].(string); ok && id != "" {
		return id
	}
	id := uuid.NewString()
	props["id"] = id
	return id
}

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func buildDeleteNodeEdgeCountQuery() string {
	return `
		MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
		OPTIONAL MATCH (n)-[r]-()
		RETURN count(r) AS edge_count
	`
}

func buildDeleteNodeQuery() string {
	return `
		MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: $node_id})
		DETACH DELETE n
	`
}

func buildBatchDeleteNodesCountQuery() string {
	return `
		UNWIND $node_ids AS node_id
		OPTIONAL MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: node_id})
		WITH [x IN collect(DISTINCT n) WHERE x IS NOT NULL] AS nodes
		OPTIONAL MATCH (m)-[r]-()
		WHERE m IN nodes
		RETURN size(nodes) AS deleted, count(DISTINCT r) AS edges_removed
	`
}

func buildBatchDeleteNodesQuery() string {
	return `
		UNWIND $node_ids AS node_id
		MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: node_id})
		DETACH DELETE n
	`
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

var cypherIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func sanitizeCypherIdentifier(input string) (string, error) {
	if !cypherIdentifierPattern.MatchString(input) {
		return "", fmt.Errorf("invalid cypher identifier: %q", input)
	}
	return input, nil
}

func (r *graphRepo) ensureGlobalFullTextIndex(ctx context.Context) error {
	traceCtx, span := observability.StartDependencySpan(ctx, "neo4j", "neo4j.ensure_fulltext_index")
	defer span.End()
	session := r.data.neo4j.NewSession(traceCtx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(traceCtx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(traceCtx, "CREATE FULLTEXT INDEX kgs_fti_global IF NOT EXISTS FOR (n) ON EACH [n.name, n.content, n.description]", nil)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		observability.RecordSpanError(span, err)
	}
	return err
}
