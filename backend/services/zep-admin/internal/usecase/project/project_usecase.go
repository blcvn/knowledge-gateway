package project

import (
\t"context"
\t"errors"
\t"time"

\t"vnp-memory/services/zep-admin/internal/domain/model"
)

// ProjectRepository defines the data access port for Project entities.
type ProjectRepository interface {
\tSave(ctx context.Context, p *model.Project) error
\tFindByID(ctx context.Context, id string) (*model.Project, error)
\tListByOwner(ctx context.Context, ownerID string) ([]*model.Project, error)
}

// ProjectUseCase orchestrates project management logic.
type ProjectUseCase struct {
\trepo ProjectRepository
}

// NewProjectUseCase creates a new instance of ProjectUseCase.
func NewProjectUseCase(repo ProjectRepository) *ProjectUseCase {
\treturn &ProjectUseCase{
\t\trepo: repo,
\t}
}

// CreateProject creates a new project ensuring business rules are met.
func (uc *ProjectUseCase) CreateProject(ctx context.Context, name, ownerID string) (*model.Project, error) {
\tif name == "" {
\t\treturn nil, errors.New("project name cannot be empty")
\t}
\tif ownerID == "" {
\t\treturn nil, errors.New("owner ID must be provided")
\t}

\tp := &model.Project{
\t\t// Note: ID should ideally be generated via UUID/ULID
\t\tID:        generateID(),
\t\tName:      name,
\t\tOwnerID:   ownerID,
\t\tCreatedAt: time.Now(),
\t}

\tif err := uc.repo.Save(ctx, p); err != nil {
\t\treturn nil, err
\t}

\treturn p, nil
}

// GetProject retrieves a project and ensures the requester has access.
func (uc *ProjectUseCase) GetProject(ctx context.Context, projectID, requesterID string) (*model.Project, error) {
\tp, err := uc.repo.FindByID(ctx, projectID)
\tif err != nil {
\t\treturn nil, err
\t}
\tif p == nil {
\t\treturn nil, errors.New("project not found")
\t}

\t// Basic authorization check
\tif p.OwnerID != requesterID {
\t\treturn nil, errors.New("unauthorized access to project")
\t}

\treturn p, nil
}

// generateID is a placeholder for a robust ID generator like google/uuid.
func generateID() string {
\treturn "proj_" + time.Now().Format("20060102150405")
}

// Extension to model to satisfy the compilation locally.
// Normally this sits in domain/model.
type modelProjectExtension struct {
\tName    string
\tOwnerID string
}
