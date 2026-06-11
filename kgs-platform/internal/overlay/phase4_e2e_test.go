package overlay

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/conf"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/data"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"

	"github.com/go-kratos/kratos/v2/log"
)

func TestPhase4E2EOverlayCommitPublishesNATSEvent(t *testing.T) {
	natsURL := runNATSServerURL(t)
	natsClient, err := data.NewNATSClientFromConfig(&conf.Data_NATS{
		Url:    natsURL,
		Stream: "kgs-events",
	}, log.NewHelper(log.DefaultLogger))
	if err != nil {
		t.Fatalf("new nats client: %v", err)
	}
	defer natsClient.Close()

	store := newMemoryStore()
	vm := &fakeVersionManager{
		versions: []version.GraphVersion{{ID: "v1", Namespace: "graph/app/tenant"}},
	}
	manager := &Manager{
		store:      store,
		versionMgr: vm,
		publisher:  natsClient,
	}

	ctx := context.Background()
	item, err := manager.Create(ctx, "graph/app/tenant", "phase4-e2e-session", "")
	if err != nil {
		t.Fatalf("create overlay: %v", err)
	}
	if _, err := manager.AddEntityDelta(ctx, item.OverlayID, "graph/app/tenant", "Requirement", map[string]any{"name": "N1"}); err != nil {
		t.Fatalf("add entity delta: %v", err)
	}

	eventCh := make(chan map[string]any, 1)
	unsubscribe, err := natsClient.Subscribe("overlay.committed.*", func(ctx context.Context, payload []byte) {
		_ = ctx
		var body map[string]any
		if json.Unmarshal(payload, &body) == nil {
			select {
			case eventCh <- body:
			default:
			}
		}
	})
	if err != nil {
		t.Fatalf("subscribe committed topic: %v", err)
	}
	defer unsubscribe()

	if _, err := manager.Commit(ctx, item.OverlayID, PolicyKeepOverlay); err != nil {
		t.Fatalf("commit overlay: %v", err)
	}

	if !waitFor(2*time.Second, func() bool { return len(eventCh) > 0 }) {
		t.Fatalf("expected committed event on overlay.committed.*")
	}

	event := <-eventCh
	if got := asString(event["overlay_id"]); got != item.OverlayID {
		t.Fatalf("event overlay_id mismatch: got=%q want=%q", got, item.OverlayID)
	}
	if got := asString(event["status"]); got != string(StatusCommitted) {
		t.Fatalf("event status mismatch: got=%q want=%q", got, StatusCommitted)
	}
	if asString(event["new_version_id"]) == "" {
		t.Fatalf("event missing new_version_id")
	}
}

func asString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
