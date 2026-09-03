package postgres

import (
\t"context"
\t"database/sql"
\t"errors"

\t"vnp-memory/services/sm-project/internal/domain/model"
)

// SpaceRepositoryImpl implements the persistence layer for Spaces.
type SpaceRepositoryImpl struct {
\tdb *sql.DB
}

// NewSpaceRepository creates a new SpaceRepositoryImpl.
func NewSpaceRepository(db *sql.DB) *SpaceRepositoryImpl {
\treturn &SpaceRepositoryImpl{
\t\tdb: db,
\t}
}

// Save persists a new Space to the database.
func (r *SpaceRepositoryImpl) Save(ctx context.Context, space *model.Space) error {
\tquery := `
\t\tINSERT INTO spaces (id, name, org_id, visibility, created_at)
\t\tVALUES ($1, $2, $3, $4, $5)
\t`
\t// Assuming space.Visibility maps to a string enum
\t_, err := r.db.ExecContext(ctx, query, space.ID, space.Name, space.OrgID, "private", space.CreatedAt)
\treturn err
}

// FindByID retrieves a Space by its ID.
func (r *SpaceRepositoryImpl) FindByID(ctx context.Context, id string) (*model.Space, error) {
\tquery := `
\t\tSELECT id, name, org_id, created_at
\t\tFROM spaces
\t\tWHERE id = $1
\t`
\t
\tvar space model.Space
\terr := r.db.QueryRowContext(ctx, query, id).Scan(
\t\t&space.ID,
\t\t&space.Name,
\t\t&space.OrgID, // Ensure model.Space has OrgID
\t\t&space.CreatedAt,
\t)
\tif err != nil {
\t\tif errors.Is(err, sql.ErrNoRows) {
\t\t\treturn nil, nil // Not found
\t\t}
\t\treturn nil, err
\t}

\treturn &space, nil
}
