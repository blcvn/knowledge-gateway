package usecase

import "context"

// Enterprise Usecases for zep-memory
type ZepMemoryUseCase interface {
	Summarize(ctx context.Context, req interface{}) (interface{}, error)
	GetContext(ctx context.Context, req interface{}) (interface{}, error)

}
