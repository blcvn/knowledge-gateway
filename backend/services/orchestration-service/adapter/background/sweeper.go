package background

import (
	"context"
	"log"
	"time"

	"vnp-memory/services/orchestration-service/internal/orchestration"
)

type BackgroundSweeper struct {
	leases      *orchestration.LeaseService
	signals     *orchestration.SignalService
	sentinels   *orchestration.SentinelService
	sketches    *orchestration.SketchService
	checkpoints *orchestration.CheckpointService
}

func NewBackgroundSweeper(
	leases *orchestration.LeaseService,
	signals *orchestration.SignalService,
	sentinels *orchestration.SentinelService,
	sketches *orchestration.SketchService,
	checkpoints *orchestration.CheckpointService,
) *BackgroundSweeper {
	return &BackgroundSweeper{leases, signals, sentinels, sketches, checkpoints}
}

func (s *BackgroundSweeper) Start(ctx context.Context) {
	log.Println("[orchestration] background sweeper started")
	go s.runEvery(ctx, 60*time.Second, func() { s.leases.SweepExpired(ctx) })
	go s.runEvery(ctx, 300*time.Second, func() { s.signals.DeleteExpired(ctx) })
	go s.runEvery(ctx, 30*time.Second, func() { s.sentinels.EvaluateAll(ctx) })
	go s.runEvery(ctx, 1*time.Hour, func() { s.sketches.ReapExpired(ctx) })
	go s.runEvery(ctx, 1*time.Hour, func() { s.checkpoints.AutoRejectExpired(ctx) })
}

func (s *BackgroundSweeper) runEvery(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			fn()
		case <-ctx.Done():
			return
		}
	}
}
