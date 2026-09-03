package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vnp-memory/services/memobase-admin/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

// ─── Ports ─────────────────────────────────────────────────────────────────────

type ProjectRepository interface {
	Create(ctx context.Context, project *domain.Project) error
	GetByID(ctx context.Context, projectID string) (*domain.Project, error)
}

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, userID, projectID string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, userID, projectID string) error
	ListByProject(ctx context.Context, projectID string, limit int, cursor string) ([]*domain.User, string, bool, error)
}

type EventPublisher interface {
	PublishUserDeleted(ctx context.Context, projectID, userID string) error
	PublishProfileConfigUpdated(ctx context.Context, projectID string) error
}

// ─── CreateProject ─────────────────────────────────────────────────────────────

type CreateProjectUseCase struct {
	projectRepo ProjectRepository
}

func NewCreateProjectUseCase(repo ProjectRepository) *CreateProjectUseCase {
	return &CreateProjectUseCase{projectRepo: repo}
}

type CreateProjectResult struct {
	ProjectID    string
	ProjectToken string // plaintext, returned ONCE
}

func (uc *CreateProjectUseCase) Execute(ctx context.Context) (*CreateProjectResult, error) {
	projectID := uuid.New().String()
	token, hashedSecret, err := domain.GenerateProjectToken(projectID)
	if err != nil {
		return nil, err
	}

	defaultCfg := domain.DefaultProfileConfig()
	cfgYAML, _ := domain.MarshalProfileConfig(defaultCfg)

	project := &domain.Project{
		ProjectID:     projectID,
		ProjectSecret: hashedSecret,
		ProfileConfig: cfgYAML,
		Status:        domain.ProjectStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := uc.projectRepo.Create(ctx, project); err != nil {
		return nil, err
	}
	return &CreateProjectResult{ProjectID: projectID, ProjectToken: token}, nil
}

// ─── ValidateProjectToken ──────────────────────────────────────────────────────

type ValidateProjectTokenUseCase struct {
	projectRepo ProjectRepository
}

func NewValidateProjectTokenUseCase(repo ProjectRepository) *ValidateProjectTokenUseCase {
	return &ValidateProjectTokenUseCase{projectRepo: repo}
}

func (uc *ValidateProjectTokenUseCase) Execute(ctx context.Context, token string) (*domain.ProjectContext, error) {
	projectID, secret, err := domain.ParseProjectToken(token)
	if err != nil {
		return nil, domain.ErrUnauthorized
	}

	project, err := uc.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.ErrUnauthorized // Never expose 404 to caller
	}

	if err := bcrypt.CompareHashAndPassword([]byte(project.ProjectSecret), []byte(secret)); err != nil {
		return nil, domain.ErrUnauthorized
	}

	if project.Status != domain.ProjectStatusActive {
		return nil, domain.ErrProjectSuspended
	}

	return &domain.ProjectContext{ProjectID: projectID}, nil
}

// ─── CreateUser ────────────────────────────────────────────────────────────────

type CreateUserUseCase struct {
	userRepo    UserRepository
	projectRepo ProjectRepository
}

func NewCreateUserUseCase(userRepo UserRepository, projectRepo ProjectRepository) *CreateUserUseCase {
	return &CreateUserUseCase{userRepo: userRepo, projectRepo: projectRepo}
}

type CreateUserRequest struct {
	ProjectID string
	Metadata  map[string]any
}

type CreateUserResult struct {
	UserID string
}

func (uc *CreateUserUseCase) Execute(ctx context.Context, req CreateUserRequest) (*CreateUserResult, error) {
	project, err := uc.projectRepo.GetByID(ctx, req.ProjectID)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}
	if project.Status != domain.ProjectStatusActive {
		return nil, domain.ErrProjectSuspended
	}

	user := &domain.User{
		ID:        uuid.New().String(),
		ProjectID: req.ProjectID,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return &CreateUserResult{UserID: user.ID}, nil
}

// ─── DeleteUser ────────────────────────────────────────────────────────────────

type DeleteUserUseCase struct {
	userRepo  UserRepository
	publisher EventPublisher
}

func NewDeleteUserUseCase(userRepo UserRepository, publisher EventPublisher) *DeleteUserUseCase {
	return &DeleteUserUseCase{userRepo: userRepo, publisher: publisher}
}

func (uc *DeleteUserUseCase) Execute(ctx context.Context, userID, projectID string) error {
	if _, err := uc.userRepo.GetByID(ctx, userID, projectID); err != nil {
		return domain.ErrUserNotFound
	}
	// PostgreSQL CASCADE handles child tables
	if err := uc.userRepo.Delete(ctx, userID, projectID); err != nil {
		return err
	}
	_ = uc.publisher.PublishUserDeleted(ctx, projectID, userID)
	return nil
}

// ─── ListProjectUsers ──────────────────────────────────────────────────────────

type ListProjectUsersUseCase struct {
	userRepo UserRepository
}

func NewListProjectUsersUseCase(repo UserRepository) *ListProjectUsersUseCase {
	return &ListProjectUsersUseCase{userRepo: repo}
}

type ListProjectUsersRequest struct {
	ProjectID string
	Limit     int
	Cursor    string
}

type ListProjectUsersResult struct {
	Users      []*domain.User
	NextCursor string
	HasMore    bool
}

func (uc *ListProjectUsersUseCase) Execute(ctx context.Context, req ListProjectUsersRequest) (*ListProjectUsersResult, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	users, nextCursor, hasMore, err := uc.userRepo.ListByProject(ctx, req.ProjectID, limit, req.Cursor)
	if err != nil {
		return nil, err
	}
	return &ListProjectUsersResult{Users: users, NextCursor: nextCursor, HasMore: hasMore}, nil
}
