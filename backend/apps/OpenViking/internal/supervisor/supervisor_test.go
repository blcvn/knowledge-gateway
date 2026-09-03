package supervisor

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"
)

func TestSupervisor_RunSuccess(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	sv := New(logger)

	sv.Register(ServiceSpec{
		Name:  "test-success",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			timer := time.NewTimer(10 * time.Millisecond)
			select {
			case <-ctx.Done():
				return nil
			case <-timer.C:
				return nil
			}
		},
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- sv.Run()
	}()

	// Since the task completes immediately, supervisor Run should also complete
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected nil error, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Test timed out")
	}
}

func TestSupervisor_RunError(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	sv := New(logger)

	expectedErr := errors.New("simulated failure")

	sv.Register(ServiceSpec{
		Name:  "test-failure",
		Phase: 1,
		StartFn: func(ctx context.Context) error {
			return expectedErr
		},
	})

	err := sv.Run()
	if err == nil {
		t.Error("Expected error from Run, got nil")
	}
}
