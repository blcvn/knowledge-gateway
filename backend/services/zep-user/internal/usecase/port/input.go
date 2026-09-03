package usecase

import "context"

// Enterprise Usecases for zep-user
type ZepUserUseCase interface {
	CreateUser(ctx context.Context, req interface{}) (interface{}, error)
	GetUser(ctx context.Context, req interface{}) (interface{}, error)
	UpdateUser(ctx context.Context, req interface{}) (interface{}, error)

}
