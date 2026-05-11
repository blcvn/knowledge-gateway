package usecase

import (
	"context"
)

// Enterprise Usecases for sm-document
type SmDocumentUseCase interface {
	CreateDocument(ctx context.Context, req interface{}) (interface{}, error)
	GetDocument(ctx context.Context, req interface{}) (interface{}, error)
	DeleteDocument(ctx context.Context, req interface{}) (interface{}, error)
	ListDocuments(ctx context.Context, req interface{}) (interface{}, error)
	GetChunks(ctx context.Context, req interface{}) (interface{}, error)

}
