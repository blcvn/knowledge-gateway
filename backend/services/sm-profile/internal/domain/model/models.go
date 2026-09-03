package model

import "time"

// Enterprise Domain Models for sm-profile
type Profile struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type StaticPreference struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type DynamicTrait struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


