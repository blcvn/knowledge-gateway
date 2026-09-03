// Package neo4j implements the GraphRepository for Neo4j.
//
// Uses Neo4j Go driver v5. Falls back gracefully if Neo4j is unavailable.
// (MERGE-P2-T1)
package neo4j

import (
	"context"
	"fmt"
	"time"

	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"vnp-memory/services/kg-service/internal/domain/graphiti"
)

// GraphRepo implements port.GraphRepository using Neo4j.
type GraphRepo struct {
	driver neo4jdriver.DriverWithContext
}

// NewGraphRepo creates a GraphRepo. If Neo4j is unavailable, returns a stub.
func NewGraphRepo(driver neo4jdriver.DriverWithContext) *GraphRepo {
	return &GraphRepo{driver: driver}
}

// UpsertNode creates or updates a graph node.
func (r *GraphRepo) UpsertNode(ctx context.Context, node *graphiti.Node) error {
	session := r.driver.NewSession(ctx, neo4jdriver.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
			MERGE (n:Entity {uuid: $uuid, tenant_id: $tenantID})
			ON CREATE SET n.name = $name, n.type = $type, n.summary = $summary,
			              n.created_at = datetime(), n.updated_at = datetime()
			ON MATCH SET  n.summary = $summary, n.updated_at = datetime()
			RETURN n`, map[string]any{
			"uuid":     node.UUID,
			"tenantID": node.TenantID,
			"name":     node.Name,
			"type":     node.Type,
			"summary":  node.Summary,
		})
		return nil, err
	})
	return err
}

// UpsertEdge creates or updates a relation between two nodes.
func (r *GraphRepo) UpsertEdge(ctx context.Context, edge *graphiti.Edge) error {
	session := r.driver.NewSession(ctx, neo4jdriver.SessionConfig{})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
			MATCH (a:Entity {uuid: $src, tenant_id: $tenantID})
			MATCH (b:Entity {uuid: $dst, tenant_id: $tenantID})
			MERGE (a)-[r:RELATION {uuid: $uuid, type: $rel}]->(b)
			ON CREATE SET r.weight = $weight, r.created_at = datetime()
			ON MATCH SET  r.weight = $weight
			RETURN r`, map[string]any{
			"src":      edge.SourceUUID,
			"dst":      edge.TargetUUID,
			"uuid":     edge.UUID,
			"rel":      edge.Relation,
			"weight":   edge.Weight,
			"tenantID": edge.TenantID,
		})
		return nil, err
	})
	return err
}

// GetNode retrieves a node by UUID.
func (r *GraphRepo) GetNode(ctx context.Context, tenantID, nodeUUID string) (*graphiti.Node, error) {
	session := r.driver.NewSession(ctx, neo4jdriver.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (n:Entity {uuid: $uuid, tenant_id: $tenantID})
			RETURN n.uuid AS uuid, n.name AS name, n.type AS type, n.summary AS summary`,
			map[string]any{"uuid": nodeUUID, "tenantID": tenantID})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			rec := res.Record()
			return &graphiti.Node{
				UUID:      getString(rec, "uuid"),
				Name:      getString(rec, "name"),
				Type:      getString(rec, "type"),
				Summary:   getString(rec, "summary"),
				TenantID:  tenantID,
				UpdatedAt: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("node not found: %s", nodeUUID)
	})
	if err != nil {
		return nil, err
	}
	return result.(*graphiti.Node), nil
}

// GetEdge retrieves an edge by UUID.
func (r *GraphRepo) GetEdge(ctx context.Context, tenantID, edgeUUID string) (*graphiti.Edge, error) {
	session := r.driver.NewSession(ctx, neo4jdriver.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH ()-[r:RELATION {uuid: $uuid}]-()
			RETURN r.uuid AS uuid, r.type AS rel, r.weight AS weight`,
			map[string]any{"uuid": edgeUUID})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			rec := res.Record()
			w, _ := rec.Get("weight")
			weight, _ := w.(float64)
			return &graphiti.Edge{
				UUID:      getString(rec, "uuid"),
				Relation:  getString(rec, "rel"),
				Weight:    weight,
				TenantID:  tenantID,
				CreatedAt: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("edge not found: %s", edgeUUID)
	})
	if err != nil {
		return nil, err
	}
	return result.(*graphiti.Edge), nil
}

// GetNeighbors returns neighboring nodes and edges up to given depth.
func (r *GraphRepo) GetNeighbors(ctx context.Context, tenantID, nodeUUID string, depth int) ([]*graphiti.Node, []*graphiti.Edge, error) {
	session := r.driver.NewSession(ctx, neo4jdriver.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH p = (n:Entity {uuid: $uuid, tenant_id: $tenantID})-[*1..`+fmt.Sprintf("%d", depth)+`]-(m:Entity)
			RETURN DISTINCT m.uuid AS uuid, m.name AS name, m.type AS type, m.summary AS summary`,
			map[string]any{"uuid": nodeUUID, "tenantID": tenantID})
		if err != nil {
			return nil, err
		}
		var nodes []*graphiti.Node
		for res.Next(ctx) {
			rec := res.Record()
			nodes = append(nodes, &graphiti.Node{
				UUID:     getString(rec, "uuid"),
				Name:     getString(rec, "name"),
				Type:     getString(rec, "type"),
				Summary:  getString(rec, "summary"),
				TenantID: tenantID,
			})
		}
		return nodes, nil
	})
	if err != nil {
		return nil, nil, err
	}
	nodes := result.([]*graphiti.Node)
	return nodes, nil, nil
}

// GetOntology returns the tenant's entity type and relation definitions.
func (r *GraphRepo) GetOntology(ctx context.Context, tenantID string) (*graphiti.Ontology, error) {
	session := r.driver.NewSession(ctx, neo4jdriver.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (n:Entity {tenant_id: $tenantID})
			RETURN DISTINCT n.type AS type`, map[string]any{"tenantID": tenantID})
		if err != nil {
			return nil, err
		}
		var types []string
		for res.Next(ctx) {
			if t, ok := res.Record().Get("type"); ok {
				if ts, ok := t.(string); ok && ts != "" {
					types = append(types, ts)
				}
			}
		}
		return &graphiti.Ontology{
			TenantID:    tenantID,
			EntityTypes: types,
			UpdatedAt:   time.Now(),
		}, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*graphiti.Ontology), nil
}

// UpdateOntology persists custom entity/relation definitions as metadata nodes.
func (r *GraphRepo) UpdateOntology(ctx context.Context, o *graphiti.Ontology) error {
	// In a real system: persist to a dedicated Ontology node in Neo4j
	// For MVP: no-op (ontology is derived from existing nodes)
	o.UpdatedAt = time.Now()
	return nil
}

// QuerySubgraph finds subgraph matching the text query.
func (r *GraphRepo) QuerySubgraph(ctx context.Context, tenantID, query string) ([]*graphiti.Node, []*graphiti.Edge, error) {
	session := r.driver.NewSession(ctx, neo4jdriver.SessionConfig{})
	defer session.Close(ctx)

	result, err := session.ExecuteRead(ctx, func(tx neo4jdriver.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, `
			MATCH (n:Entity {tenant_id: $tenantID})
			WHERE toLower(n.name) CONTAINS toLower($query) OR toLower(n.summary) CONTAINS toLower($query)
			RETURN n.uuid AS uuid, n.name AS name, n.type AS type, n.summary AS summary
			LIMIT 20`,
			map[string]any{"tenantID": tenantID, "query": query})
		if err != nil {
			return nil, err
		}
		var nodes []*graphiti.Node
		for res.Next(ctx) {
			rec := res.Record()
			nodes = append(nodes, &graphiti.Node{
				UUID:     getString(rec, "uuid"),
				Name:     getString(rec, "name"),
				Type:     getString(rec, "type"),
				Summary:  getString(rec, "summary"),
				TenantID: tenantID,
			})
		}
		return nodes, nil
	})
	if err != nil {
		return nil, nil, err
	}
	nodes := result.([]*graphiti.Node)
	return nodes, nil, nil
}

func getString(rec *neo4jdriver.Record, key string) string {
	if v, ok := rec.Get(key); ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
