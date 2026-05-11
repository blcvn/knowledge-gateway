package repository

import "context"

// Enterprise Repository Ports for zep-core
type CoreRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


