package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"vnp-memory/shared/pkg/forward"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"
	
	smauthv1 "vnp-memory/services/sm-auth/api/proto/v1"
	smgrpc "vnp-memory/services/sm-auth/internal/adapter/grpc"
	"vnp-memory/services/sm-auth/internal/adapter/repo"
	"vnp-memory/services/sm-auth/internal/usecase"
)

// Enterprise-grade bootstrap for sm-auth
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize Structured Logging
	telemetry.InitLogger("info")
	slog.Info("Initializing sm-auth at enterprise-grade...")

	// 2. Initialize OpenTelemetry Distributed Tracing
	shutdownTracer, err := telemetry.InitProvider(ctx, "sm-auth")
	if err != nil {
		slog.Error("failed to initialize OTel provider", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			slog.Error("failed to shutdown OTel provider gracefully", slog.String("error", err.Error()))
		}
	}()

	// 3. Setup Usecases & Handlers
	userRepo := repo.NewInMemoryUserRepository()
	
	jwtSecret := os.Getenv("AUTH_JWT_PRIVATE_KEY")
	if jwtSecret == "" {
		slog.Error("AUTH_JWT_PRIVATE_KEY is missing from environment")
		os.Exit(1)
	}
	// Support both raw multiline and \n encoded strings from env
	jwtSecret = strings.ReplaceAll(jwtSecret, "\\n", "\n")
	
	googleClientID := os.Getenv("GOOGLE_CLIENT_ID")
	
	adminEmail := os.Getenv("DEFAULT_ADMIN_EMAIL")
	adminPassword := os.Getenv("DEFAULT_ADMIN_PASSWORD")

	authUC, err := usecase.NewAuthUseCase(userRepo, jwtSecret, googleClientID)
	if err != nil {
		slog.Error("Failed to initialize AuthUseCase", slog.String("error", err.Error()))
		os.Exit(1)
	}
	
	// Create default admin user if specified
	if adminEmail != "" && adminPassword != "" {
		_, _, err := authUC.Register(ctx, "Admin", adminEmail, adminPassword)
		if err != nil {
			slog.Info("Default admin user may already exist", "email", adminEmail)
		} else {
			slog.Info("Created default admin user", "email", adminEmail)
		}
	}

	authHandler := smgrpc.NewAuthHandler(authUC)

	// 4. Setup gRPC Server with Interceptors (Tenant isolation & OTel traces)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)

	// Register SmAuthService
	smauthv1.RegisterSmAuthServiceServer(grpcServer, authHandler)

	// 5. Setup Health Probes
	// Setup ForwardService Router
	router := forward.NewRouter(slog.Default())
	
	router.Handle("POST", "/v1/auth/register", func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		var req smauthv1.RegisterRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		resp, err := authHandler.Register(ctx, &req)
		if err != nil {
			return nil, err
		}
		return json.Marshal(resp)
	})

	router.Handle("POST", "/v1/auth/login", func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		var req smauthv1.LoginRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		resp, err := authHandler.Login(ctx, &req)
		if err != nil {
			return nil, err
		}
		return json.Marshal(resp)
	})

	router.Handle("POST", "/v1/auth/sso/google", func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		var req smauthv1.GoogleLoginRequest
		if err := json.Unmarshal(body, &req); err != nil {
			return nil, err
		}
		resp, err := authHandler.LoginWithGoogle(ctx, &req)
		if err != nil {
			return nil, err
		}
		return json.Marshal(resp)
	})

	forward.RegisterForwardService(grpcServer, router)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)

	// 5. Start HTTP Health/Metrics Server
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

	// 6. Start gRPC Server
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		slog.Error("failed to listen", slog.String("error", err.Error()))
		os.Exit(1)
	}
	healthCheck.SetServingStatus("sm-auth", grpc_health_v1.HealthCheckResponse_SERVING)
	
	go func() {
		slog.Info("Starting gRPC server on :9090")
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("failed to serve gRPC", slog.String("error", err.Error()))
		}
	}()

	// 7. Graceful Shutdown Management
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down gracefully...")
	grpcServer.GracefulStop()
	slog.Info("Server exited properly")
}
