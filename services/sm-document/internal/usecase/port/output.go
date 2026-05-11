package repository

import (
	"context"
)

// Enterprise Repository Ports for sm-document
type DocumentRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}

type ChunkRepository interface {
	Save(ctx context.Context, entity interface{}) error
	FindByID(ctx context.Context, id string) (interface{}, error)
}


