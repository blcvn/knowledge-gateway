package usecase

import (
	"context"
)

// Enterprise Usecases for sm-profile
type SmProfileUseCase interface {
	GetProfile(ctx context.Context, req interface{}) (interface{}, error)
	UpdateProfile(ctx context.Context, req interface{}) (interface{}, error)
	GetDynamicTraits(ctx context.Context, req interface{}) (interface{}, error)

}
