package neo4j

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

type sagaNodeRepo struct{ driver *Neo4jDriver }

func (r *sagaNodeRepo) Save(ctx context.Context, node graph.SagaNode, tx port.Transaction) error {
	cypher := `
		MERGE (s:Saga {uuid: $uuid})
		SET s.name                = $name,
		    s.group_id            = $group_id,
		    s.summary             = $summary,
		    s.first_episode_uuid  = $first_episode_uuid,
		    s.last_episode_uuid   = $last_episode_uuid,
		    s.last_summarized_at  = $last_summarized_at,
		    s.updated_at          = datetime()
		ON CREATE SET s.created_at = datetime()
	`
	params := map[string]any{
		"uuid":               node.UUID,
		"name":               node.Name,
		"group_id":           node.GroupID,
		"summary":            node.Summary,
		"first_episode_uuid": node.FirstEpisodeUUID,
		"last_episode_uuid":  node.LastEpisodeUUID,
		"last_summarized_at": node.LastSummarizedAt,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *sagaNodeRepo) GetByUUID(ctx context.Context, uuid, groupID string) (*graph.SagaNode, error) {
	records, err := r.driver.ExecuteQuery(ctx,
		`MATCH (s:Saga {uuid: $uuid, group_id: $group_id}) RETURN s`,
		map[string]any{"uuid": uuid, "group_id": groupID},
	)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return mapRecordToSagaNode(records[0])
}

func (r *sagaNodeRepo) GetByGroupID(ctx context.Context, groupID string) ([]*graph.SagaNode, error) {
	records, err := r.driver.ExecuteQuery(ctx,
		`MATCH (s:Saga {group_id: $group_id}) RETURN s ORDER BY s.created_at DESC`,
		map[string]any{"group_id": groupID},
	)
	if err != nil {
		return nil, err
	}
	sagas := make([]*graph.SagaNode, 0, len(records))
	for _, rec := range records {
		s, err := mapRecordToSagaNode(rec)
		if err == nil && s != nil {
			sagas = append(sagas, s)
		}
	}
	return sagas, nil
}

func mapRecordToSagaNode(rec port.Record) (*graph.SagaNode, error) {
	nodeVal, ok := rec.Values[0].(neo4j.Node)
	if !ok {
		return nil, nil
	}
	props := nodeVal.Props
	node := &graph.SagaNode{
		UUID:             getString(props, "uuid"),
		Name:             getString(props, "name"),
		GroupID:          getString(props, "group_id"),
		Summary:          getString(props, "summary"),
		FirstEpisodeUUID: getString(props, "first_episode_uuid"),
		LastEpisodeUUID:  getString(props, "last_episode_uuid"),
		CreatedAt:        parseTime(props["created_at"]),
		UpdatedAt:        parseTime(props["updated_at"]),
	}
	if v := parseTimePtr(props["last_summarized_at"]); v != nil {
		node.LastSummarizedAt = v
	}
	return node, nil
}
