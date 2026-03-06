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
