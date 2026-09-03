package repository

import "context"

// Enterprise Repository Ports for sm-connector
type ConnectorRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


