package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/integrity"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/read"
	"kg-service/internal/search"
	"kg-service/internal/write"
)

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
	if !containsTool(result["tools"].([]any), "kg_list_templates") {
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

func TestMCPCheckAccessMatchesRESTVisibility(t *testing.T) {
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

	writeStore := write.NewMemoryStore()
	writeSvc := write.NewService(writeStore, ontologySvc, accessResolver, &postgres.SessionManager{}, nil)
	readSvc := read.NewService(writeStore, ontologySvc, accessResolver, accessSvc)
	searchSvc := search.NewService(writeStore, ontologySvc, accessResolver, accessSvc)
	integritySvc := integrity.NewService(writeStore, ontologyStore)
	mcpSvc := NewService(readSvc, searchSvc, writeSvc, ontologySvc, accessResolver, integritySvc)
	handler := NewHandler(mcpSvc)

	connectReq := httptest.NewRequest(http.MethodGet, "/v1/mcp/connect", nil)
	connectReq = connectReq.WithContext(access.ContextWithIdentity(connectReq.Context(), actor))
	connectRec := httptest.NewRecorder()
	handler.Connect(connectRec, connectReq)
	sessionID := extractSessionID(connectRec.Body.String())
	if sessionID == "" {
		t.Fatalf("session body = %s", connectRec.Body.String())
	}

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

	restHandler := access.NewHandler(accessResolver, accessSvc)
	restReq := httptest.NewRequest(http.MethodGet, "/v1/access/resolve", nil)
	restReq = restReq.WithContext(access.ContextWithIdentity(restReq.Context(), actor))
	restRec := httptest.NewRecorder()
	restHandler.GetResolve(restRec, restReq)
	if restRec.Code != http.StatusOK {
		t.Fatalf("rest status = %d body=%s", restRec.Code, restRec.Body.String())
	}

	var restPayload access.ResolveResponse
	if err := json.Unmarshal(restRec.Body.Bytes(), &restPayload); err != nil {
		t.Fatalf("json.Unmarshal() rest error = %v", err)
	}
	rawVisibleOwners, err := json.Marshal(result["visible_owners"])
	if err != nil {
		t.Fatalf("json.Marshal() mcp visible owners error = %v", err)
	}
	var mcpVisibleOwners []access.VisibleOwner
	if err := json.Unmarshal(rawVisibleOwners, &mcpVisibleOwners); err != nil {
		t.Fatalf("json.Unmarshal() mcp visible owners error = %v", err)
	}
	if !reflect.DeepEqual(restPayload.VisibleOwners, mcpVisibleOwners) {
		t.Fatalf("REST owners = %#v, MCP owners = %#v", restPayload.VisibleOwners, mcpVisibleOwners)
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
	writeSvc := write.NewService(writeStore, ontologySvc, accessResolver, &postgres.SessionManager{}, nil)
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
	return NewHandler(mcpSvc), actor
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
	writeSvc := write.NewService(writeStore, ontologySvc, accessResolver, &postgres.SessionManager{}, nil)
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
