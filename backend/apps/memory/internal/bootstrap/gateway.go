package bootstrap

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/config"
	gwHandler "github.com/vnp-community/vnp-memory/gateway/adapter/handler"
	gwMCP "github.com/vnp-community/vnp-memory/gateway/adapter/mcp"
	gwDomain "github.com/vnp-community/vnp-memory/gateway/domain"
	"github.com/vnp-community/vnp-memory/gateway/infra/middleware"
	gwUsecase "github.com/vnp-community/vnp-memory/gateway/usecase"
	gwPort "github.com/vnp-community/vnp-memory/gateway/usecase/port"
)

type GatewayServers struct {
	Router http.Handler
	MCP    *gwMCP.Server
}

// Gateway bootstraps the API gateway with all handlers.
// When spaFS is non-nil, the router serves embedded UI assets on non-API routes.
func Gateway(registry gwPort.ServiceRegistry, infra *Infra, cfg *config.Config, logger *slog.Logger, spaFS fs.FS) *GatewayServers {
	logger.Info("Bootstrapping Gateway...")

	// Route usecase — nil publisher for now (events not yet wired)
	routeUC := gwUsecase.NewRouteUseCase(registry, nil, logger)

	// HTTP Handlers (reusing gateway code without modification)
	memoryH := gwHandler.NewMemoryHandler(routeUC, registry, logger)
	cogneeH := gwHandler.NewCogneeHandler(registry, logger)
	graphitiH := gwHandler.NewGraphitiHandler(registry, logger)
	memobaseH := gwHandler.NewMemobaseHandler(registry, logger)
	ovH := gwHandler.NewOpenVikingHandler(registry, logger)
	zepH := gwHandler.NewZepHandler(registry, logger)
	smH := gwHandler.NewSMHandler(registry, logger)
	adminH := gwHandler.NewAdminHandler(registry, logger)
	authH := gwHandler.NewAuthHandler(registry, logger)

	// Console Handlers
	dashboardH := gwHandler.NewDashboardHandler(registry, logger)
	explorerH := gwHandler.NewExplorerHandler(registry, logger)
	graphH := gwHandler.NewGraphHandler(registry, logger)
	profileH := gwHandler.NewProfileHandler(registry, logger)
	adaptiveH := gwHandler.NewAdaptiveHandler(registry, logger)
	debuggerH := gwHandler.NewDebuggerHandler(registry, logger)
	sessionH := gwHandler.NewSessionHandler(registry, logger)
	governanceH := gwHandler.NewGovernanceHandler(registry, logger)
	pipelineH := gwHandler.NewPipelineHandler(registry, logger)
	infraH := gwHandler.NewInfraHandler(registry, logger)
	observabilityH := gwHandler.NewObservabilityHandler(registry, logger)
	wsH := gwHandler.NewWSHandler(logger)

	agentmemH := gwHandler.NewAgentMemoryHandler(registry, logger)

	// Org & SDK Handlers (for completeness in monolith mode)
	orgH := gwHandler.NewOrgHandler(registry, logger)
	sdkH := gwHandler.NewSDKHandler(registry, logger)

	router := gwHandler.Router(
		memoryH, cogneeH, graphitiH, memobaseH, ovH, zepH, smH, adminH, authH,
		dashboardH, explorerH, graphH, profileH, adaptiveH,
		debuggerH, sessionH, governanceH, pipelineH, infraH,
		observabilityH, wsH,
		orgH, sdkH,
		agentmemH,
		logger,
		spaFS,
	)

	// Auth Usecase integration (REAL, no mock)
	authUC, err := gwUsecase.NewAuthUseCase(
		&noopKeyStore{}, &noopPublisher{},
		[]byte(cfg.Auth.JWTPublicKey),
		cfg.Auth.JWTIssuer, cfg.Auth.JWTAudience,
		cfg.Auth.DevMode, logger,
	)
	if err != nil {
		logger.Error("Auth usecase init failed in monolith", "error", err)
	}

	var finalRouter http.Handler = router
	if authUC != nil {
		// Apply the real Auth middleware from the gateway package
		finalRouter = middleware.Auth(authUC, logger)(router)
	}

	mcpSrv := gwMCP.NewServer(registry, logger)

	return &GatewayServers{
		Router: finalRouter,
		MCP:    mcpSrv,
	}
}

// ──── Noop implementations for monolith Auth initialization ────

type noopPublisher struct{}

func (p *noopPublisher) Publish(_ context.Context, _ string, _ any) error { return nil }

type noopKeyStore struct{}

func (s *noopKeyStore) ResolveAPIKey(_ context.Context, _ string) (*gwDomain.AuthContext, error) {
	return nil, gwDomain.ErrUnauthenticated.WithMessage("key store unavailable in monolith")
}

