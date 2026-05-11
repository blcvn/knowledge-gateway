package usecase

import (
	"context"
)

// Enterprise Usecases for sm-memory
type SmMemoryUseCase interface {
	CreateMemory(ctx context.Context, req interface{}) (interface{}, error)
	GetMemory(ctx context.Context, req interface{}) (interface{}, error)
	ForgetMemory(ctx context.Context, req interface{}) (interface{}, error)
	ListMemories(ctx context.Context, req interface{}) (interface{}, error)
	CreateRelation(ctx context.Context, req interface{}) (interface{}, error)

}
