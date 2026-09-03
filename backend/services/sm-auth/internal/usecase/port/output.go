package port

import "context"

// Enterprise Repository Ports for sm-auth
type AuthRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


