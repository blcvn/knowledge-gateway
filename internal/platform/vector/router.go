package vector

import (
	"context"
	"strings"
)

type EmbeddingRouter interface {
	EmbeddingProvider
	RouteContext(tenantID, domainID string) EmbeddingProvider
}

type DirectRouter struct {
	Provider EmbeddingProvider
}

func (r DirectRouter) RouteContext(_, _ string) EmbeddingProvider {
	return r.Provider
}

func (r DirectRouter) Embed(ctx context.Context, text string) ([]float64, error) {
	if r.Provider == nil {
		return nil, nil
	}
	return r.Provider.Embed(ctx, text)
}

func (r DirectRouter) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if r.Provider == nil {
		return nil, nil
	}
	return r.Provider.EmbedBatch(ctx, texts)
}

func (r DirectRouter) Dimensions() int {
	if r.Provider == nil {
		return 0
	}
	return r.Provider.Dimensions()
}

func (r DirectRouter) ModelID() string {
	if r.Provider == nil {
		return ""
	}
	return r.Provider.ModelID()
}

type RoutingRouter struct {
	Default EmbeddingProvider
	Tenants map[string]EmbeddingProvider
	Domains map[string]EmbeddingProvider
}

func (r RoutingRouter) RouteContext(tenantID, domainID string) EmbeddingProvider {
	if provider, ok := r.Tenants[strings.TrimSpace(tenantID)]; ok && provider != nil {
		return provider
	}
	if provider, ok := r.Domains[strings.TrimSpace(domainID)]; ok && provider != nil {
		return provider
	}
	return r.Default
}

func (r RoutingRouter) Embed(ctx context.Context, text string) ([]float64, error) {
	if r.Default == nil {
		return nil, nil
	}
	return r.Default.Embed(ctx, text)
}

func (r RoutingRouter) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	if r.Default == nil {
		return nil, nil
	}
	return r.Default.EmbedBatch(ctx, texts)
}

func (r RoutingRouter) Dimensions() int {
	if r.Default == nil {
		return 0
	}
	return r.Default.Dimensions()
}

func (r RoutingRouter) ModelID() string {
	if r.Default == nil {
		return ""
	}
	return r.Default.ModelID()
}
