package overlay

import (
	"context"
	"fmt"
	"testing"
	"time"

	"kgs-platform/internal/conf"
	"kgs-platform/internal/data"
	"kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2/log"
	natsserver "github.com/nats-io/nats-server/v2/server"
)

func TestSessionCloseListener(t *testing.T) {
	store := newMemoryStore()
	manager := &Manager{
		store:      store,
		versionMgr: &fakeVersionManager{},
	}
	ctx := context.Background()
	item, err := manager.Create(ctx, "graph/app/tenant", "session-close-1", "")
	if err != nil {
		t.Fatalf("create overlay: %v", err)
	}

	natsClient, err := data.NewNATSClientFromConfig(&conf.Data_NATS{
		Url:    runNATSServerURL(t),
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	defer natsClient.Close()
	listener := &SessionCloseListener{
		nats:    natsClient,
		manager: manager,
	}
	if err := listener.Start(ctx); err != nil {
		t.Fatalf("listener start: %v", err)
	}
	defer listener.Stop(ctx)

	if err := natsClient.Publish(ctx, "session.close.session-close-1", []byte(`{"session_id":"session-close-1"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if !waitFor(2*time.Second, func() bool {
		_, ok := store.items[item.OverlayID]
		return !ok
	}) {
		t.Fatalf("overlay should be auto-discarded on session close")
	}
}

func TestSessionCloseListenerCommitWhenOverlayHasDelta(t *testing.T) {
	store := newMemoryStore()
	vm := &fakeVersionManager{
		versions: []version.GraphVersion{{ID: "v1", Namespace: "graph/app/tenant"}},
	}
	manager := &Manager{
		store:      store,
		versionMgr: vm,
	}
	ctx := context.Background()
	item, err := manager.Create(ctx, "graph/app/tenant", "session-close-2", "")
	if err != nil {
		t.Fatalf("create overlay: %v", err)
	}
	if _, err := manager.AddEntityDelta(ctx, item.OverlayID, "graph/app/tenant", "Requirement", map[string]any{"name": "N1"}); err != nil {
		t.Fatalf("add entity delta: %v", err)
	}

	natsClient, err := data.NewNATSClientFromConfig(&conf.Data_NATS{
		Url:    runNATSServerURL(t),
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	defer natsClient.Close()
	listener := NewSessionCloseListener(natsClient, manager, log.DefaultLogger)
	if err := listener.Start(ctx); err != nil {
		t.Fatalf("listener start: %v", err)
	}
	defer listener.Stop(ctx)

	if err := natsClient.Publish(ctx, data.TopicSessionClose("session-close-2"), []byte(`{"session_id":"session-close-2"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if !waitFor(2*time.Second, func() bool {
		updated, getErr := store.Get(ctx, item.OverlayID)
		return getErr == nil && updated.Status == StatusCommitted
	}) {
		t.Fatalf("expected committed status")
	}
}

func TestBudgetStopListenerCommitPartial(t *testing.T) {
	store := newMemoryStore()
	vm := &fakeVersionManager{
		versions: []version.GraphVersion{{ID: "v1", Namespace: "graph/app/tenant"}},
	}
	manager := &Manager{
		store:      store,
		versionMgr: vm,
	}
	ctx := context.Background()
	item, err := manager.Create(ctx, "graph/app/tenant", "session-budget-1", "")
	if err != nil {
		t.Fatalf("create overlay: %v", err)
	}
	if _, err := manager.AddEntityDelta(ctx, item.OverlayID, "graph/app/tenant", "Requirement", map[string]any{"name": "N1"}); err != nil {
		t.Fatalf("add entity delta: %v", err)
	}

	natsClient, err := data.NewNATSClientFromConfig(&conf.Data_NATS{
		Url:    runNATSServerURL(t),
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	defer natsClient.Close()
	listener := NewSessionCloseListener(natsClient, manager, log.DefaultLogger)
	if err := listener.Start(ctx); err != nil {
		t.Fatalf("listener start: %v", err)
	}
	defer listener.Stop(ctx)

	if err := natsClient.Publish(ctx, data.TopicBudgetStop("session-budget-1"), []byte(`{"session_id":"session-budget-1"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if !waitFor(2*time.Second, func() bool {
		updated, getErr := store.Get(ctx, item.OverlayID)
		return getErr == nil && updated.Status == StatusPartial
	}) {
		t.Fatalf("expected partial status")
	}
}

func runNATSServerURL(t *testing.T) string {
	t.Helper()
	opts := &natsserver.Options{
		Host:      "127.0.0.1",
		Port:      -1,
		JetStream: true,
		NoLog:     true,
		NoSigs:    true,
	}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatalf("new nats server: %v", err)
	}
	go srv.Start()
	if !srv.ReadyForConnections(5 * time.Second) {
		t.Fatalf("nats server not ready")
	}
	t.Cleanup(srv.Shutdown)
	return fmt.Sprintf("nats://%s", srv.Addr().String())
}

func waitFor(timeout time.Duration, fn func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fn()
}
