package model

import "time"

type EventGist struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProjectID string    `json:"project_id"`
	GistData  string    `json:"gist_data"`
	Embedding []float32 `json:"embedding"`
	CreatedAt time.Time `json:"created_at"`
}
