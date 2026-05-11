package config

import (
	"os"
	"time"
)

type Config struct {
	StoreAddr        string
	RedisURL         string
	NatsURL          string
	CacheTTL         time.Duration
	RRFKValue        int
	MMRLambda        float64
	NodeDistanceWeight float64
	GRPCPort         string
	HealthPort       string
}

func LoadConfig() *Config {
	return &Config{
		StoreAddr:        getEnv("STORE_ADDR", "localhost:9024"),
		RedisURL:         getEnv("REDIS_URL", "localhost:6379"),
		NatsURL:          getEnv("NATS_URL", "nats://localhost:4222"),
		CacheTTL:         5 * time.Minute,
		RRFKValue:        60,
		MMRLambda:        0.7,
		NodeDistanceWeight: 0.5,
		GRPCPort:         ":9022",
		HealthPort:       ":9095",
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
