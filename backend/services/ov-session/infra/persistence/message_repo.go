package persistence

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/vnp-memory/services/ov-session/domain/model"
	"github.com/vnp-memory/services/ov-session/domain/repository"
)

type MessageRepoImpl struct {
	db *sql.DB
}

func NewMessageRepository(db *sql.DB) repository.MessageRepository {
	return &MessageRepoImpl{db: db}
}

func (r *MessageRepoImpl) AddMessage(ctx context.Context, msg *model.Message) error {
	tools, _ := json.Marshal(msg.ToolCalls)
	query := `INSERT INTO ov_messages (id, session_id, role, content, tool_calls, token_count, sequence, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query, msg.ID, msg.SessionID, msg.Role, msg.Content, tools, msg.TokenCount, msg.Sequence, msg.CreatedAt)
	return err
}

func (r *MessageRepoImpl) GetMessagesBySession(ctx context.Context, sessionID string) ([]*model.Message, error) {
	query := `SELECT id, session_id, role, content, tool_calls, token_count, sequence, created_at FROM ov_messages WHERE session_id = $1 ORDER BY sequence ASC`
	rows, err := r.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*model.Message
	for rows.Next() {
		var m model.Message
		var tools []byte
		err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &tools, &m.TokenCount, &m.Sequence, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(tools, &m.ToolCalls)
		messages = append(messages, &m)
	}
	return messages, nil
}
