package config

import "time"

type Config struct {
	Service   ServiceConfig
	GRPC      GRPCConfig
	Health    HealthConfig
	Neo4j     Neo4jConfig
	Qdrant    QdrantConfig
	Redis     RedisConfig
	NATS      NATSConfig
	Bifrost   BifrostConfig
	Telemetry TelemetryConfig
	Search    SearchConfig
}

type ServiceConfig struct {
	Name string
}

type GRPCConfig struct {
	Port int
}

type HealthConfig struct {
	Port int
}

type Neo4jConfig struct{}
type QdrantConfig struct{}
type RedisConfig struct{}
type NATSConfig struct{}
type BifrostConfig struct{}
type TelemetryConfig struct{}

type SearchConfig struct {
	CacheTTL             time.Duration
	MaxConcurrentQueries int
	DefaultTopK          int
	RerankModelID        string
}

func LoadConfig() (*Config, error) {
	// Simple mock config
	return &Config{
		Service: ServiceConfig{Name: "cognee-search"},
		GRPC:    GRPCConfig{Port: 9013},
		Health:  HealthConfig{Port: 9093},
		Search: SearchConfig{
			CacheTTL:             5 * time.Minute,
			MaxConcurrentQueries: 50,
			DefaultTopK:          10,
			RerankModelID:        "bge-reranker-v2-m3",
		},
	}, nil
}
