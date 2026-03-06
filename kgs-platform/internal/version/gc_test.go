package version

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

func TestGCCompact(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()
	ns := "graph/app/tenant"

	old := time.Now().UTC().Add(-48 * time.Hour)
	newer := time.Now().UTC().Add(-2 * time.Hour)

	seed := []GraphVersion{
		{ID: "v1", Namespace: ns, CreatedAt: old},
		{ID: "v2", Namespace: ns, ParentID: "v1", CreatedAt: old.Add(1 * time.Hour)},
		{ID: "v3", Namespace: ns, ParentID: "v2", CreatedAt: newer},
	}
	if err := m.db.Create(&seed).Error; err != nil {
		t.Fatalf("seed versions: %v", err)
	}

	gc := NewGC(m, log.DefaultLogger)
	deleted, err := gc.Compact(ctx, ns, 24*time.Hour, 1)
	if err != nil {
		t.Fatalf("Compact failed: %v", err)
	}
	if deleted != 2 {
		t.Fatalf("expected 2 deletions, got %d", deleted)
	}

	remaining, err := m.ListVersions(ctx, ns)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}
	if len(remaining) != 1 || remaining[0].ID != "v3" {
		t.Fatalf("unexpected remaining versions: %#v", remaining)
	}
}
