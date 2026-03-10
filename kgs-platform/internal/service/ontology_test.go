package service

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	pb "github.com/blcvn/knowledge-gateway/kgs-platform/api/ontology/v1"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/biz"
	"github.com/blcvn/knowledge-gateway/kgs-platform/internal/server/middleware"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOntologyServiceForTest(t *testing.T, dbPath string) (*OntologyService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&biz.EntityType{}, &biz.RelationType{}); err != nil {
		t.Fatalf("failed to migrate schema: %v", err)
	}
	return NewOntologyService(db, nil), db
}

func closeTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	_ = sqlDB.Close()
}

func testAppContext() context.Context {
	return context.WithValue(context.Background(), middleware.AppContextKey, middleware.AppContext{
		AppID:    "app-test",
		TenantID: "tenant-test",
	})
}

func testMemoryDBDSN(t *testing.T) string {
	t.Helper()
	return "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
}

func TestOntologyServiceCreateEntityType_PersistsAndUpserts(t *testing.T) {
	service, db := newOntologyServiceForTest(t, testMemoryDBDSN(t))
	defer closeTestDB(t, db)

	ctx := testAppContext()
	first, err := service.CreateEntityType(ctx, &pb.CreateEntityTypeRequest{
		Name:        "Requirement",
		Description: "desc-v1",
		Schema:      `{"type":"object","properties":{"priority":{"type":"string"}}}`,
	})
	if err != nil {
		t.Fatalf("CreateEntityType first call failed: %v", err)
	}
	if first.GetId() == 0 {
		t.Fatalf("expected id > 0, got %d", first.GetId())
	}

	second, err := service.CreateEntityType(ctx, &pb.CreateEntityTypeRequest{
		Name:        "Requirement",
		Description: "desc-v2",
		Schema:      `{"type":"object","properties":{"status":{"type":"string"}}}`,
	})
	if err != nil {
		t.Fatalf("CreateEntityType second call failed: %v", err)
	}
	if second.GetId() == 0 {
		t.Fatalf("expected upserted id > 0, got %d", second.GetId())
	}

	var count int64
	if err := db.Model(&biz.EntityType{}).
		Where("app_id = ? AND tenant_id = ? AND name = ?", "app-test", "tenant-test", "Requirement").
		Count(&count).Error; err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one upserted row, got %d", count)
	}

	var stored biz.EntityType
	if err := db.Where("app_id = ? AND tenant_id = ? AND name = ?", "app-test", "tenant-test", "Requirement").
		First(&stored).Error; err != nil {
		t.Fatalf("failed to load persisted entity: %v", err)
	}
	if stored.Description != "desc-v2" {
		t.Fatalf("expected description updated to desc-v2, got %q", stored.Description)
	}
	if string(stored.Schema) != `{"type":"object","properties":{"status":{"type":"string"}}}` {
		t.Fatalf("expected schema updated, got %s", string(stored.Schema))
	}
}

func TestOntologyServiceListEntityTypes_ReadsFromDatabase(t *testing.T) {
	service, db := newOntologyServiceForTest(t, testMemoryDBDSN(t))
	defer closeTestDB(t, db)

	seed := []biz.EntityType{
		{
			AppID:    "app-test",
			TenantID: "tenant-test",
			Name:     "Requirement",
			Schema:   datatypes.JSON([]byte(`{"type":"object"}`)),
		},
		{
			AppID:    "app-test",
			TenantID: "tenant-test",
			Name:     "UseCase",
			Schema:   datatypes.JSON([]byte(`{"type":"object"}`)),
		},
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("failed to seed entity types: %v", err)
	}

	reply, err := service.ListEntityTypes(testAppContext(), &pb.ListEntityTypesRequest{})
	if err != nil {
		t.Fatalf("ListEntityTypes failed: %v", err)
	}
	if len(reply.GetEntities()) != 2 {
		t.Fatalf("expected 2 entities, got %d", len(reply.GetEntities()))
	}
}

func TestOntologyServiceCreateRelationType_PersistsJSONArrays(t *testing.T) {
	service, db := newOntologyServiceForTest(t, testMemoryDBDSN(t))
	defer closeTestDB(t, db)

	ctx := testAppContext()
	reply, err := service.CreateRelationType(ctx, &pb.CreateRelationTypeRequest{
		Name:             "DEPENDS_ON",
		Description:      "dep relation",
		PropertiesSchema: `{"type":"object"}`,
		SourceTypes:      []string{"Requirement", "UseCase"},
		TargetTypes:      []string{"Requirement", "NFR"},
	})
	if err != nil {
		t.Fatalf("CreateRelationType failed: %v", err)
	}
	if reply.GetId() == 0 {
		t.Fatalf("expected id > 0, got %d", reply.GetId())
	}

	var relation biz.RelationType
	if err := db.Where("app_id = ? AND tenant_id = ? AND name = ?", "app-test", "tenant-test", "DEPENDS_ON").
		First(&relation).Error; err != nil {
		t.Fatalf("failed to load relation: %v", err)
	}

	var gotSource []string
	var gotTarget []string
	if err := json.Unmarshal(relation.SourceTypes, &gotSource); err != nil {
		t.Fatalf("failed to decode source types: %v", err)
	}
	if err := json.Unmarshal(relation.TargetTypes, &gotTarget); err != nil {
		t.Fatalf("failed to decode target types: %v", err)
	}
	if !reflect.DeepEqual(gotSource, []string{"Requirement", "UseCase"}) {
		t.Fatalf("unexpected source types: %#v", gotSource)
	}
	if !reflect.DeepEqual(gotTarget, []string{"Requirement", "NFR"}) {
		t.Fatalf("unexpected target types: %#v", gotTarget)
	}
}

func TestOntologyServiceListRelationTypes_DecodesJSONArrays(t *testing.T) {
	service, db := newOntologyServiceForTest(t, testMemoryDBDSN(t))
	defer closeTestDB(t, db)

	seed := biz.RelationType{
		AppID:       "app-test",
		TenantID:    "tenant-test",
		Name:        "IMPLEMENTS",
		Properties:  datatypes.JSON([]byte(`{"type":"object"}`)),
		SourceTypes: datatypes.JSON([]byte(`["APIEndpoint"]`)),
		TargetTypes: datatypes.JSON([]byte(`["DataModel"]`)),
	}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("failed to seed relation type: %v", err)
	}

	reply, err := service.ListRelationTypes(testAppContext(), &pb.ListRelationTypesRequest{})
	if err != nil {
		t.Fatalf("ListRelationTypes failed: %v", err)
	}
	if len(reply.GetRelations()) != 1 {
		t.Fatalf("expected 1 relation, got %d", len(reply.GetRelations()))
	}
	rel := reply.GetRelations()[0]
	if !reflect.DeepEqual(rel.GetSourceTypes(), []string{"APIEndpoint"}) {
		t.Fatalf("unexpected source types: %#v", rel.GetSourceTypes())
	}
	if !reflect.DeepEqual(rel.GetTargetTypes(), []string{"DataModel"}) {
		t.Fatalf("unexpected target types: %#v", rel.GetTargetTypes())
	}
}

func TestOntologyServiceDataPersistsAcrossRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ontology.db")
	service1, db1 := newOntologyServiceForTest(t, dbPath)

	if _, err := service1.CreateEntityType(testAppContext(), &pb.CreateEntityTypeRequest{
		Name:        "Actor",
		Description: "seed actor",
		Schema:      `{"type":"object"}`,
	}); err != nil {
		t.Fatalf("failed to seed entity before restart: %v", err)
	}
	closeTestDB(t, db1)

	service2, db2 := newOntologyServiceForTest(t, dbPath)
	defer closeTestDB(t, db2)

	reply, err := service2.ListEntityTypes(testAppContext(), &pb.ListEntityTypesRequest{})
	if err != nil {
		t.Fatalf("ListEntityTypes after restart failed: %v", err)
	}
	if len(reply.GetEntities()) != 1 || reply.GetEntities()[0].GetName() != "Actor" {
		t.Fatalf("expected persisted entity Actor, got %#v", reply.GetEntities())
	}
}
