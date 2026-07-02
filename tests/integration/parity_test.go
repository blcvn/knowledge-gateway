package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/integrity"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/session"
	"kg-service/internal/read"
	"kg-service/internal/search"
	"kg-service/internal/workers"
	"kg-service/internal/write"
)

type recordingSessionManager struct {
	lastScope session.SessionScope
}

func (m *recordingSessionManager) Within(ctx context.Context, identity session.WriteIdentity, fn func(session.SessionScope) error) (session.SessionScope, error) {
	scope := session.SessionScope{
		Identity: identity,
		Statements: []string{
			"BEGIN",
			"SET LOCAL app.tenant_id = '" + identity.TenantID + "'",
			"SET LOCAL app.app_id = '" + identity.AppID + "'",
			"COMMIT",
		},
		Transactional: true,
	}
	m.lastScope = scope
	return scope, fn(scope)
}

func TestEndToEndParityFlow(t *testing.T) {
	fixture := newIntegrationFixture(t)

	created, err := fixture.writeSvc.CreateNode(fixture.actor, write.NodeCreateRequest{
		DomainID:   "integration-domain",
		NodeType:   "Doc",
		Properties: map[string]any{"title": "Integration Doc", "status": "active"},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	if len(fixture.sessionMgr.lastScope.Statements) != 4 {
		t.Fatalf("session statements = %#v, want 4 statements", fixture.sessionMgr.lastScope.Statements)
	}

	readReq := httptest.NewRequest(http.MethodPost, "/v1/kg/read/template/integration-domain/doc_lookup", strings.NewReader(`{"params":{"title":"Integration Doc"}}`))
	readReq.SetPathValue("domain_id", "integration-domain")
	readReq.SetPathValue("template_name", "doc_lookup")
	readReq = readReq.WithContext(access.ContextWithIdentity(readReq.Context(), fixture.actor))
	readRec := httptest.NewRecorder()
	fixture.readHandler.ExecuteTemplate(readRec, readReq)
	if readRec.Code != http.StatusOK {
		t.Fatalf("ExecuteTemplate() status = %d body=%s", readRec.Code, readRec.Body.String())
	}

	searchReq := httptest.NewRequest(http.MethodPost, "/v1/kg/search/semantic", strings.NewReader(`{"query":"Integration Doc","domain_ids":["integration-domain"],"top_k":1}`))
	searchReq = searchReq.WithContext(access.ContextWithIdentity(searchReq.Context(), fixture.actor))
	searchRec := httptest.NewRecorder()
	fixture.searchHandler.SemanticSearch(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("SemanticSearch() status = %d body=%s", searchRec.Code, searchRec.Body.String())
	}

	ragReq := httptest.NewRequest(http.MethodPost, "/v1/kg/search/rag", strings.NewReader(`{"query":"Integration Doc","domain_ids":["integration-domain"],"top_k":1}`))
	ragReq = ragReq.WithContext(access.ContextWithIdentity(ragReq.Context(), fixture.actor))
	ragRec := httptest.NewRecorder()
	fixture.searchHandler.RagSearch(ragRec, ragReq)
	if ragRec.Code != http.StatusOK {
		t.Fatalf("RagSearch() status = %d body=%s", ragRec.Code, ragRec.Body.String())
	}

	report := fixture.runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatal("expected worker to process outbox event")
	}
	if got := fixture.runtime.Graph().Nodes[created.NodeID].StatusValue; got != "active" {
		t.Fatalf("projected status = %q, want active", got)
	}
	healthy := fixture.runtime.Reconcile()
	if healthy.Overall != "pass" {
		t.Fatalf("healthy reconcile = %+v", healthy)
	}

	fixture.runtime.Graph().Nodes[created.NodeID] = workers.GraphNode{ID: created.NodeID, NodeType: "Mismatch", DomainID: "integration-domain"}
	drift := fixture.runtime.Reconcile()
	if drift.Overall != "fail" || drift.GraphDriftCount == 0 {
		t.Fatalf("drift reconcile = %+v, want drift", drift)
	}
}

func TestEndToEndSearchPipelineReturnsProjectedNode(t *testing.T) {
	fixture := newIntegrationFixture(t)

	created, err := fixture.writeSvc.CreateNode(fixture.actor, write.NodeCreateRequest{
		DomainID:   "integration-domain",
		NodeType:   "Doc",
		Properties: map[string]any{"title": "Projected Search Doc", "status": "active"},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	if report := fixture.runtime.PollOnce(); report.Processed == 0 {
		t.Fatalf("PollOnce() = %+v, want processed work", report)
	}

	semanticReq := httptest.NewRequest(http.MethodPost, "/v1/kg/search/semantic", strings.NewReader(`{"query":"Projected Search Doc","domain_ids":["integration-domain"],"top_k":5}`))
	semanticReq = semanticReq.WithContext(access.ContextWithIdentity(semanticReq.Context(), fixture.actor))
	semanticRec := httptest.NewRecorder()
	fixture.searchHandler.SemanticSearch(semanticRec, semanticReq)
	if semanticRec.Code != http.StatusOK || !strings.Contains(semanticRec.Body.String(), created.NodeID) {
		t.Fatalf("SemanticSearch() status=%d body=%s", semanticRec.Code, semanticRec.Body.String())
	}

	fulltextReq := httptest.NewRequest(http.MethodPost, "/v1/kg/search/fulltext", strings.NewReader(`{"query":"Projected Search Doc","domain_ids":["integration-domain"],"top_k":5,"mode":"phrase"}`))
	fulltextReq = fulltextReq.WithContext(access.ContextWithIdentity(fulltextReq.Context(), fixture.actor))
	fulltextRec := httptest.NewRecorder()
	fixture.searchHandler.FullTextSearch(fulltextRec, fulltextReq)
	if fulltextRec.Code != http.StatusOK || !strings.Contains(fulltextRec.Body.String(), created.NodeID) {
		t.Fatalf("FullTextSearch() status=%d body=%s", fulltextRec.Code, fulltextRec.Body.String())
	}

	hybridReq := httptest.NewRequest(http.MethodPost, "/v1/kg/search/hybrid", strings.NewReader(`{"query":"Projected Search Doc","domain_ids":["integration-domain"],"top_k":5,"semantic_weight":0.5,"fts_operator":"phrase"}`))
	hybridReq = hybridReq.WithContext(access.ContextWithIdentity(hybridReq.Context(), fixture.actor))
	hybridRec := httptest.NewRecorder()
	fixture.searchHandler.HybridSearch(hybridRec, hybridReq)
	if hybridRec.Code != http.StatusOK || !strings.Contains(hybridRec.Body.String(), created.NodeID) {
		t.Fatalf("HybridSearch() status=%d body=%s", hybridRec.Code, hybridRec.Body.String())
	}
}

func TestReplaySurvivesRestart(t *testing.T) {
	fixture := newIntegrationFixture(t)

	_, err := fixture.writeSvc.CreateNode(fixture.actor, write.NodeCreateRequest{
		DomainID:   "integration-domain",
		NodeType:   "Doc",
		Properties: map[string]any{"title": "Restart Doc", "status": "active"},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}

	first := fixture.runtime.PollOnce()
	if first.Processed == 0 {
		t.Fatal("first poll expected processed work")
	}
	restarted := workers.NewRuntime(fixture.store, fixture.ontologySvc, &fixture.cache)
	second := restarted.PollOnce()
	if second.Processed != 0 || second.Failed != 0 || second.DeadLetter != 0 {
		t.Fatalf("restarted poll = %+v, want zero work", second)
	}
}

type integrationFixture struct {
	store         *write.MemoryStore
	ontologySvc   *ontology.Service
	writeSvc      *write.Service
	readSvc       *read.Service
	searchSvc     *search.Service
	integritySv   *integrity.Service
	runtime       *workers.Runtime
	readHandler   read.Handler
	searchHandler search.Handler
	actor         access.Identity
	cache         rediscache.Client
	sessionMgr    *recordingSessionManager
}

func newIntegrationFixture(t testing.TB) integrationFixture {
	t.Helper()

	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}
	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)
	accessSvc := access.NewService(accessStore, &cache)

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(
		ontology.SeedDomains(),
		ontology.SeedVersions(),
		ontology.SeedNodeTypes(),
		ontology.SeedRelTypes(),
		ontology.SeedCrossDomainRules(),
		ontology.SeedQueryTemplates(),
		ontology.SeedStatusFieldConfigs(),
	)
	ontologySvc := ontology.NewService(ontologyStore, accessResolver)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	if _, err := ontologySvc.CreateDomain(actor, actor.TenantID, ontology.DomainCreateRequest{ID: "integration-domain", Name: "Integration Domain"}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "integration-domain", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "Doc",
		RequiredProps: []ontology.PropertySchema{{Name: "title", Type: "string"}},
		OptionalProps: []ontology.PropertySchema{{Name: "status", Type: "string"}},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType() error = %v", err)
	}
	if _, err := ontologySvc.CreateQueryTemplate(actor, actor.TenantID, "integration-domain", ontology.QueryTemplateCreateRequest{
		TemplateName: "doc_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "Doc",
				"match":     map[string]any{"title": "$title"},
			},
		},
		ParamSchema:  []ontology.ParameterSchema{{Name: "title", Type: "string", Required: true}},
		ReturnFields: []string{"Doc.title"},
	}); err != nil {
		t.Fatalf("CreateQueryTemplate() error = %v", err)
	}
	if _, err := ontologySvc.ActivateQueryTemplate(actor, actor.TenantID, "integration-domain", "doc_lookup"); err != nil {
		t.Fatalf("ActivateQueryTemplate() error = %v", err)
	}
	if _, err := ontologySvc.UpsertStatusFieldConfig(actor, actor.TenantID, "integration-domain", ontology.StatusFieldConfigRequest{
		StatusFieldName:   "status",
		ValidStatusValues: []string{"active"},
	}); err != nil {
		t.Fatalf("UpsertStatusFieldConfig() error = %v", err)
	}
	sessionMgr := &recordingSessionManager{}
	store := write.NewMemoryStore()
	writeSvc := write.NewService(store, ontologySvc, accessResolver, sessionMgr, accessSvc)
	readSvc := read.NewService(store, ontologySvc, accessResolver, accessSvc)
	searchSvc := search.NewService(store, ontologySvc, accessResolver, accessSvc)
	integritySvc := integrity.NewService(store, ontologyStore, nil)
	runtime := workers.NewRuntime(store, ontologySvc, &cache)
	searchSvc.SetVectorAdapter(runtime.VectorAdapter())
	searchSvc.SetFTSAdapter(runtime.FTSAdapter())

	return integrationFixture{
		store:         store,
		ontologySvc:   ontologySvc,
		writeSvc:      writeSvc,
		readSvc:       readSvc,
		searchSvc:     searchSvc,
		integritySv:   integritySvc,
		runtime:       runtime,
		readHandler:   read.NewHandler(readSvc),
		searchHandler: search.NewHandler(searchSvc),
		actor:         actor,
		cache:         cache,
		sessionMgr:    sessionMgr,
	}
}
