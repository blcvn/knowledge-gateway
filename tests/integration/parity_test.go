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
	"kg-service/internal/runtimeobs"
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

func TestEndToEndWriteHttpToWorkerCarriesRequestMeta(t *testing.T) {
	fixture := newIntegrationFixture(t)

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{"domain_id":"integration-domain","node_type":"Doc","properties":{"title":"HTTP Meta Doc","status":"active"}}`))
	req = req.WithContext(access.ContextWithIdentity(req.Context(), fixture.actor))
	req = req.WithContext(runtimeobs.WithRequestMeta(req.Context(), runtimeobs.NewRequestMeta("req-1", "trace-1", "span-1")))
	rec := httptest.NewRecorder()
	fixture.writeHandler.CreateNode(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("CreateNode() status = %d body=%s", rec.Code, rec.Body.String())
	}

	events := fixture.store.ListOutboxEvents()
	if len(events) != 1 {
		t.Fatalf("outbox len = %d, want 1", len(events))
	}
	payload := events[0].Payload
	if payload["request_id"] != "req-1" || payload["trace_id"] != "trace-1" || payload["span_id"] != "span-1" {
		t.Fatalf("outbox payload missing request meta: %#v", payload)
	}

	report := fixture.runtime.PollOnce()
	if report.Processed == 0 {
		t.Fatalf("PollOnce() = %+v, want processed work", report)
	}
	if got := fixture.runtime.Graph().Nodes[events[0].AggregateID].StatusValue; got != "active" {
		t.Fatalf("projected status = %q, want active", got)
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
	accessStore   *access.MemoryStore
	accessSvc     *access.Service
	store         *write.MemoryStore
	ontologySvc   *ontology.Service
	writeSvc      *write.Service
	readSvc       *read.Service
	searchSvc     *search.Service
	integritySv   *integrity.Service
	runtime       *workers.Runtime
	writeHandler  write.Handler
	readHandler   read.Handler
	searchHandler search.Handler
	actor         access.Identity
	cache         rediscache.Client
	sessionMgr    *recordingSessionManager
}

type stubOwnerRegistry struct {
	tenants map[string]access.Tenant
	apps    map[string]access.App
}

func (s *stubOwnerRegistry) GetTenant(id string) (access.Tenant, bool) {
	if s == nil {
		return access.Tenant{}, false
	}
	tenant, ok := s.tenants[id]
	return tenant, ok
}

func (s *stubOwnerRegistry) GetAppByID(id string) (access.App, bool) {
	if s == nil {
		return access.App{}, false
	}
	app, ok := s.apps[id]
	return app, ok
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
			AppID:    access.TestAlphaAdminAppID,
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
	writeSvc := write.NewService(store, ontologySvc, accessResolver, accessStore, sessionMgr, accessSvc)
	readSvc := read.NewService(store, ontologySvc, accessResolver, accessSvc)
	searchSvc := search.NewService(store, ontologySvc, accessResolver, accessSvc)
	integritySvc := integrity.NewService(store, ontologyStore, nil)
	runtime := workers.NewRuntime(store, ontologySvc, &cache)
	searchSvc.SetVectorAdapter(runtime.VectorAdapter())
	searchSvc.SetFTSAdapter(runtime.FTSAdapter())

	return integrationFixture{
		accessStore:   accessStore,
		accessSvc:     accessSvc,
		store:         store,
		ontologySvc:   ontologySvc,
		writeSvc:      writeSvc,
		readSvc:       readSvc,
		searchSvc:     searchSvc,
		integritySv:   integritySvc,
		runtime:       runtime,
		writeHandler:  write.NewHandler(writeSvc),
		readHandler:   read.NewHandler(readSvc),
		searchHandler: search.NewHandler(searchSvc),
		actor:         actor,
		cache:         cache,
		sessionMgr:    sessionMgr,
	}
}

func TestOnboardedAppCanAuthenticateAndWrite(t *testing.T) {
	fixture := newIntegrationFixture(t)

	platformAdmin := access.Identity{
		TenantID: access.PlatformTenantID,
		AppID:    access.PlatformAdminAppID,
		AppType:  "admin_tool",
	}
	tenantResp, err := fixture.accessSvc.CreateTenant(platformAdmin, access.TenantCreateRequest{
		Slug: "integration-onboarded",
		Name: "Integration Onboarded Tenant",
		Tier: "pro",
	})
	if err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}
	appResp, err := fixture.accessSvc.CreateApp(platformAdmin, tenantResp.ID, access.AppCreateRequest{
		Slug: "integration-writer",
		Name: "Integration Writer",
		Type: "admin_tool",
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	if appResp.APIKey == "" {
		t.Fatal("CreateApp() returned empty api key")
	}

	actor := access.Identity{}
	authReq, err := http.NewRequest(http.MethodGet, "/v1/access/resolve", nil)
	if err != nil {
		t.Fatalf("http.NewRequest() error = %v", err)
	}
	authReq.Header.Set("Authorization", "Bearer "+appResp.APIKey)
	authRec := httptest.NewRecorder()
	authHandler := access.NewMiddleware(access.NewIdentityResolver(fixture.accessStore, &fixture.cache)).RequireIdentity(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		actor, ok = access.IdentityFromContext(r.Context())
		if !ok {
			http.Error(w, "missing identity", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	authHandler.ServeHTTP(authRec, authReq)
	if authRec.Code != http.StatusNoContent {
		t.Fatalf("auth status = %d body=%s", authRec.Code, authRec.Body.String())
	}
	if actor.TenantID != tenantResp.ID || actor.AppID != appResp.ID {
		t.Fatalf("resolved identity = %+v, want tenant=%s app=%s", actor, tenantResp.ID, appResp.ID)
	}

	_, err = fixture.ontologySvc.CreateDomain(platformAdmin, tenantResp.ID, ontology.DomainCreateRequest{
		ID:         "integration-onboarded-domain",
		Name:       "Integration Onboarded Domain",
		Status:     "active",
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	_, err = fixture.ontologySvc.CreateNodeType(platformAdmin, tenantResp.ID, "integration-onboarded-domain", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "Doc",
		RequiredProps: []ontology.PropertySchema{{Name: "title", Type: "string"}},
		OptionalProps: []ontology.PropertySchema{{Name: "status", Type: "string"}},
	})
	if err != nil {
		t.Fatalf("CreateNodeType() error = %v", err)
	}

	writeReq := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{"domain_id":"integration-onboarded-domain","node_type":"Doc","properties":{"title":"Onboarded Doc","status":"active"}}`))
	writeReq.Header.Set("Authorization", "Bearer "+appResp.APIKey)
	writeRec := httptest.NewRecorder()
	middleware := access.NewMiddleware(access.NewIdentityResolver(fixture.accessStore, &fixture.cache))
	middleware.RequireIdentity(http.HandlerFunc(fixture.writeHandler.CreateNode)).ServeHTTP(writeRec, writeReq)
	if writeRec.Code != http.StatusAccepted {
		t.Fatalf("CreateNode() status = %d body=%s", writeRec.Code, writeRec.Body.String())
	}
	nodes := fixture.store.ListNodes()
	if len(nodes) != 1 {
		t.Fatalf("ListNodes() len = %d, want 1", len(nodes))
	}
	if nodes[0].OwnerTenantID != tenantResp.ID || nodes[0].OwnerAppID != appResp.ID {
		t.Fatalf("stored owner identity = tenant=%s app=%s, want tenant=%s app=%s", nodes[0].OwnerTenantID, nodes[0].OwnerAppID, tenantResp.ID, appResp.ID)
	}
}

func TestAuthenticatedWriteReturnsServiceUnavailableWhenDurableOwnerAppIsMissing(t *testing.T) {
	fixture := newIntegrationFixture(t)

	platformAdmin := access.Identity{
		TenantID: access.PlatformTenantID,
		AppID:    access.PlatformAdminAppID,
		AppType:  "admin_tool",
	}
	tenantResp, err := fixture.accessSvc.CreateTenant(platformAdmin, access.TenantCreateRequest{
		Slug: "integration-stale-owner",
		Name: "Integration Stale Owner Tenant",
		Tier: "pro",
	})
	if err != nil {
		t.Fatalf("CreateTenant() error = %v", err)
	}
	appResp, err := fixture.accessSvc.CreateApp(platformAdmin, tenantResp.ID, access.AppCreateRequest{
		Slug: "integration-stale-writer",
		Name: "Integration Stale Writer",
		Type: "admin_tool",
	})
	if err != nil {
		t.Fatalf("CreateApp() error = %v", err)
	}
	tenant, ok := fixture.accessStore.GetTenant(tenantResp.ID)
	if !ok {
		t.Fatalf("GetTenant(%s) = missing", tenantResp.ID)
	}
	_, ok = fixture.accessStore.GetAppByID(appResp.ID)
	if !ok {
		t.Fatalf("GetAppByID(%s) = missing", appResp.ID)
	}
	_, err = fixture.ontologySvc.CreateDomain(platformAdmin, tenantResp.ID, ontology.DomainCreateRequest{
		ID:         "integration-stale-owner-domain",
		Name:       "Integration Stale Owner Domain",
		Status:     "active",
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	_, err = fixture.ontologySvc.CreateNodeType(platformAdmin, tenantResp.ID, "integration-stale-owner-domain", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "Doc",
		RequiredProps: []ontology.PropertySchema{{Name: "title", Type: "string"}},
	})
	if err != nil {
		t.Fatalf("CreateNodeType() error = %v", err)
	}

	staleRegistry := &stubOwnerRegistry{
		tenants: map[string]access.Tenant{tenantResp.ID: tenant},
		apps:    map[string]access.App{},
	}
	accessResolver := access.NewAccessResolver(fixture.accessStore, fixture.accessStore, &fixture.cache)
	writeSvc := write.NewService(fixture.store, fixture.ontologySvc, accessResolver, staleRegistry, fixture.sessionMgr, fixture.accessSvc)
	writeHandler := write.NewHandler(writeSvc)

	writeReq := httptest.NewRequest(http.MethodPost, "/v1/kg/write/nodes", strings.NewReader(`{"domain_id":"integration-stale-owner-domain","node_type":"Doc","properties":{"title":"Should Fail Before FK"}}`))
	writeReq.Header.Set("Authorization", "Bearer "+appResp.APIKey)
	writeRec := httptest.NewRecorder()
	middleware := access.NewMiddleware(access.NewIdentityResolver(fixture.accessStore, &fixture.cache))
	middleware.RequireIdentity(http.HandlerFunc(writeHandler.CreateNode)).ServeHTTP(writeRec, writeReq)
	if writeRec.Code != http.StatusServiceUnavailable {
		t.Fatalf("CreateNode() status = %d body=%s", writeRec.Code, writeRec.Body.String())
	}
}
