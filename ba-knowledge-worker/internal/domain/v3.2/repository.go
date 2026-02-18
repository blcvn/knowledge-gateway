package v32

import (
	"context"
)

// GraphRepository defines the interface for graph persistence
type GraphRepository interface {
	Save(ctx context.Context, docID string, graph *RequirementGraph) error
	SaveGraph(ctx context.Context, projectID string, graph *RequirementGraph) error
	GetGraph(ctx context.Context, projectID string) (*RequirementGraph, error)
	GetByDocumentID(ctx context.Context, docID string) (*RequirementGraph, error)
	AddNode(ctx context.Context, projectID string, node RequirementNode) error
	AddEdge(ctx context.Context, projectID string, edge DependencyEdge) error
}

// DocumentRepository defines the interface for document persistence
type DocumentRepository interface {
	Create(ctx context.Context, doc *Document) error
	Get(ctx context.Context, id string) (*Document, error)
	GetByParentId(ctx context.Context, parentId string, tier RequirementTier) (*Document, error)
	Update(ctx context.Context, doc *Document) error
	Delete(ctx context.Context, id string) error
	ListByProject(ctx context.Context, projectID string) ([]*Document, error)
	GetByParent(ctx context.Context, parentID string) ([]*Document, error)
}
