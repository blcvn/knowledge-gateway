package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
)

// mockDeleter implements EngineDeleter for testing.
type mockDeleter struct {
	name    string
	records int
	fail    bool
}

func (m *mockDeleter) Name() string { return m.name }

func (m *mockDeleter) Count(_ context.Context, _ string) (int, error) {
	if m.fail {
		return 0, fmt.Errorf("engine %s unavailable", m.name)
	}
	return m.records, nil
}

func (m *mockDeleter) Delete(_ context.Context, _ string) (int, error) {
	if m.fail {
		return 0, fmt.Errorf("engine %s unavailable", m.name)
	}
	deleted := m.records
	m.records = 0
	return deleted, nil
}

func TestGDPRService_Preview(t *testing.T) {
	engines := []EngineDeleter{
		&mockDeleter{name: "cognee", records: 10},
		&mockDeleter{name: "graphiti", records: 5},
		&mockDeleter{name: "memobase", records: 20},
	}

	svc := NewGDPRService(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	preview, err := svc.Preview(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("Preview failed: %v", err)
	}
	if preview.TotalRecords != 35 {
		t.Errorf("expected 35 total records, got %d", preview.TotalRecords)
	}
	if preview.DataSummary["cognee"] != 10 {
		t.Errorf("expected 10 cognee records, got %d", preview.DataSummary["cognee"])
	}
}

func TestGDPRService_Forget_Success(t *testing.T) {
	engines := []EngineDeleter{
		&mockDeleter{name: "cognee", records: 10},
		&mockDeleter{name: "graphiti", records: 5},
	}

	svc := NewGDPRService(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	result, err := svc.Forget(context.Background(), ForgetRequest{
		UserID:      "user-1",
		Reason:      "GDPR request",
		RequestedBy: "admin",
	})
	if err != nil {
		t.Fatalf("Forget failed: %v", err)
	}
	if result.Status != "completed" {
		t.Errorf("expected status=completed, got %s", result.Status)
	}
	if result.TotalDeleted != 15 {
		t.Errorf("expected 15 deleted, got %d", result.TotalDeleted)
	}
	if len(result.AffectedServices) != 2 {
		t.Errorf("expected 2 services, got %d", len(result.AffectedServices))
	}
}

func TestGDPRService_Forget_PartialFailure(t *testing.T) {
	engines := []EngineDeleter{
		&mockDeleter{name: "cognee", records: 10},
		&mockDeleter{name: "graphiti", records: 5, fail: true}, // Fails
		&mockDeleter{name: "memobase", records: 20},
	}

	svc := NewGDPRService(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	result, err := svc.Forget(context.Background(), ForgetRequest{
		UserID: "user-1",
		Reason: "GDPR request",
	})
	if err != nil {
		t.Fatalf("Forget should not return error on partial failure: %v", err)
	}
	if result.Status != "partial_failure" {
		t.Errorf("expected status=partial_failure, got %s", result.Status)
	}
	if result.TotalDeleted != 30 {
		t.Errorf("expected 30 deleted (cognee+memobase), got %d", result.TotalDeleted)
	}
}

func TestGDPRService_Forget_AllFail(t *testing.T) {
	engines := []EngineDeleter{
		&mockDeleter{name: "cognee", fail: true},
		&mockDeleter{name: "graphiti", fail: true},
	}

	svc := NewGDPRService(engines, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	result, err := svc.Forget(context.Background(), ForgetRequest{
		UserID: "user-1",
		Reason: "GDPR request",
	})
	if err != nil {
		t.Fatalf("Forget should not return error even on total failure: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("expected status=failed, got %s", result.Status)
	}
	if result.TotalDeleted != 0 {
		t.Errorf("expected 0 deleted, got %d", result.TotalDeleted)
	}
}
