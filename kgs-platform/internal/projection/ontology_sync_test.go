package projection

import (
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type ontologyEntityTypeRecord struct {
	ID       uint   `gorm:"primaryKey"`
	AppID    string `gorm:"size:128"`
	TenantID string `gorm:"size:128"`
	Name     string `gorm:"size:128"`
}

func (ontologyEntityTypeRecord) TableName() string {
	return "kgs_entity_types"
}

func setupProjectionSyncDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ontologyEntityTypeRecord{}, &ViewDefinitionRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func TestOntologyProjectionSyncSyncRoleView_MergesOntologyTypes(t *testing.T) {
	db := setupProjectionSyncDB(t)
	syncer := NewOntologyProjectionSync(db, log.NewStdLogger(io.Discard))
	ctx := context.Background()

	if err := db.Create(&ontologyEntityTypeRecord{AppID: "app-1", TenantID: "tenant-1", Name: "Requirement"}).Error; err != nil {
		t.Fatalf("seed ontology type: %v", err)
	}
	if err := db.Create(&ontologyEntityTypeRecord{AppID: "app-1", TenantID: "tenant-1", Name: "UseCase"}).Error; err != nil {
		t.Fatalf("seed ontology type: %v", err)
	}
	if err := db.Create(&ViewDefinitionRecord{
		ID:                     "view-ba",
		AppID:                  "app-1",
		TenantID:               "tenant-1",
		RoleName:               "BA",
		AllowedEntityTypesJSON: `["Requirement","Actor"]`,
	}).Error; err != nil {
		t.Fatalf("seed view definition: %v", err)
	}

	if err := syncer.SyncRoleView(ctx, "app-1", "tenant-1", "BA"); err != nil {
		t.Fatalf("sync role view failed: %v", err)
	}

	var updated ViewDefinitionRecord
	if err := db.Where("id = ?", "view-ba").Take(&updated).Error; err != nil {
		t.Fatalf("load updated view: %v", err)
	}
	got := decodeJSONStringArray(updated.AllowedEntityTypesJSON)
	want := []string{"Requirement", "Actor", "UseCase"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected allowed types: got=%v want=%v", got, want)
	}
}

func TestOntologyProjectionSyncSyncRoleView_NoViewDefinitionSkips(t *testing.T) {
	db := setupProjectionSyncDB(t)
	syncer := NewOntologyProjectionSync(db, log.NewStdLogger(io.Discard))

	if err := db.Create(&ontologyEntityTypeRecord{AppID: "app-1", TenantID: "tenant-1", Name: "Requirement"}).Error; err != nil {
		t.Fatalf("seed ontology type: %v", err)
	}
	if err := syncer.SyncRoleView(context.Background(), "app-1", "tenant-1", "PO"); err != nil {
		t.Fatalf("expected skip without error, got: %v", err)
	}
}

func TestMergeStringSlices_DeduplicatesValues(t *testing.T) {
	got := mergeStringSlices(
		[]string{"Requirement", "UseCase"},
		[]string{"Requirement", " Actor ", "usecase", ""},
	)
	want := []string{"Requirement", "UseCase", "Actor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected merge result: got=%v want=%v", got, want)
	}
}
