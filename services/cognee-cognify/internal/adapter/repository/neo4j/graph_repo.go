package neo4j

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/domain"
	"github.com/vnp-community/vnp-memory/services/cognee-cognify/internal/usecase/port"
)

type graphRepository struct {
	driver neo4j.DriverWithContext
}

// NewGraphRepository creates a new Neo4j graph repository
func NewGraphRepository(driver neo4j.DriverWithContext) port.GraphRepository {
	return &graphRepository{driver: driver}
}

func (r *graphRepository) UpsertEntity(ctx context.Context, tenantID string, entity *domain.Entity) (string, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := fmt.Sprintf(upsertEntityQuery, tenantID)
	params := map[string]any{
		"id":          entity.ID.String(),
		"name":        entity.Name,
		"type":        string(entity.EntityType),
		"description": entity.Description,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})
	if err != nil {
		return "", fmt.Errorf("upsert entity %s: %w", entity.Name, err)
	}
	return entity.ID.String(), nil
}

func (r *graphRepository) UpsertRelationship(ctx context.Context, tenantID string, rel *domain.Relationship) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	query := fmt.Sprintf(upsertRelationshipQuery, tenantID, tenantID)
	params := map[string]any{
		"source_id": rel.SourceID.String(),
		"target_id": rel.TargetID.String(),
		"relation":  rel.Relation,
		"weight":    rel.Weight,
		"chunk_id":  rel.SourceChunk.String(),
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})
	if err != nil {
		return fmt.Errorf("upsert relationship %s: %w", rel.Relation, err)
	}
	return nil
}

func (r *graphRepository) UpsertCommunity(ctx context.Context, tenantID string, community *domain.Community) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)

	entityIDs := make([]string, len(community.EntityIDs))
	for i, id := range community.EntityIDs {
		entityIDs[i] = id.String()
	}

	query := fmt.Sprintf(upsertCommunityQuery, tenantID)
	params := map[string]any{
		"id":         community.ID.String(),
		"summary":    community.Summary,
		"level":      community.Level,
		"entity_ids": entityIDs,
	}

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		return result.Consume(ctx)
	})
	return err
}

func (r *graphRepository) GetEntityByName(ctx context.Context, tenantID, name string) (*domain.Entity, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	query := fmt.Sprintf(getEntityByNameQuery, tenantID)
	params := map[string]any{"name": name}

	res, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		if !result.Next(ctx) {
			return nil, nil // Not found
		}
		record := result.Record()
		return record, nil
	})
	if err != nil {
		return nil, err
	}
	if res == nil {
		return nil, nil
	}

	record := res.(*neo4j.Record)
	
	idStr, _ := record.Get("id")
	nameStr, _ := record.Get("name")
	typeStr, _ := record.Get("type")
	descStr, _ := record.Get("description")
	
	entityID, _ := uuid.Parse(idStr.(string))

	entity := &domain.Entity{
		ID:          entityID,
		Name:        nameStr.(string),
		EntityType:  domain.EntityType(typeStr.(string)),
		Description: descStr.(string),
	}
	return entity, nil
}
