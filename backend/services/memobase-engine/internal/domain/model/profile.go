package model

import (
	"time"

	"github.com/google/uuid"
)

// Profile represents a structured memory or fact about a user.
type Profile struct {
	ID         uuid.UUID
	UserID     string
	ProjectID  string
	Topic      string
	SubTopic   string
	Content    string
	Attributes map[string]interface{}
	UpdatedAt  time.time
}

// ProfileTopic defines the structure for profile schema extraction.
type ProfileTopic struct {
	Topic       string `json:"topic"`
	SubTopic    string `json:"sub_topic"`
	Description string `json:"description"`
}
