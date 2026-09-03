package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-pipeline/internal/domain/engine"
)

type ProfileRepo struct {
	db *pgxpool.Pool
}

func NewProfileRepo(db *pgxpool.Pool) *ProfileRepo {
	return &ProfileRepo{db: db}
}

func (r *ProfileRepo) FindByUser(ctx context.Context, tenantID, userID uuid.UUID) (*engine.Profile, error) {
	q := `SELECT id, topics, traits, version, updated_at FROM profiles WHERE tenant_id = $1 AND user_id = $2`
	
	var p engine.Profile
	p.TenantID = tenantID
	p.UserID = userID

	var topicsData, traitsData []byte
	err := r.db.QueryRow(ctx, q, tenantID, userID).Scan(&p.ID, &topicsData, &traitsData, &p.Version, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("profile not found")
	} else if err != nil {
		return nil, fmt.Errorf("query profile: %w", err)
	}

	_ = json.Unmarshal(topicsData, &p.Topics)
	_ = json.Unmarshal(traitsData, &p.Traits)

	return &p, nil
}

func (r *ProfileRepo) Upsert(ctx context.Context, profile *engine.Profile) error {
	q := `INSERT INTO profiles (id, tenant_id, user_id, topics, traits, version, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7)
	      ON CONFLICT (tenant_id, user_id) 
	      DO UPDATE SET topics = EXCLUDED.topics, traits = EXCLUDED.traits, version = profiles.version + 1, updated_at = EXCLUDED.updated_at`
	
	topicsData, _ := json.Marshal(profile.Topics)
	traitsData, _ := json.Marshal(profile.Traits)

	_, err := r.db.Exec(ctx, q, profile.ID, profile.TenantID, profile.UserID, topicsData, traitsData, profile.Version, time.Now())
	if err != nil {
		return fmt.Errorf("upsert profile: %w", err)
	}
	return nil
}
