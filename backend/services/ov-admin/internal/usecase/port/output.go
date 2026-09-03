package port

import (
	"context"
)

type HashPort interface {
	HashPassword(ctx context.Context, password string) (string, error)
	ComparePassword(ctx context.Context, hash, password string) (bool, error)
}

type HealthCheckerPort interface {
	CheckHealth(ctx context.Context) (map[string]string, error)
}
