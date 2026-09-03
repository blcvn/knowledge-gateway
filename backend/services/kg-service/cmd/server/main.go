// Package main — kg-service entry point.
//
// Merged services: graphiti-ingestion + graphiti-knowledge + graphiti-pipeline +
//                  graphiti-search + graphiti-store + cognee-ingestion + cognee-cognify +
//                  cognee-pipeline + cognee-search
// (MERGE-P2-T1 + MERGE-P2-T2)
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

	"github.com/jackc/pgx/v5/pgxpool"
	neo4jdriver "github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	"vnp-memory/shared/pkg/forward"
	"vnp-memory/shared/pkg/telemetry"
	"vnp-memory/shared/pkg/tenant"

	cogclient "vnp-memory/services/kg-service/internal/adapter/cognee"
	kggrpc "vnp-memory/services/kg-service/internal/adapter/grpc"
	graphitidomain "vnp-memory/services/kg-service/internal/domain/graphiti"
	"vnp-memory/services/kg-service/internal/infra/config"
	"vnp-memory/services/kg-service/internal/infra/neo4j"
	"vnp-memory/services/kg-service/internal/infra/pgvector"
	uc_cognee "vnp-memory/services/kg-service/internal/usecase/cognee"
	uc_graphiti "vnp-memory/services/kg-service/internal/usecase/graphiti"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "error", err)
		os.Exit(1)
	}

	telemetry.InitLogger(cfg.LogLevel)
	slog.Info("Starting kg-service",
		"grpc_port", cfg.GRPCPort,
		"neo4j_enabled", cfg.Neo4jURL != "",
		"cognee_enabled", cfg.CogneeEnabled,
	)

	if shutdownTracer, err := telemetry.InitProvider(ctx, "kg-service"); err == nil {
		defer func() { _ = shutdownTracer(context.Background()) }()
	}

	// ─── PostgreSQL ────────────────────────────────────────────────────────
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connect failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		slog.Error("database ping failed", "error", err)
		os.Exit(1)
	}

	// ─── Neo4j (optional) ──────────────────────────────────────────────────
	var gRepo uc_graphiti.GraphRepoInterface = &noopGraphRepo{}
	if cfg.Neo4jURL != "" {
		driver, err := neo4jdriver.NewDriverWithContext(cfg.Neo4jURL,
			neo4jdriver.BasicAuth(cfg.Neo4jUser, cfg.Neo4jPassword, ""))
		if err == nil {
			if pingErr := driver.VerifyConnectivity(ctx); pingErr == nil {
				slog.Info("Connected to Neo4j")
				gRepo = neo4j.NewGraphRepo(driver)
			} else {
				slog.Warn("Neo4j not reachable, using no-op graph", "error", pingErr)
			}
		} else {
			slog.Warn("Neo4j driver init failed", "error", err)
		}
	}

	// ─── Infra Repos ───────────────────────────────────────────────────────
	episodeRepo := pgvector.NewEpisodeRepo(pool)
	datasetRepo := pgvector.NewDatasetRepo(pool)

	// ─── Graphiti Usecases ─────────────────────────────────────────────────
	ingestUC := uc_graphiti.NewIngestUseCase(episodeRepo, gRepo, nil, nil)
	storeUC := uc_graphiti.NewStoreUseCase(gRepo)
	searchUC := uc_graphiti.NewSearchUseCase(episodeRepo, gRepo, nil)
	knowledgeUC := uc_graphiti.NewKnowledgeUseCase(gRepo)

	// ─── Cognee Client ─────────────────────────────────────────────────────
	var cClient cogclient.Interface = &cogclient.NoopClient{}
	if cfg.CogneeEnabled && cfg.CogneeURL != "" {
		cClient = cogclient.NewHTTPClient(cfg.CogneeURL, cfg.CogneeAPIKey, cfg.CogneeTimeoutSec)
		slog.Info("Cognee client enabled", "url", cfg.CogneeURL)
	}

	datasetUC := uc_cognee.NewDatasetUseCase(cClient, datasetRepo, cfg.CogneeEnabled)
	cognifyUC := uc_cognee.NewCognifyUseCase(cClient, datasetRepo, cfg.CogneeEnabled)
	csearchUC := uc_cognee.NewCogneeSearchUseCase(cClient, cfg.CogneeEnabled)
	// CR-COGNEE-001: Non-destructive graph enrichment
	memifyUC := uc_cognee.NewMemifyUseCase(cClient, datasetRepo, cfg.CogneeEnabled)
	// CR-COGNEE-002: NodeSet-scoped search
	nsSearchUC := uc_cognee.NewNodeSetsSearchUseCase(cClient, cfg.CogneeEnabled)
	// CR-COGNEE-003: Schema-defined DataPoint ingestion (zero LLM tokens)
	datapointsUC := uc_cognee.NewAddDataPointsUseCase(cClient, datasetRepo, cfg.CogneeEnabled)

	// ─── Handler + Router ──────────────────────────────────────────────────
	handler := kggrpc.NewKGHandler(
		ingestUC, storeUC, searchUC, knowledgeUC,
		datasetUC, cognifyUC, csearchUC,
		memifyUC, nsSearchUC, datapointsUC,
	)
	logger := slog.Default()
	router := forward.NewRouter(logger)
	kggrpc.RegisterRoutes(router, handler)

	// ─── gRPC Server ────────────────────────────────────────────────────────
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(tenant.UnaryServerInterceptor()),
	)
	forward.RegisterForwardService(grpcServer, router)
	healthSrv := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthSrv)
	healthSrv.SetServingStatus("kg-service", grpc_health_v1.HealthCheckResponse_SERVING)

	// ─── HTTP Health ────────────────────────────────────────────────────────
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","service":"kg-service"}`))
		})
		addr := fmt.Sprintf(":%d", cfg.HealthPort)
		slog.Info("HTTP health server", "addr", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Error("health server failed", "error", err)
		}
	}()

	// ─── Serve ──────────────────────────────────────────────────────────────
	grpcAddr := fmt.Sprintf(":%d", cfg.GRPCPort)
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		slog.Error("listen failed", "addr", grpcAddr, "error", err)
		os.Exit(1)
	}
	go func() {
		slog.Info("gRPC ForwardService listening", "addr", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("gRPC serve failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down kg-service...")
	grpcServer.GracefulStop()
}

// noopGraphRepo is a no-op stub when Neo4j is unavailable.
type noopGraphRepo struct{}

func (r *noopGraphRepo) UpsertNode(_ context.Context, _ *graphitidomain.Node) error { return nil }
func (r *noopGraphRepo) UpsertEdge(_ context.Context, _ *graphitidomain.Edge) error { return nil }
func (r *noopGraphRepo) GetNode(_ context.Context, _, _ string) (*graphitidomain.Node, error) {
	return nil, fmt.Errorf("neo4j: not available")
}
func (r *noopGraphRepo) GetEdge(_ context.Context, _, _ string) (*graphitidomain.Edge, error) {
	return nil, fmt.Errorf("neo4j: not available")
}
func (r *noopGraphRepo) GetNeighbors(_ context.Context, _, _ string, _ int) ([]*graphitidomain.Node, []*graphitidomain.Edge, error) {
	return nil, nil, nil
}
func (r *noopGraphRepo) GetOntology(_ context.Context, tenantID string) (*graphitidomain.Ontology, error) {
	return &graphitidomain.Ontology{TenantID: tenantID}, nil
}
func (r *noopGraphRepo) UpdateOntology(_ context.Context, _ *graphitidomain.Ontology) error {
	return nil
}
func (r *noopGraphRepo) QuerySubgraph(_ context.Context, _, _ string) ([]*graphitidomain.Node, []*graphitidomain.Edge, error) {
	return nil, nil, nil
}
