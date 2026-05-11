package usecase

import "context"

// Enterprise Usecases for zep-core
type ZepCoreUseCase interface {
	AddMemory(ctx context.Context, req interface{}) (interface{}, error)
	GetMemory(ctx context.Context, req interface{}) (interface{}, error)

}
