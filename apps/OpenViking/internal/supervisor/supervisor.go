package supervisor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

type StartFn func(ctx context.Context) error

type ServiceSpec struct {
	Name    string
	Phase   int
	StartFn StartFn
}

type Supervisor struct {
	services []ServiceSpec
	logger   *slog.Logger
}

func New(logger *slog.Logger) *Supervisor {
	return &Supervisor{
		logger: logger,
	}
}

func (s *Supervisor) Register(spec ServiceSpec) {
	s.services = append(s.services, spec)
}

func (s *Supervisor) Run() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	g, gCtx := errgroup.WithContext(ctx)

	// In a real implementation we would start Phase 1, wait for readiness, then Phase 2, etc.
	// For simplicity, we just start them all as goroutines and let them run.
	for _, svc := range s.services {
		svc := svc
		g.Go(func() error {
			s.logger.Info("starting service", "name", svc.Name, "phase", svc.Phase)
			if err := svc.StartFn(gCtx); err != nil {
				s.logger.Error("service failed", "name", svc.Name, "error", err)
				return fmt.Errorf("service %s failed: %w", svc.Name, err)
			}
			s.logger.Info("service exited", "name", svc.Name)
			return nil
		})
	}

	s.logger.Info("supervisor is waiting for services or interrupt")
	if err := g.Wait(); err != nil {
		s.logger.Error("supervisor stopped with error", "error", err)
		return err
	}

	s.logger.Info("supervisor stopped cleanly")
	return nil
}
