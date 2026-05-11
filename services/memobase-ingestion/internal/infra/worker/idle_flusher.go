package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/domain/repository"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/dto"
	"github.com/vnp-community/vnp-memory/services/memobase-ingestion/internal/usecase/port"
)

type IdleFlusherWorker struct {
	bufferRepo repository.BufferZoneRepository
	flusher    port.BufferFlusher
	interval   time.Duration
	idleTime   time.Duration
}

func NewIdleFlusherWorker(
	bufferRepo repository.BufferZoneRepository,
	flusher port.BufferFlusher,
	interval time.Duration,
	idleTime time.Duration,
) *IdleFlusherWorker {
	return &IdleFlusherWorker{
		bufferRepo: bufferRepo,
		flusher:    flusher,
		interval:   interval,
		idleTime:   idleTime,
	}
}

func (w *IdleFlusherWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("IdleFlusherWorker stopping", "reason", ctx.Err())
			return
		case <-ticker.C:
			w.processIdleEntries(ctx)
		}
	}
}

func (w *IdleFlusherWorker) processIdleEntries(ctx context.Context) {
	thresholdTime := time.Now().Add(-w.idleTime)
	entries, err := w.bufferRepo.GetIdleSince(ctx, thresholdTime)
	if err != nil {
		slog.Error("Failed to fetch idle entries", "error", err)
		return
	}

	// Group by user and project
	grouped := make(map[string]map[string]bool)
	for _, e := range entries {
		if grouped[e.ProjectID] == nil {
			grouped[e.ProjectID] = make(map[string]bool)
		}
		grouped[e.ProjectID][e.UserID] = true
	}

	for pID, users := range grouped {
		for uID := range users {
			// Trigger flush for this user/project
			_, err := w.flusher.FlushBuffer(ctx, &dto.FlushBufferRequest{
				UserID:    uID,
				ProjectID: pID,
			})
			if err != nil {
				slog.Error("Failed to flush idle entries", "project_id", pID, "user_id", uID, "error", err)
			}
		}
	}
}
