package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"vnp-memory/services/graphiti-search/internal/adapter/cache"
	"vnp-memory/services/graphiti-search/internal/adapter/client"
	"vnp-memory/services/graphiti-search/internal/adapter/event"
	searchgrpc "vnp-memory/services/graphiti-search/internal/adapter/grpc"
	"vnp-memory/services/graphiti-search/internal/infra/config"
	"vnp-memory/services/graphiti-search/internal/infra/server"
	"vnp-memory/services/graphiti-search/internal/usecase"
	"vnp-memory/services/graphiti-search/internal/usecase/reranker"
)

func main() {
	cfg := config.LoadConfig()

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Printf("Warning: failed to connect to NATS: %v", err)
	}

	storeConn, err := grpc.Dial(cfg.StoreAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("failed to connect to store service: %v", err)
	}

	storeClient := client.NewStoreClientAdapter(storeConn)
	redisCache := cache.NewRedisCacheAdapter(rdb)

	rerankers := []usecase.Reranker{
		reranker.NewRRFReranker(cfg.RRFKValue),
		reranker.NewMMRReranker(cfg.MMRLambda),
		reranker.NewCrossEncoderReranker(storeConn),
		reranker.NewNodeDistanceReranker(cfg.NodeDistanceWeight),
		reranker.NewEpisodeMentionsReranker(),
	}

	hybridUC := usecase.NewHybridSearchUseCase(storeClient, nil, redisCache, rerankers, cfg.CacheTTL)
	nodeUC := usecase.NewNodeSearchUseCase(storeClient)
	edgeUC := usecase.NewEdgeSearchUseCase(storeClient)
	communityUC := usecase.NewCommunitySearchUseCase(storeClient)

	searchServer := searchgrpc.NewSearchServiceServer(hybridUC, nodeUC, edgeUC, communityUC)
	srv := server.NewGRPCServer(cfg.GRPCPort, searchServer)

	natsSub := event.NewNatsSubscriber(nc, redisCache)
	if nc != nil {
		if err := natsSub.Listen(context.Background()); err != nil {
			log.Printf("Warning: NATS listener failed: %v", err)
		} else {
			log.Println("NATS cache invalidation listener started")
		}
	}

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("failed to start gRPC server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	srv.Stop()
	rdb.Close()
	if nc != nil {
		nc.Close()
	}
	storeConn.Close()
	log.Println("Server gracefully stopped")
}
