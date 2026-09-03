package neo4j

import (
	"context"

	"vnp-memory/pkg/graph"
	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

// ─── Episodic Edges (MENTIONS: episode → entity) ─────────────────────────────

type episodicEdgeRepo struct{ driver *Neo4jDriver }

func (r *episodicEdgeRepo) Save(ctx context.Context, edge graph.EpisodicEdge, tx port.Transaction) error {
	cypher := `
		MATCH (ep:Episodic {uuid: $src}), (entity:Entity {uuid: $tgt})
		MERGE (ep)-[e:MENTIONS {uuid: $uuid}]->(entity)
		ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
	`
	params := map[string]any{
		"uuid": edge.UUID, "src": edge.SourceUUID,
		"tgt": edge.TargetUUID, "group_id": edge.GroupID,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *episodicEdgeRepo) SaveBulk(ctx context.Context, edges []graph.EpisodicEdge, tx port.Transaction) error {
	for _, e := range edges {
		if err := r.Save(ctx, e, tx); err != nil {
			return err
		}
	}
	return nil
}

func (r *episodicEdgeRepo) DeleteByEpisodeUUID(ctx context.Context, episodeUUID string, tx port.Transaction) error {
	cypher := `MATCH (ep:Episodic {uuid: $uuid})-[e:MENTIONS]->() DELETE e`
	params := map[string]any{"uuid": episodeUUID}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

// ─── Community Edges (HAS_MEMBER: community → entity) ────────────────────────

type communityEdgeRepo struct{ driver *Neo4jDriver }

func (r *communityEdgeRepo) Save(ctx context.Context, edge graph.CommunityEdge, tx port.Transaction) error {
	cypher := `
		MATCH (c:Community {uuid: $src}), (n:Entity {uuid: $tgt})
		MERGE (c)-[e:HAS_MEMBER {uuid: $uuid}]->(n)
		ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
	`
	params := map[string]any{
		"uuid": edge.UUID, "src": edge.SourceUUID,
		"tgt": edge.TargetUUID, "group_id": edge.GroupID,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

func (r *communityEdgeRepo) DeleteByCommunityUUID(ctx context.Context, communityUUID string, tx port.Transaction) error {
	cypher := `MATCH (c:Community {uuid: $uuid})-[e:HAS_MEMBER]->() DELETE e`
	params := map[string]any{"uuid": communityUUID}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

// ─── Saga Edges ───────────────────────────────────────────────────────────────

type hasEpisodeEdgeRepo struct{ driver *Neo4jDriver }

func (r *hasEpisodeEdgeRepo) Save(ctx context.Context, edge graph.HasEpisodeEdge, tx port.Transaction) error {
	cypher := `
		MATCH (s:Saga {uuid: $src}), (ep:Episodic {uuid: $tgt})
		MERGE (s)-[e:HAS_EPISODE {uuid: $uuid}]->(ep)
		ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
	`
	params := map[string]any{
		"uuid": edge.UUID, "src": edge.SourceUUID,
		"tgt": edge.TargetUUID, "group_id": edge.GroupID,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}

type nextEpisodeEdgeRepo struct{ driver *Neo4jDriver }

func (r *nextEpisodeEdgeRepo) Save(ctx context.Context, edge graph.NextEpisodeEdge, tx port.Transaction) error {
	cypher := `
		MATCH (src:Episodic {uuid: $src}), (tgt:Episodic {uuid: $tgt})
		MERGE (src)-[e:NEXT_EPISODE {uuid: $uuid}]->(tgt)
		ON CREATE SET e.group_id = $group_id, e.created_at = datetime()
	`
	params := map[string]any{
		"uuid": edge.UUID, "src": edge.SourceUUID,
		"tgt": edge.TargetUUID, "group_id": edge.GroupID,
	}
	if tx != nil {
		_, err := tx.Run(ctx, cypher, params)
		return err
	}
	_, err := r.driver.ExecuteQuery(ctx, cypher, params)
	return err
}
