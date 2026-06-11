package version

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&GraphVersion{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewManager(db, log.DefaultLogger)
}

func TestVersionManagerCreateDiffRollback(t *testing.T) {
	m := newTestManager(t)
	ctx := context.Background()
	ns := "graph/app/tenant"

	v1, err := m.CreateDelta(ctx, ns, ChangeSet{
		EntitiesAdded: 1,
		EdgesAdded:    1,
		CommitMessage: "initial",
	})
	if err != nil {
		t.Fatalf("CreateDelta v1 failed: %v", err)
	}
	v2, err := m.CreateDelta(ctx, ns, ChangeSet{
		EntitiesAdded:    2,
		EntitiesModified: 1,
		EdgesAdded:       3,
		CommitMessage:    "second",
	})
	if err != nil {
		t.Fatalf("CreateDelta v2 failed: %v", err)
	}

	versions, err := m.ListVersions(ctx, ns)
	if err != nil {
		t.Fatalf("ListVersions failed: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}

	diff, err := m.DiffVersions(ctx, ns, v1, v2)
	if err != nil {
		t.Fatalf("DiffVersions failed: %v", err)
	}
	if diff.EntitiesAdded != 2 || diff.EntitiesModified != 1 || diff.EdgesAdded != 3 {
		t.Fatalf("unexpected diff: %#v", diff)
	}

	rollbackVersionID, err := m.Rollback(ctx, ns, v1, "manual rollback")
	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if rollbackVersionID == "" {
		t.Fatalf("expected rollback version id")
	}
}
