package overlay

import (
	"context"
	"testing"

	"kgs-platform/internal/conf"
	"kgs-platform/internal/data"
	"kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2/log"
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
		Url:    "nats://localhost:4222",
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
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

	if _, ok := store.items[item.OverlayID]; ok {
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
		Url:    "nats://localhost:4222",
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	listener := NewSessionCloseListener(natsClient, manager, log.DefaultLogger)
	if err := listener.Start(ctx); err != nil {
		t.Fatalf("listener start: %v", err)
	}
	defer listener.Stop(ctx)

	if err := natsClient.Publish(ctx, data.TopicSessionClose("session-close-2"), []byte(`{"session_id":"session-close-2"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	updated, err := store.Get(ctx, item.OverlayID)
	if err != nil {
		t.Fatalf("get overlay: %v", err)
	}
	if updated.Status != StatusCommitted {
		t.Fatalf("expected committed status, got %s", updated.Status)
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
		Url:    "nats://localhost:4222",
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	listener := NewSessionCloseListener(natsClient, manager, log.DefaultLogger)
	if err := listener.Start(ctx); err != nil {
		t.Fatalf("listener start: %v", err)
	}
	defer listener.Stop(ctx)

	if err := natsClient.Publish(ctx, data.TopicBudgetStop("session-budget-1"), []byte(`{"session_id":"session-budget-1"}`)); err != nil {
		t.Fatalf("publish: %v", err)
	}

	updated, err := store.Get(ctx, item.OverlayID)
	if err != nil {
		t.Fatalf("get overlay: %v", err)
	}
	if updated.Status != StatusPartial {
		t.Fatalf("expected partial status, got %s", updated.Status)
	}
}
