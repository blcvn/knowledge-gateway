// Package supervisor manages the lifecycle of embedded services running as
// goroutines within the memobase monolith process.
//
// It provides phased startup (data→intelligence→application→gateway) and
// ordered shutdown (gateway→application→intelligence→data) with panic
// recovery per goroutine.
//
// Phase ordering ensures that downstream dependencies (e.g., ingestion for
// blob storage) are ready before upstream services (e.g., engine, context)
// attempt to connect.
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
	// PhaseData — Ingestion service starts first (blob storage, buffer zone).
	PhaseData Phase = 0
	// PhaseIntelligence — Engine service starts after ingestion is ready.
	PhaseIntelligence Phase = 1
	// PhaseApplication — Context and pipeline start after engine is ready.
	PhaseApplication Phase = 2
	// PhaseGateway — HTTP/MCP gateway starts after all gRPC services are ready.
	PhaseGateway Phase = 3
)

// ServiceSpec describes a single embedded service.
type ServiceSpec struct {
	Name    string                          // Human-readable service name (e.g. "memobase-ingestion")
	Phase   Phase                           // Startup phase
	Port    int                             // TCP port this service listens on (used for readiness probing)
	StartFn func(ctx context.Context) error // Blocking start function; returns when ctx is cancelled
}

// serviceState tracks runtime state for one embedded service.
type serviceState struct {
	spec   ServiceSpec
	cancel context.CancelFunc
	status string       // "registered", "starting", "serving", "stopping", "stopped", "failed"
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
//  1. Phase 0 (data): memobase-ingestion
//  2. Phase 1 (intelligence): memobase-engine
//  3. Phase 2 (application): memobase-context, memobase-pipeline
//  4. Phase 3 (gateway): vnp-gateway REST + MCP
//
// Each phase waits for TCP port readiness before proceeding.
// It blocks until ctx is cancelled or a fatal error is received.
func (s *Supervisor) StartAll(parentCtx context.Context) error {
	s.mu.Lock()
	// Sort by phase (ascending)
	sort.Slice(s.services, func(i, j int) bool {
		return s.services[i].spec.Phase < s.services[j].spec.Phase
	})
	s.mu.Unlock()

	// Group by phase
	phases := s.groupByPhase()

	// Sort phase keys ascending to ensure deterministic startup order
	phaseKeys := make([]int, 0, len(phases))
	for k := range phases {
		phaseKeys = append(phaseKeys, k)
	}
	sort.Ints(phaseKeys)

	for _, phase := range phaseKeys {
		group := phases[phase]
		s.logger.Info("starting phase",
			"phase", phase,
			"services", len(group),
			"names", serviceNames(group),
		)

		// Launch all services in this phase with independent contexts
		for _, svc := range group {
			s.startService(svc)
		}

		// Wait for all ports in this phase to be ready
		for _, svc := range group {
			if svc.spec.Port > 0 {
				addr := fmt.Sprintf("localhost:%d", svc.spec.Port)
				if err := waitForPort(addr, 30*time.Second); err != nil {
					return fmt.Errorf("service %s port %d not ready: %w",
						svc.spec.Name, svc.spec.Port, err)
				}
				s.logger.Info("service ready",
					"service", svc.spec.Name,
					"port", svc.spec.Port,
				)
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
// Gateway (Phase 3) stops first, then Application, Intelligence, Data.
func (s *Supervisor) Shutdown(timeout time.Duration) {
	s.logger.Info("supervisor shutdown starting", "timeout", timeout)

	s.mu.RLock()
	phases := s.groupByPhase()
	s.mu.RUnlock()

	// Shutdown in reverse phase order (gateway first, then app, intelligence, data)
	keys := make([]int, 0, len(phases))
	for k := range phases {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(keys)))

	for _, phase := range keys {
		group := phases[phase]
		s.logger.Info("shutting down phase",
			"phase", phase,
			"services", serviceNames(group),
		)

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
// Uses an independent context so Shutdown controls cancellation.
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

		s.logger.Info("starting service",
			"service", svc.spec.Name,
			"port", svc.spec.Port,
		)

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
		s.logger.Info("service stopped", "service", svc.spec.Name)
	}()
}

// groupByPhase returns services grouped by their phase.
func (s *Supervisor) groupByPhase() map[int][]*serviceState {
	groups := make(map[int][]*serviceState)
	for _, svc := range s.services {
		p := int(svc.spec.Phase)
		groups[p] = append(groups[p], svc)
	}
	return groups
}

// serviceNames extracts names from a slice of serviceState.
func serviceNames(svcs []*serviceState) []string {
	names := make([]string, len(svcs))
	for i, svc := range svcs {
		names[i] = svc.spec.Name
	}
	return names
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
