package model

import "time"

// Enterprise Domain Models for sm-analytics
type Metric struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Report struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


