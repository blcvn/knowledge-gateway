package model

import "time"

// Enterprise Domain Models for sm-document
type Document struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Chunk struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type ContentExtraction struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


