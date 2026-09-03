// Package supervisor manages the lifecycle of embedded services running as
// goroutines within the cognee monolith process.
//
// It provides phased startup (services first, gateway second) and ordered
// shutdown (gateway first, services second) with panic recovery per goroutine.
package supervisor

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"sync"
	"time"
)

// Phase controls the startup order of services.
// Lower-phase services start first and stop last.
type Phase int

const (
	// PhaseInfra — gRPC services start first so they are ready before gateway.
	PhaseInfra Phase = 0
	// PhaseGateway — HTTP gateway starts after all infra services are accepting connections.
	PhaseGateway Phase = 1
)

// ServiceSpec describes a single embedded service.
type ServiceSpec struct {
	Name    string                          // Human-readable service name (e.g. "cognee-ingestion")
	Phase   Phase                           // Startup phase (PhaseInfra or PhaseGateway)
	Port    int                             // TCP port this service listens on (used for readiness probing)
	StartFn func(ctx context.Context) error // Blocking start function; returns when ctx is cancelled
}

// serviceState tracks runtime state for one embedded service.
type serviceState struct {
	spec   ServiceSpec
	cancel context.CancelFunc
	status string       // "starting", "serving", "stopping", "stopped", "failed"
	err    error
	done   chan struct{} // closed when goroutine exits
}

// Supervisor orchestrates multiple embedded services.
type Supervisor struct {
	mu       sync.RWMutex
	services []*serviceState
	logger   *slog.Logger
	errors   chan error
}

// New creates a Supervisor.
func New(logger *slog.Logger) *Supervisor {
	return &Supervisor{
		logger: logger,
		errors: make(chan error, 16),
	}
}

// Register adds a service to the supervisor. Must be called before StartAll.
func (s *Supervisor) Register(spec ServiceSpec) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services = append(s.services, &serviceState{
		spec:   spec,
		status: "registered",
		done:   make(chan struct{}),
	})
}

// setStatus safely updates the status of a service state.
func (s *Supervisor) setStatus(svc *serviceState, status string) {
	s.mu.Lock()
	svc.status = status
	s.mu.Unlock()
}

// StartAll launches services in phase order:
//  1. Start all Phase 0 (infra/gRPC) services and wait for their ports to accept TCP.
//  2. Start all Phase 1 (gateway) services and wait for their ports.
//
// Services get independent contexts (not derived from parentCtx) so that
// Shutdown can cancel them in reverse phase order.
// parentCtx is only used to signal StartAll to stop blocking.
func (s *Supervisor) StartAll(parentCtx context.Context) error {
	s.mu.Lock()
	sort.Slice(s.services, func(i, j int) bool {
		return s.services[i].spec.Phase < s.services[j].spec.Phase
	})
	s.mu.Unlock()

	phases := s.groupByPhase()

	for phase, group := range phases {
		s.logger.Info("starting phase", "phase", phase, "services", len(group))

		for _, svc := range group {
			// Each service gets its own independent context (not derived from parentCtx)
			// so that Shutdown() can cancel them individually in reverse phase order.
			s.startService(svc)
		}

		for _, svc := range group {
			if svc.spec.Port > 0 {
				addr := fmt.Sprintf("localhost:%d", svc.spec.Port)
				if err := waitForPort(addr, 30*time.Second); err != nil {
					return fmt.Errorf("service %s port %d not ready: %w", svc.spec.Name, svc.spec.Port, err)
				}
				s.logger.Info("service ready", "service", svc.spec.Name, "port", svc.spec.Port)
				s.setStatus(svc, "serving")
			}
		}

		s.logger.Info("phase ready", "phase", phase)
	}

	s.logger.Info("all services started")

	// Block until parentCtx signals stop or fatal error
	select {
	case <-parentCtx.Done():
		return nil
	case err := <-s.errors:
		return fmt.Errorf("service error: %w", err)
	}
}

// Shutdown stops services in reverse phase order with the given timeout.
// Gateway (Phase 1) stops first, then infrastructure services (Phase 0).
// Each phase's contexts are cancelled, then we wait for their goroutines to exit
// before proceeding to the previous phase.
func (s *Supervisor) Shutdown(timeout time.Duration) {
	s.logger.Info("supervisor shutdown starting", "timeout", timeout)

	s.mu.RLock()
	phases := s.groupByPhaseUnsafe()
	s.mu.RUnlock()

	keys := make([]int, 0, len(phases))
	for k := range phases {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))

	for _, phase := range keys {
		group := phases[phase]
		s.logger.Info("shutting down phase", "phase", phase, "services", len(group))

		// Cancel all services in this phase
		for _, svc := range group {
			s.setStatus(svc, "stopping")
			if svc.cancel != nil {
				svc.cancel()
			}
		}

		// Wait for each service in this phase to finish before moving to next phase
		phaseDeadline := time.After(timeout)
		for _, svc := range group {
			select {
			case <-svc.done:
				// service goroutine exited
			case <-phaseDeadline:
				s.logger.Warn("service shutdown timed out", "service", svc.spec.Name)
			}
		}

		s.logger.Info("phase stopped", "phase", phase)
	}

	s.logger.Info("supervisor shutdown complete")
}

// HealthCheck returns the status of each embedded service.
func (s *Supervisor) HealthCheck() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]string, len(s.services))
	for _, svc := range s.services {
		result[svc.spec.Name] = svc.status
	}
	return result
}

// startService launches a single service in a goroutine with panic recovery.
// Uses an independent context (not derived from any parent) so Shutdown controls cancellation.
func (s *Supervisor) startService(svc *serviceState) {
	ctx, cancel := context.WithCancel(context.Background())
	svc.cancel = cancel
	s.setStatus(svc, "starting")

	go func() {
		defer close(svc.done)
		defer func() {
			if r := recover(); r != nil {
				s.mu.Lock()
				svc.status = "failed"
				svc.err = fmt.Errorf("panic: %v", r)
				s.mu.Unlock()
				s.logger.Error("service panicked",
					"service", svc.spec.Name,
					"error", r,
				)
				s.errors <- svc.err
			}
		}()

		s.logger.Info("starting service", "service", svc.spec.Name, "port", svc.spec.Port)

		if err := svc.spec.StartFn(ctx); err != nil {
			s.mu.Lock()
			svc.status = "failed"
			svc.err = err
			s.mu.Unlock()
			s.logger.Error("service failed",
				"service", svc.spec.Name,
				"error", err,
			)
			s.errors <- err
			return
		}

		s.setStatus(svc, "stopped")
	}()
}

// groupByPhase returns services grouped by their phase.
func (s *Supervisor) groupByPhase() map[int][]*serviceState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.groupByPhaseUnsafe()
}

// groupByPhaseUnsafe returns services grouped by phase without locking.
func (s *Supervisor) groupByPhaseUnsafe() map[int][]*serviceState {
	groups := make(map[int][]*serviceState)
	for _, svc := range s.services {
		p := int(svc.spec.Phase)
		groups[p] = append(groups[p], svc)
	}
	return groups
}

// waitForPort polls a TCP address until it accepts a connection or timeout expires.
func waitForPort(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("port %s not ready after %v", addr, timeout)
}
