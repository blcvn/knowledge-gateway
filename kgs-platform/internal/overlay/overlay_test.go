package overlay

import (
	"context"
	"testing"
	"time"

	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/version"
)

type memoryStore struct {
	items       map[string]*OverlayGraph
	sessionBind map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		items:       map[string]*OverlayGraph{},
		sessionBind: map[string]string{},
	}
}

func (s *memoryStore) Save(ctx context.Context, overlay *OverlayGraph, ttl time.Duration) error {
	_ = ctx
	_ = ttl
	copy := *overlay
	s.items[overlay.OverlayID] = &copy
	return nil
}

func (s *memoryStore) Get(ctx context.Context, overlayID string) (*OverlayGraph, error) {
	_ = ctx
	out, ok := s.items[overlayID]
	if !ok {
		return nil, context.Canceled
	}
	copy := *out
	return &copy, nil
}

func (s *memoryStore) Delete(ctx context.Context, overlayID string) error {
	_ = ctx
	delete(s.items, overlayID)
	return nil
}

func (s *memoryStore) BindSession(ctx context.Context, sessionID, overlayID string, ttl time.Duration) error {
	_ = ctx
	_ = ttl
	s.sessionBind[sessionID] = overlayID
	return nil
}

func (s *memoryStore) UnbindSession(ctx context.Context, sessionID string) error {
	_ = ctx
	delete(s.sessionBind, sessionID)
	return nil
}

func (s *memoryStore) FindBySession(ctx context.Context, sessionID string) (string, error) {
	_ = ctx
	return s.sessionBind[sessionID], nil
}

type fakeVersionManager struct {
	versions []version.GraphVersion
}

func (f *fakeVersionManager) CreateDelta(ctx context.Context, namespace string, changes version.ChangeSet) (string, error) {
	_ = ctx
	id := "v" + time.Now().UTC().Format("150405")
	f.versions = append([]version.GraphVersion{{
		ID:            id,
		Namespace:     namespace,
		CommitMessage: changes.CommitMessage,
	}}, f.versions...)
	return id, nil
}

func (f *fakeVersionManager) GetVersion(ctx context.Context, namespace, versionID string) (*version.GraphVersion, error) {
	_ = ctx
	for _, item := range f.versions {
		if item.ID == versionID && item.Namespace == namespace {
			found := item
			return &found, nil
		}
	}
	return nil, context.Canceled
}

func (f *fakeVersionManager) ListVersions(ctx context.Context, namespace string) ([]version.GraphVersion, error) {
	_ = ctx
	var out []version.GraphVersion
	for _, item := range f.versions {
		if item.Namespace == namespace {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *fakeVersionManager) DiffVersions(ctx context.Context, namespace, fromVersionID, toVersionID string) (*version.DiffResult, error) {
	_ = ctx
	_ = namespace
	_ = fromVersionID
	_ = toVersionID
	return &version.DiffResult{}, nil
}

func (f *fakeVersionManager) Rollback(ctx context.Context, namespace, targetVersionID, reason string) (string, error) {
	_ = ctx
	_ = namespace
	_ = targetVersionID
	_ = reason
	return "rollback", nil
}

type fakePublisher struct {
	published int
	subjects  []string
}

func (f *fakePublisher) Publish(ctx context.Context, subject string, payload []byte) error {
	_ = ctx
	_ = payload
	f.published++
	f.subjects = append(f.subjects, subject)
	return nil
}

func TestOverlayLifecycle(t *testing.T) {
	store := newMemoryStore()
	vm := &fakeVersionManager{
		versions: []version.GraphVersion{{ID: "v1", Namespace: "graph/app/tenant"}},
	}
	pub := &fakePublisher{}
	manager := &Manager{
		store:      store,
		versionMgr: vm,
		publisher:  pub,
	}

	ctx := context.Background()
	item, err := manager.Create(ctx, "graph/app/tenant", "session-1", "current")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if item.BaseVersionID != "v1" {
		t.Fatalf("expected base version v1, got %s", item.BaseVersionID)
	}

	commit, err := manager.Commit(ctx, item.OverlayID, PolicyKeepOverlay)
	if err != nil {
		t.Fatalf("Commit failed: %v", err)
	}
	if commit.NewVersionID == "" {
		t.Fatalf("expected new version id")
	}
	if pub.published == 0 {
		t.Fatalf("expected publish event on commit")
	}
	if len(pub.subjects) == 0 || pub.subjects[0] != "overlay.committed.tenant" {
		t.Fatalf("unexpected commit topic: %#v", pub.subjects)
	}

	item2, err := manager.Create(ctx, "graph/app/tenant", "session-2", "")
	if err != nil {
		t.Fatalf("Create#2 failed: %v", err)
	}
	if err := manager.Discard(ctx, item2.OverlayID); err != nil {
		t.Fatalf("Discard failed: %v", err)
	}
	if _, ok := store.items[item2.OverlayID]; ok {
		t.Fatalf("overlay should be removed after discard")
	}
	if len(pub.subjects) < 2 || pub.subjects[1] != "overlay.discarded.tenant" {
		t.Fatalf("unexpected discard topic: %#v", pub.subjects)
	}
}

func TestOverlayDiscardBySession(t *testing.T) {
	store := newMemoryStore()
	manager := &Manager{
		store:      store,
		versionMgr: &fakeVersionManager{},
	}
	ctx := context.Background()
	item, err := manager.Create(ctx, "graph/app/tenant", "session-3", "")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := manager.DiscardBySession(ctx, "session-3"); err != nil {
		t.Fatalf("DiscardBySession failed: %v", err)
	}
	if _, ok := store.items[item.OverlayID]; ok {
		t.Fatalf("overlay should be removed")
	}
}
