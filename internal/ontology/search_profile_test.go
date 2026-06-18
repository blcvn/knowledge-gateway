package ontology

import (
	"bytes"
	"log"
	"strings"
	"testing"

	"kg-service/internal/access"
)

func TestDefaultSearchProfileResolverUsesSystemDefaults(t *testing.T) {
	store := NewMemoryStore()
	store.CreateDomain(Domain{ID: "domain-1", OwnerTenantID: access.PlatformTenantID})

	resolver := NewDefaultSearchProfileResolver(store)
	resolved, err := resolver.Resolve("domain-1", "tenant-1", "app-1")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.FTSLanguage != "simple" {
		t.Fatalf("FTSLanguage = %q, want simple", resolved.FTSLanguage)
	}
	if resolved.QueryStrategy.Key != "default" {
		t.Fatalf("QueryStrategy.Key = %q, want default", resolved.QueryStrategy.Key)
	}
	if len(resolved.SemanticFields) == 0 {
		t.Fatal("SemanticFields is empty")
	}
	want := defaultSemanticFields()
	if len(resolved.SemanticFields) != len(want) {
		t.Fatalf("default fields len = %d, want %d", len(resolved.SemanticFields), len(want))
	}
	for i := range want {
		if resolved.SemanticFields[i] != want[i] {
			t.Fatalf("default field[%d] = %#v, want %#v", i, resolved.SemanticFields[i], want[i])
		}
	}
}

func TestDefaultSearchProfileResolverPrecedenceAndFallback(t *testing.T) {
	store := NewMemoryStore()
	store.CreateDomain(Domain{
		ID:            "domain-1",
		OwnerTenantID: access.PlatformTenantID,
		SearchProfile: &SearchProfile{
			SemanticFields:   []IndexedField{{FieldName: "id", Weight: 1}},
			FTSLanguage:      "simple",
			QueryStrategyRef: "missing_strategy",
			TenantOverrides: map[string]SearchProfileOverride{
				"tenant-1": {
					FTSLanguage: "vi",
				},
			},
			AppOverrides: map[string]SearchProfileOverride{
				"tenant-1:app-1": {
					SemanticFields: []IndexedField{{FieldName: "node_type", Weight: 2}},
				},
			},
		},
	})

	var buf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(orig) })

	resolver := NewDefaultSearchProfileResolver(store)
	resolved, err := resolver.Resolve("domain-1", "tenant-1", "app-1")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.FTSLanguage != "vi" {
		t.Fatalf("FTSLanguage = %q, want vi", resolved.FTSLanguage)
	}
	if resolved.SemanticFields[0].FieldName != "node_type" {
		t.Fatalf("SemanticFields[0] = %#v, want app override", resolved.SemanticFields[0])
	}
	if resolved.QueryStrategy.Key != "default" {
		t.Fatalf("QueryStrategy.Key = %q, want default", resolved.QueryStrategy.Key)
	}
	if !strings.Contains(buf.String(), "missing query strategy") {
		t.Fatalf("log output = %q, want missing strategy warning", buf.String())
	}
}

func TestUpsertSearchProfileValidation(t *testing.T) {
	store := NewMemoryStore()
	store.CreateDomain(Domain{ID: "domain-1", OwnerTenantID: access.PlatformTenantID})
	service := NewService(store, nil)

	_, err := service.UpsertSearchProfile(access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}, access.PlatformTenantID, "domain-1", SearchProfile{
		SemanticFields: []IndexedField{{FieldName: "unknown_field", Weight: 1}},
		FTSLanguage:    "simple",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown semantic field") {
		t.Fatalf("unknown field error = %v, want unknown semantic field", err)
	}

	_, err = service.UpsertSearchProfile(access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}, access.PlatformTenantID, "domain-1", SearchProfile{
		SemanticFields: []IndexedField{},
		FTSLanguage:    "simple",
	})
	if err == nil || !strings.Contains(err.Error(), "must be nil") {
		t.Fatalf("empty semantic fields error = %v, want nil/default validation", err)
	}

	_, err = service.UpsertSearchProfile(access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}, access.PlatformTenantID, "domain-1", SearchProfile{
		FTSLanguage: "",
	})
	if err == nil || !strings.Contains(err.Error(), "fts_language") {
		t.Fatalf("empty language error = %v, want fts_language validation", err)
	}
}

func TestUpsertQueryStrategyIncrementsVersion(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store, nil)

	created, err := service.UpsertQueryStrategy(access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}, access.PlatformTenantID, QueryStrategy{
		Key:      "finance_deep",
		MaxDepth: 8,
		Params:   map[string]any{"direction": "out"},
	})
	if err != nil {
		t.Fatalf("UpsertQueryStrategy() error = %v", err)
	}
	if created.Version != 1 {
		t.Fatalf("Version = %d, want 1", created.Version)
	}

	updated, err := service.UpsertQueryStrategy(access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}, access.PlatformTenantID, QueryStrategy{
		Key:      "finance_deep",
		MaxDepth: 10,
		Params:   map[string]any{"direction": "out"},
	})
	if err != nil {
		t.Fatalf("UpsertQueryStrategy(update) error = %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("Version = %d, want 2", updated.Version)
	}
}

func TestBuiltInQueryStrategiesRejectMutation(t *testing.T) {
	store := NewMemoryStore()
	service := NewService(store, nil)
	actor := access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}

	if _, err := service.UpsertQueryStrategy(actor, access.PlatformTenantID, QueryStrategy{Key: "default", MaxDepth: 7}); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("update built-in error = %v, want forbidden", err)
	}
	if err := service.DeleteQueryStrategy(actor, access.PlatformTenantID, "default"); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("delete built-in error = %v, want forbidden", err)
	}
	if _, err := service.UpsertQueryStrategy(actor, access.PlatformTenantID, QueryStrategy{Key: "custom", MaxDepth: 2}); err != nil {
		t.Fatalf("custom strategy create error = %v", err)
	}
	if err := service.DeleteQueryStrategy(actor, access.PlatformTenantID, "custom"); err != nil {
		t.Fatalf("custom strategy delete error = %v", err)
	}
	if _, ok := store.GetQueryStrategy("custom"); ok {
		t.Fatal("custom strategy should be deleted")
	}
}
