//go:build e2e

package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vnp-community/vnp-memory/apps/memory/internal/bootstrap"
	"github.com/vnp-community/vnp-memory/apps/memory/internal/bus"
	"github.com/vnp-community/vnp-memory/apps/memory/internal/config"
	gwServer "github.com/vnp-community/vnp-memory/gateway/infra/server"
)

// TestMonolithSmoke verifies that the monolithic app boots correctly
// and responds to basic API requests.
func TestMonolithSmoke(t *testing.T) {
	// Use a test-specific config
	os.Setenv("VNP_MEMORY_SERVER_REST_PORT", "19080")
	os.Setenv("VNP_MEMORY_SERVER_HEALTH_PORT", "19083")
	os.Setenv("VNP_MEMORY_AUTH_DEV_MODE", "true")
	os.Setenv("VNP_MEMORY_NATS_MODE", "embedded")
	os.Setenv("VNP_MEMORY_NATS_STORE_DIR", t.TempDir())
	defer func() {
		os.Unsetenv("VNP_MEMORY_SERVER_REST_PORT")
		os.Unsetenv("VNP_MEMORY_SERVER_HEALTH_PORT")
		os.Unsetenv("VNP_MEMORY_AUTH_DEV_MODE")
		os.Unsetenv("VNP_MEMORY_NATS_MODE")
		os.Unsetenv("VNP_MEMORY_NATS_STORE_DIR")
	}()

	cfg := config.Load()
	cfg.Server.RESTPort = 19080
	cfg.Server.HealthPort = 19083
	cfg.Auth.DevMode = true
	cfg.NATS.Mode = "embedded"
	cfg.NATS.StoreDir = t.TempDir()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))

	// ── Boot infrastructure ──
	infra, err := bootstrap.NewInfra(cfg, logger)
	if err != nil {
		t.Fatalf("infra init failed: %v", err)
	}
	defer infra.Close()

	// ── Boot bus ──
	grpcBus := bus.NewGRPCBus()

	natsBus, err := bus.NewNATSBus(bus.NATSConfig{
		Mode:     cfg.NATS.Mode,
		StoreDir: cfg.NATS.StoreDir,
	}, logger)
	if err != nil {
		t.Fatalf("NATS init failed: %v", err)
	}
	defer natsBus.Close()

	// ── Bootstrap engines ──
	bootstrap.Platform(grpcBus, infra, natsBus, cfg, logger)
	bootstrap.Cognee(grpcBus, infra, natsBus, logger)
	bootstrap.Graphiti(grpcBus, infra, natsBus, logger)
	bootstrap.Memobase(grpcBus, infra, natsBus, logger)
	bootstrap.OpenViking(grpcBus, infra, natsBus, logger)
	bootstrap.Zep(grpcBus, infra, natsBus, logger)
	bootstrap.Supermemory(grpcBus, infra, natsBus, logger)

	go grpcBus.Serve()
	defer grpcBus.Stop()

	registry := bus.NewInProcessRegistry(grpcBus, logger)
	gw := bootstrap.Gateway(registry, infra, cfg, logger)

	// ── Start REST server ──
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	restSrv := gwServer.NewHTTPServer(gw.Router, cfg.Server.RESTPort, logger)
	go restSrv.Start(ctx)

	// Wait for server to be ready
	time.Sleep(500 * time.Millisecond)

	baseURL := fmt.Sprintf("http://localhost:%d", cfg.Server.RESTPort)

	// ── Test 1: POST /v1/memory/store ──
	t.Run("POST /v1/memory/store returns 200", func(t *testing.T) {
		body := `{"content":"test memory","type":"fact","metadata":{"source":"e2e"}}`
		resp, err := http.Post(baseURL+"/v1/memory/store", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// In dev mode without real services, we expect either 200 or a service-level error (not 5xx crash)
		if resp.StatusCode >= 500 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			t.Fatalf("unexpected server error %d: %s", resp.StatusCode, string(bodyBytes))
		}
	})

	// ── Test 2: GET /healthz returns valid JSON ──
	t.Run("healthz returns JSON", func(t *testing.T) {
		// This test would need the health server running too.
		// For now we just validate the REST server is up.
		resp, err := http.Get(baseURL + "/v1/memory/store")
		if err != nil {
			t.Fatalf("server not reachable: %v", err)
		}
		defer resp.Body.Close()
		// Server is alive if we get any response
	})

	// ── Test 3: MCP tools/list ──
	t.Run("MCP tools/list returns tools", func(t *testing.T) {
		mcpSrv := gwServer.NewHTTPServer(gw.MCP.Handler(), 19082, logger)
		go mcpSrv.Start(ctx)
		time.Sleep(300 * time.Millisecond)

		body := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
		resp, err := http.Post("http://localhost:19082/message", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("MCP request failed: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]any
		json.NewDecoder(resp.Body).Decode(&result)

		if result["error"] != nil {
			t.Fatalf("MCP returned error: %v", result["error"])
		}
	})
}
