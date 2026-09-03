package model

import "time"

// Enterprise Domain Models for sm-memory
type Memory struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Relation struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type ForgettingCurve struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


