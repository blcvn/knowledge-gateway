// Package main is the entry point for vnp-gateway.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vnp-community/vnp-memory/gateway/adapter/client"
	"github.com/vnp-community/vnp-memory/gateway/adapter/handler"
	"github.com/vnp-community/vnp-memory/gateway/adapter/mcp"
	"github.com/vnp-community/vnp-memory/gateway/adapter/webdav"
	"github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/infra/config"
	"github.com/vnp-community/vnp-memory/gateway/infra/persistence"
	"github.com/vnp-community/vnp-memory/gateway/infra/server"
	"github.com/vnp-community/vnp-memory/gateway/usecase"
	"github.com/vnp-community/vnp-memory/gateway/usecase/port"
)

func main() {
	// Load config
	cfg := config.Load()

	// Setup structured logger
	logLevel := slog.LevelInfo
	switch cfg.Server.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	logger.Info("starting vnp-gateway",
		"rest_port", cfg.Server.RESTPort,
		"grpc_port", cfg.Server.GRPCPort,
		"mcp_port", cfg.Server.MCPPort,
		"health_port", cfg.Server.HealthPort,
		"dev_mode", cfg.Auth.DevMode,
	)

	// Validate config
	if err := cfg.Validate(); err != nil {
		logger.Error("config validation failed", "error", err)
		os.Exit(1)
	}

	// ──── Infrastructure Setup ────────────────────

	var cleanups []func()
	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()

	// Event Publisher (NATS or noop)
	var publisher port.EventPublisher
	if cfg.NATS.URL != "" {
		natsPub, cleanup, err := persistence.NewNATSPublisher(cfg.NATS.URL, cfg.NATS.CredsFile, logger)
		if err != nil {
			logger.Warn("NATS unavailable, using noop publisher", "error", err)
			publisher = &noopPublisher{}
		} else {
			publisher = natsPub
			cleanups = append(cleanups, cleanup)
		}
	} else {
		publisher = &noopPublisher{}
	}

	// Key Store + Console Stores (PostgreSQL or noop)
	var keyStore port.KeyStore
	var auditStore port.AuditStore
	var policyStore port.PolicyStore
	var pgPool *pgxpool.Pool // Shared pool for direct DB access (auth handler)
	if cfg.Postgres.DSN != "" {
		pool, err := persistence.NewPGPool(cfg.Postgres.DSN, cfg.Postgres.MaxConns, cfg.Postgres.MinConns)
		if err != nil {
			logger.Warn("PostgreSQL unavailable, using noop key store", "error", err)
			keyStore = &noopKeyStore{}
		} else {
			cleanups = append(cleanups, pool.Close)
			pgPool = pool // Share pool reference

			// Auto-migrate schema
			migrateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := persistence.MigrateSchema(migrateCtx, pool); err != nil {
				logger.Error("schema migration failed", "error", err)
			} else {
				logger.Info("database schema migrated")
			}
			// Console schema migration (T08-T09)
			if err := persistence.MigrateConsoleSchema(migrateCtx, pool); err != nil {
				logger.Error("console schema migration failed", "error", err)
			} else {
				logger.Info("console schema migrated")
			}
			// Auth schema migration (TASK-BE-001: console_users, refresh_tokens, sessions, messages)
			if err := persistence.MigrateAuthSchema(migrateCtx, pool); err != nil {
				logger.Error("auth schema migration failed", "error", err)
			} else {
				logger.Info("auth schema migrated (console_users, refresh_tokens, sessions, messages)")
			}
			// Governance schema migration (TASK-BE-009: audit_log_entries, opa_policies)
			if err := persistence.MigrateGovernanceSchema(migrateCtx, pool); err != nil {
				logger.Error("governance schema migration failed", "error", err)
			} else {
				logger.Info("governance schema migrated (audit_log_entries, opa_policies)")
			}
			// Observability schema migration (TASK-BE-010: error_aggregates, llm_cost_events)
			if err := persistence.MigrateObservabilitySchema(migrateCtx, pool); err != nil {
				logger.Error("observability schema migration failed", "error", err)
			} else {
				logger.Info("observability schema migrated (error_aggregates, llm_cost_events)")
			}
			// Org & SDK schema migration (TASK-BE-013: sdk_api_keys, webhooks)
			if err := persistence.MigrateOrgSDKSchema(migrateCtx, pool); err != nil {
				logger.Error("org/sdk schema migration failed", "error", err)
			} else {
				logger.Info("org/sdk schema migrated (sdk_api_keys, webhooks)")
			}
			cancel()

			pgStore := persistence.NewPGTenantStore(pool, logger)
			keyStore = pgStore

			// Console stores (T08-T09)
			auditStore = persistence.NewPGAuditStore(pool, logger)
			policyStore = persistence.NewPGPolicyStore(pool, logger)
		}
	} else {
		keyStore = &noopKeyStore{}
		auditStore = &noopAuditStore{}
		policyStore = &noopPolicyStore{}
	}

	// Rate Limit Store (Redis or noop)
	var rateLimitStore port.RateLimitStore
	if cfg.Redis.Addr != "" {
		redisClient, err := persistence.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			logger.Warn("Redis unavailable, using noop rate limiter", "error", err)
			rateLimitStore = &noopRateLimitStore{}
		} else {
			cleanups = append(cleanups, func() { redisClient.Close() })
			rateLimitStore = persistence.NewRedisRateLimiter(redisClient, logger)
		}
	} else {
		rateLimitStore = &noopRateLimitStore{}
	}

	// Service Registry (gRPC + Circuit Breaker)
	var registry port.ServiceRegistry
	grpcReg, grpcCleanup, err := client.NewGRPCRegistry(cfg.Services, cfg.Timeout.Default, logger)
	if err != nil {
		logger.Warn("gRPC registry init failed, using noop", "error", err)
		registry = &noopRegistry{}
	} else {
		cleanups = append(cleanups, grpcCleanup)
		circuitReg := client.NewCircuitRegistry(grpcReg, client.CircuitConfig{
			MaxFailures: cfg.Circuit.MaxFailures,
			Timeout:     cfg.Circuit.Timeout,
			MaxRequests: cfg.Circuit.MaxRequests,
		}, logger)
		registry = circuitReg
	}

	// ──── Usecase Layer ───────────────────────────

	// Auth
	authUC, err := usecase.NewAuthUseCase(
		keyStore, publisher,
		[]byte(cfg.Auth.JWTPublicKey),
		cfg.Auth.JWTIssuer, cfg.Auth.JWTAudience,
		cfg.Auth.DevMode, logger,
	)
	if err != nil {
		logger.Error("auth usecase init failed", "error", err)
		os.Exit(1)
	}

	// Route (auto-classification)
	routeUC := usecase.NewRouteUseCase(registry, publisher, logger)

	// Rate Limit
	rateLimitUC := usecase.NewRateLimitUseCase(rateLimitStore, logger)
	_ = rateLimitUC // Used in middleware
	_ = authUC      // Used in middleware

	// Console Usecases (T08-T13)
	auditUC := usecase.NewAuditUseCase(auditStore, publisher, logger)
	_ = usecase.NewPolicyUseCase(policyStore, publisher, logger)
	_ = usecase.NewPipelineUseCase(registry, logger)
	_ = usecase.NewInfraUseCase(registry, logger)
	_ = usecase.NewSearchUseCase(registry, logger)
	_ = usecase.NewForgetUseCase(registry, publisher, auditUC, logger)

	// ──── Adapter Layer ───────────────────────────

	// HTTP Handlers
	memoryH := handler.NewMemoryHandler(routeUC, registry, logger)
	cogneeH := handler.NewCogneeHandler(registry, logger)
	graphitiH := handler.NewGraphitiHandler(registry, logger)
	memobaseH := handler.NewMemobaseHandler(registry, logger)
	ovH := handler.NewOpenVikingHandler(registry, logger)
	zepH := handler.NewZepHandler(registry, logger)
	smH := handler.NewSMHandler(registry, logger)
	adminH := handler.NewAdminHandler(registry, logger)
	// Auth Handler — sử dụng DB pool nếu có (TASK-BE-002)
	var authH *handler.AuthHandler
	if pgPool != nil {
		authH = handler.NewAuthHandlerWithDB(registry, pgPool, authUC, logger)
	} else {
		authH = handler.NewAuthHandler(registry, logger)
	}

	// Console Handlers (SOL-002)
	dashboardH := handler.NewDashboardHandler(registry, logger)
	explorerH := handler.NewExplorerHandler(registry, logger)
	graphH := handler.NewGraphHandler(registry, logger)
	profileH := handler.NewProfileHandler(registry, logger)
	adaptiveH := handler.NewAdaptiveHandler(registry, logger)
	debuggerH := handler.NewDebuggerHandler(registry, logger)
	sessionH := handler.NewSessionHandler(registry, logger)
	governanceH := handler.NewGovernanceHandler(registry, logger)
	pipelineH := handler.NewPipelineHandler(registry, logger)
	infraH := handler.NewInfraHandler(registry, logger)
	observabilityH := handler.NewObservabilityHandler(registry, logger)
	wsH := handler.NewWSHandler(logger)
	// Org & SDK Handlers (TASK-BE-013)
	orgH := handler.NewOrgHandler(registry, logger)
	sdkH := handler.NewSDKHandler(registry, logger)

	// AgentMemory Handler (TASK-AM-002/005)
	agentmemH := handler.NewAgentMemoryHandler(registry, logger)

	// HTTP Router
	router := handler.Router(
		memoryH, cogneeH, graphitiH, memobaseH, ovH, zepH, smH, adminH, authH,
		dashboardH, explorerH, graphH, profileH, adaptiveH,
		debuggerH, sessionH, governanceH, pipelineH, infraH,
		observabilityH, wsH,
		orgH, sdkH,
		agentmemH,
		logger,
		nil, // no embedded UI in standalone gateway mode
	)

	// MCP Server
	mcpSrv := mcp.NewServer(registry, logger)

	// WebDAV Proxy
	webdavProxy := webdav.NewProxy(registry, logger)
	_ = webdavProxy // Mounted at /webdav in router

	// ──── Server Lifecycle ────────────────────────

	// Graceful shutdown context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var wg sync.WaitGroup

	// REST server
	restSrv := server.NewHTTPServer(router, cfg.Server.RESTPort, logger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := restSrv.Start(ctx); err != nil {
			logger.Error("REST server error", "error", err)
		}
	}()

	// MCP server (separate port)
	mcpHTTPSrv := server.NewHTTPServer(mcpSrv.Handler(), cfg.Server.MCPPort, logger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := mcpHTTPSrv.Start(ctx); err != nil {
			logger.Error("MCP server error", "error", err)
		}
	}()

	// Observability server (health + metrics)
	obsSrv := server.NewObservabilityServer(cfg.Server.HealthPort, registry, logger)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := obsSrv.Start(ctx); err != nil {
			logger.Error("observability server error", "error", err)
		}
	}()

	// Background health checker
	if grpcReg != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			grpcReg.StartHealthCheck(ctx, 30*time.Second)
		}()
	}

	logger.Info("vnp-gateway running",
		"rest", cfg.Server.RESTPort,
		"mcp", cfg.Server.MCPPort,
		"health", cfg.Server.HealthPort,
	)
	wg.Wait()
	logger.Info("vnp-gateway stopped")
}

// ──── Noop implementations for graceful degradation ────

type noopPublisher struct{}

func (p *noopPublisher) Publish(_ context.Context, _ string, _ any) error { return nil }

type noopKeyStore struct{}

func (s *noopKeyStore) ResolveAPIKey(_ context.Context, _ string) (*domain.AuthContext, error) {
	return nil, domain.ErrUnauthenticated.WithMessage("key store unavailable")
}

type noopRateLimitStore struct{}

func (s *noopRateLimitStore) CheckAndIncrement(_ context.Context, _ string, _ int, _ int) (bool, int, error) {
	return true, 9999, nil // fail-open
}

type noopRegistry struct{}

func (r *noopRegistry) Resolve(service string) (*domain.RouteTarget, error) {
	return &domain.RouteTarget{Service: service, Address: "localhost:0", Timeout: 30 * time.Second}, nil
}
func (r *noopRegistry) Forward(_ context.Context, _ *domain.RouteTarget, _ []byte) ([]byte, error) {
	return []byte(`{"status":"not_connected"}`), nil
}
func (r *noopRegistry) ForwardWithContext(_ context.Context, _ *domain.RouteTarget, _ *domain.ForwardRequest) ([]byte, error) {
	return []byte(`{"status":"not_connected"}`), nil
}
func (r *noopRegistry) HealthCheck(_ string) (bool, error) { return false, nil }

type noopAuditStore struct{}

func (s *noopAuditStore) Insert(_ context.Context, _ *domain.AuditEntry) error { return nil }
func (s *noopAuditStore) Search(_ context.Context, _ *domain.AuditFilter) ([]*domain.AuditEntry, int, error) {
	return nil, 0, nil
}

type noopPolicyStore struct{}

func (s *noopPolicyStore) List(_ context.Context, _ string) ([]*domain.Policy, error) {
	return nil, nil
}
func (s *noopPolicyStore) Get(_ context.Context, _ string) (*domain.Policy, error) {
	return nil, domain.ErrNotFound.WithMessage("policy store unavailable")
}
func (s *noopPolicyStore) Create(_ context.Context, _ *domain.Policy) error { return nil }
func (s *noopPolicyStore) Update(_ context.Context, _ *domain.Policy) error { return nil }
