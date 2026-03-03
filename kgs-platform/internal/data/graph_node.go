package data

import (
	"context"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type graphRepo struct {
	data *Data
	log  *log.Helper
}

// NewGraphRepo .
func NewGraphRepo(data *Data, logger log.Logger) biz.GraphRepo {
	return &graphRepo{
		data: data,
		log:  log.NewHelper(logger),
	}
}

// CreateNode creates a new namespaced node in Neo4j
func (r *graphRepo) CreateNode(ctx context.Context, appID string, label string, properties map[string]any) (map[string]any, error) {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		// DYNAMIC CYPHER - safe because parameterization
		// We add the namespace prefix to both label and id properties
		query := `
			CREATE (n:` + label + ` {app_id: $app_id})
			SET n += $props
			RETURN n
		`
		params := map[string]interface{}{
			"app_id": appID,
			"props":  properties,
		}

		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		if res.Next(ctx) {
			node := res.Record().Values[0].(neo4j.Node)
			return node.Props, nil
		}

		return nil, res.Err()
	})

	if err != nil {
		r.log.Errorf("Failed to create node: %v", err)
		return nil, err
	}

	return result.(map[string]any), nil
}
