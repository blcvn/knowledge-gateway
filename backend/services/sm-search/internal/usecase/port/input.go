package usecase

import "context"

// Enterprise Usecases for sm-search
type SmSearchUseCase interface {
	ExecuteSearch(ctx context.Context, req interface{}) (interface{}, error)
	IndexDocument(ctx context.Context, req interface{}) (interface{}, error)

}
