package model

import "time"

// Enterprise Domain Models for sm-connector
type Connection struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type SyncJob struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


