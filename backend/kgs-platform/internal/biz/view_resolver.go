package biz

import (
	"context"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/projection"
)

// ViewResolver shapes raw graph data via role-based projection rules.
type ViewResolver struct {
	projection projection.ProjectionEngine
}

// NewViewResolver builds a view resolver instance.
func NewViewResolver(engine projection.ProjectionEngine) *ViewResolver {
	return &ViewResolver{projection: engine}
}

// Resolve applies role-based filtering/masking for a given namespace and role.
func (vr *ViewResolver) Resolve(ctx context.Context, namespace, role string, queryResult map[string]any) (map[string]any, error) {
	if vr == nil || vr.projection == nil {
		return queryResult, nil
	}
	return vr.projection.Apply(ctx, namespace, role, queryResult)
}
