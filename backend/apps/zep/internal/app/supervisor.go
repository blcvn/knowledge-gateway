package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Runnable interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Name() string
}

type Supervisor struct {
	components []Runnable
	mu         sync.Mutex
	timeout    time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewSupervisor(timeout time.Duration) *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &Supervisor{
		components: make([]Runnable, 0),
		timeout:    timeout,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (s *Supervisor) Register(c Runnable) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.components = append(s.components, c)
}

func (s *Supervisor) StartAll() error {
	for _, c := range s.components {
		slog.Info("[Supervisor] Starting component", "name", c.Name())
		if err := c.Start(s.ctx); err != nil {
			slog.Error("[Supervisor] Error starting component", "name", c.Name(), "error", err)
			return err
		}
	}
	return nil
}

func (s *Supervisor) WaitAndStop() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("[Supervisor] Shutdown signal received. Stopping components...")

	// Cancel the context to signal background tasks to shut down
	s.cancel()

	// Wait for grace period
	shutdownCtx, cancelWait := context.WithTimeout(context.Background(), s.timeout)
	defer cancelWait()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Optionally call Stop in reverse order if components have explicit Wait logic
	for i := len(s.components) - 1; i >= 0; i-- {
		c := s.components[i]
		
		errCh := make(chan error, 1)
		go func(comp Runnable) {
			errCh <- comp.Stop(shutdownCtx)
		}(c)

		select {
		case err := <-errCh:
			if err != nil {
				slog.Error("[Supervisor] Error stopping component", "name", c.Name(), "error", err)
			}
		case <-shutdownCtx.Done():
			slog.Warn("[Supervisor] Timeout while stopping component", "name", c.Name())
		}
	}
	
	slog.Info("[Supervisor] Shutdown complete")
}
