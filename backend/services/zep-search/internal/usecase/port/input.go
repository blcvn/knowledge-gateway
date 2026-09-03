package usecase

import "context"

// Enterprise Usecases for zep-search
type ZepSearchUseCase interface {
	VectorSearch(ctx context.Context, req interface{}) (interface{}, error)
	BM25Search(ctx context.Context, req interface{}) (interface{}, error)
	MMRSearch(ctx context.Context, req interface{}) (interface{}, error)

}
