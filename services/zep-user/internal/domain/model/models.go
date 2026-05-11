package model

import "time"

// Enterprise Domain Models for zep-user
type User struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type UserPreference struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


