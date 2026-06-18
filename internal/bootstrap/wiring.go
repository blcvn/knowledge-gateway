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

func buildEmbeddingRouter(cfg config.Config) (vector.EmbeddingRouter, error) {
	var provider vector.EmbeddingProvider
	switch cfg.Embedding.Provider {
	case "", "deterministic":
		provider = vector.NewDeterministicProvider(8)
	case "http":
		provider = vector.HTTPEmbeddingProvider{
			URL:     cfg.Embedding.URL,
			Model:   cfg.Embedding.Model,
			APIKey:  cfg.Embedding.APIKey,
			Timeout: 30 * time.Second,
		}
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Embedding.Provider)
	}

	var chain vector.EmbeddingProvider = provider
	if cfg.Embedding.ProxyURL != "" {
		chain = vector.ProxyHTTPProvider{Inner: chain, ProxyURL: cfg.Embedding.ProxyURL}
	}
	chain = &vector.RetryProvider{Inner: chain, MaxAttempts: 3}
	if cfg.Embedding.CacheTTL > 0 {
		chain = &vector.CachingProvider{Inner: chain, TTL: cfg.Embedding.CacheTTL}
	}
	return vector.DirectRouter{Provider: chain}, nil
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
	return labels
}

func buildVectorAdapter(kind string, db *sql.DB) (vectorstore.VectorAdapter, error) {
	switch kind {
	case "", "memory":
		return vectorstore.NewInMemoryVectorAdapter(), nil
	case "pgvector":
		return vectorstore.NewPgVectorAdapter(db), nil
	default:
		return nil, fmt.Errorf("unsupported vector adapter: %s", kind)
	}
}

func buildGraphAdapter(kind string) (graphstore.GraphAdapter, error) {
	switch kind {
	case "", "memory":
		return graphstore.NewInMemoryGraphAdapter(), nil
	case "neo4j":
		return nil, fmt.Errorf("graph adapter neo4j is not wired in bootstrap yet")
	default:
		return nil, fmt.Errorf("unsupported graph adapter: %s", kind)
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
