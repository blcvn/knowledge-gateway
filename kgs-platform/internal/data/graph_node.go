package data

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"kgs-platform/internal/biz"
	"kgs-platform/internal/observability"

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
			return node.Props, nil
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
			return node.Props, nil
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
