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
	delegate := GraphAdapter(NewNeo4jGraphAdapter(cfg))
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
