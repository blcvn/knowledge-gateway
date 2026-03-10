package conf

import (
	"path/filepath"
	"testing"

	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
)

func TestConfigYAMLOntologySection_IsLoadedByGetOntology(t *testing.T) {
	cfgPath := filepath.Join("..", "..", "configs", "config.yaml")
	c := config.New(config.WithSource(file.NewSource(cfgPath)))
	defer c.Close()

	if err := c.Load(); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatalf("failed to scan config bootstrap: %v", err)
	}

	ontology := bc.GetData().GetOntology()
	if ontology == nil {
		t.Fatalf("expected ontology config, got nil")
	}
	if ontology.GetValidationEnabled() {
		t.Fatalf("expected validation_enabled=false")
	}
	if ontology.GetStrictMode() {
		t.Fatalf("expected strict_mode=false")
	}
	if ontology.GetSchemaValidation() {
		t.Fatalf("expected schema_validation=false")
	}
	if ontology.GetEdgeConstraintCheck() {
		t.Fatalf("expected edge_constraint_check=false")
	}
	if ontology.GetSyncProjection() {
		t.Fatalf("expected sync_projection=false")
	}
}
