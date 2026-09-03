package neo4j

import (
	"context"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

// ─── Community Node Repo ──────────────────────────────────────────────────────

type communityNodeRepo struct{ driver *Neo4jDriver }

func (r *communityNodeRepo) Save(ctx context.Context, node graph.CommunityNode, tx port.Transaction) error {
	cypher := `
		MERGE (c:Community {uuid: $uuid})
		SET c.name           = $name,
		    c.summary        = $summary,
		    c.name_embedding = $name_embedding,
		    c.group_id       = $group_id,
		    c.updated_at     = datetime()
		ON CREATE SET c.created_at = datetime()
	`
	params := map[string]any{
		"uuid":           node.UUID,
		"name":           node.Name,
		"summary":        node.Summary,
		"name_embedding": node.NameEmbedding,
		"group_id":       node.GroupID,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *communityNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.CommunityNode, error) {
	records, err := r.driver.ExecuteQuery(ctx,
		`MATCH (c:Community {uuid: $uuid}) RETURN c`,
		map[string]any{"uuid": uuid},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return mapRecordToCommunityNode(records[0])
}

func (r *communityNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction) error {
	cypher := `MATCH (c:Community {group_id: $group_id}) DETACH DELETE c`
	params := map[string]any{"group_id": groupID}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}
