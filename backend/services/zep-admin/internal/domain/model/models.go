package model

import "time"

// Enterprise Domain Models for zep-admin
type Project struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type APIKey struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


