package usecase

import "context"

// Enterprise Usecases for zep-graph
type ZepGraphUseCase interface {
	AddNode(ctx context.Context, req interface{}) (interface{}, error)
	AddEdge(ctx context.Context, req interface{}) (interface{}, error)
	Traverse(ctx context.Context, req interface{}) (interface{}, error)

}
