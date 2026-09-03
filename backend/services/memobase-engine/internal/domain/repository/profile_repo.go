package repository

import (
	"context"

	"github.com/google/uuid"
	"vnp-memory/services/memobase-engine/internal/domain/model"
)

// ProfileRepository defines the interface for interacting with user profiles.
type ProfileRepository interface {
	// UpsertProfiles inserts new profiles or updates existing ones.
	UpsertProfiles(ctx context.Context, profiles []model.Profile) error
	
	// GetProfiles retrieves profiles for a user/project.
	GetProfiles(ctx context.Context, userID, projectID string) ([]model.Profile, error)
	
	// DeleteProfiles deletes profiles by ID.
	DeleteProfiles(ctx context.Context, userID, projectID string, ids []uuid.UUID) error
}
