package domain

import "time"

type AccountCreated struct {
	AccountID string    `json:"account_id"`
	Timestamp time.Time `json:"timestamp"`
}

type UserDeleted struct {
	UserID    string    `json:"user_id"`
	AccountID string    `json:"account_id"`
	Timestamp time.Time `json:"timestamp"`
}
