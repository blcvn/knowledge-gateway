---
id: TASK-005
title: "Health Aggregation + main.go Entry Point"
app: apps/graphiti
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-004]
---

## Mục Tiêu

Implement aggregated health check HTTP server và main.go entry point kết nối tất cả components.

## Scope

### In Scope
- `internal/health/server.go` — Aggregated health HTTP server (/healthz, /readyz)
- `cmd/graphiti/main.go` — Main entry point, wires everything together
- Signal handling (SIGTERM, SIGINT)
- Startup banner và summary logging

### Out of Scope
- Dockerfile, Makefile (TASK-006)
- Documentation (TASK-007)

## Thiết Kế Kỹ Thuật

### Health Server

```go
// internal/health/server.go
package health

type Server struct {
    supervisor *supervisor.Supervisor
    port       int
    logger     *slog.Logger
}

// Handler returns http.Handler with:
// GET /healthz — liveness (always 200 if process running)
// GET /readyz  — readiness (200 only if all services SERVING)
// GET /status  — JSON with per-service health details
func (s *Server) Handler() http.Handler

func (s *Server) Start(ctx context.Context) error
```

### main.go

```go
// cmd/graphiti/main.go
func main() {
    // 1. Load config
    cfg := config.Load()

    // 2. Setup logger
    logger := setupLogger(cfg)

    // 3. Set ENV vars for embedded services
    cfg.SetServiceEnvVars()

    // 4. Create supervisor
    sv := supervisor.New(logger)

    // 5. Register services (phase-ordered)
    sv.Register(supervisor.ServiceSpec{
        Name: "graphiti-store", Port: cfg.StoreGRPCPort, Phase: 1,
        StartFn: func(ctx context.Context) error {
            return embed.StartStoreService(ctx, cfg)
        },
    })
    sv.Register(supervisor.ServiceSpec{
        Name: "graphiti-knowledge", Port: cfg.KnowledgeGRPCPort, Phase: 2,
        StartFn: func(ctx context.Context) error {
            return embed.StartKnowledgeService(ctx, cfg)
        },
    })
    // ... ingestion, search, pipeline (Phase 3)
    // ... gateway (Phase 4, Port: 0)

    // 6. Signal context
    ctx, cancel := signal.NotifyContext(context.Background(),
        os.Interrupt, syscall.SIGTERM)
    defer cancel()

    // 7. Start health server
    healthSrv := health.NewServer(sv, cfg.HealthPort, logger)
    go healthSrv.Start(ctx)

    // 8. Start all services (blocking until all phases ready)
    if err := sv.StartAll(ctx); err != nil {
        logger.Error("startup failed", "error", err)
        os.Exit(1)
    }

    logger.Info("graphiti-app running",
        "rest", cfg.GatewayRESTPort,
        "mcp", cfg.GatewayMCPPort,
        "health", cfg.HealthPort,
    )

    // 9. Wait for shutdown signal
    <-ctx.Done()
    logger.Info("shutdown signal received")

    // 10. Graceful shutdown
    shutdownCtx, shutdownCancel := context.WithTimeout(
        context.Background(), time.Duration(cfg.ShutdownTimeout)*time.Second)
    defer shutdownCancel()
    sv.Shutdown(shutdownCtx)

    logger.Info("graphiti-app stopped")
}
```

## Acceptance Criteria

- [x] AC-1: `go build ./cmd/graphiti/` produces single binary
- [x] AC-2: Binary starts all 5 services + gateway in correct order
- [x] AC-3: `GET /healthz` returns 200 when process running
- [x] AC-4: `GET /readyz` returns 200 only when all services SERVING
- [x] AC-5: `GET /status` returns JSON with per-service health
- [x] AC-6: SIGTERM triggers ordered graceful shutdown
- [x] AC-7: Startup log shows all services + ports

## Definition of Done

- [x] `go build` succeeds
- [x] Smoke test: binary starts → health check passes → SIGTERM → clean exit
- [x] Không có lint errors
