package model

import "time"

// Enterprise Domain Models for sm-auth
type Session struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type Token struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


