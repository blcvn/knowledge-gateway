package persistence

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/vnp-memory/services/ov-session/internal/domain/model"
	"github.com/vnp-memory/services/ov-session/internal/domain/repository"
)

type SessionRepoImpl struct {
	db *sql.DB
}

func NewSessionRepository(db *sql.DB) repository.SessionRepository {
	return &SessionRepoImpl{db: db}
}

func (r *SessionRepoImpl) Create(ctx context.Context, session *model.Session) error {
	metaJSON, _ := json.Marshal(session.Metadata)
	query := `INSERT INTO ov_sessions (id, account_id, user_id, agent_id, title, status, archive_path, memories_count, compression_version, metadata, created_at, committed_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query, session.ID, session.AccountID, session.UserID, session.AgentID, session.Title, session.Status, session.ArchivePath, session.MemoriesCount, session.CompressionVersion, metaJSON, session.CreatedAt, session.CommittedAt)
	return err
}

func (r *SessionRepoImpl) GetByID(ctx context.Context, id string) (*model.Session, error) {
	query := `SELECT id, account_id, user_id, agent_id, title, status, archive_path, memories_count, compression_version, metadata, created_at, committed_at FROM ov_sessions WHERE id = $1`
	var s model.Session
	var metaJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(&s.ID, &s.AccountID, &s.UserID, &s.AgentID, &s.Title, &s.Status, &s.ArchivePath, &s.MemoriesCount, &s.CompressionVersion, &metaJSON, &s.CreatedAt, &s.CommittedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(metaJSON, &s.Metadata)
	return &s, nil
}

func (r *SessionRepoImpl) Update(ctx context.Context, session *model.Session) error {
	metaJSON, _ := json.Marshal(session.Metadata)
	query := `UPDATE ov_sessions SET status = $1, archive_path = $2, memories_count = $3, metadata = $4, committed_at = $5 WHERE id = $6`
	_, err := r.db.ExecContext(ctx, query, session.Status, session.ArchivePath, session.MemoriesCount, metaJSON, session.CommittedAt, session.ID)
	return err
}

func (r *SessionRepoImpl) GetWorkingMemory(ctx context.Context, sessionID string) (*model.WorkingMemory, error) {
	query := `SELECT session_id, title, state, goals, facts, errors, context, updated_at FROM ov_working_memory WHERE session_id = $1`
	var wm model.WorkingMemory
	var goals, facts, errs, ctxJSON []byte
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&wm.SessionID, &wm.Title, &wm.State, &goals, &facts, &errs, &ctxJSON, &wm.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(goals, &wm.Goals)
	json.Unmarshal(facts, &wm.Facts)
	json.Unmarshal(errs, &wm.Errors)
	json.Unmarshal(ctxJSON, &wm.Context)
	return &wm, nil
}

func (r *SessionRepoImpl) UpdateWorkingMemory(ctx context.Context, wm *model.WorkingMemory) error {
	goals, _ := json.Marshal(wm.Goals)
	facts, _ := json.Marshal(wm.Facts)
	errs, _ := json.Marshal(wm.Errors)
	ctxJSON, _ := json.Marshal(wm.Context)

	query := `INSERT INTO ov_working_memory (session_id, title, state, goals, facts, errors, context, updated_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  ON CONFLICT (session_id) DO UPDATE SET title = $2, state = $3, goals = $4, facts = $5, errors = $6, context = $7, updated_at = $8`
	_, err := r.db.ExecContext(ctx, query, wm.SessionID, wm.Title, wm.State, goals, facts, errs, ctxJSON, wm.UpdatedAt)
	return err
}

func (r *SessionRepoImpl) SaveMemory(ctx context.Context, memory *model.CandidateMemory) error {
	query := `INSERT INTO ov_extracted_memories (id, session_id, account_id, category, content, confidence, dedup_action, fs_path, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(ctx, query, memory.ID, memory.SessionID, memory.AccountID, memory.Category, memory.Content, memory.Confidence, memory.DedupAction, memory.FSPath, memory.CreatedAt)
	return err
}
