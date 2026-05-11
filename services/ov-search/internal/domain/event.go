package domain

import "time"

type SearchCompleted struct {
	AccountID string    `json:"account_id"`
	Query     string    `json:"query"`
	Timestamp time.Time `json:"timestamp"`
}

type HotnessUpdated struct {
	Path         string  `json:"path"`
	AccountID    string  `json:"account_id"`
	NewScore     float64 `json:"new_score"`
	UpdatedByJob bool    `json:"updated_by_job"`
}
