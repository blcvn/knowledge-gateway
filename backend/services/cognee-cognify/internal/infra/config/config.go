package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Service   ServiceConfig
	GRPC      GRPCConfig
	Health    HealthConfig
	Postgres  PostgresConfig
	Neo4j     Neo4jConfig
	Qdrant    QdrantConfig
	NATS      NATSConfig
	Bifrost   BifrostConfig
	Telemetry TelemetryConfig
	Pipeline  PipelineConfig
}

type ServiceConfig struct {
	Name    string
	Version string
}

type GRPCConfig struct {
	Port int
}

type HealthConfig struct {
	Port int
}

type PostgresConfig struct {
	DSN string
}

type Neo4jConfig struct {
	URI      string
	User     string
	Password string
}

type QdrantConfig struct {
	URL              string
	ChunkCollection  string
	EntityCollection string
}

type NATSConfig struct {
	URL string
}

type BifrostConfig struct {
	URL   string
	Model string
}

type TelemetryConfig struct {
	OTLPEndpoint string
}

type PipelineConfig struct {
	MaxConcurrentLLMCalls int           `mapstructure:"max_concurrent_llm"`
	StageTimeout          time.Duration `mapstructure:"stage_timeout"`
	ChunkSize             int           `mapstructure:"chunk_size"`
	ChunkOverlap          int           `mapstructure:"chunk_overlap"`
}

func Load() (*Config, error) {
	viper.SetEnvPrefix("COGNEE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("Service.Name", "cognee-cognify")
	viper.SetDefault("Service.Version", "1.0.0")
	viper.SetDefault("GRPC.Port", 9012)
	viper.SetDefault("Health.Port", 9092)
	viper.SetDefault("Postgres.DSN", "host=localhost user=postgres password=postgres dbname=cognee port=5432 sslmode=disable")
	viper.SetDefault("Neo4j.URI", "neo4j://localhost:7687")
	viper.SetDefault("Neo4j.User", "neo4j")
	viper.SetDefault("Neo4j.Password", "password")
	viper.SetDefault("Qdrant.URL", "http://localhost:6333")
	viper.SetDefault("Qdrant.ChunkCollection", "chunks")
	viper.SetDefault("Qdrant.EntityCollection", "entities")
	viper.SetDefault("NATS.URL", "nats://localhost:4222")
	viper.SetDefault("Bifrost.URL", "http://localhost:8080")
	viper.SetDefault("Bifrost.Model", "gpt-4o-mini")
	viper.SetDefault("Telemetry.OTLPEndpoint", "localhost:4317")
	viper.SetDefault("Pipeline.MaxConcurrentLLMCalls", 5)
	viper.SetDefault("Pipeline.StageTimeout", 5*time.Minute)
	viper.SetDefault("Pipeline.ChunkSize", 512)
	viper.SetDefault("Pipeline.ChunkOverlap", 50)

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	return &cfg, nil
}
