package model

import "time"

// Enterprise Domain Models for zep-thread
type Thread struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Message struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


