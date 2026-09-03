package config

import "os"

// Config holds vnp-search-hub service configuration.
type Config struct {
	GRPCPort   string
	HealthPort string
}

func Load() *Config {
	return &Config{
		GRPCPort:   getEnv("GRPC_PORT", "9042"),
		HealthPort: getEnv("HEALTH_PORT", "9102"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
