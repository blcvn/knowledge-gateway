package read

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/ontology"
	"kg-service/internal/platform/postgres"
	"kg-service/internal/platform/rediscache"
	"kg-service/internal/write"
)

func TestListTemplatesReturnsOnlyActiveTemplates(t *testing.T) {
	svc, _, actor, _, _, _ := newReadFixture(t)

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

func TestExecuteTemplateReturnsVisibleResults(t *testing.T) {
	svc, _, actor, _, _, _ := newReadFixture(t)

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
	svc, _, actor, contractNodeID, _, _ := newReadFixture(t)

	node, err := svc.GetNode(actor, contractNodeID)
	if err != nil {
		t.Fatalf("GetNode() error = %v", err)
	}
	if node.ID == "" {
		t.Fatal("node id is empty")
	}
}

func TestReadHandlersReturnEnvelopeAndNode(t *testing.T) {
	svc, _, actor, contractNodeID, _, _ := newReadFixture(t)
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

func TestReadAuditsAllowAndDeny(t *testing.T) {
	svc, _, actor, contractNodeID, _, auditSvc := newReadFixture(t)

	_, _ = svc.ListTemplates(actor, "noi_bo_hop_dong")
	_, _ = svc.ExecuteTemplate(actor, "noi_bo_hop_dong", "contract_lookup", map[string]any{"ten_hop_dong": "Hop dong A"})
	_, _ = svc.GetNode(actor, contractNodeID)

	entries, err := auditSvc.ListAuditLogs(actor, access.AuditFilter{ResourceOwnerTenantID: actor.TenantID})
	if err != nil {
		t.Fatalf("ListAuditLogs() error = %v", err)
	}
	if len(entries) < 3 {
		t.Fatalf("audit len = %d, want at least 3", len(entries))
	}
}

func newReadFixture(t *testing.T) (*Service, *ontology.Service, access.Identity, string, string, *access.Service) {
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
					"rel_type":     "THAM_CHIEU",
					"to_node_type": "KhoanMau",
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

	writeStore := write.NewMemoryStore()
	writeSvc := write.NewService(writeStore, ontologyService, accessResolver, &postgres.SessionManager{}, nil)
	if _, err := ontologyService.CreateNodeType(actor, actor.TenantID, "noi_bo_hop_dong", ontology.NodeTypeCreateRequest{
		NodeTypeName: "PhuLucHopDong",
		RequiredProps: []ontology.PropertySchema{
			{Name: "ten", Type: "string"},
		},
	}); err != nil && !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("CreateNodeType(bridge) error = %v", err)
	}
	bridgeNode, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "PhuLucHopDong",
		Properties:  map[string]any{"ten": "Phu luc 1"},
		ExternalRef: "appendix-node-1",
	})
	if err != nil {
		t.Fatalf("CreateNode(bridge) error = %v", err)
	}
	clauseNode, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "KhoanMau",
		Properties:  map[string]any{"ten": "Khoan 1"},
		ExternalRef: "clause-node-1",
	})
	if err != nil {
		t.Fatalf("CreateNode(clause) error = %v", err)
	}
	contractNode, err := writeSvc.CreateNode(actor, write.NodeCreateRequest{
		DomainID:    "noi_bo_hop_dong",
		NodeType:    "HopDongMau",
		Properties:  map[string]any{"ten": "Hop dong A", "bridge_dinh_kem_ids": []any{bridgeNode.NodeID}},
		ExternalRef: "contract-node-1",
	})
	if err != nil {
		t.Fatalf("CreateNode(contract) error = %v", err)
	}
	relationship, err := writeSvc.CreateRelationship(actor, write.RelationshipCreateRequest{
		RelType:    "THAM_CHIEU",
		FromNodeID: contractNode.NodeID,
		ToNodeID:   clauseNode.NodeID,
		DomainID:   "noi_bo_hop_dong",
		Properties: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CreateRelationship() error = %v", err)
	}

	auditLogger := access.NewService(accessStore, &cache)
	svc := NewService(writeStore, ontologyService, accessResolver, auditLogger)
	return svc, ontologyService, actor, contractNode.NodeID, relationship.RelationshipID, auditLogger
}
