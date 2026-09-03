package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	contextv1 "github.com/vnp-community/vnp-memory/services/memobase-context/api/proto/memobase/context/v1"
	"github.com/vnp-community/vnp-memory/services/memobase-context/adapter/event"
	"github.com/vnp-community/vnp-memory/services/memobase-context/infra/config"
	"github.com/vnp-community/vnp-memory/services/memobase-context/infra/telemetry"
	"github.com/vnp-community/vnp-memory/services/memobase-context/infra/wire"
	redisadapter "github.com/vnp-community/vnp-memory/services/memobase-context/adapter/repository/redis"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	telemetry.InitLogger(cfg.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Init DB
	pool, err := pgxpool.New(ctx, cfg.DBDSN)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		// Assuming DB is required
		// os.Exit(1) // Usually we might let it reconnect
	}
	if pool != nil {
		defer pool.Close()
	}

	// Init Redis
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("Failed to parse redis url", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(opts)
	defer redisClient.Close()

	// Init NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		slog.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Close()

	// Setup NATS Subscriber
	profileCache := redisadapter.NewProfileCache(redisClient)
	natsSub := event.NewNatsSubscriber(nc, profileCache)
	if err := natsSub.Start(); err != nil {
		slog.Error("Failed to start NATS subscriber", "error", err)
		os.Exit(1)
	}

	// Init Wire (DI)
	handler := wire.InitializeHandler(
		pool,
		redisClient,
		cfg.ProfileCacheTTL,
		cfg.DefaultMaxTokenSize,
		cfg.ProfileEventRatio,
		cfg.EventSearchThreshold,
		cfg.EventSearchWindowDays,
		cfg.EventSearchTopK,
	)

	// Setup gRPC
	grpcServer := grpc.NewServer()
	contextv1.RegisterMemobaseContextServiceServer(grpcServer, handler)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	// Start HTTP health/ready probes
	go func() {
		http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		})
		http.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			if pool == nil || pool.Ping(r.Context()) != nil {
				http.Error(w, "db not ready", http.StatusServiceUnavailable)
				return
			}
			if err := redisClient.Ping(r.Context()).Err(); err != nil {
				http.Error(w, "redis not ready", http.StatusServiceUnavailable)
				return
			}
			if !nc.IsConnected() {
				http.Error(w, "nats not ready", http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		})
		
		addr := fmt.Sprintf(":%d", cfg.HealthPort)
		slog.Info("Starting HTTP health server", "addr", addr)
		if err := http.ListenAndServe(addr, nil); err != nil {
			slog.Error("HTTP health server failed", "error", err)
		}
	}()

	// Start gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
	if err != nil {
		slog.Error("Failed to listen", "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("Starting gRPC server", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
	time.Sleep(1 * time.Second) // wait for pending requests
	slog.Info("Shutdown complete")
}
