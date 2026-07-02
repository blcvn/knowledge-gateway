package graphstore

import "context"

type CypherConfig struct {
	Endpoint string
	Database string
}

type delegatedGraphAdapter struct {
	delegate GraphAdapter
}

func (a delegatedGraphAdapter) UpsertNode(ctx context.Context, node GraphNode) error {
	return a.delegate.UpsertNode(ctx, node)
}

func (a delegatedGraphAdapter) DeleteNode(ctx context.Context, nodeID string) error {
	return a.delegate.DeleteNode(ctx, nodeID)
}

func (a delegatedGraphAdapter) UpsertRelationship(ctx context.Context, rel GraphRelationship) error {
	return a.delegate.UpsertRelationship(ctx, rel)
}

func (a delegatedGraphAdapter) DeleteRelationship(ctx context.Context, relID string) error {
	return a.delegate.DeleteRelationship(ctx, relID)
}

func (a delegatedGraphAdapter) UpsertNodesBatch(ctx context.Context, nodes []GraphNode) error {
	if batch, ok := a.delegate.(BatchGraphAdapter); ok {
		return batch.UpsertNodesBatch(ctx, nodes)
	}
	for _, node := range nodes {
		if err := a.delegate.UpsertNode(ctx, node); err != nil {
			return err
		}
	}
	return nil
}

func (a delegatedGraphAdapter) DeleteNodesBatch(ctx context.Context, nodes []GraphNode) error {
	if batch, ok := a.delegate.(BatchGraphAdapter); ok {
		return batch.DeleteNodesBatch(ctx, nodes)
	}
	for _, node := range nodes {
		if err := a.delegate.DeleteNode(ctx, node.ID); err != nil {
			return err
		}
	}
	return nil
}

func (a delegatedGraphAdapter) UpsertRelationshipsBatch(ctx context.Context, rels []GraphRelationship) error {
	if batch, ok := a.delegate.(BatchGraphAdapter); ok {
		return batch.UpsertRelationshipsBatch(ctx, rels)
	}
	for _, rel := range rels {
		if err := a.delegate.UpsertRelationship(ctx, rel); err != nil {
			return err
		}
	}
	return nil
}

func (a delegatedGraphAdapter) DeleteRelationshipsBatch(ctx context.Context, rels []GraphRelationship) error {
	if batch, ok := a.delegate.(BatchGraphAdapter); ok {
		return batch.DeleteRelationshipsBatch(ctx, rels)
	}
	for _, rel := range rels {
		if err := a.delegate.DeleteRelationship(ctx, rel.ID); err != nil {
			return err
		}
	}
	return nil
}

func (a delegatedGraphAdapter) ExecuteQuery(ctx context.Context, query GraphQuery, params map[string]any) ([]map[string]any, error) {
	return a.delegate.ExecuteQuery(ctx, query, params)
}

func (a delegatedGraphAdapter) ListNodes(ctx context.Context) ([]GraphNode, error) {
	return a.delegate.ListNodes(ctx)
}

func (a delegatedGraphAdapter) ListRelationships(ctx context.Context) ([]GraphRelationship, error) {
	return a.delegate.ListRelationships(ctx)
}

func (a delegatedGraphAdapter) ReadSyncVersion(ctx context.Context, entityID string) (int64, error) {
	return a.delegate.ReadSyncVersion(ctx, entityID)
}

type MemgraphGraphAdapter struct {
	delegatedGraphAdapter
	Endpoint string
}

func NewMemgraphGraphAdapter(cfg CypherConfig) *MemgraphGraphAdapter {
	// Memgraph community mode does not rely on Neo4j-style named databases.
	// Keep the endpoint, but always use the default graph context so writes
	// land in the same place manual mgconsole checks can read from.
	delegate := GraphAdapter(NewNeo4jGraphAdapter(CypherConfig{
		Endpoint: cfg.Endpoint,
	}))
	return &MemgraphGraphAdapter{delegatedGraphAdapter: delegatedGraphAdapter{delegate: delegate}, Endpoint: cfg.Endpoint}
}

type NebulaGraphAdapter struct {
	delegatedGraphAdapter
	Endpoint string
}

func NewNebulaGraphAdapter(cfg CypherConfig) *NebulaGraphAdapter {
	return &NebulaGraphAdapter{
		delegatedGraphAdapter: delegatedGraphAdapter{delegate: newNebulaDelegate(cfg)},
		Endpoint:              cfg.Endpoint,
	}
}
