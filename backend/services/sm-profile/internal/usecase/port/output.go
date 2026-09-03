package repository

import (
	"context"
)

// Enterprise Repository Ports for sm-profile
type ProfileRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


