package port

import (
	"context"

	"vnp-memory/pkg/graph"
)

// CommunityPort — graph operations needed by community builder
type CommunityPort interface {
	GetClusters(ctx context.Context, groupIDs []string) ([][]string, error)
	GetEntityByUUIDs(ctx context.Context, uuids []string) (map[string]*graph.EntityNode, error)
}
