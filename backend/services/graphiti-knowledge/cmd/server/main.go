package main

import (
	"vnp-memory/shared/pkg/forward"
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"

	graphitigrpc "vnp-memory/services/graphiti-knowledge/adapter/grpc"
	pb "vnp-memory/services/graphiti-knowledge/adapter/grpc/pb"
	"vnp-memory/services/graphiti-knowledge/adapter/llm"
	"vnp-memory/services/graphiti-knowledge/adapter/repository/neo4j"
	"vnp-memory/services/graphiti-knowledge/usecase/knowledge"
)

// Enterprise-grade bootstrap for graphiti-knowledge
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize Structured Logging
	telemetry.InitLogger("info")
	slog.Info("Initializing graphiti-knowledge at enterprise-grade...")

	// 2. Initialize OpenTelemetry Distributed Tracing
	shutdownTracer, err := telemetry.InitProvider(ctx, "graphiti-knowledge")
	if err != nil {
		slog.Error("failed to initialize OTel provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			slog.Error("failed to shutdown OTel provider gracefully", slog.String("error", err.Error()))
		}
	}()

	// 3. Setup Dependencies
	neo4jURI := getEnv("NEO4J_URI", "neo4j://localhost:7687")
	neo4jUser := getEnv("NEO4J_USERNAME", "neo4j")
	neo4jPass := getEnv("NEO4J_PASSWORD", "password")
	graphRepo, err := neo4j.NewGraphRepository(neo4jURI, neo4jUser, neo4jPass)
	if err != nil {
		slog.Error("Failed to init Neo4j", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer graphRepo.Close(context.Background())

	apiKey := os.Getenv("OPENAI_API_KEY")
	llmClient := llm.NewOpenAIClient(apiKey, "gpt-4o")

	// 4. Initialize Usecase & Handler
	knowledgeUsecase := knowledge.NewKnowledgeUsecase(llmClient, graphRepo)
	knowledgeHandler := graphitigrpc.NewHandler(knowledgeUsecase)

	// 5. Setup gRPC Server with Interceptors (Tenant isolation & OTel traces)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)

	// Register the Knowledge Service
	pb.RegisterGraphitiKnowledgeServiceServer(grpcServer, knowledgeHandler)

	// 6. Setup Health Probes
	// Setup ForwardService Router
	router := forward.NewRouter()
	forward.RegisterForwardService(grpcServer, router)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	// 7. Start HTTP Health/Metrics Server
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		slog.Info("Starting HTTP probe server on :9199")
		if err := http.ListenAndServe(":9199", mux); err != nil {
			slog.Error("HTTP probe server failed", slog.String("error", err.Error()))
		}
	}()

	// 8. Start gRPC Server
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		slog.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}
	healthCheck.SetServingStatus("graphiti-knowledge", grpc_health_v1.HealthCheckResponse_SERVING)
	
	go func() {
		slog.Info("Starting gRPC server on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", slog.String("error", err.Error()))
		}
	}()

	// 9. Graceful Shutdown Management
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
	slog.Info("Server exited properly")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
