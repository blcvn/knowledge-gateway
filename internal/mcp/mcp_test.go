package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/httpapi/respond"
	"kg-service/internal/integrity"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/platform/session"
	"kg-service/internal/read"
	"kg-service/internal/search"
	"kg-service/internal/write"
)

type recordingSessionManager struct{}

func (recordingSessionManager) Within(ctx context.Context, identity session.WriteIdentity, fn func(session.SessionScope) error) (session.SessionScope, error) {
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
	return scope, fn(scope)
}

func TestMCPConnectListAndCallTools(t *testing.T) {
	handler, actor := newMCPFixture(t)

	connectReq := httptest.NewRequest(http.MethodGet, "/v1/mcp/connect", nil)
	connectReq = connectReq.WithContext(access.ContextWithIdentity(connectReq.Context(), actor))
	connectRec := httptest.NewRecorder()
	handler.Connect(connectRec, connectReq)
	if connectRec.Code != http.StatusOK {
		t.Fatalf("connect status = %d body=%s", connectRec.Code, connectRec.Body.String())
	}
	sessionID := extractSessionID(connectRec.Body.String())
	if sessionID == "" {
		t.Fatalf("session body = %s", connectRec.Body.String())
	}

	listResp := callMCP(t, handler, sessionID, JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"})
	var listPayload map[string]any
	mustDecodeJSON(t, listResp.Body.Bytes(), &listPayload)
	result := listPayload["result"].(map[string]any)
	if !containsTool(result["tools"].([]any), "kg_list_templates") || !containsTool(result["tools"].([]any), "kg_entity_sync_status") {
		t.Fatalf("tools = %#v", result["tools"])
	}

	callResp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_list_templates",
			"arguments": map[string]any{
				"domain_id": "noi_bo_hop_dong",
			},
		},
	})
	mustDecodeJSON(t, callResp.Body.Bytes(), &listPayload)
	if listPayload["error"] != nil {
		t.Fatalf("kg_list_templates error = %#v", listPayload["error"])
	}
	if _, ok := listPayload["result"].(map[string]any)["templates"]; !ok {
		t.Fatalf("kg_list_templates result = %#v", listPayload["result"])
	}

	searchResp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_search",
			"arguments": map[string]any{
				"query":      "Hop dong",
				"domain_ids": []any{"noi_bo_hop_dong"},
				"top_k":      1,
			},
		},
	})
	mustDecodeJSON(t, searchResp.Body.Bytes(), &listPayload)
	if listPayload["error"] != nil {
		t.Fatalf("kg_search error = %#v", listPayload["error"])
	}
}

func TestMCPCallEntitySyncStatus(t *testing.T) {
	fixture := newParityFixture(t)
	handler := fixture.mcpHandler

	connectReq := httptest.NewRequest(http.MethodGet, "/v1/mcp/connect", nil)
	connectReq = connectReq.WithContext(access.ContextWithIdentity(connectReq.Context(), fixture.actor))
	connectRec := httptest.NewRecorder()
	handler.Connect(connectRec, connectReq)
	sessionID := extractSessionID(connectRec.Body.String())
	if sessionID == "" {
		t.Fatalf("session body = %s", connectRec.Body.String())
	}

	resp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      99,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_entity_sync_status",
			"arguments": map[string]any{
				"entity_id":   fixture.bridgeID,
				"entity_kind": "kg_node",
			},
		},
	})
	var payload map[string]any
	mustDecodeJSON(t, resp.Body.Bytes(), &payload)
	if payload["error"] != nil {
		t.Fatalf("kg_entity_sync_status error = %#v", payload["error"])
	}
	result := payload["result"].(map[string]any)
	if result["graph_lag_class"] != "SYNCED" || result["vector_lag_class"] != "SYNCED" {
		t.Fatalf("result = %#v", result)
	}
}

func TestMCPCheckAccessMatchesRESTVisibility(t *testing.T) {
	fixture := newParityFixture(t)
	handler := fixture.mcpHandler

	connectReq := httptest.NewRequest(http.MethodGet, "/v1/mcp/connect", nil)
	connectReq = connectReq.WithContext(access.ContextWithIdentity(connectReq.Context(), fixture.actor))
	connectRec := httptest.NewRecorder()
	handler.Connect(connectRec, connectReq)
	sessionID := extractSessionID(connectRec.Body.String())
	if sessionID == "" {
		t.Fatalf("session body = %s", connectRec.Body.String())
	}

	restAccessReq := httptest.NewRequest(http.MethodGet, "/v1/access/resolve", nil)
	restAccessReq = restAccessReq.WithContext(access.ContextWithIdentity(restAccessReq.Context(), fixture.actor))
	restAccessRec := httptest.NewRecorder()
	fixture.accessHandler.GetResolve(restAccessRec, restAccessReq)
	if restAccessRec.Code != http.StatusOK {
		t.Fatalf("access REST status = %d body=%s", restAccessRec.Code, restAccessRec.Body.String())
	}
	var restAccess access.ResolveResponse
	mustDecodeJSON(t, restAccessRec.Body.Bytes(), &restAccess)

	mcpResp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_check_access",
		},
	})
	var mcpPayload map[string]any
	mustDecodeJSON(t, mcpResp.Body.Bytes(), &mcpPayload)
	if mcpPayload["error"] != nil {
		t.Fatalf("kg_check_access error = %#v", mcpPayload["error"])
	}
	result := mcpPayload["result"].(map[string]any)
	rawVisibleOwners, err := json.Marshal(result["visible_owners"])
	if err != nil {
		t.Fatalf("json.Marshal() mcp visible owners error = %v", err)
	}
	var mcpVisibleOwners []access.VisibleOwner
	if err := json.Unmarshal(rawVisibleOwners, &mcpVisibleOwners); err != nil {
		t.Fatalf("json.Unmarshal() mcp visible owners error = %v", err)
	}
	if !reflect.DeepEqual(restAccess.VisibleOwners, mcpVisibleOwners) {
		t.Fatalf("REST owners = %#v, MCP owners = %#v", restAccess.VisibleOwners, mcpVisibleOwners)
	}

	restListReq := httptest.NewRequest(http.MethodGet, "/v1/kg/read/templates?domain_id=noi_bo_hop_dong", nil)
	restListReq = restListReq.WithContext(access.ContextWithIdentity(restListReq.Context(), fixture.actor))
	restListRec := httptest.NewRecorder()
	fixture.readHandler.ListTemplates(restListRec, restListReq)
	if restListRec.Code != http.StatusOK {
		t.Fatalf("read REST list status = %d body=%s", restListRec.Code, restListRec.Body.String())
	}
	var restTemplates respond.ListEnvelope[read.TemplateListItem]
	mustDecodeJSON(t, restListRec.Body.Bytes(), &restTemplates)

	mcpTemplatesResp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      11,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_list_templates",
			"arguments": map[string]any{
				"domain_id": "noi_bo_hop_dong",
			},
		},
	})
	mustDecodeJSON(t, mcpTemplatesResp.Body.Bytes(), &mcpPayload)
	if mcpPayload["error"] != nil {
		t.Fatalf("kg_list_templates error = %#v", mcpPayload["error"])
	}
	var mcpTemplates []read.TemplateListItem
	rawTemplates, err := json.Marshal(mcpPayload["result"].(map[string]any)["templates"])
	if err != nil {
		t.Fatalf("json.Marshal() mcp templates error = %v", err)
	}
	if err := json.Unmarshal(rawTemplates, &mcpTemplates); err != nil {
		t.Fatalf("json.Unmarshal() mcp templates error = %v", err)
	}
	if !reflect.DeepEqual(restTemplates.Data, mcpTemplates) {
		t.Fatalf("REST templates = %#v, MCP templates = %#v", restTemplates.Data, mcpTemplates)
	}

	restReadReq := httptest.NewRequest(http.MethodPost, "/v1/kg/read/template/noi_bo_hop_dong/contract_lookup", strings.NewReader(`{"params":{"ten":"Hop dong A"}}`))
	restReadReq.SetPathValue("domain_id", "noi_bo_hop_dong")
	restReadReq.SetPathValue("template_name", "contract_lookup")
	restReadReq = restReadReq.WithContext(access.ContextWithIdentity(restReadReq.Context(), fixture.actor))
	restReadRec := httptest.NewRecorder()
	fixture.readHandler.ExecuteTemplate(restReadRec, restReadReq)
	if restReadRec.Code != http.StatusOK {
		t.Fatalf("read REST execute status = %d body=%s", restReadRec.Code, restReadRec.Body.String())
	}
	var restRead read.TemplateExecutionResponse
	mustDecodeJSON(t, restReadRec.Body.Bytes(), &restRead)

	mcpReadResp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      12,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_read_pattern",
			"arguments": map[string]any{
				"domain_id":     "noi_bo_hop_dong",
				"template_name": "contract_lookup",
				"params":        map[string]any{"ten": "Hop dong A"},
			},
		},
	})
	mustDecodeJSON(t, mcpReadResp.Body.Bytes(), &mcpPayload)
	if mcpPayload["error"] != nil {
		t.Fatalf("kg_read_pattern error = %#v", mcpPayload["error"])
	}
	var mcpRead read.TemplateExecutionResponse
	rawRead, err := json.Marshal(mcpPayload["result"])
	if err != nil {
		t.Fatalf("json.Marshal() mcp read result error = %v", err)
	}
	if err := json.Unmarshal(rawRead, &mcpRead); err != nil {
		t.Fatalf("json.Unmarshal() mcp read result error = %v", err)
	}
	if !reflect.DeepEqual(restRead.Results, mcpRead.Results) {
		t.Fatalf("REST read = %#v, MCP read = %#v", restRead.Results, mcpRead.Results)
	}

	restSearchReq := httptest.NewRequest(http.MethodPost, "/v1/kg/search/semantic", strings.NewReader(`{"query":"Hop dong","domain_ids":["noi_bo_hop_dong"],"top_k":1}`))
	restSearchReq = restSearchReq.WithContext(access.ContextWithIdentity(restSearchReq.Context(), fixture.actor))
	restSearchRec := httptest.NewRecorder()
	fixture.searchHandler.SemanticSearch(restSearchRec, restSearchReq)
	if restSearchRec.Code != http.StatusOK {
		t.Fatalf("search REST status = %d body=%s", restSearchRec.Code, restSearchRec.Body.String())
	}
	var restSearch search.SemanticSearchResponse
	mustDecodeJSON(t, restSearchRec.Body.Bytes(), &restSearch)

	mcpSearchResp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      13,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_search",
			"arguments": map[string]any{
				"query":      "Hop dong",
				"domain_ids": []any{"noi_bo_hop_dong"},
				"top_k":      1,
			},
		},
	})
	mustDecodeJSON(t, mcpSearchResp.Body.Bytes(), &mcpPayload)
	if mcpPayload["error"] != nil {
		t.Fatalf("kg_search error = %#v", mcpPayload["error"])
	}
	var mcpSearch search.SemanticSearchResponse
	rawSearch, err := json.Marshal(mcpPayload["result"])
	if err != nil {
		t.Fatalf("json.Marshal() mcp search result error = %v", err)
	}
	if err := json.Unmarshal(rawSearch, &mcpSearch); err != nil {
		t.Fatalf("json.Unmarshal() mcp search result error = %v", err)
	}
	if !reflect.DeepEqual(restSearch.Results, mcpSearch.Results) {
		t.Fatalf("REST search = %#v, MCP search = %#v", restSearch.Results, mcpSearch.Results)
	}

	restIntegrityReq := httptest.NewRequest(http.MethodGet, "/v1/kg/integrity/tenant/"+fixture.actor.TenantID, nil)
	restIntegrityReq.SetPathValue("tenant_id", fixture.actor.TenantID)
	restIntegrityReq = restIntegrityReq.WithContext(access.ContextWithIdentity(restIntegrityReq.Context(), fixture.actor))
	restIntegrityRec := httptest.NewRecorder()
	fixture.integrityHandler.TenantIntegrity(restIntegrityRec, restIntegrityReq)
	if restIntegrityRec.Code != http.StatusOK {
		t.Fatalf("integrity REST status = %d body=%s", restIntegrityRec.Code, restIntegrityRec.Body.String())
	}
	var restIntegrity integrity.TenantIntegrityResponse
	mustDecodeJSON(t, restIntegrityRec.Body.Bytes(), &restIntegrity)

	restMissingReq := httptest.NewRequest(http.MethodGet, "/v1/kg/integrity/missing-bridges?tenant_id="+fixture.actor.TenantID, nil)
	restMissingReq = restMissingReq.WithContext(access.ContextWithIdentity(restMissingReq.Context(), fixture.actor))
	restMissingRec := httptest.NewRecorder()
	fixture.integrityHandler.MissingBridges(restMissingRec, restMissingReq)
	if restMissingRec.Code != http.StatusOK {
		t.Fatalf("integrity REST missing bridges status = %d body=%s", restMissingRec.Code, restMissingRec.Body.String())
	}
	var restMissing respond.ListEnvelope[integrity.MissingBridgeItem]
	mustDecodeJSON(t, restMissingRec.Body.Bytes(), &restMissing)

	mcpIntegrityResp := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      14,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_integrity",
			"arguments": map[string]any{
				"tenant_id": fixture.actor.TenantID,
			},
		},
	})
	mustDecodeJSON(t, mcpIntegrityResp.Body.Bytes(), &mcpPayload)
	if mcpPayload["error"] != nil {
		t.Fatalf("kg_integrity error = %#v", mcpPayload["error"])
	}
	integrityResult := mcpPayload["result"].(map[string]any)
	rawSummary, err := json.Marshal(integrityResult["summary"])
	if err != nil {
		t.Fatalf("json.Marshal() mcp summary error = %v", err)
	}
	var mcpIntegrity integrity.TenantIntegrityResponse
	if err := json.Unmarshal(rawSummary, &mcpIntegrity); err != nil {
		t.Fatalf("json.Unmarshal() mcp summary error = %v", err)
	}
	if !reflect.DeepEqual(restIntegrity.Checks, mcpIntegrity.Checks) || restIntegrity.Overall != mcpIntegrity.Overall {
		t.Fatalf("REST integrity = %#v, MCP integrity = %#v", restIntegrity, mcpIntegrity)
	}
	rawMissing, err := json.Marshal(integrityResult["missing_bridges"])
	if err != nil {
		t.Fatalf("json.Marshal() mcp missing bridges error = %v", err)
	}
	var mcpMissing []integrity.MissingBridgeItem
	if err := json.Unmarshal(rawMissing, &mcpMissing); err != nil {
		t.Fatalf("json.Unmarshal() mcp missing bridges error = %v", err)
	}
	if !reflect.DeepEqual(restMissing.Data, mcpMissing) {
		t.Fatalf("REST missing bridges = %#v, MCP missing bridges = %#v", restMissing.Data, mcpMissing)
	}
}

func TestMCPSessionRateLimitsToolCallsByTenantTier(t *testing.T) {
	handler, actor := newMCPFixtureWithLimiter(t, map[string]int{"pro": 1})

	connectReq := httptest.NewRequest(http.MethodGet, "/v1/mcp/connect", nil)
	connectReq = connectReq.WithContext(access.ContextWithIdentity(connectReq.Context(), actor))
	connectRec := httptest.NewRecorder()
	handler.Connect(connectRec, connectReq)
	sessionID := extractSessionID(connectRec.Body.String())
	if sessionID == "" {
		t.Fatalf("session body = %s", connectRec.Body.String())
	}

	first := callMCP(t, handler, sessionID, JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_list_templates",
			"arguments": map[string]any{
				"domain_id": "noi_bo_hop_dong",
			},
		},
	})
	var firstPayload map[string]any
	mustDecodeJSON(t, first.Body.Bytes(), &firstPayload)
	if firstPayload["error"] != nil {
		t.Fatalf("first call error = %#v", firstPayload["error"])
	}

	secondReq := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params: map[string]any{
			"name": "kg_list_templates",
			"arguments": map[string]any{
				"domain_id": "noi_bo_hop_dong",
			},
		},
	}
	secondBody, err := json.Marshal(secondReq)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	secondHTTP := httptest.NewRequest(http.MethodPost, "/v1/mcp/messages/"+sessionID, bytes.NewReader(secondBody))
	secondHTTP.SetPathValue("session_id", sessionID)
	secondRec := httptest.NewRecorder()
	handler.Message(secondRec, secondHTTP)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("message status = %d body=%s", secondRec.Code, secondRec.Body.String())
	}

	var secondPayload map[string]any
	mustDecodeJSON(t, secondRec.Body.Bytes(), &secondPayload)
	errObj, ok := secondPayload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error payload, got %#v", secondPayload)
	}
	if got := int(errObj["code"].(float64)); got != -32029 {
		t.Fatalf("error code = %d, want -32029", got)
	}
}

func newMCPFixture(t *testing.T) (Handler, access.Identity) {
	t.Helper()
	fixture := newParityFixture(t)
	return fixture.mcpHandler, fixture.actor
}

type parityFixture struct {
	accessHandler    access.Handler
	readHandler      read.Handler
	searchHandler    search.Handler
	integrityHandler integrity.Handler
	mcpHandler       Handler
	actor            access.Identity
	bridgeID         string
	createdID        string
}

func newParityFixture(t *testing.T) parityFixture {
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
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologySvc := ontology.NewService(ontologyStore, accessResolver)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	if _, err := ontologySvc.CreateDomain(actor, actor.TenantID, ontology.DomainCreateRequest{ID: "noi_bo_hop_dong", Name: "Noi Bo Hop Dong"}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "HopDongMau",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "PhuLucHopDong",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(bridge) error = %v", err)
	}
	if _, err := ontologySvc.CreateQueryTemplate(actor, actor.TenantID, "noi_bo_hop_dong", ontology.QueryTemplateCreateRequest{
		TemplateName: "contract_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "HopDongMau",
				"match":     map[string]any{"ten": "$ten"},
			},
		},
		ParamSchema:  []ontology.ParameterSchema{{Name: "ten", Type: "string", Required: true}},
		ReturnFields: []string{"HopDongMau.ten"},
	}); err != nil {
		t.Fatalf("CreateQueryTemplate() error = %v", err)
	}
	if _, err := ontologySvc.ActivateQueryTemplate(actor, actor.TenantID, "noi_bo_hop_dong", "contract_lookup"); err != nil {
		t.Fatalf("ActivateQueryTemplate() error = %v", err)
	}

	writeStore := write.NewMemoryStore()
	writeSvc := write.NewService(writeStore, ontologySvc, accessResolver, &recordingSessionManager{}, nil)
	bridge, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "PhuLucHopDong",
		Properties: map[string]any{"ten": "Phu luc A"},
	})
	if err != nil {
		t.Fatalf("CreateNode(bridge) error = %v", err)
	}
	created, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{bridge.NodeID}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	_ = created
	now := time.Now().UTC()
	_ = writeStore.UpsertProjectionVersion(context.Background(), write.ProjectionVersionRecord{
		EntityID:           bridge.NodeID,
		EntityKind:         "kg_node",
		SourceVersion:      int64(bridge.DomainVersion),
		SourceEventID:      "evt-bridge",
		SourceUpdatedAt:    now.Add(-2 * time.Second),
		GraphBackend:       "graph",
		GraphVersion:       int64(bridge.DomainVersion),
		LastGraphSyncedAt:  now.Add(-1 * time.Second),
		VectorBackend:      "vector",
		VectorVersion:      int64(bridge.DomainVersion),
		LastVectorSyncedAt: now.Add(-1 * time.Second),
	})

	readSvc := read.NewService(writeStore, ontologySvc, accessResolver, accessSvc)
	searchSvc := search.NewService(writeStore, ontologySvc, accessResolver, accessSvc)
	integritySvc := integrity.NewService(writeStore, ontologyStore)
	mcpSvc := NewService(readSvc, searchSvc, writeSvc, ontologySvc, accessResolver, integritySvc)
	return parityFixture{
		accessHandler:    access.NewHandler(accessResolver, accessSvc),
		readHandler:      read.NewHandler(readSvc),
		searchHandler:    search.NewHandler(searchSvc),
		integrityHandler: integrity.NewHandler(integritySvc),
		mcpHandler:       NewHandler(mcpSvc),
		actor:            actor,
		bridgeID:         bridge.NodeID,
		createdID:        created.NodeID,
	}
}

func newMCPFixtureWithLimiter(t *testing.T, tierLimits map[string]int) (Handler, access.Identity) {
	t.Helper()

	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}

	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)
	accessSvc := access.NewService(accessStore, &cache)
	limiter := access.NewRateLimiter(accessStore, tierLimits)
	limiter.SetNow(func() time.Time { return time.Date(2026, 6, 17, 10, 0, 0, 0, time.UTC) })

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologySvc := ontology.NewService(ontologyStore, accessResolver)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	if _, err := ontologySvc.CreateDomain(actor, actor.TenantID, ontology.DomainCreateRequest{ID: "noi_bo_hop_dong", Name: "Noi Bo Hop Dong"}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "HopDongMau",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName:  "PhuLucHopDong",
		RequiredProps: []ontology.PropertySchema{{Name: "ten", Type: "string"}},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(bridge) error = %v", err)
	}
	if _, err := ontologySvc.CreateQueryTemplate(actor, actor.TenantID, "noi_bo_hop_dong", ontology.QueryTemplateCreateRequest{
		TemplateName: "contract_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "HopDongMau",
				"match":     map[string]any{"ten": "$ten"},
			},
		},
		ParamSchema:  []ontology.ParameterSchema{{Name: "ten", Type: "string", Required: true}},
		ReturnFields: []string{"HopDongMau.ten"},
	}); err != nil {
		t.Fatalf("CreateQueryTemplate() error = %v", err)
	}
	if _, err := ontologySvc.ActivateQueryTemplate(actor, actor.TenantID, "noi_bo_hop_dong", "contract_lookup"); err != nil {
		t.Fatalf("ActivateQueryTemplate() error = %v", err)
	}

	writeStore := write.NewMemoryStore()
	writeSvc := write.NewService(writeStore, ontologySvc, accessResolver, &recordingSessionManager{}, nil)
	bridge, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "PhuLucHopDong",
		Properties: map[string]any{"ten": "Phu luc A"},
	})
	if err != nil {
		t.Fatalf("CreateNode(bridge) error = %v", err)
	}
	created, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:   "noi_bo_hop_dong",
		NodeType:   "HopDongMau",
		Properties: map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{bridge.NodeID}},
	})
	if err != nil {
		t.Fatalf("CreateNode() error = %v", err)
	}
	_ = created

	readSvc := read.NewService(writeStore, ontologySvc, accessResolver, accessSvc)
	searchSvc := search.NewService(writeStore, ontologySvc, accessResolver, accessSvc)
	integritySvc := integrity.NewService(writeStore, ontologyStore)
	mcpSvc := NewService(readSvc, searchSvc, writeSvc, ontologySvc, accessResolver, integritySvc)
	return NewHandler(mcpSvc, limiter), actor
}

func callMCP(t *testing.T, handler Handler, sessionID string, req JSONRPCRequest) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/v1/mcp/messages/"+sessionID, bytes.NewReader(body))
	httpReq.SetPathValue("session_id", sessionID)
	rec := httptest.NewRecorder()
	handler.Message(rec, httpReq)
	if rec.Code != http.StatusOK {
		t.Fatalf("message status = %d body=%s", rec.Code, rec.Body.String())
	}
	return rec
}

func extractSessionID(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if strings.Contains(line, "\"session_id\"") {
			start := strings.Index(line, "\"session_id\":\"")
			if start < 0 {
				continue
			}
			start += len("\"session_id\":\"")
			end := strings.Index(line[start:], "\"")
			if end < 0 {
				continue
			}
			return line[start : start+end]
		}
	}
	return ""
}

func mustDecodeJSON(t *testing.T, data []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}

func containsTool(tools []any, name string) bool {
	for _, tool := range tools {
		item, ok := tool.(map[string]any)
		if !ok {
			continue
		}
		if item["name"] == name {
			return true
		}
	}
	return false
}
