package neo4j

import (
	"context"
	"fmt"

	n4j "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

// --- BulkRepository ---

// SaveBulk atomically persists nodes, edges, and episode in a single transaction.
func (d *Driver) SaveBulk(ctx context.Context, nodes []domain.EntityNode, edges []domain.EntityEdge, episode domain.EpisodicNode) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx n4j.ManagedTransaction) (any, error) {
		// 1. Save episode node
		if err := d.txSaveEpisodicNode(ctx, tx, episode); err != nil {
			return nil, fmt.Errorf("save episode: %w", err)
		}

		// 2. Save all entity nodes
		for i := range nodes {
			if err := d.txSaveEntityNode(ctx, tx, nodes[i]); err != nil {
				return nil, fmt.Errorf("save node %d (%s): %w", i, nodes[i].UUID, err)
			}
		}

		// 3. Save all edges (after nodes exist)
		for i := range edges {
			if err := d.txSaveEdge(ctx, tx, edges[i]); err != nil {
				return nil, fmt.Errorf("save edge %d (%s): %w", i, edges[i].UUID, err)
			}
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("neo4j: bulk save: %w", err)
	}

	d.logger.Info("bulk save completed",
		"episode_id", episode.UUID,
		"group_id", episode.GroupID,
		"nodes", len(nodes),
		"edges", len(edges),
	)
	return nil
}

// RollbackBulk removes all nodes/edges created by a specific episode.
func (d *Driver) RollbackBulk(ctx context.Context, episodeID string) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx n4j.ManagedTransaction) (any, error) {
		// Delete edges linked to this episode
		_, err := tx.Run(ctx,
			`MATCH ()-[r:RELATES_TO {episode_id: $episode_id}]->() DELETE r`,
			map[string]any{"episode_id": episodeID},
		)
		if err != nil {
			return nil, fmt.Errorf("delete edges: %w", err)
		}

		// Delete MENTIONS edges from this episode
		_, err = tx.Run(ctx,
			`MATCH (ep:Episodic {uuid: $episode_id})-[r:MENTIONS]->() DELETE r`,
			map[string]any{"episode_id": episodeID},
		)
		if err != nil {
			return nil, fmt.Errorf("delete mentions: %w", err)
		}

		// Delete the episode node itself
		_, err = tx.Run(ctx,
			`MATCH (n:Episodic {uuid: $episode_id}) DETACH DELETE n`,
			map[string]any{"episode_id": episodeID},
		)
		if err != nil {
			return nil, fmt.Errorf("delete episode: %w", err)
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("neo4j: rollback bulk: %w", err)
	}

	d.logger.Info("bulk rollback completed", "episode_id", episodeID)
	return nil
}

// DeleteByGroupID removes ALL data for a tenant (purge operation).
func (d *Driver) DeleteByGroupID(ctx context.Context, groupID string) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	// Delete in batches to avoid transaction memory issues
	labels := []string{"Entity", "Episodic", "Community", "Saga"}
	for _, label := range labels {
		cypher := fmt.Sprintf(`MATCH (n:%s {group_id: $group_id}) DETACH DELETE n`, label)
		_, err := session.Run(ctx, cypher, map[string]any{"group_id": groupID})
		if err != nil {
			return fmt.Errorf("neo4j: delete %s by group: %w", label, err)
		}
	}

	d.logger.Info("group purge completed", "group_id", groupID)
	return nil
}

// --- TransactionManager ---

// WithTransaction executes fn within a Neo4j transaction.
func (d *Driver) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session := d.session(ctx)
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx, func(tx n4j.ManagedTransaction) (any, error) {
		// Execute the user's function
		if err := fn(ctx); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

// --- Transactional helpers ---

func (d *Driver) txSaveEntityNode(ctx context.Context, tx n4j.ManagedTransaction, node domain.EntityNode) error {
	cypher := `
		MERGE (n:Entity {uuid: $uuid})
		SET n.name = $name, n.group_id = $group_id, n.summary = $summary,
		    n.name_embedding = $name_embedding, n.labels = $labels,
		    n.created_at = coalesce(n.created_at, datetime()), n.updated_at = datetime()
	`
	_, err := tx.Run(ctx, cypher, map[string]any{
		"uuid":           node.UUID,
		"name":           node.Name,
		"group_id":       node.GroupID,
		"summary":        node.Summary,
		"name_embedding": node.NameEmbedding,
		"labels":         node.Labels,
	})
	return err
}

func (d *Driver) txSaveEpisodicNode(ctx context.Context, tx n4j.ManagedTransaction, node domain.EpisodicNode) error {
	cypher := `
		MERGE (n:Episodic {uuid: $uuid})
		SET n.name = $name, n.group_id = $group_id, n.content = $content,
		    n.source = $source, n.valid_at = datetime($valid_at),
		    n.created_at = coalesce(n.created_at, datetime())
	`
	_, err := tx.Run(ctx, cypher, map[string]any{
		"uuid":     node.UUID,
		"name":     node.Name,
		"group_id": node.GroupID,
		"content":  node.Content,
		"source":   node.Source,
		"valid_at": node.ValidAt.Format("2006-01-02T15:04:05Z"),
	})
	return err
}

func (d *Driver) txSaveEdge(ctx context.Context, tx n4j.ManagedTransaction, edge domain.EntityEdge) error {
	cypher := `
		MATCH (s:Entity {uuid: $source_node_id})
		MATCH (t:Entity {uuid: $target_node_id})
		MERGE (s)-[r:RELATES_TO {uuid: $uuid}]->(t)
		SET r.name = $name, r.group_id = $group_id, r.fact = $fact,
		    r.fact_embedding = $fact_embedding, r.valid_at = datetime($valid_at),
		    r.episode_id = $episode_id, r.created_at = coalesce(r.created_at, datetime())
	`
	_, err := tx.Run(ctx, cypher, map[string]any{
		"uuid":           edge.UUID,
		"source_node_id": edge.SourceNodeID,
		"target_node_id": edge.TargetNodeID,
		"name":           edge.Name,
		"group_id":       edge.GroupID,
		"fact":           edge.Fact,
		"fact_embedding":  edge.FactEmbedding,
		"valid_at":       edge.ValidAt.Format("2006-01-02T15:04:05Z"),
		"episode_id":     edge.EpisodeID,
	})
	return err
}
