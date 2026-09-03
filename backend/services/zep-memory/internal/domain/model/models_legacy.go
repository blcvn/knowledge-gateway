package model

import "time"

// Enterprise Domain Models for zep-core
type Memory struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


