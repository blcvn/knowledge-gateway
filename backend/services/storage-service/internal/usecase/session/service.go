// Package session implements the session usecase for storage-service.
//
// Provides: Create, AddMessage, Commit, GetHistory
// Absorbed from: ov-session (MERGE-P1-T4)
package session

import (
	"context"
	"fmt"
	"time"

	"vnp-memory/services/storage-service/internal/domain/session"
)

// Repository is the output port for session persistence.
type Repository interface {
	Create(ctx context.Context, s *session.ChatSession) error
	FindByID(ctx context.Context, id string) (*session.ChatSession, error)
	Update(ctx context.Context, s *session.ChatSession) error
	AddMessage(ctx context.Context, msg *session.Message) error
	GetMessages(ctx context.Context, sessionID string) ([]*session.Message, error)
	SaveCommit(ctx context.Context, record *session.CommitRecord) error
}

// Service implements session use cases.
type Service struct {
	repo Repository
}

// NewService creates a session Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Create initializes a new chat session.
func (s *Service) Create(ctx context.Context, tenantID, baseDir string) (*session.ChatSession, error) {
	sess := &session.ChatSession{
		ID:        generateID("sess"),
		TenantID:  tenantID,
		BaseDir:   baseDir,
		Status:    "open",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, sess); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	return sess, nil
}

// AddMessage appends a message to a session.
func (s *Service) AddMessage(ctx context.Context, sessionID string, msg *session.Message) error {
	msg.ID = generateID("msg")
	msg.SessionID = sessionID
	msg.CreatedAt = time.Now()
	if err := s.repo.AddMessage(ctx, msg); err != nil {
		return fmt.Errorf("add message: %w", err)
	}
	return nil
}

// Commit finalizes the session and records file changes.
func (s *Service) Commit(ctx context.Context, sessionID string) (*session.CommitRecord, error) {
	sess, err := s.repo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("find session: %w", err)
	}
	if sess.Status == "committed" {
		return nil, fmt.Errorf("session already committed")
	}

	record := &session.CommitRecord{
		SessionID:   sessionID,
		FilesDiff:   []session.FileDiff{},
		Summary:     "Session committed",
		CommittedAt: time.Now(),
	}

	if err := s.repo.SaveCommit(ctx, record); err != nil {
		return nil, fmt.Errorf("save commit: %w", err)
	}

	sess.Status = "committed"
	sess.UpdatedAt = time.Now()
	_ = s.repo.Update(ctx, sess)

	return record, nil
}

// GetHistory retrieves all messages for a session.
func (s *Service) GetHistory(ctx context.Context, sessionID string) ([]*session.Message, error) {
	return s.repo.GetMessages(ctx, sessionID)
}

func generateID(prefix string) string {
	return prefix + "_" + fmt.Sprintf("%d", time.Now().UnixNano())
}
