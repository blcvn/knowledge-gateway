package usecase

import "context"

// Enterprise Usecases for sm-auth
type SmAuthUseCase interface {
	Login(ctx context.Context, req interface{}) (interface{}, error)
	ValidateToken(ctx context.Context, req interface{}) (interface{}, error)

}
