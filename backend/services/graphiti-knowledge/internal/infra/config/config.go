package config

import (
	"errors"
	"net/url"
	"os"
	"strings"
)

type Config struct {
	LLMProvider     string
	LLMModel        string
	LLMAPIKey       string
	EmbedderURL     string
	EmbedderAPIKey  string
	StoreAddr       string
	NatsURL         string
	GRPCPort        string
	HealthPort      string
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		LLMProvider:    getEnv("LLM_PROVIDER", "bifrost"),
		LLMModel:       getEnv("LLM_MODEL", "gpt-4o"),
		LLMAPIKey:      os.Getenv("LLM_API_KEY"),
		EmbedderURL:    getEnv("EMBEDDER_URL", "http://bifrost:8443/v1/embeddings"),
		EmbedderAPIKey: os.Getenv("EMBEDDER_API_KEY"),
		StoreAddr:      os.Getenv("STORE_ADDR"),
		NatsURL:        getEnv("NATS_URL", "nats://nats:4222"),
		GRPCPort:       getEnv("GRPC_PORT", "9023"),
		HealthPort:     getEnv("HEALTH_PORT", "9096"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.LLMAPIKey) == "" {
		return errors.New("LLM_API_KEY is required")
	}
	if len(c.LLMAPIKey) < 10 {
		return errors.New("LLM_API_KEY is too short, invalid format")
	}

	if strings.TrimSpace(c.StoreAddr) == "" {
		return errors.New("STORE_ADDR is required")
	}

	if _, err := url.ParseRequestURI(c.EmbedderURL); err != nil {
		return errors.New("EMBEDDER_URL is an invalid URL format")
	}

	if _, err := url.ParseRequestURI(c.NatsURL); err != nil {
		return errors.New("NATS_URL is an invalid URL format")
	}

	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
