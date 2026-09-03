package model

import "time"

// Enterprise Domain Models for sm-project
type Space struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type SpaceMember struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

type ContainerTag struct {
	ID string `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}


