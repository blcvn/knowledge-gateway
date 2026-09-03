package config

import "os"

// Config holds vnp-admin service configuration.
type Config struct {
	GRPCPort    string
	HealthPort  string
	DatabaseURL string
	RedisURL    string
	NatsURL     string
}

func Load() *Config {
	return &Config{
		GRPCPort:    getEnv("GRPC_PORT", "9050"),
		HealthPort:  getEnv("HEALTH_PORT", "9100"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://vnp:vnppassword@localhost:5432/vnp_memory?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379/0"),
		NatsURL:     getEnv("NATS_URL", "nats://localhost:4222"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
