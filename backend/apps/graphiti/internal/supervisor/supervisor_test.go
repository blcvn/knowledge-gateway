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

// getFreePort returns an available TCP port on localhost.
func getFreePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	port := lis.Addr().(*net.TCPAddr).Port
	lis.Close()
	return port
}

// makeListenerStartFn creates a StartFn that opens a TCP listener on the given port,
// records its name into startOrder, then blocks until ctx is done.
func makeListenerStartFn(name string, port int, mu *sync.Mutex, startOrder *[]string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			return err
		}
		defer lis.Close()

		mu.Lock()
		*startOrder = append(*startOrder, name)
		mu.Unlock()

		<-ctx.Done()
		return nil
	}
}

func TestRegisterAndStartPhaseOrder(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	sv := New(logger)

	var mu sync.Mutex
	startOrder := make([]string, 0, 2)

	portStore := getFreePort(t)
	portGW := getFreePort(t)

	// Register in reverse order to test sorting
	sv.Register(ServiceSpec{
		Name:    "gateway",
		Phase:   PhaseGateway,
		Port:    portGW,
		StartFn: makeListenerStartFn("gateway", portGW, &mu, &startOrder),
	})
	sv.Register(ServiceSpec{
		Name:    "store",
		Phase:   PhaseData,
		Port:    portStore,
		StartFn: makeListenerStartFn("store", portStore, &mu, &startOrder),
	})

	ctx, cancel := context.WithCancel(context.Background())
	startDone := make(chan error, 1)
	go func() {
		startDone <- sv.StartAll(ctx)
	}()

	// Wait for both services to become ready
	time.Sleep(2 * time.Second)

	mu.Lock()
	defer mu.Unlock()

	if len(startOrder) < 2 {
		t.Fatalf("expected 2 services started, got %d", len(startOrder))
	}
	// Store (PhaseData=0) should start before gateway (PhaseGateway=3)
	if startOrder[0] != "store" {
		t.Errorf("expected store first, got %v", startOrder)
	}
	if startOrder[1] != "gateway" {
		t.Errorf("expected gateway second, got %v", startOrder)
	}

	// Cancel parent context to unblock StartAll, then Shutdown cancels goroutines
	cancel()
	<-startDone
	sv.Shutdown(2 * time.Second)
}

func TestMultiPhaseOrder(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	sv := New(logger)

	var mu sync.Mutex
	startOrder := make([]string, 0, 4)

	portData := getFreePort(t)
	portIntel := getFreePort(t)
	portApp := getFreePort(t)
	portGW := getFreePort(t)

	// Register all 4 phases in mixed order
	sv.Register(ServiceSpec{
		Name: "app", Phase: PhaseApplication, Port: portApp,
		StartFn: makeListenerStartFn("app", portApp, &mu, &startOrder),
	})
	sv.Register(ServiceSpec{
		Name: "gw", Phase: PhaseGateway, Port: portGW,
		StartFn: makeListenerStartFn("gw", portGW, &mu, &startOrder),
	})
	sv.Register(ServiceSpec{
		Name: "data", Phase: PhaseData, Port: portData,
		StartFn: makeListenerStartFn("data", portData, &mu, &startOrder),
	})
	sv.Register(ServiceSpec{
		Name: "intel", Phase: PhaseIntelligence, Port: portIntel,
		StartFn: makeListenerStartFn("intel", portIntel, &mu, &startOrder),
	})

	ctx, cancel := context.WithCancel(context.Background())
	startDone := make(chan error, 1)
	go func() {
		startDone <- sv.StartAll(ctx)
	}()

	time.Sleep(3 * time.Second)

	mu.Lock()
	order := make([]string, len(startOrder))
	copy(order, startOrder)
	mu.Unlock()

	if len(order) != 4 {
		t.Fatalf("expected 4 services, got %d: %v", len(order), order)
	}
	// Phase order: data(0) → intel(1) → app(2) → gw(3)
	if order[0] != "data" {
		t.Errorf("phase 0 should be 'data', got %q in order %v", order[0], order)
	}
	if order[1] != "intel" {
		t.Errorf("phase 1 should be 'intel', got %q in order %v", order[1], order)
	}
	if order[2] != "app" {
		t.Errorf("phase 2 should be 'app', got %q in order %v", order[2], order)
	}
	if order[3] != "gw" {
		t.Errorf("phase 3 should be 'gw', got %q in order %v", order[3], order)
	}

	cancel()
	<-startDone
	sv.Shutdown(2 * time.Second)
}

func TestHealthCheck(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	sv := New(logger)

	sv.Register(ServiceSpec{
		Name:  "test-svc",
		Phase: PhaseData,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	})

	status := sv.HealthCheck()
	if status["test-svc"] != "registered" {
		t.Errorf("expected 'registered', got %q", status["test-svc"])
	}
}

func TestPanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	sv := New(logger)

	sv.Register(ServiceSpec{
		Name:  "panic-svc",
		Phase: PhaseData,
		Port:  0,
		StartFn: func(ctx context.Context) error {
			panic("test panic")
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := sv.StartAll(ctx)
	if err == nil {
		t.Error("expected error from panicking service")
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
		t.Errorf("waitForPort should succeed for open port: %v", err)
	}

	// Test timeout on closed port
	lis.Close()
	err = waitForPort(fmt.Sprintf("localhost:%d", 19999), 500*time.Millisecond)
	if err == nil {
		t.Error("waitForPort should timeout on closed port")
	}
}

func TestShutdownCompletes(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	sv := New(logger)

	portStore := getFreePort(t)
	portGW := getFreePort(t)

	sv.Register(ServiceSpec{
		Name: "store", Phase: PhaseData, Port: portStore,
		StartFn: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", fmt.Sprintf(":%d", portStore))
			if err != nil {
				return err
			}
			defer lis.Close()
			<-ctx.Done()
			return nil
		},
	})
	sv.Register(ServiceSpec{
		Name: "gw", Phase: PhaseGateway, Port: portGW,
		StartFn: func(ctx context.Context) error {
			lis, err := net.Listen("tcp", fmt.Sprintf(":%d", portGW))
			if err != nil {
				return err
			}
			defer lis.Close()
			<-ctx.Done()
			return nil
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	startDone := make(chan error, 1)
	go func() {
		startDone <- sv.StartAll(ctx)
	}()

	time.Sleep(2 * time.Second)
	cancel()
	<-startDone

	// Shutdown should complete within timeout, not hang
	done := make(chan struct{})
	go func() {
		sv.Shutdown(5 * time.Second)
		close(done)
	}()

	select {
	case <-done:
		// Verify all services show stopped/stopping status
		status := sv.HealthCheck()
		for name, s := range status {
			if s != "stopped" && s != "stopping" {
				t.Errorf("service %s should be stopped/stopping, got %q", name, s)
			}
		}
	case <-time.After(10 * time.Second):
		t.Fatal("shutdown timed out — possible deadlock")
	}
}
