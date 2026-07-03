package config

import (
	"fmt"
	"time"
)

type Config struct {
	Observability       ObservabilityConfig
	HTTP                HTTPConfig
	Postgres            PostgresConfig
	Redis               RedisConfig
	Embedding           EmbeddingConfig
	Vector              AdapterConfig
	Graph               AdapterConfig
	FTS                 AdapterConfig
	RateLimit           RateLimitConfig
	SyncLagToleranceMs  int
	SyncLagStuckRetries int
	SyncEtaDefaultMs    int
}

type ObservabilityConfig struct {
	LogLevel             string
	LogFormat            string
	ServiceName          string
	ServiceVersion       string
	OTELExporterEndpoint string
	OTELExporterProtocol string
	TraceSamplingRatio   float64
	MetricsEnabled       bool
	MetricsAddress       string
}

type HTTPConfig struct {
	Host string
	Port int
}

func (c HTTPConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DB       int
}

func (c RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type EmbeddingConfig struct {
	Provider string
	URL      string
	Model    string
	APIKey   string
	ProxyURL string
	CacheTTL time.Duration
	// Dimensions is the vector dimension returned by the configured provider.
	// Required when using VECTOR_ADAPTER=pgvector and HNSW indexing.
	// Set via EMBEDDING_DIMENSIONS (default 0 = inferred from response).
	Dimensions int
	// TenantRoutes maps tenant IDs to per-tenant provider overrides.
	// Set via EMBEDDING_TENANT_ROUTES (JSON: {"tenant-id": EmbeddingRoute}).
	TenantRoutes map[string]EmbeddingRoute
	// DomainRoutes maps domain IDs to per-domain provider overrides.
	// Set via EMBEDDING_DOMAIN_ROUTES (JSON: {"domain-id": EmbeddingRoute}).
	DomainRoutes map[string]EmbeddingRoute
}

// EmbeddingRoute configures a per-tenant or per-domain embedding provider
// override used by RoutingRouter. Fields mirror EmbeddingConfig.
type EmbeddingRoute struct {
	Provider   string `json:"provider"`
	URL        string `json:"url"`
	Model      string `json:"model"`
	APIKey     string `json:"api_key"`
	Dimensions int    `json:"dimensions"`
}

type AdapterConfig struct {
	Kind       string
	Endpoint   string
	Database   string
	Collection string
}

type RateLimitConfig struct {
	FreePerMinute       int
	ProPerMinute        int
	EnterprisePerMinute int
}
