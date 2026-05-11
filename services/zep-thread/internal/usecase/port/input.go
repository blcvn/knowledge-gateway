package usecase

import "context"

// Enterprise Usecases for zep-thread
type ZepThreadUseCase interface {
	CreateThread(ctx context.Context, req interface{}) (interface{}, error)
	AddMessage(ctx context.Context, req interface{}) (interface{}, error)
	ListThreads(ctx context.Context, req interface{}) (interface{}, error)

}
