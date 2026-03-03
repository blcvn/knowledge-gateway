package data

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// The Graph Usecase relies on these methods to execute planned strings

func (r *graphRepo) ExecuteQuery(ctx context.Context, cypher string, params map[string]any) (map[string]any, error) {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}

		var rows []map[string]any
		for res.Next(ctx) {
			rows = append(rows, res.Record().AsMap())
		}

		if err = res.Err(); err != nil {
			return nil, err
		}

		return map[string]any{"data": rows}, nil
	})

	if err != nil {
		r.log.Errorf("Failed to execute read query: %v\nCypher: %s", err, cypher)
		return nil, err
	}

	return result.(map[string]any), nil
}
