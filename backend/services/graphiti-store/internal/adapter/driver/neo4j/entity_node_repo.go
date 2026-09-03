package neo4j

import (
	"context"
	"fmt"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

type entityNodeRepo struct{ driver *Neo4jDriver }

func (r *entityNodeRepo) Save(ctx context.Context, node graph.EntityNode, tx port.Transaction) error {
	cypher := `
		MERGE (n:Entity {uuid: $uuid})
		SET n.name           = $name,
		    n.labels         = $labels,
		    n.summary        = $summary,
		    n.group_id       = $group_id,
		    n.name_embedding = $name_embedding,
		    n.updated_at     = datetime()
		ON CREATE SET n.created_at = datetime()
	`
	params := map[string]any{
		"uuid":           node.UUID,
		"name":           node.Name,
		"labels":         node.Labels,
		"summary":        node.Summary,
		"group_id":       node.GroupID,
		"name_embedding": node.NameEmbedding,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *entityNodeRepo) SaveBulk(ctx context.Context, nodes []graph.EntityNode, tx port.Transaction, batchSize int) error {
	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}
		batch := nodes[i:end]
		for _, node := range batch {
			if err := r.Save(ctx, node, tx); err != nil {
				return fmt.Errorf("save entity node %s: %w", node.UUID, err)
			}
		}
	}
	return nil
}

func (r *entityNodeRepo) GetByUUID(ctx context.Context, uuid string) (*graph.EntityNode, error) {
	cypher := `MATCH (n:Entity {uuid: $uuid}) RETURN n`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuid": uuid})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return mapRecordToEntityNode(records[0])
}

func (r *entityNodeRepo) GetByUUIDs(ctx context.Context, uuids []string) ([]*graph.EntityNode, error) {
	cypher := `MATCH (n:Entity) WHERE n.uuid IN $uuids RETURN n`
	records, err := r.driver.ExecuteQuery(ctx, cypher, map[string]any{"uuids": uuids})
	if err != nil {
		return nil, err
	}
	nodes := make([]*graph.EntityNode, 0, len(records))
	for _, rec := range records {
		n, err := mapRecordToEntityNode(rec)
		if err == nil && n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (r *entityNodeRepo) Delete(ctx context.Context, uuid string, tx port.Transaction) error {
	cypher := `
		MATCH (n:Entity {uuid: $uuid})
		DETACH DELETE n
	`
	params := map[string]any{"uuid": uuid}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *entityNodeRepo) DeleteByGroupID(ctx context.Context, groupID string, tx port.Transaction, batchSize int) error {
	// Batch delete to avoid large transactions
	for {
		cypher := fmt.Sprintf(`
			MATCH (n:Entity {group_id: $group_id})
			WITH n LIMIT %d
			DETACH DELETE n
			RETURN count(n) as deleted
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
