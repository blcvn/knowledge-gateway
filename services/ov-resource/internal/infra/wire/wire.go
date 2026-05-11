//go:build wireinject
// +build wireinject

package wire

import (
	"database/sql"
	"github.com/google/wire"

	"openviking.com/ov-resource/internal/adapter/client"
	"openviking.com/ov-resource/internal/adapter/event"
	"openviking.com/ov-resource/internal/adapter/grpc"
	"openviking.com/ov-resource/internal/adapter/parser"
	"openviking.com/ov-resource/internal/domain/model"
	"openviking.com/ov-resource/internal/infra/config"
	"openviking.com/ov-resource/internal/infra/persistence"
	"openviking.com/ov-resource/internal/usecase"
	"openviking.com/ov-resource/internal/usecase/port"
	"openviking.com/ov-resource/internal/domain/repository"
)

func provideParserConfig(cfg config.Config) model.ParserConfig {
	return model.ParserConfig{
		ChunkSizeTokens:    cfg.ChunkSizeTokens,
		ChunkOverlapTokens: cfg.ChunkOverlapTokens,
		TreesitterEnabled:  cfg.TreesitterEnabled,
	}
}

func provideParserRegistry(pCfg model.ParserConfig) port.ParserPort {
	return parser.NewRegistry(pCfg)
}

func provideFileWriter(cfg config.Config) port.FileWriterPort {
	return client.NewFsClient(cfg.OvFsAddr)
}

func provideEventPublisher(cfg config.Config) port.EventPublisherPort {
	return event.NewNatsPublisher(cfg.NatsUrl)
}

func provideResourceRepo(db *sql.DB) repository.ResourceRepository {
	return persistence.NewResourceRepository(db)
}

func provideWatchRepo(db *sql.DB) repository.WatchRepository {
	return persistence.NewWatchRepository(db)
}

func provideIngestUseCase(registry port.ParserPort, writer port.FileWriterPort, pub port.EventPublisherPort, repo repository.ResourceRepository, cfg config.Config) port.IngestUseCase {
	return usecase.NewIngestUseCase(registry, writer, pub, repo, cfg.MaxIngestionSizeMb)
}

func provideParseUseCase(registry port.ParserPort) port.ParseUseCase {
	return usecase.NewParseUseCase(registry)
}

func provideWatchUseCase(repo repository.WatchRepository, ingest port.IngestUseCase, cfg config.Config) port.WatchUseCase {
	return usecase.NewWatchUseCase(repo, ingest, cfg.WatchMaxTasks, cfg.WatchDefaultPollMs)
}

func provideRefreshUseCase(ingest port.IngestUseCase) port.RefreshUseCase {
	return usecase.NewRefreshUseCase(ingest)
}

func InitializeApp(cfg config.Config, db *sql.DB) (*grpc.Handler, error) {
	wire.Build(
		provideParserConfig,
		provideParserRegistry,
		provideFileWriter,
		provideEventPublisher,
		provideResourceRepo,
		provideWatchRepo,
		provideIngestUseCase,
		provideParseUseCase,
		provideWatchUseCase,
		provideRefreshUseCase,
		grpc.NewHandler,
	)
	return &grpc.Handler{}, nil
}
