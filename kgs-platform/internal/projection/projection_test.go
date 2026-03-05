package projection

import (
	"context"
	"io"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestProjectionApplyFiltersByRoleAndMasksPII(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ViewDefinitionRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	engine := NewEngine(db, log.NewStdLogger(io.Discard))
	_, err = engine.CreateViewDefinition(context.Background(), "graph/app-1/tenant-1", ViewDefinition{
		RoleName:           "BA",
		AllowedEntityTypes: []string{"Requirement"},
		AllowedFields:      []string{"id", "name", "email", "phone"},
		PIIMaskFields:      []string{"email", "phone"},
	})
	if err != nil {
		t.Fatalf("create view: %v", err)
	}

	raw := map[string]any{
		"nodes": []map[string]any{
			{
				"id":    "n1",
				"label": "Requirement",
				"properties": map[string]any{
					"id":    "n1",
					"name":  "FR-001",
					"email": "alice@example.com",
					"phone": "+84 912345678",
					"owner": "alice",
				},
			},
			{
				"id":    "n2",
				"label": "UseCase",
				"properties": map[string]any{
					"id":   "n2",
					"name": "UC-001",
				},
			},
		},
		"edges": []map[string]any{
			{"id": "e1", "source": "n1", "target": "n2", "type": "IMPLEMENTS", "properties": map[string]any{}},
		},
	}

	projected, err := engine.Apply(context.Background(), "graph/app-1/tenant-1", "BA", raw)
	if err != nil {
		t.Fatalf("apply projection: %v", err)
	}

	nodes := toNodeMaps(projected["nodes"])
	if len(nodes) != 1 {
		t.Fatalf("expected one visible node, got %d", len(nodes))
	}
	props := toMap(nodes[0]["properties"])
	if _, ok := props["owner"]; ok {
		t.Fatalf("unexpected filtered-out field owner in properties: %#v", props)
	}
	if props["email"] != "a***@***.com" {
		t.Fatalf("expected masked email, got %#v", props["email"])
	}
	if props["phone"] != "***-***-5678" {
		t.Fatalf("expected masked phone, got %#v", props["phone"])
	}

	edges := toEdgeMaps(projected["edges"])
	if len(edges) != 0 {
		t.Fatalf("expected edge dropped when target node hidden, got %#v", edges)
	}
}

func TestProjectionApplyNoRoleViewReturnsRawData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ViewDefinitionRecord{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	engine := NewEngine(db, log.NewStdLogger(io.Discard))

	raw := map[string]any{"nodes": []map[string]any{{"id": "n1", "label": "Requirement", "properties": map[string]any{"name": "FR-001"}}}, "edges": []map[string]any{}}
	projected, err := engine.Apply(context.Background(), "graph/app-1/tenant-1", "UNKNOWN", raw)
	if err != nil {
		t.Fatalf("apply projection: %v", err)
	}
	if len(toNodeMaps(projected["nodes"])) != 1 {
		t.Fatalf("expected raw data to stay unchanged: %#v", projected)
	}
}
