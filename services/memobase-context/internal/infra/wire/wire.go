//go:build wireinject
// +build wireinject

package wire

import (
	"github.com/google/wire"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/adapter/external"
	grpchandler "github.com/vnp-community/vnp-memory/services/memobase-context/internal/adapter/grpc"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/adapter/repository/postgres"
	redisadapter "github.com/vnp-community/vnp-memory/services/memobase-context/internal/adapter/repository/redis"
	"github.com/vnp-community/vnp-memory/services/memobase-context/internal/usecase"
)

func InitializeHandler(
	pool *pgxpool.Pool, 
	redisClient *redis.Client,
	cacheTTL int,
	defaultMaxTokenSize int32,
	profileEventRatio float32,
	eventThreshold float32,
	eventWindowDays int,
	eventTopK int,
) *grpchandler.Handler {
	wire.Build(
		postgres.NewProfileReadRepository,
		postgres.NewEventGistSearchRepository,
		redisadapter.NewProfileCache,
		external.NewDummyEmbedder,

		usecase.NewGetProfilesUseCase,
		usecase.NewGetContextUseCase,
		usecase.NewSearchProfilesUseCase,

		grpchandler.NewHandler,
	)
	return nil
}
