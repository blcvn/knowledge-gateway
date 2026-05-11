package model

import "time"

// Enterprise Domain Models for zep-memory
type MemorySummary struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Context struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


