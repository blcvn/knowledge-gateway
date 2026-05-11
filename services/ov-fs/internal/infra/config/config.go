package config

type Config struct {
	GRPCPort           int    `mapstructure:"GRPC_PORT"`
	HealthPort         int    `mapstructure:"HEALTH_PORT"`
	LogLevel           string `mapstructure:"LOG_LEVEL"`
	OTelEndpoint       string `mapstructure:"OTEL_ENDPOINT"`
	NatsURL            string `mapstructure:"NATS_URL"`
	DbDSN              string `mapstructure:"DB_DSN"`
	DbBackend          string `mapstructure:"DB_BACKEND"`
	DbMaxConnections   int    `mapstructure:"DB_MAX_CONNECTIONS"`
	CryptoServiceAddr  string `mapstructure:"CRYPTO_SERVICE_ADDR"`
	CryptoEnabled      bool   `mapstructure:"CRYPTO_ENABLED"`
	PathLockTimeoutMs  int    `mapstructure:"PATHLOCK_TIMEOUT_MS"`
	MaxFileSizeMb      int    `mapstructure:"MAX_FILE_SIZE_MB"`
	AbstractGeneration string `mapstructure:"ABSTRACT_GENERATION"`
	LlmEndpoint        string `mapstructure:"LLM_ENDPOINT"`

	// PostgreSQL
	PgHost     string `mapstructure:"PG_HOST"`
	PgPort     int    `mapstructure:"PG_PORT"`
	PgDatabase string `mapstructure:"PG_DATABASE"`
	PgSSLMode  string `mapstructure:"PG_SSLMODE"`

	// SurrealDB
	SurrealURL string `mapstructure:"SURREAL_URL"`
	SurrealNS  string `mapstructure:"SURREAL_NS"`
	SurrealDB  string `mapstructure:"SURREAL_DB"`
}

func Load() (*Config, error) {
	// viper loading logic
	return &Config{}, nil
}
