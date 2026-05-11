package wire

import (
	"database/sql"

	"github.com/google/wire"
	"github.com/vnp-memory/services/ov-session/internal/adapter/client"
	"github.com/vnp-memory/services/ov-session/internal/adapter/event"
	"github.com/vnp-memory/services/ov-session/internal/adapter/grpc"
	"github.com/vnp-memory/services/ov-session/internal/infra/persistence"
	"github.com/vnp-memory/services/ov-session/internal/usecase"
)

func InitializeApp(db *sql.DB) (*grpc.OvSessionHandler, error) {
	wire.Build(
		persistence.NewSessionRepository,
		persistence.NewMessageRepository,
		client.NewFSClient,
		client.NewLLMClient,
		event.NewPublisher,
		usecase.NewSessionUseCase,
		usecase.NewWorkingMemoryUseCase,
		usecase.NewCompressorUseCase,
		usecase.NewMemoryExtractorUseCase,
		usecase.NewMemoryDeduplicatorUseCase,
		usecase.NewCommitUseCase,
		grpc.NewOvSessionHandler,
	)
	return &grpc.OvSessionHandler{}, nil
}
