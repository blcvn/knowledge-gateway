package search

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type TextSearcher struct {
	driver neo4j.DriverWithContext
	log    *log.Helper
}

func NewTextSearcher(driver neo4j.DriverWithContext, logger log.Logger) *TextSearcher {
	return &TextSearcher{
		driver: driver,
		log:    log.NewHelper(logger),
	}
}

func (s *TextSearcher) Search(ctx context.Context, namespace, query string, topK int) ([]Result, error) {
	if s == nil || s.driver == nil {
		return nil, nil
	}
	if topK <= 0 {
		topK = defaultTopK
	}
	if err := s.ensureFullTextIndex(ctx, namespace); err != nil {
		return nil, err
	}
	appID, tenantID := parseNamespace(namespace)
	indexName := fullTextIndexName(namespace)

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	outAny, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			CALL db.index.fulltext.queryNodes($index_name, $query)
			YIELD node, score
			WHERE node.app_id = $app_id AND node.tenant_id = $tenant_id
			RETURN node.id AS id, labels(node)[0] AS label, properties(node) AS props, score
			ORDER BY score DESC
			LIMIT $limit
		`
		rows, err := tx.Run(ctx, cypher, map[string]any{
			"index_name": indexName,
			"query":      query,
			"app_id":     appID,
			"tenant_id":  tenantID,
			"limit":      topK,
		})
		if err != nil {
			return nil, err
		}
		out := make([]Result, 0, topK)
		for rows.Next(ctx) {
			rec := rows.Record().AsMap()
			props, _ := rec["props"].(map[string]any)
			label := ""
			if rawLabel := rec["label"]; rawLabel != nil {
				label = fmt.Sprint(rawLabel)
			}
			score := readRecordFloat(rec, "score")
			out = append(out, Result{
				ID:         fmt.Sprint(rec["id"]),
				Label:      label,
				Properties: props,
				TextScore:  score,
				Score:      score,
			})
		}
		return out, rows.Err()
	})
	if err != nil {
		return nil, err
	}
	results, _ := outAny.([]Result)
	return results, nil
}

func (s *TextSearcher) ensureFullTextIndex(ctx context.Context, namespace string) error {
	indexName := fullTextIndexName(namespace)
	cypher := fmt.Sprintf(
		"CREATE FULLTEXT INDEX %s IF NOT EXISTS FOR (n) ON EACH [n.name, n.content, n.description]",
		indexName,
	)
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, cypher, nil)
		if err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

type Neo4jCentralityProvider struct {
	driver neo4j.DriverWithContext
}

func NewNeo4jCentralityProvider(driver neo4j.DriverWithContext) *Neo4jCentralityProvider {
	return &Neo4jCentralityProvider{driver: driver}
}

func (p *Neo4jCentralityProvider) Scores(ctx context.Context, namespace string, nodeIDs []string) (map[string]float64, error) {
	if p == nil || p.driver == nil || len(nodeIDs) == 0 {
		return map[string]float64{}, nil
	}
	appID, tenantID := parseNamespace(namespace)
	session := p.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	outAny, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		cypher := `
			UNWIND $node_ids AS id
			MATCH (n {app_id: $app_id, tenant_id: $tenant_id, id: id})
			RETURN id, toFloat(size((n)--())) AS centrality
		`
		rows, err := tx.Run(ctx, cypher, map[string]any{
			"node_ids":  nodeIDs,
			"app_id":    appID,
			"tenant_id": tenantID,
		})
		if err != nil {
			return nil, err
		}
		out := map[string]float64{}
		for rows.Next(ctx) {
			rec := rows.Record().AsMap()
			id := fmt.Sprint(rec["id"])
			out[id] = readRecordFloat(rec, "centrality")
		}
		return out, rows.Err()
	})
	if err != nil {
		return nil, err
	}
	scores, _ := outAny.(map[string]float64)
	return scores, nil
}

func readRecordFloat(m map[string]any, key string) float64 {
	switch value := m[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}
