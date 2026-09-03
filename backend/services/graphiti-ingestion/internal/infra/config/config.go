package config

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	KnowledgeAddr string `mapstructure:"KNOWLEDGE_ADDR"`
	StoreAddr     string `mapstructure:"STORE_ADDR"`
	PostgresURI   string `mapstructure:"POSTGRES_URI"`
	NatsURL       string `mapstructure:"NATS_URL"`
	Port          int    `mapstructure:"PORT"`
	HealthPort    int    `mapstructure:"HEALTH_PORT"`
}

func LoadConfig() (*Config, error) {
	viper.AutomaticEnv()
	
	viper.SetDefault("PORT", 9021)
	viper.SetDefault("HEALTH_PORT", 9094)

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.KnowledgeAddr == "" {
		return nil, errors.New("KNOWLEDGE_ADDR is required")
	}
	if cfg.StoreAddr == "" {
		return nil, errors.New("STORE_ADDR is required")
	}
	if cfg.PostgresURI == "" {
		return nil, errors.New("POSTGRES_URI is required")
	}

	return &cfg, nil
}
