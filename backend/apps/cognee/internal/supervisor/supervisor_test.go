package supervisor

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
}

// fakeService creates a StartFn that listens on a TCP port until ctx is cancelled.
func fakeService(t *testing.T, port int) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return err
		}
		defer lis.Close()

		go func() {
			for {
				conn, err := lis.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}()

		<-ctx.Done()
		return nil
	}
}

func TestSupervisor_PhasedStartup(t *testing.T) {
	sv := New(testLogger())
	var mu sync.Mutex
	var order []string

	sv.Register(ServiceSpec{
		Name:  "svc-a",
		Phase: PhaseInfra,
		Port:  19401,
		StartFn: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "svc-a")
			mu.Unlock()
			return fakeService(t, 19401)(ctx)
		},
	})

	sv.Register(ServiceSpec{
		Name:  "gateway",
		Phase: PhaseGateway,
		Port:  19402,
		StartFn: func(ctx context.Context) error {
			mu.Lock()
			order = append(order, "gateway")
			mu.Unlock()
			return fakeService(t, 19402)(ctx)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(2 * time.Second)
		cancel() // unblock StartAll
	}()

	_ = sv.StartAll(ctx)
	sv.Shutdown(5 * time.Second)

	mu.Lock()
	defer mu.Unlock()
	if len(order) >= 2 && order[0] != "svc-a" {
		t.Errorf("expected svc-a to start first, got %v", order)
	}
}

func TestSupervisor_HealthCheck(t *testing.T) {
	sv := New(testLogger())
	sv.Register(ServiceSpec{
		Name:    "test-svc",
		Phase:   PhaseInfra,
		Port:    19403,
		StartFn: fakeService(t, 19403),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = sv.StartAll(ctx)
	}()

	time.Sleep(1 * time.Second)

	status := sv.HealthCheck()
	if s, ok := status["test-svc"]; !ok || (s != "serving" && s != "starting") {
		t.Errorf("expected test-svc to be serving/starting, got %q", s)
	}

	cancel()
	sv.Shutdown(2 * time.Second)
}

func TestSupervisor_PanicRecovery(t *testing.T) {
	sv := New(testLogger())

	sv.Register(ServiceSpec{
		Name:  "panic-svc",
		Phase: PhaseInfra,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			panic("intentional test panic")
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := sv.StartAll(ctx)
	if err == nil {
		t.Log("StartAll returned nil (panic caught in goroutine)")
	}
	sv.Shutdown(1 * time.Second)
}

func TestWaitForPort(t *testing.T) {
	lis, err := net.Listen("tcp", ":19404")
	if err != nil {
		t.Fatal(err)
	}
	defer lis.Close()

	if err := waitForPort("localhost:19404", 2*time.Second); err != nil {
		t.Errorf("expected port to be ready: %v", err)
	}

	if err := waitForPort("localhost:19499", 500*time.Millisecond); err == nil {
		t.Error("expected timeout for non-existent port")
	}
}

func TestSupervisor_ShutdownOrder(t *testing.T) {
	sv := New(testLogger())
	var mu sync.Mutex
	var stopOrder []string

	// Phase 0: infra service
	sv.Register(ServiceSpec{
		Name:  "infra-svc",
		Phase: PhaseInfra,
		Port:  19405,
		StartFn: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", ":19405")
			if err != nil {
				return err
			}
			defer lis.Close()
			go func() {
				for {
					conn, err := lis.Accept()
					if err != nil {
						return
					}
					conn.Close()
				}
			}()
			<-ctx.Done()
			mu.Lock()
			stopOrder = append(stopOrder, "infra-svc")
			mu.Unlock()
			return nil
		},
	})

	// Phase 1: gateway
	sv.Register(ServiceSpec{
		Name:  "gw",
		Phase: PhaseGateway,
		Port:  19406,
		StartFn: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", ":19406")
			if err != nil {
				return err
			}
			defer lis.Close()
			go func() {
				for {
					conn, err := lis.Accept()
					if err != nil {
						return
					}
					conn.Close()
				}
			}()
			<-ctx.Done()
			mu.Lock()
			stopOrder = append(stopOrder, "gw")
			mu.Unlock()
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = sv.StartAll(ctx)
	}()

	// Wait for both services to be ready
	time.Sleep(1500 * time.Millisecond)

	// Cancel the parent ctx just to unblock StartAll
	cancel()
	// Let StartAll return
	time.Sleep(100 * time.Millisecond)

	// Shutdown does the actual ordered cancellation:
	// Phase 1 (gw) cancelled first → wait for done → Phase 0 (infra) cancelled
	sv.Shutdown(5 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if len(stopOrder) < 2 {
		t.Fatalf("expected 2 stop events, got %d: %v", len(stopOrder), stopOrder)
	}
	if stopOrder[0] != "gw" {
		t.Errorf("expected gateway to stop first, got %v", stopOrder)
	}
	if stopOrder[1] != "infra-svc" {
		t.Errorf("expected infra-svc to stop second, got %v", stopOrder)
	}
}
