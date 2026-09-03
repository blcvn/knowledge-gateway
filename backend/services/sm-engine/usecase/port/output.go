package repository

import "context"

// Enterprise Repository Ports for sm-engine
type EngineRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


