package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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

func (c Client) Open() (*sql.DB, error) {
	db, err := sql.Open("pgx", c.DSN)
	if err != nil {
		return nil, err
	}
	if c.MaxOpenConns > 0 {
		db.SetMaxOpenConns(c.MaxOpenConns)
	}
	if c.MaxIdleConns > 0 {
		db.SetMaxIdleConns(c.MaxIdleConns)
	}
	if c.ConnMaxLifetime != "" {
		if d, err := time.ParseDuration(c.ConnMaxLifetime); err == nil {
			db.SetConnMaxLifetime(d)
		}
	}
	if err := db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
