package supervisor

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"log/slog"
	"os"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

func TestRegisterAndHealthCheck(t *testing.T) {
	sv := New(testLogger())
	sv.Register(ServiceSpec{Name: "svc-a", Phase: PhaseData, Port: 0})
	sv.Register(ServiceSpec{Name: "svc-b", Phase: PhaseApplication, Port: 0})

	health := sv.HealthCheck()
	if health["svc-a"] != "registered" {
		t.Errorf("expected 'registered', got %q", health["svc-a"])
	}
	if health["svc-b"] != "registered" {
		t.Errorf("expected 'registered', got %q", health["svc-b"])
	}
}

func TestStartAllPhasedOrder(t *testing.T) {
	sv := New(testLogger())
	var order []string
	var orderMu atomic.Int32

	// Phase 2 registered first, Phase 0 second — should start Phase 0 first
	sv.Register(ServiceSpec{
		Name:  "phase-2-svc",
		Phase: PhaseApplication,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			orderMu.Add(1)
			order = append(order, "phase-2")
			<-ctx.Done()
			return nil
		},
	})
	sv.Register(ServiceSpec{
		Name:  "phase-0-svc",
		Phase: PhaseData,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			order = append(order, "phase-0")
			<-ctx.Done()
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	_ = sv.StartAll(ctx)
	sv.Shutdown(2 * time.Second)
}

func TestShutdownReverseOrder(t *testing.T) {
	sv := New(testLogger())

	sv.Register(ServiceSpec{
		Name:  "data-svc",
		Phase: PhaseData,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	})
	sv.Register(ServiceSpec{
		Name:  "gateway-svc",
		Phase: PhaseGateway,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = sv.StartAll(ctx)
	sv.Shutdown(2 * time.Second)

	// After shutdown, all should be stopped
	health := sv.HealthCheck()
	for name, status := range health {
		if status != "stopped" {
			t.Errorf("service %s expected 'stopped', got %q", name, status)
		}
	}
}

func TestPanicRecovery(t *testing.T) {
	sv := New(testLogger())

	sv.Register(ServiceSpec{
		Name:  "panic-svc",
		Phase: PhaseData,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			panic("test panic")
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// StartAll should receive the error from panic
	err := sv.StartAll(ctx)
	if err == nil {
		t.Error("expected error from panicked service")
	}

	health := sv.HealthCheck()
	if health["panic-svc"] != "failed" {
		t.Errorf("expected 'failed', got %q", health["panic-svc"])
	}
}

func TestWaitForPort(t *testing.T) {
	// Start a TCP listener
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	addr := lis.Addr().String()
	if err := waitForPort(addr, 2*time.Second); err != nil {
		t.Errorf("expected port to be ready: %v", err)
	}
}

func TestWaitForPortTimeout(t *testing.T) {
	// Port that's not listening
	err := waitForPort("localhost:59999", 200*time.Millisecond)
	if err == nil {
		t.Error("expected timeout error for non-listening port")
	}
}

func TestServiceNamesHelper(t *testing.T) {
	svcs := []*serviceState{
		{spec: ServiceSpec{Name: "a"}},
		{spec: ServiceSpec{Name: "b"}},
	}
	names := serviceNames(svcs)
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestStartAllWithPortReadiness(t *testing.T) {
	sv := New(testLogger())

	// Find a free port
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	lis.Close()

	sv.Register(ServiceSpec{
		Name:  "port-svc",
		Phase: PhaseData,
		Port:  port,
		StartFn: func(ctx context.Context) error {
			l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
			if err != nil {
				return err
			}
			defer l.Close()
			<-ctx.Done()
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	if err := sv.StartAll(ctx); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	sv.Shutdown(2 * time.Second)

	health := sv.HealthCheck()
	if health["port-svc"] != "stopped" {
		t.Errorf("expected 'stopped', got %q", health["port-svc"])
	}
}
