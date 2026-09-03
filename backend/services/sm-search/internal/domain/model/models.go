package model

import "time"

// Enterprise Domain Models for sm-search
type SearchQuery struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchResult struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


