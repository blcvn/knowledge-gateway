package usecase

import "context"

// Enterprise Usecases for sm-project
type SmProjectUseCase interface {
	CreateSpace(ctx context.Context, req interface{}) (interface{}, error)
	CheckPermission(ctx context.Context, req interface{}) (interface{}, error)

}
