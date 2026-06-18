package read

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/write"
)

func TestListTemplatesReturnsOnlyActiveTemplates(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	templates, err := svc.ListTemplates(actor, "noi_bo_hop_dong")
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("templates len = %d, want 1", len(templates))
	}
	if templates[0].TemplateName != "contract_lookup" {
		t.Fatalf("template_name = %q", templates[0].TemplateName)
	}
}

func TestSampleSeedTemplatesExecuteThroughGenericRoute(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	templates, err := svc.ListTemplates(actor, "sample-policy")
	if err != nil {
		t.Fatalf("ListTemplates() error = %v", err)
	}
	if len(templates) != 5 {
		t.Fatalf("templates len = %d, want 5", len(templates))
	}

	cases := map[string]map[string]any{
		"action-guide":       {"topic_key": "returns"},
		"topic-routing":      {"topic_key": "returns"},
		"reference-check":    {"record_key": "record-101"},
		"obligation-summary": {"obligation_key": "obligation-7"},
		"schedule-trace":     {"schedule_key": "schedule-9"},
	}
	for _, template := range templates {
		params := cases[template.TemplateName]
		if params == nil {
			t.Fatalf("missing params for template %q", template.TemplateName)
		}
		resp, err := svc.ExecuteTemplate(actor, "sample-policy", template.TemplateName, params)
		if err != nil {
			t.Fatalf("ExecuteTemplate(%s) error = %v", template.TemplateName, err)
		}
		if resp.Results == nil {
			t.Fatalf("ExecuteTemplate(%s) results = nil", template.TemplateName)
		}
	}
}

func TestExecuteTemplateReturnsVisibleResults(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	resp, err := svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": "Hop dong A"})
	if err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(resp.Results))
	}
	if got := resp.Results[0]["HopDongMau.ten"]; got != "Hop dong A" {
		t.Fatalf("result field = %v", got)
	}
}

func TestGetNodeRespectsVisibility(t *testing.T) {
	svc, _, actor, contractNodeID, _, _, _ := newReadFixture(t)

	node, err := svc.GetNode(actor, contractNodeID)
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}
	if node.ID == "" {
		t.Fatal("node id is empty")
	}
}

func TestReadHandlersReturnEnvelopeAndNode(t *testing.T) {
	svc, _, actor, contractNodeID, _, _, _ := newReadFixture(t)
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/kg/read/templates?domain_id=noi_bo_hop_dong", nil)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()
	handler.ListTemplates(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ListTemplates() status = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/v1/kg/read/template/noi_bo_hop_dong/contract_lookup", strings.NewReader(`{"params":{"ten_hop_dong":"Hop dong A"}}`))
	req.SetPathValue("domain_id", "noi_bo_hop_dong")
	req.SetPathValue("template_name", "contract_lookup")
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec = httptest.NewRecorder()
	handler.ExecuteTemplate(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ExecuteTemplate() status = %d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/kg/read/nodes/"+contractNodeID, nil)
	req.SetPathValue("id", contractNodeID)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec = httptest.NewRecorder()
	handler.GetNode(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GetNode() status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReadTemplateRejectsRawCypherNameAtHTTPBoundary(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/kg/read/template/noi_bo_hop_dong/raw-cypher-query", strings.NewReader(`{"params":{"ten_hop_dong":"Hop dong A"}}`))
	req.SetPathValue("domain_id", "noi_bo_hop_dong")
	req.SetPathValue("template_name", "raw-cypher-query")
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()

	handler.ExecuteTemplate(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestReadListTemplatesReturnsStandardListEnvelope(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/kg/read/templates?domain_id=noi_bo_hop_dong&limit=1", nil)
	req = req.WithContext(access.ContextWithIdentity(req.Context(), actor))
	rec := httptest.NewRecorder()
	handler.ListTemplates(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("ListTemplates() status = %d body=%s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := payload["data"].([]any); !ok {
		t.Fatalf("payload = %#v", payload)
	}
	if _, ok := payload["has_more"].(bool); !ok {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestReadAuditsAllowAndDeny(t *testing.T) {
	svc, _, actor, contractNodeID, _, auditSvc, _ := newReadFixture(t)

	_, _ = svc.ListTemplates(actor, "noi_bo_hop_dong")
	_, _ = svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": "Hop dong A"})
	_, _ = svc.GetNode(actor, contractNodeID)

	entries, err := auditSvc.ListAuditLogs(actor, access.AuditListFilter{ResourceOwnerTenantID: actor.TenantID})
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}
	if len(entries) < 3 {
		t.Fatalf("audit len = %d, want at least 3", len(entries))
	}
}

func TestReadTemplateRejectsInvisibleStartNode(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	resp, err := svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": "Hop dong Bi Mat"})
	if err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("results len = %d, want 0", len(resp.Results))
	}
}

func TestReadTemplateRejectsInvisibleHopNode(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	resp, err := svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": "Hop dong Hidden Hop"})
	if err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	if len(resp.Results) != 0 {
		t.Fatalf("results len = %d, want 0", len(resp.Results))
	}
}

func TestReadTemplateValidatesParams(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	_, err := svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{})
	if err == nil {
		t.Fatal("ExecuteTemplate() error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Fatalf("ExecuteTemplate() error = %v, want required param failure", err)
	}

	_, err = svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": 123})
	if err == nil {
		t.Fatal("ExecuteTemplate() type error = nil, want validation failure")
	}
	if !strings.Contains(err.Error(), "must be a string") {
		t.Fatalf("ExecuteTemplate() error = %v, want type validation failure", err)
	}
}

func TestReadTemplateRejectsInactiveTemplate(t *testing.T) {
	svc, _, actor, _, _, _, _ := newReadFixture(t)

	_, err := svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "draft_only", map[string]any{})
	if err == nil {
		t.Fatal("ExecuteTemplate() error = nil, want inactive template rejection")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("ExecuteTemplate() error = %v, want not found", err)
	}
}

func TestReadTemplateIgnoresMissingLifecycleConfig(t *testing.T) {
	svc, ontologySvc, actor, _, _, _, store := newReadFixture(t)
	if _, err := ontologySvc.CreateDomain(actor, actor.TenantID, ontology.DomainCreateRequest{
		ID:         "no_status_domain",
		Name:       "No Status Domain",
		Visibility: "public",
		Status:     "active",
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologySvc.CreateNodeType(actor, actor.TenantID, "no_status_domain", ontology.NodeTypeCreateRequest{
		NodeTypeName: "NoStatusDoc",
		RequiredProps: []ontology.PropertySchema{
			{Name: "summary", Type: "string"},
		},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType() error = %v", err)
	}

	seedVisibleNode(t, store, write.NodeRecord{
		ID:            "no-status-node",
		NodeType:      "NoStatusDoc",
		DomainID:      "no_status_domain",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"summary": "No lifecycle config"},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 11, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 11, 0, 0, 0, time.UTC),
	})

	if _, err := ontologySvc.CreateQueryTemplate(actor, actor.TenantID, "no_status_domain", ontology.QueryTemplateCreateRequest{
		TemplateName: "no_status_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "NoStatusDoc",
				"match": map[string]any{
					"summary": "No lifecycle config",
				},
			},
		},
		ReturnFields: []string{"NoStatusDoc.summary"},
	}); err != nil {
		t.Fatalf("CreateQueryTemplate() error = %v", err)
	}
	if _, err := ontologySvc.ActivateQueryTemplate(actor, actor.TenantID, "no_status_domain", "no_status_lookup"); err != nil {
		t.Fatalf("ActivateQueryTemplate() error = %v", err)
	}

	resp, err := svc.ExecuteTemplate(actor, "no_status_domain", "no_status_lookup", map[string]any{})
	if err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results len = %d, want 1", len(resp.Results))
	}
}

func TestExecuteTemplateEnforcesMaxRows(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	svc.maxRows = 1

	seedVisibleNode(t, store, write.NodeRecord{
		ID:            "maxrows-contract-2",
		NodeType:      "HopDongMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 42, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 42, 0, 0, time.UTC),
	})
	seedVisibleRelationship(t, store, write.RelationshipRecord{
		ID:            "maxrows-rel-2",
		RelType:       "THAM_CHIEU",
		FromNodeID:    "maxrows-contract-2",
		ToNodeID:      "clause-node-1",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		Properties:    map[string]any{},
		CreatedAt:     time.Date(2026, 6, 17, 10, 42, 30, 0, time.UTC),
	})

	resp, err := svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": "Hop dong A"})
	if err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results len = %d, want 1 due to max_rows guard", len(resp.Results))
	}
}

func TestExecuteTemplateTimesOutWhenClockAdvancesPastLimit(t *testing.T) {
	svc, _, actor, _, _, _, store := newReadFixture(t)
	svc.queryTimeout = 2 * time.Microsecond
	ticks := []time.Time{
		time.Date(2026, 6, 17, 11, 30, 0, 0, time.UTC),
		time.Date(2026, 6, 17, 11, 30, 0, 1_000, time.UTC),
		time.Date(2026, 6, 17, 11, 30, 0, 2_000, time.UTC),
		time.Date(2026, 6, 17, 11, 30, 0, 3_000, time.UTC),
		time.Date(2026, 6, 17, 11, 30, 0, 4_000, time.UTC),
	}
	idx := 0
	svc.now = func() time.Time {
		if idx >= len(ticks) {
			return ticks[len(ticks)-1]
		}
		t := ticks[idx]
		idx++
		return t
	}

	seedVisibleNode(t, store, write.NodeRecord{
		ID:            "timeout-contract-2",
		NodeType:      "HopDongMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 43, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 43, 0, 0, time.UTC),
	})
	seedVisibleRelationship(t, store, write.RelationshipRecord{
		ID:            "timeout-rel-2",
		RelType:       "THAM_CHIEU",
		FromNodeID:    "timeout-contract-2",
		ToNodeID:      "clause-node-1",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		Properties:    map[string]any{},
		CreatedAt:     time.Date(2026, 6, 17, 10, 43, 30, 0, time.UTC),
	})
	seedVisibleNode(t, store, write.NodeRecord{
		ID:            "timeout-contract-3",
		NodeType:      "HopDongMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{"appendix-1"}},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 44, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 44, 0, 0, time.UTC),
	})
	seedVisibleRelationship(t, store, write.RelationshipRecord{
		ID:            "timeout-rel-3",
		RelType:       "THAM_CHIEU",
		FromNodeID:    "timeout-contract-3",
		ToNodeID:      "clause-node-1",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		Properties:    map[string]any{},
		CreatedAt:     time.Date(2026, 6, 17, 10, 44, 30, 0, time.UTC),
	})

	_, err := svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": "Hop dong A"})
	if err == nil {
		t.Fatal("ExecuteTemplate() error = nil, want timeout")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("ExecuteTemplate() error = %v, want timeout", err)
	}
}

func newReadFixture(t *testing.T) (*Service, *ontology.Service, access.Identity, string, string, *access.Service, *write.MemoryStore) {
	t.Helper()

	cache, err := rediscache.New(config.RedisConfig{Host: "127.0.0.1", Port: 6379, DB: 0})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}
	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)

	ontologyStore := ontology.NewMemoryStore()
	ontologyStore.Seed(ontology.SeedDomains(), ontology.SeedVersions(), ontology.SeedNodeTypes(), ontology.SeedRelTypes(), ontology.SeedCrossDomainRules(), ontology.SeedQueryTemplates(), ontology.SeedStatusFieldConfigs())
	ontologyService := ontology.NewService(ontologyStore, accessResolver)

	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	if _, err := ontologyService.CreateDomain(actor, actor.TenantID, ontology.DomainCreateRequest{
		ID:   "noi_bo_hop_dong",
		Name: "Noi Bo Hop Dong",
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if _, err := ontologyService.CreateQueryTemplate(actor, actor.TenantID, "noi_bo_hop_dong", ontology.QueryTemplateCreateRequest{
		TemplateName: "contract_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "HopDongMau",
				"match": map[string]any{
					"ten": "$ten_hop_dong",
				},
			},
			"hops": []any{
				map[string]any{
					"rel_type":      "THAM_CHIEU",
					"to_node_type":  "KhoanMau",
					"filter_status": "valid_only",
				},
			},
		},
		ParamSchema: []ontology.ParameterSchema{
			{Name: "ten_hop_dong", Type: "string", Required: true},
		},
		ReturnFields: []string{"HopDongMau.ten", "KhoanMau.ten"},
	}); err != nil {
		t.Fatalf("CreateQueryTemplate() error = %v", err)
	}
	if _, err := ontologyService.ActivateQueryTemplate(actor, actor.TenantID, "noi_bo_hop_dong", "contract_lookup"); err != nil {
		t.Fatalf("ActivateQueryTemplate() error = %v", err)
	}
	if _, err := ontologyService.CreateQueryTemplate(actor, actor.TenantID, "noi_bo_hop_dong", ontology.QueryTemplateCreateRequest{
		TemplateName: "draft_only",
		PatternSpec: map[string]any{
			"start": map[string]any{"node_type": "HopDongMau"},
		},
		ReturnFields: []string{"HopDongMau.ten"},
	}); err != nil {
		t.Fatalf("CreateQueryTemplate(draft) error = %v", err)
	}

	if _, err := ontologyService.CreateNodeType(actor, actor.TenantID, "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName: "PhuLucHopDong",
		RequiredProps: []ontology.PropertySchema{
			{Name: "ten", Type: "string"},
		},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(bridge) error = %v", err)
	}
	writeStore := write.NewMemoryStore()
	_ = seedVisibleNode(t, writeStore, write.NodeRecord{
		ID:            "bridge-node-1",
		NodeType:      "PhuLucHopDong",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Phu luc 1"},
		ExternalRef:   "appendix-node-1",
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 31, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 31, 0, 0, time.UTC),
	})
	clauseNode := seedVisibleNode(t, writeStore, write.NodeRecord{
		ID:            "clause-node-1",
		NodeType:      "KhoanMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Khoan 1"},
		ExternalRef:   "clause-node-1",
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 32, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 32, 0, 0, time.UTC),
	})
	contractNode := seedVisibleNode(t, writeStore, write.NodeRecord{
		ID:            "contract-node-1",
		NodeType:      "HopDongMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Hop dong A"},
		ExternalRef:   "contract-node-1",
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 33, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 33, 0, 0, time.UTC),
	})
	relationship := write.RelationshipRecord{
		ID:            "rel-1",
		RelType:       "THAM_CHIEU",
		FromNodeID:    contractNode.ID,
		ToNodeID:      clauseNode.ID,
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		Properties:    map[string]any{},
		CreatedAt:     time.Date(2026, 6, 17, 10, 34, 0, 0, time.UTC),
	}
	seedVisibleRelationship(t, writeStore, relationship)

	hiddenClause := seedVisibleNode(t, writeStore, write.NodeRecord{
		ID:            "hidden-clause-node",
		NodeType:      "KhoanMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: "22222222-2222-2222-2222-222222222222",
		OwnerAppID:    "22222222-aaaa-2222-aaaa-222222222222",
		ACLVisibleTo:  []string{"22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222"},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Khoan Bi Mat"},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 40, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 40, 0, 0, time.UTC),
	})
	seedVisibleRelationship(t, writeStore, write.RelationshipRecord{
		ID:            "hidden-hop-rel",
		RelType:       "THAM_CHIEU",
		FromNodeID:    "hidden-hop-contract-node",
		ToNodeID:      hiddenClause.ID,
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		Properties:    map[string]any{},
		CreatedAt:     time.Date(2026, 6, 17, 10, 41, 0, 0, time.UTC),
	})
	seedVisibleNode(t, writeStore, write.NodeRecord{
		ID:            "hidden-hop-contract-node",
		NodeType:      "HopDongMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: actor.TenantID,
		OwnerAppID:    actor.AppID,
		ACLVisibleTo:  []string{actor.TenantID + ":" + actor.AppID},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Hop dong Hidden Hop"},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 39, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 39, 0, 0, time.UTC),
	})
	seedVisibleNode(t, writeStore, write.NodeRecord{
		ID:            "invisible-contract-node",
		NodeType:      "HopDongMau",
		DomainID:      "noi_bo_hop_dong",
		OwnerTenantID: "22222222-2222-2222-2222-222222222222",
		OwnerAppID:    "22222222-aaaa-2222-aaaa-222222222222",
		ACLVisibleTo:  []string{"22222222-2222-2222-2222-222222222222:22222222-aaaa-2222-aaaa-222222222222"},
		Visibility:    "private",
		Properties:    map[string]any{"ten": "Hop dong Bi Mat"},
		DomainVersion: 1,
		CreatedAt:     time.Date(2026, 6, 17, 10, 38, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2026, 6, 17, 10, 38, 0, 0, time.UTC),
	})

	auditLogger := access.NewService(accessStore, &cache)
	svc := NewService(writeStore, ontologyService, accessResolver, auditLogger)
	return svc, ontologyService, actor, contractNode.ID, relationship.ID, auditLogger, writeStore
}

func seedVisibleNode(t *testing.T, store *write.MemoryStore, node write.NodeRecord) write.NodeRecord {
	t.Helper()
	if err := store.CreateNodeWithOutbox(node, write.OutboxEvent{
		ID:            "evt-" + node.ID,
		AggregateType: "kg_node",
		AggregateID:   node.ID,
		EventType:     "NODE_UPSERTED",
		Payload: map[string]any{
			"node_id":   node.ID,
			"domain_id": node.DomainID,
		},
		Status:    "PENDING",
		CreatedAt: node.CreatedAt,
	}); err != nil {
		t.Fatalf("CreateNodeWithOutbox(%s) error = %v", node.ID, err)
	}
	return node
}

func seedVisibleRelationship(t *testing.T, store *write.MemoryStore, rel write.RelationshipRecord) {
	t.Helper()
	if err := store.CreateRelationshipWithOutbox(rel, write.OutboxEvent{
		ID:            "evt-" + rel.ID,
		AggregateType: "kg_relationship",
		AggregateID:   rel.ID,
		EventType:     "REL_UPSERTED",
		Payload: map[string]any{
			"relationship_id": rel.ID,
			"domain_id":       rel.DomainID,
		},
		Status:    "PENDING",
		CreatedAt: rel.CreatedAt,
	}); err != nil {
		t.Fatalf("CreateRelationshipWithOutbox(%s) error = %v", rel.ID, err)
	}
}
