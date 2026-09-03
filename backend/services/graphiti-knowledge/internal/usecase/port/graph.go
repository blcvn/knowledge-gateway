package port

import (
	"context"
	"time"

	"vnp-memory/pkg/graph"
)

// EntityCandidate — candidate entity from similarity search for dedupe
type EntityCandidate struct {
	UUID      string
	Name      string
	Labels    []string
	Summary   string
	CreatedAt time.Time
}

// EntityNodePort — read-only operations used by knowledge layer
type EntityNodePort interface {
	FindCandidates(ctx context.Context, nameVec []float32, groupID string) ([]EntityCandidate, error)
	GetByUUIDs(ctx context.Context, uuids []string) ([]*graph.EntityNode, error)
}

// EntityEdgePort — read operations for edge deduplication
type EntityEdgePort interface {
	GetBetweenNodes(ctx context.Context, srcUUID, tgtUUID string) ([]*graph.EntityEdge, error)
	GetByNodeUUID(ctx context.Context, nodeUUID string) ([]*graph.EntityEdge, error)
}
