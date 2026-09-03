package tests

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/adapter/repository/postgres"
	"github.com/vnp-community/vnp-memory/services/graphiti-ingestion/internal/domain"
)

// SetupPostgresContainer spins up a real postgres instance via Docker for integration testing
func SetupPostgresContainer(ctx context.Context) (testcontainers.Container, *pgxpool.Pool, error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "user",
			"POSTGRES_PASSWORD": "password",
			"POSTGRES_DB":       "graphiti",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, err
	}

	host, err := container.Host(ctx)
	if err != nil {
		return container, nil, err
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return container, nil, err
	}

	connStr := "postgres://user:password@" + host + ":" + port.Port() + "/graphiti?sslmode=disable"
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return container, nil, err
	}

	// Initialize tables (simulating migrations/001_init.sql)
	initSQL := `
		CREATE TABLE graphiti_saga_state (
			id VARCHAR PRIMARY KEY,
			episode_id VARCHAR,
			group_id VARCHAR,
			current_step VARCHAR,
			status VARCHAR,
			step_history JSONB,
			retry_count INT,
			error_message TEXT,
			started_at TIMESTAMP,
			completed_at TIMESTAMP
		);
		CREATE TABLE graphiti_episodes (
			uuid VARCHAR PRIMARY KEY,
			name VARCHAR,
			group_id VARCHAR,
			body TEXT,
			source VARCHAR,
			reference_time TIMESTAMP,
			content_hash VARCHAR,
			saga_id VARCHAR,
			entity_types JSONB,
			edge_types JSONB,
			created_at TIMESTAMP
		);
		CREATE TABLE graphiti_episode_dedup (
			content_hash VARCHAR PRIMARY KEY,
			episode_id VARCHAR
		);
	`
	_, err = pool.Exec(ctx, initSQL)
	
	return container, pool, err
}

func TestSagaRepo_Integration(t *testing.T) {
	// Skip in CI environments if docker is not available
	// if os.Getenv("CI") != "" { t.Skip("Skipping testcontainers in CI") }
	
	ctx := context.Background()
	container, pool, err := SetupPostgresContainer(ctx)
	if err != nil {
		t.Skipf("Failed to start postgres container: %v", err)
	}
	defer func() {
		if container != nil {
			container.Terminate(ctx)
		}
	}()
	defer pool.Close()

	repo := postgres.NewSagaRepo(pool)

	// Test Create
	state := domain.NewSagaState("ep-123", "grp-456")
	err = repo.Create(ctx, state)
	if err != nil {
		t.Fatalf("failed to create saga state: %v", err)
	}

	// Test Get
	fetched, err := repo.Get(ctx, state.ID)
	if err != nil {
		t.Fatalf("failed to get saga state: %v", err)
	}
	if fetched.ID != state.ID {
		t.Errorf("expected ID %s, got %s", state.ID, fetched.ID)
	}

	// Test GetStuckSagas (simulate stuck by backdating started_at)
	_, _ = pool.Exec(ctx, "UPDATE graphiti_saga_state SET started_at = NOW() - INTERVAL '10 minutes'")
	stuck, err := repo.GetStuckSagas(ctx, 5, 10)
	if err != nil {
		t.Fatalf("failed to get stuck sagas: %v", err)
	}
	if len(stuck) != 1 {
		t.Errorf("expected 1 stuck saga, got %d", len(stuck))
	}
}

func TestEpisodeRepo_Integration(t *testing.T) {
	ctx := context.Background()
	container, pool, err := SetupPostgresContainer(ctx)
	if err != nil {
		t.Skipf("Failed to start postgres container: %v", err)
	}
	defer func() {
		if container != nil {
			container.Terminate(ctx)
		}
	}()
	defer pool.Close()

	repo := postgres.NewEpisodeRepo(pool)

	ep, _ := domain.NewEpisode("Test Ep", "grp-1", "Hello World", domain.SourceText, time.Now())

	// Test Create
	err = repo.Create(ctx, ep)
	if err != nil {
		t.Fatalf("failed to create episode: %v", err)
	}

	// Test Duplicate handling
	err = repo.Create(ctx, ep)
	if err != domain.ErrDuplicateEpisode {
		t.Errorf("expected ErrDuplicateEpisode, got %v", err)
	}

	// Test GetByHash
	fetched, err := repo.GetByHash(ctx, ep.ContentHash)
	if err != nil {
		t.Fatalf("failed to get episode by hash: %v", err)
	}
	if fetched.UUID != ep.UUID {
		t.Errorf("expected UUID %s, got %s", ep.UUID, fetched.UUID)
	}
}
