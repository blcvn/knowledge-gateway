package usecase

import (
	"context"
	"log"
	"time"
)

type SagaWorker struct {
	repo         SagaStateRepo
	orchestrator *SagaOrchestrator
	ticker       *time.Ticker
	quit         chan struct{}
}

func NewSagaWorker(repo SagaStateRepo, orch *SagaOrchestrator, interval time.Duration) *SagaWorker {
	return &SagaWorker{
		repo:         repo,
		orchestrator: orch,
		ticker:       time.NewTicker(interval),
		quit:         make(chan struct{}),
	}
}

func (w *SagaWorker) Start() {
	go func() {
		for {
			select {
			case <-w.ticker.C:
				w.resumeStuckSagas()
			case <-w.quit:
				w.ticker.Stop()
				return
			}
		}
	}()
}

func (w *SagaWorker) Stop() {
	close(w.quit)
}

func (w *SagaWorker) resumeStuckSagas() {
	ctx := context.Background()
	
	// Sagas stuck for more than 5 minutes
	states, err := w.repo.GetStuckSagas(ctx, 5, 50)
	if err != nil {
		log.Printf("Failed to get stuck sagas: %v", err)
		return
	}

	for _, state := range states {
		log.Printf("Resuming stuck saga %s from step %s", state.ID, state.CurrentStep)
		
		// In a real system, you'd load the original Episode from the DB here
		// For the sake of this scaffold, we trigger compensate immediately or attempt resume
		// if we have the Episode payload. Since we don't load Episode here, 
		// we'll fail it for safety to trigger rollback.
		w.orchestrator.compensateSaga(ctx, state, context.DeadlineExceeded)
	}
}
