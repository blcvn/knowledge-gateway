package usecase

import (
	"context"
	"log/slog"
	"math"
	"time"

	"vnp-memory/ov-search/internal/domain/model"
	"vnp-memory/ov-search/internal/domain/repository"
	"vnp-memory/ov-search/internal/usecase/port"
)

type hotnessUseCase struct {
	repo         repository.HotnessRepository
	decay        model.DecayConfig
	recomputeInt time.Duration
}

func NewHotnessUseCase(r repository.HotnessRepository, decay model.DecayConfig, interval time.Duration) port.HotnessUseCase {
	return &hotnessUseCase{
		repo:         r,
		decay:        decay,
		recomputeInt: interval,
	}
}

func (u *hotnessUseCase) Get(ctx context.Context, accountID string, paths []string) (map[string]float64, error) {
	scores, err := u.repo.GetMulti(ctx, accountID, paths)
	if err != nil {
		return nil, err
	}

	res := make(map[string]float64)
	for p, s := range scores {
		res[p] = s.ComputedHotness
	}
	return res, nil
}

func (u *hotnessUseCase) BoostSession(ctx context.Context, accountID string, paths []string) error {
	scores, err := u.repo.GetMulti(ctx, accountID, paths)
	if err != nil {
		return err
	}

	for _, s := range scores {
		s.SessionRefCount++
		// Recompute immediately on boost
		s.ComputedHotness = u.calculateHotness(s)
		s.UpdatedAt = time.Now()
		_ = u.repo.Save(ctx, s)
	}
	return nil
}

func (u *hotnessUseCase) StartWorker(ctx context.Context) {
	ticker := time.NewTicker(u.recomputeInt)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			u.recomputeAll(context.Background())
		}
	}
}

func (u *hotnessUseCase) recomputeAll(ctx context.Context) {
	slog.Info("Starting hotness recompute job")
	scores, err := u.repo.GetAll(ctx)
	if err != nil {
		slog.Error("Failed to fetch hotness scores for recompute", "err", err)
		return
	}

	for _, s := range scores {
		newHotness := u.calculateHotness(s)
		if math.Abs(newHotness-s.ComputedHotness) > 0.001 {
			s.ComputedHotness = newHotness
			s.UpdatedAt = time.Now()
			_ = u.repo.Save(ctx, s)
		}
	}
	slog.Info("Completed hotness recompute job")
}

func (u *hotnessUseCase) calculateHotness(s *model.HotnessScore) float64 {
	// H(t) = H_0 * exp(-lambda * dt) + session_boost
	lambda := math.Ln2 / u.decay.HalfLifeHours
	dt := time.Since(s.LastAccessedAt).Hours()
	
	decayed := s.BaseScore * math.Exp(-lambda*dt)
	boost := float64(s.SessionRefCount) * u.decay.SessionBoost
	
	return decayed + boost
}
