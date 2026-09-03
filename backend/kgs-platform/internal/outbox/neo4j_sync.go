package outbox

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Neo4jSyncer struct {
	driver neo4j.DriverWithContext
	log    *log.Helper
}

func NewNeo4jSyncer(driver neo4j.DriverWithContext, logger log.Logger) *Neo4jSyncer {
	return &Neo4jSyncer{driver: driver, log: log.NewHelper(logger)}
}

func (s *Neo4jSyncer) UpsertEntity(ctx context.Context, entity data.KGEntity) error {
	if s == nil || s.driver == nil {
		return nil
	}
	started := time.Now()
	s.log.Infof("[KGS][Neo4jSyncer] UpsertEntity start app_id=%s tenant_id=%s entity_id=%s entity_type=%s version=%d",
		entity.AppID, entity.TenantID, entity.EntityID, entity.EntityType, entity.Version)
	label, err := data.SanitizeCypherIdentifier(defaultEntityType(entity.EntityType))
	if err != nil {
		s.log.Errorf("[KGS][Neo4jSyncer] UpsertEntity sanitize failed app_id=%s tenant_id=%s entity_id=%s err=%v",
			entity.AppID, entity.TenantID, entity.EntityID, err)
		return err
	}

	query := data.BuildCreateNodeQuery(label)
	normalizedProps, normalizedCount := data.NormalizePropertiesForNeo4j(entity.Properties)
	if normalizedCount > 0 {
		s.log.Warnf("[KGS][Neo4jSyncer] UpsertEntity normalized properties app_id=%s tenant_id=%s entity_id=%s converted=%d",
			entity.AppID, entity.TenantID, entity.EntityID, normalizedCount)
	}
	props := map[string]any(normalizedProps)
	props["id"] = entity.EntityID
	props["name"] = entity.Name
	props["entity_type"] = entity.EntityType
	props["app_id"] = entity.AppID
	props["tenant_id"] = entity.TenantID
	props["version"] = entity.Version
	props["isDeleted"] = entity.IsDeleted
	props["provenance_type"] = entity.ProvenanceType

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"app_id":     entity.AppID,
			"tenant_id":  entity.TenantID,
			"node_id":    entity.EntityID,
			"unique_key": buildUniqueKey(entity.AppID, entity.TenantID, entity.EntityID),
			"props":      props,
		})
		return nil, err
	})
	if err != nil {
		s.log.Errorf("[KGS][Neo4jSyncer] UpsertEntity failed app_id=%s tenant_id=%s entity_id=%s err=%v",
			entity.AppID, entity.TenantID, entity.EntityID, err)
		return err
	}
	s.log.Infof("[KGS][Neo4jSyncer] UpsertEntity done app_id=%s tenant_id=%s entity_id=%s duration=%s",
		entity.AppID, entity.TenantID, entity.EntityID, time.Since(started))
	return err
}

func (s *Neo4jSyncer) UpsertEdge(ctx context.Context, edge data.KGEdge) error {
	if s == nil || s.driver == nil {
		return nil
	}
	started := time.Now()
	s.log.Infof("[KGS][Neo4jSyncer] UpsertEdge start app_id=%s tenant_id=%s edge_id=%s relation=%s from=%s to=%s",
		edge.AppID, edge.TenantID, edge.EdgeID, edge.RelationType, edge.FromEntityID, edge.ToEntityID)
	relationType, err := data.SanitizeCypherIdentifier(defaultRelationType(edge.RelationType))
	if err != nil {
		s.log.Errorf("[KGS][Neo4jSyncer] UpsertEdge sanitize failed app_id=%s tenant_id=%s edge_id=%s err=%v",
			edge.AppID, edge.TenantID, edge.EdgeID, err)
		return err
	}
	query := data.BuildCreateEdgeQuery(relationType)
	normalizedProps, normalizedCount := data.NormalizePropertiesForNeo4j(edge.Properties)
	if normalizedCount > 0 {
		s.log.Warnf("[KGS][Neo4jSyncer] UpsertEdge normalized properties app_id=%s tenant_id=%s edge_id=%s converted=%d",
			edge.AppID, edge.TenantID, edge.EdgeID, normalizedCount)
	}
	props := map[string]any(normalizedProps)
	props["id"] = edge.EdgeID
	props["relation_type"] = edge.RelationType
	props["confidence"] = edge.Confidence
	props["version_id"] = edge.VersionID

	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err = session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, map[string]any{
			"app_id":         edge.AppID,
			"tenant_id":      edge.TenantID,
			"source_node_id": edge.FromEntityID,
			"target_node_id": edge.ToEntityID,
			"edge_id":        edge.EdgeID,
			"props":          props,
		})
		return nil, err
	})
	if err != nil {
		s.log.Errorf("[KGS][Neo4jSyncer] UpsertEdge failed app_id=%s tenant_id=%s edge_id=%s relation=%s from=%s to=%s err=%v",
			edge.AppID, edge.TenantID, edge.EdgeID, edge.RelationType, edge.FromEntityID, edge.ToEntityID, err)
		return err
	}
	s.log.Infof("[KGS][Neo4jSyncer] UpsertEdge done app_id=%s tenant_id=%s edge_id=%s duration=%s",
		edge.AppID, edge.TenantID, edge.EdgeID, time.Since(started))
	return err
}

func (s *Neo4jSyncer) SoftDeleteEntity(ctx context.Context, entityID, tenantID string) error {
	if s == nil || s.driver == nil {
		return nil
	}
	started := time.Now()
	s.log.Infof("[KGS][Neo4jSyncer] SoftDeleteEntity start tenant_id=%s entity_id=%s", tenantID, entityID)
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
			MATCH (n {tenant_id: $tenant_id, id: $entity_id})
			SET n.isDeleted = true, n.updated_at = datetime()
		`, map[string]any{
			"tenant_id": tenantID,
			"entity_id": entityID,
		})
		return nil, err
	})
	if err != nil {
		s.log.Errorf("[KGS][Neo4jSyncer] SoftDeleteEntity failed tenant_id=%s entity_id=%s err=%v", tenantID, entityID, err)
		return err
	}
	s.log.Infof("[KGS][Neo4jSyncer] SoftDeleteEntity done tenant_id=%s entity_id=%s duration=%s", tenantID, entityID, time.Since(started))
	return err
}

func (s *Neo4jSyncer) SoftDeleteEdge(ctx context.Context, edgeID, tenantID string) error {
	if s == nil || s.driver == nil {
		return nil
	}
	started := time.Now()
	s.log.Infof("[KGS][Neo4jSyncer] SoftDeleteEdge start tenant_id=%s edge_id=%s", tenantID, edgeID)
	session := s.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close(ctx)
	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, `
			MATCH ()-[r {tenant_id: $tenant_id, id: $edge_id}]->()
			DELETE r
		`, map[string]any{
			"tenant_id": tenantID,
			"edge_id":   edgeID,
		})
		return nil, err
	})
	if err != nil {
		s.log.Errorf("[KGS][Neo4jSyncer] SoftDeleteEdge failed tenant_id=%s edge_id=%s err=%v", tenantID, edgeID, err)
		return err
	}
	s.log.Infof("[KGS][Neo4jSyncer] SoftDeleteEdge done tenant_id=%s edge_id=%s duration=%s", tenantID, edgeID, time.Since(started))
	return err
}

func buildUniqueKey(appID, tenantID, id string) string {
	return fmt.Sprintf("%s|%s|%s", strings.TrimSpace(appID), strings.TrimSpace(tenantID), strings.TrimSpace(id))
}

func defaultEntityType(entityType string) string {
	entityType = strings.TrimSpace(entityType)
	if entityType == "" {
		return "Entity"
	}
	return entityType
}

func defaultRelationType(relationType string) string {
	relationType = strings.TrimSpace(relationType)
	if relationType == "" {
		return "RELATES_TO"
	}
	return relationType
}
