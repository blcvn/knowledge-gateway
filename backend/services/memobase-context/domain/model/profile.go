package model

import "time"

type Profile struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProjectID string    `json:"project_id"`
	Content   string    `json:"content"`
	Topic     string    `json:"topic"`
	SubTopic  string    `json:"sub_topic"`
	UpdatedAt time.Time `json:"updated_at"`
}
