package config

import (
	"os"
	"strconv"
)

// ServicesConfig defines ports and addresses for embedded services
type ServicesConfig struct {
	DocumentPort  int
	MemoryPort    int
	SearchPort    int
	ProfilePort   int
	ConnectorPort int
	MCPPort       int
	AuthPort      int
	AnalyticsPort int
	ProjectPort   int
	EnginePort    int
	GatewayPort   int
}

// Config represents the unified configuration for the monolith
type Config struct {
	Services ServicesConfig
	DatabaseURL string
	RedisAddr   string
	NATSURL     string
}

// Load reads the configuration from environment variables
func Load() (*Config, error) {
	return &Config{
		Services: ServicesConfig{
			DocumentPort:  getEnvAsInt("SM_DOCUMENT_PORT", 9001),
			MemoryPort:    getEnvAsInt("SM_MEMORY_PORT", 9002),
			SearchPort:    getEnvAsInt("SM_SEARCH_PORT", 9003),
			ProfilePort:   getEnvAsInt("SM_PROFILE_PORT", 9004),
			ConnectorPort: getEnvAsInt("SM_CONNECTOR_PORT", 9005),
			MCPPort:       getEnvAsInt("SM_MCP_PORT", 9006),
			AuthPort:      getEnvAsInt("SM_AUTH_PORT", 9007),
			AnalyticsPort: getEnvAsInt("SM_ANALYTICS_PORT", 9008),
			ProjectPort:   getEnvAsInt("SM_PROJECT_PORT", 9009),
			EnginePort:    getEnvAsInt("SM_ENGINE_PORT", 9010),
			GatewayPort:   getEnvAsInt("GATEWAY_PORT", 8080),
		},
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisAddr:   os.Getenv("REDIS_ADDR"),
		NATSURL:     os.Getenv("NATS_URL"),
	}, nil
}

func getEnvAsInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}
