package bootstrap

import (
	"database/sql"
	"fmt"
	"time"

	"kg-service/internal/config"
	"kg-service/internal/platform/fts"
	"kg-service/internal/platform/graphstore"
	"kg-service/internal/platform/vector"
	"kg-service/internal/platform/vectorstore"
)

// buildEmbeddingRouter constructs an EmbeddingRouter from config.
//
//   - No routes configured  →  DirectRouter (single provider for all tenants/domains)
//   - Routes configured     →  RoutingRouter (per-tenant or per-domain overrides)
//
// The default provider is always wrapped in the middleware chain
// (cache → retry → optional proxy). Route overrides inherit retry but
// run their own provider.
func buildEmbeddingRouter(cfg config.Config) (vector.EmbeddingRouter, error) {
	defaultProvider, err := buildEmbeddingProvider(cfg.Embedding.Provider, cfg.Embedding.URL, cfg.Embedding.Model, cfg.Embedding.APIKey, cfg.Embedding.Dimensions, cfg.Embedding.MaxInputChars)
	if err != nil {
		return nil, err
	}
	defaultChain := applyEmbeddingMiddleware(defaultProvider, cfg.Embedding)

	// No per-tenant/domain overrides → use fast DirectRouter.
	if len(cfg.Embedding.TenantRoutes) == 0 && len(cfg.Embedding.DomainRoutes) == 0 {
		return vector.DirectRouter{Provider: defaultChain}, nil
	}

	// Build RoutingRouter with per-tenant and per-domain provider overrides.
	router := vector.RoutingRouter{
		Default: defaultChain,
		Tenants: make(map[string]vector.EmbeddingProvider, len(cfg.Embedding.TenantRoutes)),
		Domains: make(map[string]vector.EmbeddingProvider, len(cfg.Embedding.DomainRoutes)),
	}
	for tenantID, route := range cfg.Embedding.TenantRoutes {
		p, err := buildEmbeddingProvider(route.Provider, route.URL, route.Model, route.APIKey, route.Dimensions, cfg.Embedding.MaxInputChars)
		if err != nil {
			return nil, fmt.Errorf("tenant route %q: %w", tenantID, err)
		}
		router.Tenants[tenantID] = applyEmbeddingMiddleware(p, cfg.Embedding)
	}
	for domainID, route := range cfg.Embedding.DomainRoutes {
		p, err := buildEmbeddingProvider(route.Provider, route.URL, route.Model, route.APIKey, route.Dimensions, cfg.Embedding.MaxInputChars)
		if err != nil {
			return nil, fmt.Errorf("domain route %q: %w", domainID, err)
		}
		router.Domains[domainID] = applyEmbeddingMiddleware(p, cfg.Embedding)
	}
	return router, nil
}

// buildEmbeddingProvider creates the leaf provider (no middleware).
func buildEmbeddingProvider(provider, url, model, apiKey string, dimensions, maxInputChars int) (vector.EmbeddingProvider, error) {
	switch provider {
	case "", "deterministic":
		dims := dimensions
		if dims <= 0 {
			dims = 8
		}
		return vector.NewDeterministicProvider(dims), nil
	case "http":
		return vector.HTTPEmbeddingProvider{
			URL:           url,
			Model:         model,
			APIKey:        apiKey,
			Timeout:       30 * time.Second,
			Dims:          dimensions,
			MaxInputChars: maxInputChars,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", provider)
	}
}

// applyEmbeddingMiddleware wraps a leaf provider in the middleware chain:
//
//	[CachingProvider →] RetryProvider [→ ProxyHTTPProvider]
//
// Cache and proxy are applied only when configured.
func applyEmbeddingMiddleware(inner vector.EmbeddingProvider, cfg config.EmbeddingConfig) vector.EmbeddingProvider {
	var chain vector.EmbeddingProvider = inner
	if cfg.ProxyURL != "" {
		chain = vector.ProxyHTTPProvider{Inner: chain, ProxyURL: cfg.ProxyURL}
	}
	chain = &vector.RetryProvider{Inner: chain, MaxAttempts: 3}
	if cfg.CacheTTL > 0 {
		chain = &vector.CachingProvider{Inner: chain, TTL: cfg.CacheTTL}
	}
	return chain
}

func embeddingChain(cfg config.Config) []string {
	labels := []string{}
	if cfg.Embedding.CacheTTL > 0 {
		labels = append(labels, "cache")
	}
	labels = append(labels, "retry")
	if cfg.Embedding.ProxyURL != "" {
		labels = append(labels, "proxy")
	}
	switch cfg.Embedding.Provider {
	case "", "deterministic":
		labels = append(labels, "deterministic")
	case "http":
		labels = append(labels, "http")
	}
	if len(cfg.Embedding.TenantRoutes) > 0 {
		labels = append(labels, fmt.Sprintf("+%d tenant-routes", len(cfg.Embedding.TenantRoutes)))
	}
	if len(cfg.Embedding.DomainRoutes) > 0 {
		labels = append(labels, fmt.Sprintf("+%d domain-routes", len(cfg.Embedding.DomainRoutes)))
	}
	return labels
}

func buildVectorAdapter(cfg config.Config, db *sql.DB) (vectorstore.VectorAdapter, error) {
	switch cfg.Vector.Kind {
	case "", "memory":
		return vectorstore.NewInMemoryVectorAdapter(), nil
	case "pgvector":
		return vectorstore.NewPgVectorAdapter(db), nil
	case "qdrant":
		return vectorstore.NewQdrantVectorAdapter(vectorstore.QdrantConfig{
			Endpoint:   cfg.Vector.Endpoint,
			Collection: cfg.Vector.Collection,
		}), nil
	case "milvus":
		return vectorstore.NewMilvusVectorAdapter(vectorstore.MilvusConfig{
			Endpoint:   cfg.Vector.Endpoint,
			Collection: cfg.Vector.Collection,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported vector adapter: %s", cfg.Vector.Kind)
	}
}

func buildGraphAdapter(cfg config.Config) (graphstore.GraphAdapter, error) {
	switch cfg.Graph.Kind {
	case "", "memory":
		return graphstore.NewInMemoryGraphAdapter(), nil
	case "neo4j":
		return graphstore.NewNeo4jGraphAdapter(graphstore.CypherConfig{
			Endpoint: cfg.Graph.Endpoint,
			Database: cfg.Graph.Database,
		}), nil
	case "memgraph":
		return graphstore.NewMemgraphGraphAdapter(graphstore.CypherConfig{
			Endpoint: cfg.Graph.Endpoint,
			Database: cfg.Graph.Database,
		}), nil
	case "nebula":
		return graphstore.NewNebulaGraphAdapter(graphstore.CypherConfig{
			Endpoint: cfg.Graph.Endpoint,
			Database: cfg.Graph.Database,
		}), nil
	case "surreal":
		sc := cfg.Graph.SurrealConfig()
		return graphstore.NewSurrealGraphAdapter(graphstore.SurrealConfig{
			Endpoint:  sc.Endpoint,
			Namespace: sc.Namespace,
			Database:  sc.Database,
			Username:  sc.Username,
			Password:  sc.Password,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported graph adapter: %s", cfg.Graph.Kind)
	}
}

func buildFTSAdapter(kind string, db *sql.DB) (fts.FTSAdapter, error) {
	switch kind {
	case "", "memory":
		return fts.NewInMemoryFTSAdapter(), nil
	case "postgres":
		return fts.NewPgFTSAdapter(db, "simple"), nil
	default:
		return nil, fmt.Errorf("unsupported fts adapter: %s", kind)
	}
}
