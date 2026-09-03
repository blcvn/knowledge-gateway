package neo4j

import (
	"context"
	"fmt"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

type episodeNodeRepo struct{ driver *Neo4jDriver }

func (r *episodeNodeRepo) Save(ctx context.Context, node graph.EpisodicNode, tx port.Transaction) error {
	cypher := `
		MERGE (ep:Episodic {uuid: $uuid})
		SET ep.name               = $name,
		    ep.content            = $content,
		    ep.source             = $source,
		    ep.source_description = $source_description,
		    ep.valid_at           = $valid_at,
		    ep.group_id           = $group_id,
		    ep.updated_at         = datetime()
		ON CREATE SET ep.created_at = datetime()
	`
	params := map[string]any{
		"uuid":               node.UUID,
		"name":               node.Name,
		"content":            node.Content,
		"source":             string(node.Source),
		"source_description": node.SourceDescription,
		"valid_at":           node.ValidAt,
		"group_id":           node.GroupID,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *episodeNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EpisodicNode, error) {
	records, err := r.driver.ExecuteQuery(ctx,
		`MATCH (ep:Episodic {uuid: $uuid}) RETURN ep`,
		map[string]any{"uuid": uuid},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return mapRecordToEpisodicNode(records[0])
}

// RetrieveEpisodes returns the N most recent episodes for given group_ids
func (r *episodeNodeRepo) RetrieveEpisodes(ctx context.Context, req port.RetrieveEpisodesReq) ([]*graph.EpisodicNode, error) {
	cypher := `
		MATCH (ep:Episodic)
		WHERE ep.group_id IN $group_ids
	`
	params := map[string]any{"group_ids": req.GroupIDs, "limit": req.LastN}

	if req.Source != nil {
		cypher += " AND ep.source = $source"
		params["source"] = string(*req.Source)
	}
	if req.SagaID != "" {
		cypher += " AND EXISTS((s:Saga {uuid: $saga_id})-[:HAS_EPISODE]->(ep))"
		params["saga_id"] = req.SagaID
	}

	cypher += " RETURN ep ORDER BY ep.valid_at DESC LIMIT $limit"

	records, err := r.driver.ExecuteQuery(ctx, cypher, params)
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EpisodicNode, 0, len(records))
	for _, rec := range records {
		n, err := mapRecordToEpisodicNode(rec)
		if err == nil && n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (r *episodeNodeRepo) GetByEntityNodeUUID(ctx context.Context, entityNodeUUID string) ([]*graph.EpisodicNode, error) {
	cypher := `
		MATCH (ep:Episodic)-[:MENTIONS]->(entity:Entity {uuid: $uuid})
		RETURN ep ORDER BY ep.valid_at DESC
	`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": entityNodeUUID})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EpisodicNode, 0)
	for _, rec := range records {
		n, err := mapRecordToEpisodicNode(rec)
		if err == nil && n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (r *episodeNodeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
	cypher := `MATCH (ep:Episodic {uuid: $uuid}) DETACH DELETE ep`
	params := map[string]any{"uuid": uuid}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *episodeNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction, batchSize int) error {
	for {
		cypher := fmt.Sprintf(`
			MATCH (ep:Episodic {group_id: $group_id})
			WITH ep LIMIT %d DETACH DELETE ep
			RETURN count(ep) as deleted
		`, batchSize)
		records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"group_id": groupID})
		if err != nil {
			return err
		}
		if len(records) == 0 {
			break
		}
		deleted, _ := records[0].Values[0].(int64)
		if deleted == 0 {
			break
		}
	}
	return nil
}
