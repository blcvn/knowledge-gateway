package config

import (
	"errors"
	"time"
)

func Load() (Config, error) {
	var errs []error

	httpPort, err := intEnv("KG_HTTP_PORT", 8082)
	if err != nil {
		errs = append(errs, err)
	}
	postgresPort, err := intEnv("KG_POSTGRES_PORT", 5432)
	if err != nil {
		errs = append(errs, err)
	}
	postgresMaxOpenConns, err := intEnv("KG_POSTGRES_MAX_OPEN_CONNS", 20)
	if err != nil {
		errs = append(errs, err)
	}
	postgresMaxIdleConns, err := intEnv("KG_POSTGRES_MAX_IDLE_CONNS", 5)
	if err != nil {
		errs = append(errs, err)
	}
	postgresConnMaxLifetime, err := durationEnv("KG_POSTGRES_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		errs = append(errs, err)
	}
	redisPort, err := intEnv("KG_REDIS_PORT", 6379)
	if err != nil {
		errs = append(errs, err)
	}
	redisDB, err := intEnv("KG_REDIS_DB", 0)
	if err != nil {
		errs = append(errs, err)
	}
	embeddingCacheTTLS, err := intEnv("EMBEDDING_CACHE_TTL_S", 0)
	if err != nil {
		errs = append(errs, err)
	}
	syncLagToleranceMs, err := intEnv("SYNC_LAG_TOLERANCE_MS", 30000)
	if err != nil {
		errs = append(errs, err)
	}
	syncLagStuckRetries, err := intEnv("SYNC_LAG_STUCK_RETRIES", 3)
	if err != nil {
		errs = append(errs, err)
	}
	syncEtaDefaultMs, err := intEnv("SYNC_ETA_DEFAULT_MS", 5000)
	if err != nil {
		errs = append(errs, err)
	}
	rateLimitFree, err := intEnv("KG_RATE_LIMIT_FREE", 15)
	if err != nil {
		errs = append(errs, err)
	}
	rateLimitPro, err := intEnv("KG_RATE_LIMIT_PRO", 60)
	if err != nil {
		errs = append(errs, err)
	}
	rateLimitEnterprise, err := intEnv("KG_RATE_LIMIT_ENTERPRISE", 240)
	if err != nil {
		errs = append(errs, err)
	}

	cfg := Config{
		HTTP: HTTPConfig{
			Host: stringEnv("KG_HTTP_HOST", "0.0.0.0"),
			Port: httpPort,
		},
		Postgres: PostgresConfig{
			Host:            stringEnv("KG_POSTGRES_HOST", "127.0.0.1"),
			Port:            postgresPort,
			User:            stringEnv("KG_POSTGRES_USER", "postgres"),
			Password:        stringEnv("KG_POSTGRES_PASSWORD", "postgres"),
			Database:        stringEnv("KG_POSTGRES_DATABASE", "kg_service"),
			SSLMode:         stringEnv("KG_POSTGRES_SSLMODE", "disable"),
			MaxOpenConns:    postgresMaxOpenConns,
			MaxIdleConns:    postgresMaxIdleConns,
			ConnMaxLifetime: postgresConnMaxLifetime,
		},
		Redis: RedisConfig{
			Host:     stringEnv("KG_REDIS_HOST", "127.0.0.1"),
			Port:     redisPort,
			Username: stringEnv("KG_REDIS_USERNAME", ""),
			Password: stringEnv("KG_REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		Embedding: EmbeddingConfig{
			Provider:     stringEnv("EMBEDDING_PROVIDER", "deterministic"),
			URL:          stringEnv("EMBEDDING_URL", ""),
			Model:        stringEnv("EMBEDDING_MODEL", ""),
			APIKey:       stringEnv("EMBEDDING_API_KEY", ""),
			ProxyURL:     stringEnv("EMBEDDING_PROXY_URL", ""),
			CacheTTL:     time.Duration(embeddingCacheTTLS) * time.Second,
			Dimensions:   intEnv("EMBEDDING_DIMENSIONS", 0),
			TenantRoutes: jsonMapEnv[EmbeddingRoute]("EMBEDDING_TENANT_ROUTES"),
			DomainRoutes: jsonMapEnv[EmbeddingRoute]("EMBEDDING_DOMAIN_ROUTES"),
		},
		Vector: AdapterConfig{
			Kind:       stringEnv("VECTOR_ADAPTER", "memory"),
			Endpoint:   stringEnv("KG_VECTOR_ENDPOINT", ""),
			Collection: stringEnv("KG_VECTOR_COLLECTION", "kg_vectors"),
		},
		Graph: AdapterConfig{
			Kind:     stringEnv("GRAPH_ADAPTER", "memory"),
			Endpoint: stringEnv("KG_GRAPH_ENDPOINT", ""),
			Database: stringEnv("KG_GRAPH_DATABASE", ""),
		},
		FTS: AdapterConfig{Kind: stringEnv("FTS_ADAPTER", "memory")},
		RateLimit: RateLimitConfig{
			FreePerMinute:       rateLimitFree,
			ProPerMinute:        rateLimitPro,
			EnterprisePerMinute: rateLimitEnterprise,
		},
		SyncLagToleranceMs:  syncLagToleranceMs,
		SyncLagStuckRetries: syncLagStuckRetries,
		SyncEtaDefaultMs:    syncEtaDefaultMs,
	}

	errs = append(errs, cfg.Validate())
	return cfg, errors.Join(errs...)
}
