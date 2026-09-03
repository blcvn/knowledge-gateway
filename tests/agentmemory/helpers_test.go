package agentmemory_test

import (
    "context"
    "net"
    "os"
    "testing"

    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
    "github.com/stretchr/testify/require"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "github.com/vnp-memory/apps/memory/internal/bootstrap"
    "github.com/vnp-memory/apps/memory/internal/config"
    observepb  "github.com/vnp-memory/api/proto/observe/v1"
    memorypb   "github.com/vnp-memory/api/proto/memory/v1"
    searchpb   "github.com/vnp-memory/api/proto/search/v1"
    orchpb     "github.com/vnp-memory/api/proto/orchestration/v1"
)

type testHarness struct {
    t          *testing.T
    ctx        context.Context
    cancel     context.CancelFunc
    conn       *grpc.ClientConn
    observe    observepb.ObserveServiceClient
    memory     memorypb.AgentMemoryServiceClient
    search     searchpb.ObserveSearchServiceClient
    orch       orchpb.OrchestrationServiceClient
    natsConn   *nats.Conn
    natsServer *server.Server
}

func setupHarness(t *testing.T) *testHarness {
    t.Helper()
    ctx, cancel := context.WithCancel(context.Background())

    // Embedded NATS
    ns, err := server.NewServer(&server.Options{Port: -1})
    require.NoError(t, err)
    go ns.Start()
    require.True(t, ns.ReadyForConnections(5*time.Second))
    nc, err := nats.Connect(ns.ClientURL())
    require.NoError(t, err)

    // Use test PostgreSQL (env DATABASE_URL or skip)
    if os.Getenv("DATABASE_URL") == "" {
        t.Skip("DATABASE_URL not set — skipping integration tests")
    }

    cfg := &config.Config{
        DatabaseURL: os.Getenv("DATABASE_URL"),
        AgentMemory: config.LoadAgentMemoryConfig(),
    }

    // Bootstrap monolith in-process
    err = bootstrap.BootstrapForTest(ctx, cfg, nc)
    require.NoError(t, err)

    // Connect via bufconn
    conn, err := grpc.NewClient("bufconn://agentmemory",
        grpc.WithContextDialer(bootstrap.BufconnDialer()),
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    require.NoError(t, err)

    h := &testHarness{
        t: t, ctx: ctx, cancel: cancel, conn: conn,
        observe: observepb.NewObserveServiceClient(conn),
        memory:  memorypb.NewAgentMemoryServiceClient(conn),
        search:  searchpb.NewObserveSearchServiceClient(conn),
        orch:    orchpb.NewOrchestrationServiceClient(conn),
        natsConn: nc, natsServer: ns,
    }
    t.Cleanup(h.teardown)
    return h
}

func (h *testHarness) teardown() {
    h.cancel()
    h.conn.Close()
    h.natsConn.Close()
    h.natsServer.Shutdown()
}

func (h *testHarness) newSession(project string) string {
    resp, err := h.observe.StartSession(h.ctx, &observepb.StartSessionRequest{
        TenantId: "test-tenant", Project: project,
    })
    require.NoError(h.t, err)
    return resp.SessionId
}
