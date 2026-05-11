package repository

import "context"

// Enterprise Repository Ports for zep-search
type SearchRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


