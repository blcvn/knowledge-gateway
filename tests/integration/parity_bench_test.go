package integration_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"kg-service/internal/access"
	"kg-service/internal/write"
)

func BenchmarkReadTemplate(b *testing.B) {
	fixture := newIntegrationFixture(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/kg/read/template/integration-domain/doc_lookup", strings.NewReader(`{"params":{"title":"Integration Doc"}}`))
		req.SetPathValue("domain_id", "integration-domain")
		req.SetPathValue("template_name", "doc_lookup")
		req = req.WithContext(access.ContextWithIdentity(req.Context(), fixture.actor))
		rec := httptest.NewRecorder()
		fixture.readHandler.ExecuteTemplate(rec, req)
		if rec.Code != 200 {
			b.Fatalf("ExecuteTemplate() status = %d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func BenchmarkSemanticSearch(b *testing.B) {
	fixture := newIntegrationFixture(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v1/kg/search/semantic", strings.NewReader(`{"query":"Integration Doc","domain_ids":["integration-domain"],"top_k":1}`))
		req = req.WithContext(access.ContextWithIdentity(req.Context(), fixture.actor))
		rec := httptest.NewRecorder()
		fixture.searchHandler.SemanticSearch(rec, req)
		if rec.Code != 200 {
			b.Fatalf("SemanticSearch() status = %d body=%s", rec.Code, rec.Body.String())
		}
	}
}

func BenchmarkWriteToSync(b *testing.B) {
	fixture := newIntegrationFixture(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		created, err := fixture.writeSvc.CreateNode(fixture.actor, write.NodeCreateRequest{
			DomainID:   "integration-domain",
			NodeType:   "Doc",
			Properties: map[string]any{"title": "Bench Doc", "status": "active"},
		})
		if err != nil {
			b.Fatalf("CreateNode() error = %v", err)
		}
		report := fixture.runtime.PollOnce()
		if report.Processed == 0 {
			b.Fatal("expected processed outbox event")
		}
		if got := fixture.runtime.Graph().Nodes[created.NodeID].StatusValue; got != "active" {
			b.Fatalf("projected status = %q, want active", got)
		}
	}
}
