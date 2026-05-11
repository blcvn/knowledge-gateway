package repository

import "context"

// Enterprise Repository Ports for zep-user
type UserRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


