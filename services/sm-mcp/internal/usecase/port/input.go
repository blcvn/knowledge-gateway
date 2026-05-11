package usecase

import (
	"context"
)

// Enterprise Usecases for sm-mcp
type SmMcpUseCase interface {
	HandleToolCall(ctx context.Context, req interface{}) (interface{}, error)
	ListTools(ctx context.Context, req interface{}) (interface{}, error)
	ReadResource(ctx context.Context, req interface{}) (interface{}, error)

}
