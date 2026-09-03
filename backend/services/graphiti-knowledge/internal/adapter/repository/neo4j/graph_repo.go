package neo4j

import (
	"context"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	pb "vnp-memory/services/graphiti-knowledge/adapter/grpc/pb"
	"vnp-memory/shared/pkg/tenant"
)

// GraphRepositoryImpl implements the GraphRepository interface using Neo4j.
type GraphRepositoryImpl struct {
	driver neo4j.DriverWithContext
}

// NewGraphRepository creates a new Neo4j graph repository.
func NewGraphRepository(uri, username, password string) (*GraphRepositoryImpl, error) {
	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	// Verify connection
	err = driver.VerifyConnectivity(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to neo4j: %w", err)
	}

	return &GraphRepositoryImpl{
		driver: driver,
	}, nil
}

// SearchSimilarEntities searches for nodes in Neo4j that might match the extracted entity's name.
func (r *GraphRepositoryImpl) SearchSimilarEntities(ctx context.Context, groupID string, entityName string, topK int) ([]*pb.ExtractedEntity, error) {
	tenantID, err := tenant.FromContext(ctx)
	if err != nil {
		return nil, err
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	cypher := `
		MATCH (e:Entity)
		WHERE e.tenant_id = $tenantID AND e.group_id = $groupID AND toLower(e.name) CONTAINS toLower($entityName)
		RETURN e.name AS name, e.label AS label, e.summary AS summary
		LIMIT $topK
	`

	result, err := session.Run(ctx, cypher, map[string]interface{}{
		"tenantID":   tenantID,
		"groupID":    groupID,
		"entityName": entityName,
		"topK":       topK,
	})
	if err != nil {
		return nil, err
	}

	var candidates []*pb.ExtractedEntity
	for result.Next(ctx) {
		record := result.Record()
		name, _ := record.Get("name")
		label, _ := record.Get("label")
		summary, _ := record.Get("summary")

		candidates = append(candidates, &pb.ExtractedEntity{
			Name:    name.(string),
			Label:   label.(string),
			Summary: summary.(string),
		})
	}

	if err = result.Err(); err != nil {
		return nil, err
	}

	return candidates, nil
}

// UpsertEntity creates a new node or updates an existing one based on its group_id and name.
func (r *GraphRepositoryImpl) UpsertEntity(ctx context.Context, groupID string, entity *pb.ExtractedEntity) error {
	tenantID, err := tenant.FromContext(ctx)
	if err != nil {
		return err
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	cypher := `
		MERGE (e:Entity {tenant_id: $tenantID, group_id: $groupID, name: $name})
		SET e.label = $label,
		    e.summary = $summary,
		    e.updated_at = timestamp()
	`

	_, err = session.Run(ctx, cypher, map[string]interface{}{
		"tenantID": tenantID,
		"groupID":  groupID,
		"name":     entity.Name,
		"label":    entity.Label,
		"summary":  entity.Summary,
	})

	return err
}

// UpsertEdge creates a directed relationship between two entities.
func (r *GraphRepositoryImpl) UpsertEdge(ctx context.Context, groupID string, edge *pb.ExtractedEdge) error {
	tenantID, err := tenant.FromContext(ctx)
	if err != nil {
		return err
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	cypher := `
		MATCH (source:Entity {tenant_id: $tenantID, group_id: $groupID, name: $sourceName})
		MATCH (target:Entity {tenant_id: $tenantID, group_id: $groupID, name: $targetName})
		MERGE (source)-[r:RELATED_TO {type: $relationType}]->(target)
		SET r.fact = $fact,
		    r.temporal = $temporal,
		    r.updated_at = timestamp()
	`

	_, err = session.Run(ctx, cypher, map[string]interface{}{
		"tenantID":     tenantID,
		"groupID":      groupID,
		"sourceName":   edge.Source,
		"targetName":   edge.Target,
		"relationType": edge.Relation,
		"fact":         edge.Fact,
		"temporal":     edge.Temporal,
	})

	return err
}

// Close terminates the driver connection pool.
func (r *GraphRepositoryImpl) Close(ctx context.Context) error {
	log.Println("Closing Neo4j Driver Connection...")
	return r.driver.Close(ctx)
}
