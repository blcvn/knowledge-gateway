package model

import "time"

// Enterprise Domain Models for sm-engine
type MemoryCurve struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type EbbinghausItem struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


