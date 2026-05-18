package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/vnp-community/vnp-memory/gateway/usecase/port"
)

// RateLimitUseCase checks and enforces per-tenant rate limits.
type RateLimitUseCase struct {
	store  port.RateLimitStore
	tiers  map[string]int // tier → requests per minute
	logger *slog.Logger
}

// NewRateLimitUseCase creates a new RateLimitUseCase.
func NewRateLimitUseCase(store port.RateLimitStore, logger *slog.Logger) *RateLimitUseCase {
	return &RateLimitUseCase{
		store: store,
		tiers: map[string]int{
			"free":       60,
			"pro":        600,
			"enterprise": 6000,
		},
		logger: logger,
	}
}

// Check verifies if the request is within the rate limit for the given tenant and endpoint.
func (uc *RateLimitUseCase) Check(ctx context.Context, tenantID, endpoint string) (bool, int, error) {
	tier := "free" // Default tier; real implementation looks up from AuthContext
	limit, ok := uc.tiers[tier]
	if !ok {
		limit = uc.tiers["free"]
	}

	key := fmt.Sprintf("rl:%s:%s", tenantID, endpoint)
	allowed, remaining, err := uc.store.CheckAndIncrement(ctx, key, limit, 60)
	if err != nil {
		// Fail-open: allow request if rate limit store is unavailable
		uc.logger.Error("rate limit check failed, fail-open", "error", err, "tenant", tenantID)
		return true, limit, nil
	}

	if !allowed {
		uc.logger.Warn("rate limit exceeded", "tenant", tenantID, "endpoint", endpoint, "tier", tier)
	}

	return allowed, remaining, nil
}

// CheckWithTier verifies the rate limit using a specific tier.
func (uc *RateLimitUseCase) CheckWithTier(ctx context.Context, tenantID, endpoint, tier string) (bool, int, error) {
	limit, ok := uc.tiers[tier]
	if !ok {
		limit = uc.tiers["free"]
	}

	key := fmt.Sprintf("rl:%s:%s", tenantID, endpoint)
	return uc.store.CheckAndIncrement(ctx, key, limit, 60)
}
