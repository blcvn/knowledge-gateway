---
id: TASK-005
title: "Health Aggregation + main.go Entry Point"
app: apps/cognee
version: 1.0.0
status: Done
priority: P0
created: 2026-05-12
updated: 2026-05-12
linked_sol: SOL-001
depends_on: [TASK-004]
estimated: 1.5h
---

## Mục Tiêu

Tạo main.go orchestrator, aggregated health endpoint, và wiring tất cả services + gateway qua supervisor.

## Scope

### In Scope
- `cmd/cognee/main.go` — Entry point: config → supervisor → startup → signal → shutdown
- `cmd/cognee/health.go` — Aggregated health server polling all embedded services
- Final wiring of all ServiceSpec registrations

### Out of Scope
- Docker/Makefile (TASK-006)

## Thiết Kế Kỹ Thuật

### main.go

```go
// cmd/cognee/main.go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "vnp-memory/apps/cognee/internal/config"
    "vnp-memory/apps/cognee/internal/supervisor"
    "vnp-memory/pkg/telemetry"
)

func main() {
    // 1. Load unified config
    cfg, err := config.Load()
    if err != nil {
        slog.Error("config load failed", "error", err)
        os.Exit(1)
    }
    if err := cfg.Validate(); err != nil {
        slog.Error("config validation failed", "error", err)
        os.Exit(1)
    }

    // 2. Init logger
    telemetry.InitLogger(cfg.LogLevel)
    slog.Info("starting cognee-app", "env", cfg.Environment)

    // 3. Set ENV vars for embedded services
    cfg.SetServiceEnvVars()

    // 4. Create supervisor
    sv := supervisor.New(slog.Default())

    // 5. Register Phase 0: Cognee services
    sv.Register(supervisor.ServiceSpec{
        Name:  "cognee-ingestion",
        Phase: supervisor.PhaseInfra,
        Port:  cfg.IngestionPort,
        StartFn: func(ctx context.Context) error {
            return startIngestionService(ctx, cfg)
        },
    })
    sv.Register(supervisor.ServiceSpec{
        Name:  "cognee-cognify",
        Phase: supervisor.PhaseInfra,
        Port:  cfg.CognifyPort,
        StartFn: func(ctx context.Context) error {
            return startCognifyService(ctx, cfg)
        },
    })
    sv.Register(supervisor.ServiceSpec{
        Name:  "cognee-search",
        Phase: supervisor.PhaseInfra,
        Port:  cfg.SearchPort,
        StartFn: func(ctx context.Context) error {
            return startSearchService(ctx, cfg)
        },
    })

    // 6. Register Phase 1: Gateway (after services ready)
    sv.Register(supervisor.ServiceSpec{
        Name:  "vnp-gateway",
        Phase: supervisor.PhaseGateway,
        Port:  cfg.GatewayRESTPort,
        StartFn: func(ctx context.Context) error {
            return startGateway(ctx, cfg)
        },
    })

    // 7. Start aggregated health server
    go startHealthServer(cfg, sv)

    // 8. Start all with signal handling
    ctx, cancel := signal.NotifyContext(context.Background(),
        os.Interrupt, syscall.SIGTERM)
    defer cancel()

    slog.Info("cognee-app starting all services...")
    if err := sv.StartAll(ctx); err != nil {
        slog.Error("supervisor error", "error", err)
        os.Exit(1)
    }

    // 9. Graceful shutdown
    sv.Shutdown(30 * time.Second)
    slog.Info("cognee-app stopped")
}
```

### Aggregated Health Server

```go
// cmd/cognee/health.go
package main

func startHealthServer(cfg *config.Config, sv *supervisor.Supervisor) {
    mux := http.NewServeMux()

    // /healthz — liveness (app process alive)
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
    })

    // /readyz — readiness (all services ready)
    mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
        status := sv.HealthCheck()
        allReady := true
        for _, s := range status {
            if s != "SERVING" {
                allReady = false
                break
            }
        }
        w.Header().Set("Content-Type", "application/json")
        if !allReady {
            w.WriteHeader(http.StatusServiceUnavailable)
        }
        json.NewEncoder(w).Encode(map[string]any{
            "status":   boolToStatus(allReady),
            "services": status,
        })
    })

    slog.Info("health server starting", "port", cfg.HealthPort)
    http.ListenAndServe(fmt.Sprintf(":%d", cfg.HealthPort), mux)
}
```

## Acceptance Criteria

- [x] AC-1: `go build ./cmd/cognee/` produces single binary
- [x] AC-2: Binary starts all 3 services + gateway sequentially
- [x] AC-3: `/healthz` responds 200 immediately
- [x] AC-4: `/readyz` responds 200 only when all services SERVING
- [x] AC-5: SIGTERM → ordered shutdown (gateway → services)
- [x] AC-6: Startup logs show each service name + port
- [x] AC-7: Total main.go + health.go ≤ 120 lines

## Definition of Done

- [x] main.go compiles and runs
- [x] Health endpoints verified
- [x] Ordered startup/shutdown verified
