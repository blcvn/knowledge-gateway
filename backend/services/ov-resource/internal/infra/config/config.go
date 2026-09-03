package config

import "os"

type Config struct {
	GrpcPort             string
	HealthPort           string
	LogLevel             string
	OtelEndpoint         string
	NatsUrl              string
	DbDsn                string
	DbMaxConnections     int
	OvFsAddr             string
	ChunkSizeTokens      int
	ChunkOverlapTokens   int
	MaxIngestionSizeMb   int
	WatchEnabled         bool
	WatchDefaultPollMs   int64
	WatchMaxTasks        int
	TreesitterEnabled    bool
}

func LoadConfig() Config {
	return Config{
		GrpcPort:             getEnv("GRPC_PORT", "9054"),
		HealthPort:           getEnv("HEALTH_PORT", "9107"),
		LogLevel:             getEnv("LOG_LEVEL", "info"),
		OtelEndpoint:         getEnv("OTEL_ENDPOINT", "otel-collector:4317"),
		NatsUrl:              getEnv("NATS_URL", "nats://nats:4222"),
		DbDsn:                getEnv("DB_DSN", ""),
		DbMaxConnections:     20,
		OvFsAddr:             getEnv("OV_FS_ADDR", "ov-fs:9051"),
		ChunkSizeTokens:      512,
		ChunkOverlapTokens:   50,
		MaxIngestionSizeMb:   100,
		WatchEnabled:         true,
		WatchDefaultPollMs:   30000,
		WatchMaxTasks:        100,
		TreesitterEnabled:    true,
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
