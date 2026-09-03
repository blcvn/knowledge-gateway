// Package main — vnp-platform entry point.
//
// Consolidated service hosting: auth, admin, tenant, events, analytics, spaces, dashboard.
// Absorbed from: sm-auth (T1), vnp-admin (T2), vnp-event/vnp-dashboard/sm-analytics/sm-project (T3)
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/shared/pkg/forward"
	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"

	grpchandlers "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/adapter/grpc"
	admindomain "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/domain/admin"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/infra/config"
	"github.com/vnp-community/vnp-memory/services/vnp-platform/internal/infra/persistence"
	ucadmin "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/admin"
	ucanalytics "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/analytics"
	ucauth "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/auth"
	ucevent "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/event"
	ucproject "github.com/vnp-community/vnp-memory/services/vnp-platform/internal/usecase/project"
)


func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ─── Config ───────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// ─── Telemetry ────────────────────────────────────────────────────────
	telemetry.InitLogger(cfg.LogLevel)
	slog.Info("Starting vnp-platform", "grpc_port", cfg.GRPCPort, "health_port", cfg.HealthPort)

	shutdownTracer, err := telemetry.InitProvider(ctx, "vnp-platform")
	if err != nil {
		slog.Error("failed to initialize OTel provider", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			slog.Error("failed to shutdown OTel", "error", err)
		}
	}()

	// ─── Database ─────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	slog.Info("Connected to PostgreSQL")

	// ─── Repositories ─────────────────────────────────────────────────────
	tenantRepo := persistence.NewTenantRepo(pool)
	apiKeyRepo := persistence.NewAPIKeyRepo(pool)
	adminUserRepo := persistence.NewAdminUserRepo(pool)
	authUserRepo := persistence.NewAuthUserRepo(pool)
	eventRepo := persistence.NewEventRepo(pool)
	usageRepo := persistence.NewUsageRepo(pool)
	spaceRepo := persistence.NewSpaceRepo(pool)

	// ─── NATS Publisher (optional, degrade gracefully) ─────────────────────
	// noopPublisher used when NATS_URL is not configured
	pub := &noopPublisher{}

	// ─── Usecases ─────────────────────────────────────────────────────────
	tenantSvc := ucadmin.NewTenantService(tenantRepo, pub)
	apiKeySvc := ucadmin.NewAPIKeyService(apiKeyRepo)
	userSvc := ucadmin.NewUserService(adminUserRepo)
	healthSvc := ucadmin.NewHealthService()
	eventSvc := ucevent.NewService(eventRepo)
	analyticsSvc := ucanalytics.NewService(usageRepo)
	projectSvc := ucproject.NewService(spaceRepo)

	// ─── Auth Service (T1 - sm-auth absorbed) ─────────────────────────────
	authSvc, err := ucauth.NewAuthService(authUserRepo, cfg.JWTPrivateKey, cfg.GoogleClientID)
	if err != nil {
		slog.Error("failed to initialize auth service", "error", err)
		os.Exit(1)
	}

	// ─── Handlers ─────────────────────────────────────────────────────────
	authHandler := grpchandlers.NewAuthHandler(authSvc)
	platformHandler := grpchandlers.NewPlatformForwardHandler(
		tenantSvc, apiKeySvc, userSvc, healthSvc,
		eventSvc, analyticsSvc, projectSvc,
	)

	// ─── gRPC + ForwardService Router ─────────────────────────────────────
	logger := slog.Default()
	router := forward.NewRouter(logger)
	registerRoutes(router, authHandler, platformHandler)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)
	forward.RegisterForwardService(grpcServer, router)

	healthCheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthCheck)
	healthCheck.SetServingStatus("vnp-platform", grpc_health_v1.HealthCheckResponse_SERVING)

	// ─── HTTP Health Server ────────────────────────────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		})
		mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
			if err := pool.Ping(r.Context()); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"not ready"}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		})
		addr := fmt.Sprintf(":%d", cfg.HealthPort)
		slog.Info("Starting HTTP health server", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("HTTP health server failed", "error", err)
		}
	}()

	// ─── Start gRPC Server ─────────────────────────────────────────────────
	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("failed to listen", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("Starting gRPC ForwardService", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC server failed", "error", err)
		}
	}()

	// ─── Graceful Shutdown ─────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down vnp-platform gracefully...")
	grpcServer.GracefulStop()
	slog.Info("vnp-platform stopped")
}

// registerRoutes wires all HTTP path patterns to handler functions.
func registerRoutes(router *forward.Router, auth *grpchandlers.AuthHandler, platform *grpchandlers.PlatformForwardHandler) {
	// ── Auth routes (T1 - sm-auth) ──
	router.Handle("POST", "/v1/auth/register", adapt(auth.Register))
	router.Handle("POST", "/v1/auth/login", adapt(auth.Login))
	router.Handle("POST", "/v1/auth/sso/google", adapt(auth.LoginWithGoogle))

	// ── Admin routes (T2) ──
	router.Handle("POST", "/v1/admin/tenants", adapt(platform.CreateTenant))
	router.Handle("GET", "/v1/admin/tenants", adapt(platform.ListTenants))
	router.Handle("GET", "/v1/admin/tenants/*", adapt(platform.GetTenant))
	router.Handle("PUT", "/v1/admin/tenants/*", adapt(platform.UpdateTenant))
	router.Handle("POST", "/v1/admin/tenants/*/keys", adapt(platform.IssueAPIKey))
	router.Handle("GET", "/v1/admin/health", adapt(platform.Health))
	router.Handle("GET", "/v1/admin/metrics", adapt(platform.Metrics))

	// ── Governance console routes (T2) ──
	router.Handle("GET", "/v1/console/governance/tenants", adapt(platform.ListTenants))
	router.Handle("POST", "/v1/console/governance/tenants", adapt(platform.CreateTenant))
	router.Handle("PUT", "/v1/console/governance/tenants/*", adapt(platform.UpdateTenant))
	router.Handle("GET", "/v1/console/governance/policies", adapt(platform.ListPolicies))
	router.Handle("POST", "/v1/console/governance/policies", adapt(platform.CreatePolicy))
	router.Handle("PUT", "/v1/console/governance/policies/*", adapt(platform.UpdatePolicy))
	router.Handle("GET", "/v1/console/governance/audit", adapt(platform.SearchAudit))
	router.Handle("POST", "/v1/console/governance/gdpr/forget", adapt(platform.GDPRForget))
	router.Handle("POST", "/v1/console/governance/gdpr/forget/preview", adapt(platform.GDPRForgetPreview))

	// ── Dashboard console routes (T3 - vnp-dashboard) ──
	router.Handle("GET", "/v1/console/dashboard/health", adapt(platform.DashboardHealth))
	router.Handle("GET", "/v1/console/dashboard/metrics", adapt(platform.DashboardMetrics))
	router.Handle("GET", "/v1/console/dashboard/throughput", adapt(platform.DashboardThroughput))
	router.Handle("GET", "/v1/console/dashboard/heatmap", adapt(platform.DashboardHeatmap))

	// ── Event routes (T3 - vnp-event) ──
	router.Handle("GET", "/v1/memobase/users/*/events", adapt(platform.GetUserEvents))
	router.Handle("GET", "/v1/console/profiles/*/events", adapt(platform.GetProfileEvents))

	// ── Analytics routes (T3 - sm-analytics) ──
	router.Handle("GET", "/v1/console/adaptive/analytics", adapt(platform.GetAnalytics))
	router.Handle("GET", "/v1/console/adaptive/forget-rules", adapt(platform.GetForgetRules))
	router.Handle("PUT", "/v1/console/adaptive/forget-rules", adapt(platform.UpdateForgetRules))

	// ── Space routes (T3 - sm-project) ──
	router.Handle("POST", "/v1/sm/projects/spaces", adapt(platform.CreateSpace))

	// ── Profile console routes (T3) ──
	router.Handle("GET", "/v1/console/profiles", adapt(platform.ListProfiles))
	router.Handle("GET", "/v1/console/profiles/config", adapt(platform.GetProfileConfig))
	router.Handle("PUT", "/v1/console/profiles/config", adapt(platform.UpdateProfileConfig))
	router.Handle("GET", "/v1/console/profiles/*", adapt(platform.GetProfile))
	router.Handle("GET", "/v1/console/profiles/*/context", adapt(platform.GetProfileContext))
	router.Handle("GET", "/v1/console/profiles/*/buffers", adapt(platform.GetProfileBuffers))

	// ── Debugger console routes (T3) ──
	router.Handle("POST", "/v1/console/debugger/trace", adapt(platform.CreateTrace))
	router.Handle("GET", "/v1/console/debugger/traces/*", adapt(platform.GetTrace))
	router.Handle("GET", "/v1/console/debugger/traces", adapt(platform.ListTraces))

	// ── Session console routes (T3) ──
	router.Handle("GET", "/v1/console/sessions/live", adapt(platform.ListLiveSessions))
	router.Handle("GET", "/v1/console/sessions", adapt(platform.ListSessions))
	router.Handle("GET", "/v1/console/sessions/*", adapt(platform.GetSession))
	router.Handle("GET", "/v1/console/sessions/*/timeline", adapt(platform.GetSessionTimeline))
	router.Handle("GET", "/v1/console/sessions/*/diff", adapt(platform.GetSessionDiff))
	router.Handle("GET", "/v1/console/sessions/*/working-memory", adapt(platform.GetWorkingMemory))
	router.Handle("GET", "/v1/console/sessions/*/user-summary", adapt(platform.GetUserSummary))

	// ── Pipeline status (served by platform, forwarded from gateway) ──
	router.Handle("GET", "/v1/console/pipelines/status", adapt(platform.PipelineStatus))
}

// adapt converts an http.HandlerFunc to forward.HandlerFunc.
// Bridges the ForwardService RPC protocol to standard http.HandlerFunc.
func adapt(h http.HandlerFunc) forward.HandlerFunc {
	return func(ctx context.Context, body []byte, params map[string]string) ([]byte, error) {
		// Reconstruct path from params
		path := "/"
		if p, ok := params["__path"]; ok {
			path = p
		}
		method := "GET"
		if m, ok := params["__method"]; ok {
			method = m
		}

		// Build synthetic http.Request
		u, _ := url.Parse(path)
		if q, ok := params["__query"]; ok {
			u.RawQuery = q
		}
		req, _ := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		// Inject path parameters (id, user_id, etc.)
		for k, v := range params {
			if k[0] == '_' {
				continue // skip internal keys
			}
			req.SetPathValue(k, v)
		}

		// Capture response
		rw := &responseCapture{header: make(http.Header)}
		h(rw, req)

		if rw.code >= 500 {
			var errResp map[string]string
			_ = json.Unmarshal(rw.body.Bytes(), &errResp)
			if msg, ok := errResp["error"]; ok {
				return rw.body.Bytes(), fmt.Errorf("%s", msg)
			}
			return rw.body.Bytes(), fmt.Errorf("HTTP %d", rw.code)
		}

		return rw.body.Bytes(), nil
	}
}

// responseCapture captures http.ResponseWriter output.
type responseCapture struct {
	header http.Header
	body   bytes.Buffer
	code   int
}

func (rc *responseCapture) Header() http.Header        { return rc.header }
func (rc *responseCapture) WriteHeader(code int)        { rc.code = code }
func (rc *responseCapture) Write(b []byte) (int, error) { return rc.body.Write(b) }

// noopPublisher is a no-op implementation of port.EventPublisher.
// Used when NATS is unavailable at startup.
type noopPublisher struct{}

func (p *noopPublisher) PublishTenantCreated(_ context.Context, _ *admindomain.Tenant) error {
	return nil
}
func (p *noopPublisher) PublishTenantDeleted(_ context.Context, _ uuid.UUID) error { return nil }

// Ensure io is used (for io.NopCloser in adapt)
var _ io.Reader = bytes.NewReader(nil)
