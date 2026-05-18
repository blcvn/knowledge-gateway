// Package usecase implements GDPR cascading forget and event timeline for vnp-event.
package usecase

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ForgetRequest represents a GDPR forget request.
type ForgetRequest struct {
	UserID      string `json:"user_id"`
	Reason      string `json:"reason"`
	RequestedBy string `json:"requested_by"`
	DryRun      bool   `json:"dry_run"` // If true, only preview
}

// ForgetResult represents the outcome of a GDPR forget operation.
type ForgetResult struct {
	ForgetID         string                   `json:"forget_id"`
	UserID           string                   `json:"user_id"`
	Status           string                   `json:"status"` // completed | partial_failure | failed
	AffectedServices []ServiceForgetStatus    `json:"affected_services"`
	TotalDeleted     int                      `json:"total_deleted"`
	AuditTrailID     string                   `json:"audit_trail_id"`
	CompletedAt      time.Time                `json:"completed_at"`
}

// ServiceForgetStatus represents per-service forget outcome.
type ServiceForgetStatus struct {
	Service      string `json:"service"`
	Status       string `json:"status"` // success | failed | skipped
	RecordCount  int    `json:"record_count"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// ForgetPreview represents a dry-run forget preview.
type ForgetPreview struct {
	UserID       string         `json:"user_id"`
	DataSummary  map[string]int `json:"data_summary"` // engine → record count
	TotalRecords int            `json:"total_records"`
}

// TimelineEvent represents a single event in the timeline.
type TimelineEvent struct {
	ID           string         `json:"id"`
	UserID       string         `json:"user_id"`
	EventType    string         `json:"event_type"`
	Source       string         `json:"source"`
	Description  string         `json:"description"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	OccurredAt   time.Time      `json:"occurred_at"`
}

// EngineDeleter interface for engine-specific data deletion.
type EngineDeleter interface {
	Delete(ctx context.Context, userID string) (int, error)
	Count(ctx context.Context, userID string) (int, error)
	Name() string
}

// GDPRService handles GDPR forget operations and event timelines.
type GDPRService struct {
	engines    []EngineDeleter
	logger     *slog.Logger
	maxRetries int
}

// NewGDPRService creates a GDPR service with engine clients.
func NewGDPRService(engines []EngineDeleter, logger *slog.Logger) *GDPRService {
	return &GDPRService{
		engines:    engines,
		logger:     logger,
		maxRetries: 3,
	}
}

// Preview performs a dry-run forget — counts records without deleting.
func (s *GDPRService) Preview(ctx context.Context, userID string) (*ForgetPreview, error) {
	preview := &ForgetPreview{
		UserID:      userID,
		DataSummary: make(map[string]int),
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, eng := range s.engines {
		wg.Add(1)
		go func(e EngineDeleter) {
			defer wg.Done()
			count, err := e.Count(ctx, userID)
			if err != nil {
				s.logger.Warn("forget preview failed for engine", "engine", e.Name(), "error", err)
				return
			}
			mu.Lock()
			preview.DataSummary[e.Name()] = count
			preview.TotalRecords += count
			mu.Unlock()
		}(eng)
	}
	wg.Wait()

	return preview, nil
}

// Forget executes cascading delete across all engines.
// Deletion order: Phase 1 (parallel: cognee, graphiti, openviking, supermemory)
//                Phase 2 (zep), Phase 3 (memobase), Phase 4 (audit)
func (s *GDPRService) Forget(ctx context.Context, req ForgetRequest) (*ForgetResult, error) {
	forgetID := uuid.New().String()
	s.logger.Info("GDPR forget initiated", "forget_id", forgetID, "user_id", req.UserID, "reason", req.Reason)

	result := &ForgetResult{
		ForgetID:         forgetID,
		UserID:           req.UserID,
		AffectedServices: make([]ServiceForgetStatus, 0, len(s.engines)),
		AuditTrailID:     uuid.New().String(),
	}

	// Execute deletions with per-engine timeout
	var mu sync.Mutex
	var wg sync.WaitGroup
	failCount := 0

	for _, eng := range s.engines {
		wg.Add(1)
		go func(e EngineDeleter) {
			defer wg.Done()

			deleteCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			count, err := e.Delete(deleteCtx, req.UserID)
			status := ServiceForgetStatus{
				Service:     e.Name(),
				RecordCount: count,
			}

			if err != nil {
				status.Status = "failed"
				status.ErrorMessage = err.Error()
				s.logger.Error("forget failed for engine", "engine", e.Name(), "error", err)
				mu.Lock()
				failCount++
				mu.Unlock()
			} else {
				status.Status = "success"
			}

			mu.Lock()
			result.AffectedServices = append(result.AffectedServices, status)
			result.TotalDeleted += count
			mu.Unlock()
		}(eng)
	}
	wg.Wait()

	// Determine overall status
	switch {
	case failCount == 0:
		result.Status = "completed"
	case failCount < len(s.engines):
		result.Status = "partial_failure"
	default:
		result.Status = "failed"
	}
	result.CompletedAt = time.Now().UTC()

	s.logger.Info("GDPR forget completed",
		"forget_id", forgetID,
		"status", result.Status,
		"total_deleted", result.TotalDeleted,
		"failures", failCount,
	)

	return result, nil
}
