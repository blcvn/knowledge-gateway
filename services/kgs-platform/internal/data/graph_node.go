package data

import (
	"context"

	"kgs-platform/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
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
			MERGE (n:` + label + ` {app_id: $app_id, id: $node_id})
			SET n += $props
			RETURN n
		`

		// Ensure props contains a unique id — auto-generate UUID if not provided.
		nodeID, ok := properties["id"].(string)
		if !ok || nodeID == "" || nodeID == "unknown_id" {
			nodeID = uuid.NewString()
			properties["id"] = nodeID // write-back so props is consistent
		}

		params := map[string]interface{}{
			"app_id":  appID,
			"node_id": nodeID,
			"props":   properties,
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

// UpdateNode updates an existing node's properties
func (r *graphRepo) UpdateNode(ctx context.Context, appID string, nodeID string, properties map[string]any) (map[string]any, error) {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {app_id: $app_id, id: $node_id})
			SET n += $props
			RETURN n
		`
		params := map[string]interface{}{
			"app_id":  appID,
			"node_id": nodeID,
			"props":   properties,
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
		r.log.Errorf("Failed to update node: %v", err)
		return nil, err
	}

	return result.(map[string]any), nil
}

// DeleteNode removes a node and all its connected relationships
func (r *graphRepo) DeleteNode(ctx context.Context, appID string, nodeID string) error {
	session := r.data.neo4j.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (n {app_id: $app_id, id: $node_id})
			DETACH DELETE n
		`
		params := map[string]interface{}{
			"app_id":  appID,
			"node_id": nodeID,
		}

		return tx.Run(ctx, query, params)
	})

	if err != nil {
		r.log.Errorf("Failed to delete node: %v", err)
		return err
	}

	return nil
}
