package postgres

import (
	"context"

	"github.com/google/uuid"
	"vnp-memory/services/memobase-engine/internal/domain/model"
	"vnp-memory/services/memobase-engine/internal/domain/repository"
)

type profileRepository struct {
	// db *sql.DB or *gorm.DB
}

// NewProfileRepository creates a new PostgreSQL backed profile repository.
func NewProfileRepository() repository.ProfileRepository {
	return &profileRepository{}
}

func (r *profileRepository) UpsertProfiles(ctx context.Context, profiles []model.Profile) error {
	return nil
}

func (r *profileRepository) GetProfiles(ctx context.Context, userID, projectID string) ([]model.Profile, error) {
	return nil, nil
}

func (r *profileRepository) DeleteProfiles(ctx context.Context, userID, projectID string, ids []uuid.UUID) error {
	return nil
}
