package postgres

import (
	"fmt"

	"kg-service/internal/config"
)

type Client struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime string
}

func New(cfg config.PostgresConfig) (Client, error) {
	if cfg.DSN() == "" {
		return Client{}, fmt.Errorf("postgres dsn must not be empty")
	}

	return Client{
		DSN:             cfg.DSN(),
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime.String(),
	}, nil
}
