package model

import "time"

// Enterprise Domain Models for zep-graph
type Node struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Edge struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Graph struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


