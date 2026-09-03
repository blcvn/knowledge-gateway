//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"

	"vnp-memory/ov-search/internal/adapter/client"
	"vnp-memory/ov-search/internal/adapter/event"
	"vnp-memory/ov-search/internal/adapter/grpc"
	"vnp-memory/ov-search/internal/domain/model"
	"vnp-memory/ov-search/internal/infra/config"
	"vnp-memory/ov-search/internal/infra/persistence"
	"vnp-memory/ov-search/internal/usecase"
)

func InitializeApp(cfg *config.Config) (*grpc.OvSearchHandler, *event.Subscriber, *usecase.HotnessUseCase, error) {
	wire.Build(
		// Infra
		persistence.NewQdrantRepo,
		persistence.NewHotnessRepo,
		// Adapter
		client.NewBifrostClient,
		client.NewFsClient,
		// Providers map config to primitive types (mock implementation)
		wire.FieldsOf(new(*config.Config), "QdrantURL", "QdrantCollection", "BifrostAddr", "DBDSN", "OvFsAddr"),
		// Usecase
		usecase.NewHierarchicalSearch,
		usecase.NewHotnessUseCase,
		usecase.NewEmbeddingUseCase,
		// Mock config dependencies for usecase
		wire.Struct(new(model.DecayConfig), "*"),
		// Handler / Subscriber
		grpc.NewOvSearchHandler,
		event.NewSubscriber,
	)
	return &grpc.OvSearchHandler{}, &event.Subscriber{}, nil, nil
}
