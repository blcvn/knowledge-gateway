package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/services/memobase-context/domain/model"
	"github.com/vnp-community/vnp-memory/services/memobase-context/domain/repository"
)

type profileReadRepo struct {
	pool *pgxpool.Pool
}

func NewProfileReadRepository(pool *pgxpool.Pool) repository.ProfileReadRepository {
	return &profileReadRepo{pool: pool}
}

func (r *profileReadRepo) GetProfiles(ctx context.Context, userID, projectID string) ([]*model.Profile, error) {
	query := `SELECT id, user_id, project_id, content, attributes->>'topic' as topic, attributes->>'sub_topic' as sub_topic, updated_at FROM user_profiles WHERE user_id = $1 AND project_id = $2`
	rows, err := r.pool.Query(ctx, query, userID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*model.Profile
	for rows.Next() {
		p := &model.Profile{}
		var topic, subTopic *string
		err := rows.Scan(&p.ID, &p.UserID, &p.ProjectID, &p.Content, &topic, &subTopic, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if topic != nil {
			p.Topic = *topic
		}
		if subTopic != nil {
			p.SubTopic = *subTopic
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

func (r *profileReadRepo) SearchProfiles(ctx context.Context, userID, projectID string, queryEmbedding []float32, limit int) ([]*model.Profile, error) {
	return r.GetProfiles(ctx, userID, projectID)
}
