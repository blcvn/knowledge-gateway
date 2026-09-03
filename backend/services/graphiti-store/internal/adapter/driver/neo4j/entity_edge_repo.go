package neo4j

import (
	"context"
	"fmt"
	"time"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

type entityEdgeRepo struct{ driver *Neo4jDriver }

func (r *entityEdgeRepo) Save(ctx context.Context, edge graph.EntityEdge, tx port.Transaction) error {
	cypher := `
		MATCH (src:Entity {uuid: $src_uuid}), (tgt:Entity {uuid: $tgt_uuid})
		MERGE (src)-[e:RELATES_TO {uuid: $uuid}]->(tgt)
		SET e.name           = $name,
		    e.fact           = $fact,
		    e.fact_embedding = $fact_embedding,
		    e.episodes       = $episodes,
		    e.group_id       = $group_id,
		    e.valid_at       = $valid_at,
		    e.invalid_at     = null,
		    e.expired_at     = null,
		    e.updated_at     = datetime()
		ON CREATE SET e.created_at = datetime()
	`
	params := map[string]any{
		"uuid":           edge.UUID,
		"src_uuid":       edge.SourceNodeUUID,
		"tgt_uuid":       edge.TargetNodeUUID,
		"name":           edge.Name,
		"fact":           edge.Fact,
		"fact_embedding": edge.FactEmbedding,
		"episodes":       edge.Episodes,
		"group_id":       edge.GroupID,
		"valid_at":       edge.ValidAt,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *entityEdgeRepo) SaveBulk(ctx context.Context, edges []graph.EntityEdge, tx port.Transaction, batchSize int) error {
	for i := 0; i < len(edges); i += batchSize {
		end := i + batchSize
		if end > len(edges) {
			end = len(edges)
		}
		for _, edge := range edges[i:end] {
			if err := r.Save(ctx, edge, tx); err != nil {
				return fmt.Errorf("save entity edge %s: %w", edge.UUID, err)
			}
		}
	}
	return nil
}

// Invalidate marks an EntityEdge as temporally invalid.
// NEVER deletes — preserves historical data for point-in-time queries.
func (r *entityEdgeRepo) Invalidate(ctx context.Context, uuid string, invalidAt time.Time, tx port.Transaction) error {
	cypher := `
		MATCH ()-[e:RELATES_TO {uuid: $uuid}]->()
		SET e.invalid_at = $invalid_at,
		    e.expired_at = datetime(),
		    e.updated_at = datetime()
	`
	params := map[string]any{
		"uuid":       uuid,
		"invalid_at": invalidAt,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *entityEdgeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EntityEdge, error) {
	cypher := `
		MATCH (src)-[e:RELATES_TO {uuid: $uuid}]->(tgt)
		RETURN e, src.uuid as src_uuid, tgt.uuid as tgt_uuid
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": uuid})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return mapRecordToEntityEdge(records[0])
}

// GetBetweenNodes returns ALL edges including invalidated (caller filters temporal)
func (r *entityEdgeRepo) GetBetweenNodes(ctx context.Context, srcUUID, tgtUUID string) ([]*graph.EntityEdge, error) {
	cypher := `
		MATCH (src:Entity {uuid: $src_uuid})-[e:RELATES_TO]->(tgt:Entity {uuid: $tgt_uuid})
		RETURN e, src.uuid as src_uuid, tgt.uuid as tgt_uuid
		ORDER BY e.created_at DESC
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{
		"src_uuid": srcUUID, "tgt_uuid": tgtUUID,
	})
	if err != nil {
		return nil, err
	}
	edges := make([]*graph.EntityEdge, 0, len(records))
	for _, rec := range records {
		e, err := mapRecordToEntityEdge(rec)
		if err == nil && e != nil {
			edges = append(edges, e)
		}
	}
	return edges, nil
}

func (r *entityEdgeRepo) GetByNodeUUID(ctx context.Context, nodeUUID string) ([]*graph.EntityEdge, error) {
	cypher := `
		MATCH (n:Entity {uuid: $uuid})-[e:RELATES_TO]-(other:Entity)
		RETURN e,
		       CASE WHEN startNode(e) = n THEN n.uuid ELSE other.uuid END as src_uuid,
		       CASE WHEN endNode(e) = n THEN n.uuid ELSE other.uuid END as tgt_uuid
		ORDER BY e.created_at DESC
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": nodeUUID})
	if err != nil {
		return nil, err
	}
	edges := make([]*graph.EntityEdge, 0)
	for _, rec := range records {
		e, err := mapRecordToEntityEdge(rec)
		if err == nil && e != nil {
			edges = append(edges, e)
		}
	}
	return edges, nil
}

func (r *entityEdgeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
	cypher := `MATCH ()-[e:RELATES_TO {uuid: $uuid}]->() DELETE e`
	params := map[string]any{"uuid": uuid}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}
