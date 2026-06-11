package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port         int
	DatabaseDSN  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:         8080,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	if v := os.Getenv("PORT"); v != "" {
		port, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("parse PORT: %w", err)
		}
		cfg.Port = port
	}
	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.DatabaseDSN = v
	}
	if cfg.DatabaseDSN == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}
