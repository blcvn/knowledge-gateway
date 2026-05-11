package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	GRPCPort                   int           `mapstructure:"GRPC_PORT"`
	HealthPort                 int           `mapstructure:"HEALTH_PORT"`
	LogLevel                   string        `mapstructure:"LOG_LEVEL"`
	OtelEndpoint               string        `mapstructure:"OTEL_ENDPOINT"`
	NatsURL                    string        `mapstructure:"NATS_URL"`
	DBDSN                      string        `mapstructure:"DB_DSN"`
	QdrantURL                  string        `mapstructure:"QDRANT_URL"`
	BifrostAddr                string        `mapstructure:"BIFROST_ADDR"`
	OvFsAddr                   string        `mapstructure:"OV_FS_ADDR"`
	HotnessDecayHalfLifeH      int           `mapstructure:"HOTNESS_DECAY_HALF_LIFE_H"`
	HotnessSessionBoost        float64       `mapstructure:"HOTNESS_SESSION_BOOST"`
	HotnessRecomputeIntervalM  time.Duration `mapstructure:"HOTNESS_RECOMPUTE_INTERVAL_M"`
	RerankDefaultStrategy      string        `mapstructure:"RERANK_DEFAULT_STRATEGY"`
	SearchMaxResults           int           `mapstructure:"SEARCH_MAX_RESULTS"`
	PropagationFactor          float64       `mapstructure:"PROPAGATION_FACTOR"`
	ConvergenceThreshold       float64       `mapstructure:"CONVERGENCE_THRESHOLD"`
	EmbeddingModel             string        `mapstructure:"EMBEDDING_MODEL"`
	EmbeddingDim               int           `mapstructure:"EMBEDDING_DIM"`
	QdrantCollection           string        `mapstructure:"QDRANT_COLLECTION"`
}

func LoadConfig() (*Config, error) {
	viper.SetEnvPrefix("OV_SEARCH")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set Defaults
	viper.SetDefault("GRPC_PORT", 9052)
	viper.SetDefault("HEALTH_PORT", 9105)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("NATS_URL", "nats://nats:4222")
	viper.SetDefault("QDRANT_COLLECTION", "ov_embeddings")
	viper.SetDefault("EMBEDDING_MODEL", "text-embedding-3-large")
	viper.SetDefault("EMBEDDING_DIM", 1536)
	viper.SetDefault("HOTNESS_DECAY_HALF_LIFE_H", 24)
	viper.SetDefault("HOTNESS_SESSION_BOOST", 0.3)
	viper.SetDefault("HOTNESS_RECOMPUTE_INTERVAL_M", 5*time.Minute)
	viper.SetDefault("RERANK_DEFAULT_STRATEGY", "rrf")
	viper.SetDefault("SEARCH_MAX_RESULTS", 50)
	viper.SetDefault("PROPAGATION_FACTOR", 0.7)
	viper.SetDefault("CONVERGENCE_THRESHOLD", 0.05)

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
