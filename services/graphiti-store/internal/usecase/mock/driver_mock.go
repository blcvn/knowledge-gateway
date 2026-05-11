package mock

import (
	"context"
	"time"

	"github.com/vnp-community/vnp-memory/services/graphiti-store/internal/domain"
)

type GraphDriverMock struct {
	SaveNodeFunc func(ctx context.Context, node domain.EntityNode) error
	GetNodeFunc  func(ctx context.Context, groupID, uuid string) (*domain.EntityNode, error)
	DeleteNodeFunc func(ctx context.Context, groupID, uuid string) error
	// and so on... for the sake of tests we can use a mocking framework like mockgen,
	// but let's implement a simple mock or use mockery if it's available.
}
