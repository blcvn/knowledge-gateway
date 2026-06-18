package ontology

import "testing"

func TestMemoryStoreSearchProfileAndStrategies(t *testing.T) {
	store := NewMemoryStore()
	profile := SearchProfile{
		SemanticFields:   []IndexedField{{FieldName: "id", Weight: 1}},
		FTSLanguage:      "simple",
		QueryStrategyRef: "default",
	}
	domain := store.CreateDomain(Domain{ID: "d1", OwnerTenantID: "t1", SearchProfile: &profile})

	gotDomain, ok := store.GetDomain("d1")
	if !ok {
		t.Fatal("domain missing")
	}
	if gotDomain.SearchProfile == nil || gotDomain.SearchProfile.QueryStrategyRef != "default" {
		t.Fatalf("domain search profile = %#v", gotDomain.SearchProfile)
	}
	if domain.SearchProfile == nil || domain.SearchProfile.FTSLanguage != "simple" {
		t.Fatalf("created domain search profile = %#v", domain.SearchProfile)
	}

	stored := store.UpsertQueryStrategy(QueryStrategy{Key: "finance_deep", MaxDepth: 8, Version: 1})
	if stored.Key != "finance_deep" {
		t.Fatalf("stored strategy = %#v", stored)
	}
	gotStrategy, ok := store.GetQueryStrategy("finance_deep")
	if !ok || gotStrategy.MaxDepth != 8 {
		t.Fatalf("strategy lookup = %#v ok=%v", gotStrategy, ok)
	}
	if len(store.ListQueryStrategies()) == 0 {
		t.Fatal("expected query strategies to be listed")
	}
}
