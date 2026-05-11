package usecase

import (
	"context"

	"github.com/vnp-community/vnp-memory/services/ov-admin/internal/usecase/port"
)

type healthUseCase struct {
	checker port.HealthCheckerPort
}

func NewHealthUseCase(checker port.HealthCheckerPort) port.HealthUseCase {
	return &healthUseCase{checker: checker}
}

func (u *healthUseCase) GetHealth(ctx context.Context) (map[string]string, error) {
	return u.checker.CheckHealth(ctx)
}
