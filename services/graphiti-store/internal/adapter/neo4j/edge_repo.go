package neo4j

import (
	"context"
	"fmt"
	"time"

	n4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

// --- EdgeRepository ---

// SaveEdge creates a RELATES_TO relationship between two Entity nodes.
func (d *Driver) SaveEdge(ctx context.Context, edge domain.EntityEdge) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (source:Entity {uuid: $source_node_id})
		MATCH (target:Entity {uuid: $target_node_id})
		MERGE (source)-[r:RELATES_TO {uuid: $uuid}]->(target)
		SET r.name = $name,
		    r.group_id = $group_id,
		    r.fact = $fact,
		    r.fact_embedding = $fact_embedding,
		    r.valid_at = datetime($valid_at),
		    r.episode_id = $episode_id,
		    r.created_at = coalesce(r.created_at, datetime())
	`
	params := map[string]any{
		"uuid":           edge.UUID,
		"source_node_id": edge.SourceNodeID,
		"target_node_id": edge.TargetNodeID,
		"name":           edge.Name,
		"group_id":       edge.GroupID,
		"fact":           edge.Fact,
		"fact_embedding":  edge.FactEmbedding,
		"valid_at":       edge.ValidAt.Format(time.RFC3339),
		"episode_id":     edge.EpisodeID,
	}

	if edge.InvalidAt != nil {
		cypher += `, r.invalid_at = datetime($invalid_at)`
		params["invalid_at"] = edge.InvalidAt.Format(time.RFC3339)
	}

	_, err := session.ExecuteWrite(ctx, func(tx n4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, cypher, params)
		return nil, err
	})
	if err != nil {
		return fmt.Errorf("neo4j: save edge: %w", err)
	}
	return nil
}

// GetEdge retrieves an EntityEdge by UUID.
func (d *Driver) GetEdge(ctx context.Context, uuid string) (*domain.EntityEdge, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (s)-[r:RELATES_TO {uuid: $uuid}]->(t)
		RETURN r, s.uuid AS source_uuid, t.uuid AS target_uuid
	`
	resultVal, err := session.ExecuteRead(ctx, func(tx n4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, cypher, map[string]any{"uuid": uuid})
		if err != nil {
			return nil, err
		}
		if result.Next(ctx) {
			return d.recordToEntityEdge(result.Record())
		}
		return nil, domain.ErrEdgeNotFound
	})

	if err != nil {
		return nil, fmt.Errorf("neo4j: get edge: %w", err)
	}

	edge, _ := resultVal.(*domain.EntityEdge)
	return edge, nil
}

// DeleteEdge removes an edge.
func (d *Driver) DeleteEdge(ctx context.Context, uuid string) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `MATCH ()-[r:RELATES_TO {uuid: $uuid}]->() DELETE r`
	_, err := session.Run(ctx, cypher, map[string]any{"uuid": uuid})
	if err != nil {
		return fmt.Errorf("neo4j: delete edge: %w", err)
	}
	return nil
}

// InvalidateEdge sets invalid_at on an edge without deleting it.
func (d *Driver) InvalidateEdge(ctx context.Context, uuid string, invalidAt time.Time) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH ()-[r:RELATES_TO {uuid: $uuid}]->()
		SET r.invalid_at = datetime($invalid_at)
	`
	_, err := session.Run(ctx, cypher, map[string]any{
		"uuid":       uuid,
		"invalid_at": invalidAt.Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("neo4j: invalidate edge: %w", err)
	}
	return nil
}

// GetEdgesInTimeRange returns edges whose validity window intersects [from, to].
func (d *Driver) GetEdgesInTimeRange(ctx context.Context, groupID string, from, to time.Time) ([]domain.EntityEdge, error) {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (s)-[r:RELATES_TO {group_id: $group_id}]->(t)
		WHERE r.valid_at <= datetime($to)
		  AND (r.invalid_at IS NULL OR r.invalid_at >= datetime($from))
		RETURN r, s.uuid AS source_uuid, t.uuid AS target_uuid
		ORDER BY r.valid_at DESC
	`
	result, err := session.Run(ctx, cypher, map[string]any{
		"group_id": groupID,
		"from":     from.Format(time.RFC3339),
		"to":       to.Format(time.RFC3339),
	})
	if err != nil {
		return nil, fmt.Errorf("neo4j: edges in time range: %w", err)
	}

	var edges []domain.EntityEdge
	for result.Next(ctx) {
		edge, err := d.recordToEntityEdge(result.Record())
		if err != nil {
			d.logger.Warn("skip edge", "error", err)
			continue
		}
		edges = append(edges, *edge)
	}
	return edges, nil
}

// SaveEpisodicEdge creates a MENTIONS relationship.
func (d *Driver) SaveEpisodicEdge(ctx context.Context, edge domain.EpisodicEdge) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	cypher := `
		MATCH (ep:Episodic {uuid: $episode_id})
		MATCH (en:Entity {uuid: $entity_id})
		MERGE (ep)-[r:MENTIONS {uuid: $uuid}]->(en)
		SET r.group_id = $group_id,
		    r.created_at = coalesce(r.created_at, datetime())
	`
	_, err := session.Run(ctx, cypher, map[string]any{
		"uuid":       edge.UUID,
		"episode_id": edge.EpisodeID,
		"entity_id":  edge.EntityID,
		"group_id":   edge.GroupID,
	})
	if err != nil {
		return fmt.Errorf("neo4j: save episodic edge: %w", err)
	}
	return nil
}

// recordToEntityEdge maps a Neo4j record to a domain.EntityEdge.
func (d *Driver) recordToEntityEdge(record *n4j.Record) (*domain.EntityEdge, error) {
	relVal, ok := record.Get("r")
	if !ok {
		return nil, fmt.Errorf("neo4j: missing 'r' in record")
	}

	rel, ok := relVal.(n4j.Relationship)
	if !ok {
		return nil, fmt.Errorf("neo4j: unexpected type for rel: %T", relVal)
	}

	props := rel.Props
	edge := &domain.EntityEdge{
		UUID:         getStringProp(props, "uuid"),
		Name:         getStringProp(props, "name"),
		GroupID:      getStringProp(props, "group_id"),
		Fact:         getStringProp(props, "fact"),
		EpisodeID:    getStringProp(props, "episode_id"),
	}

	if src, ok := record.Get("source_uuid"); ok {
		if s, ok := src.(string); ok {
			edge.SourceNodeID = s
		}
	}
	if tgt, ok := record.Get("target_uuid"); ok {
		if s, ok := tgt.(string); ok {
			edge.TargetNodeID = s
		}
	}

	if v, ok := props["valid_at"]; ok {
		if t, ok := v.(time.Time); ok {
			edge.ValidAt = t
		}
	}
	if v, ok := props["invalid_at"]; ok {
		if t, ok := v.(time.Time); ok {
			edge.InvalidAt = &t
		}
	}
	if v, ok := props["expired_at"]; ok {
		if t, ok := v.(time.Time); ok {
			edge.ExpiredAt = &t
		}
	}

	return edge, nil
}
