package neo4j

import (
	"context"
	"fmt"
	"time"

	"vnp-memory/services/graphiti-store/internal/usecase/port"
)

type bulkRepo struct {
	driver           *Neo4jDriver
	entityNodes      *entityNodeRepo
	entityEdges      *entityEdgeRepo
	episodeNodes     *episodeNodeRepo
	sagaNodes        *sagaNodeRepo
	episodicEdges    *episodicEdgeRepo
	hasEpisodeEdges  *hasEpisodeEdgeRepo
	nextEpisodeEdges *nextEpisodeEdgeRepo
}

// SaveBulk persists all objects for a single ingestion atomically.
// Order: invalidate old edges → save nodes → save edges → save episode → save saga
func (r *bulkRepo) SaveBulk(ctx context.Context, req port.SaveBulkReq) error {
	tx, err := r.driver.BeginTransaction(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Step 1: Invalidate old contradicted/updated edges (temporal)
	now := time.Now()
	for _, edgeID := range req.InvalidatedEdgeIDs {
		if err = r.entityEdges.Invalidate(ctx, edgeID, now, tx); err != nil {
			return fmt.Errorf("invalidate edge %s: %w", edgeID, err)
		}
	}

	// Step 2: Save entity nodes (MERGE — upsert)
	if err = r.entityNodes.SaveBulk(ctx, req.EntityNodes, tx, 100); err != nil {
		return fmt.Errorf("save entity nodes: %w", err)
	}

	// Step 3: Save new entity edges
	if err = r.entityEdges.SaveBulk(ctx, req.EntityEdges, tx, 100); err != nil {
		return fmt.Errorf("save entity edges: %w", err)
	}

	// Step 4: Save episodic node
	if err = r.episodeNodes.Save(ctx, req.Episode, tx); err != nil {
		return fmt.Errorf("save episode node: %w", err)
	}

	// Step 5: Save MENTIONS edges (episode → entity)
	if err = r.episodicEdges.SaveBulk(ctx, req.EpisodicEdges, tx); err != nil {
		return fmt.Errorf("save episodic edges: %w", err)
	}

	// Step 6: Save saga (optional)
	if req.SagaNode != nil {
		if err = r.sagaNodes.Save(ctx, *req.SagaNode, tx); err != nil {
			return fmt.Errorf("save saga node: %w", err)
		}
		for _, e := range req.HasEpisodeEdges {
			if err = r.hasEpisodeEdges.Save(ctx, e, tx); err != nil {
				return fmt.Errorf("save has_episode edge: %w", err)
			}
		}
		for _, e := range req.NextEpisodeEdges {
			if err = r.nextEpisodeEdges.Save(ctx, e, tx); err != nil {
				return fmt.Errorf("save next_episode edge: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}
