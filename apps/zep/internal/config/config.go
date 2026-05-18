package config

import (
	"os"
)

type UnifiedConfig struct {
	GatewayPort string
	UserPort    string
	ThreadPort  string
	MemoryPort  string
	GraphPort   string
	SearchPort  string
	AdminPort   string
}

func LoadConfig() *UnifiedConfig {
	return &UnifiedConfig{
		GatewayPort: getEnvOrDefault("ZEP_GATEWAY_PORT", "8080"),
		UserPort:    getEnvOrDefault("ZEP_USER_PORT", "9041"),
		ThreadPort:  getEnvOrDefault("ZEP_THREAD_PORT", "9042"),
		MemoryPort:  getEnvOrDefault("ZEP_MEMORY_PORT", "9043"),
		GraphPort:   getEnvOrDefault("ZEP_GRAPH_PORT", "9044"),
		SearchPort:  getEnvOrDefault("ZEP_SEARCH_PORT", "9045"),
		AdminPort:   getEnvOrDefault("ZEP_ADMIN_PORT", "9046"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
