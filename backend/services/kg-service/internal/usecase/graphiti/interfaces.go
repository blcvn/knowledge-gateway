// Package graphiti — graph repository interface and port definitions.
package graphiti

import (
	"context"

	"vnp-memory/services/kg-service/internal/domain/graphiti"
)

// GraphRepoInterface defines all graph repository methods used by usecases.
type GraphRepoInterface interface {
	UpsertNode(ctx context.Context, node *graphiti.Node) error
	UpsertEdge(ctx context.Context, edge *graphiti.Edge) error
	GetNode(ctx context.Context, tenantID, uuid string) (*graphiti.Node, error)
	GetEdge(ctx context.Context, tenantID, uuid string) (*graphiti.Edge, error)
	GetNeighbors(ctx context.Context, tenantID, nodeUUID string, depth int) ([]*graphiti.Node, []*graphiti.Edge, error)
	GetOntology(ctx context.Context, tenantID string) (*graphiti.Ontology, error)
	UpdateOntology(ctx context.Context, ontology *graphiti.Ontology) error
	QuerySubgraph(ctx context.Context, tenantID, query string) ([]*graphiti.Node, []*graphiti.Edge, error)
}
