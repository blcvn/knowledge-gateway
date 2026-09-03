package port

import "context"

// ISmartSearchUseCase is the input port for smart search.
type ISmartSearchUseCase interface {
    SmartSearch(ctx context.Context, query, tenantID string, limit int) (any, error)
}
