package ontology

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"kg-service/internal/access"
	"kg-service/internal/config"
	"kg-service/internal/platform/rediscache"
)

func TestCreateDomainRequiresTenantAdmin(t *testing.T) {
	service := newTestService(t)

	_, err := service.CreateDomain(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
		AppType:  "agent_consumer",
	}, "11111111-1111-1111-1111-111111111111", DomainCreateRequest{
		ID:   "noi_bo_hop_dong",
		Name: "Noi Bo Hop Dong",
	})
	if err == nil {
		t.Fatal("CreateDomain() error = nil, want forbidden")
	}
}

func TestGetEffectiveDomainsIncludesPlatformOwnedAndGrantedDomains(t *testing.T) {
	service := newTestService(t)

	domains, err := service.GetEffectiveDomains(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
		AppType:  "agent_consumer",
	}, "11111111-1111-1111-1111-111111111111")
	if err != nil {
		t.Fatalf("GetEffectiveDomains() error = %v", err)
	}

	if len(domains) < 3 {
		t.Fatalf("domains len = %d, want at least 3", len(domains))
	}

	foundPlatform := false
	foundGrant := false
	for _, domain := range domains {
		if domain.ID == "sample-registry" {
			foundPlatform = true
		}
		if domain.ID == "shared-domain" {
			foundGrant = true
		}
	}
	if !foundPlatform {
		t.Fatal("expected platform domain in effective ontology")
	}
	if !foundGrant {
		t.Fatal("expected grant-derived domain in effective ontology")
	}
}

func TestGetEffectiveDomainsExcludesUnsharedForeignDomains(t *testing.T) {
	service := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}
	platformAdmin := access.Identity{
		TenantID: access.PlatformTenantID,
		AppID:    "00000000-admin-0000-admin-000000000000",
		AppType:  "admin_tool",
	}

	_, err := service.CreateDomain(platformAdmin, "22222222-2222-2222-2222-222222222222", DomainCreateRequest{
		ID:         "private-beta-domain",
		Name:       "Private Beta",
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}

	domains, err := service.GetEffectiveDomains(actor, actor.TenantID)
	if err != nil {
		t.Fatalf("GetEffectiveDomains() error = %v", err)
	}
	for _, domain := range domains {
		if domain.ID == "private-beta-domain" {
			t.Fatalf("unexpected foreign domain in effective ontology: %#v", domains)
		}
	}
}

func TestCreateNodeTypeRegistersSchema(t *testing.T) {
	service := newTestService(t)

	schema, err := service.CreateNodeType(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", NodeTypeCreateRequest{
		NodeTypeName: "PhuLucHopDong",
		RequiredProps: []PropertySchema{
			{Name: "ten", Type: "string"},
		},
	})
	if err != nil {
		t.Fatalf("CreateNodeType() error = %v", err)
	}
	if schema.ID != "noi_bo_hop_dong.PhuLucHopDong" {
		t.Fatalf("schema id = %q", schema.ID)
	}
}

func TestGetEffectiveHandlerReturnsDomains(t *testing.T) {
	service := newTestService(t)
	handler := NewHandler(service)

	request := httptest.NewRequest(http.MethodGet, "/v1/tenants/11111111-1111-1111-1111-111111111111/ontology/effective", nil)
	request.SetPathValue("tenant_id", "11111111-1111-1111-1111-111111111111")
	request = request.WithContext(access.ContextWithIdentity(request.Context(), access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-aaaa-1111-aaaa-111111111111",
		AppType:  "agent_consumer",
	}))
	recorder := httptest.NewRecorder()

	handler.GetEffective(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"domains"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()

	cache, err := rediscache.New(config.RedisConfig{
		Host: "127.0.0.1",
		Port: 6379,
		DB:   0,
	})
	if err != nil {
		t.Fatalf("rediscache.New() error = %v", err)
	}

	accessStore := access.NewMemoryStore()
	accessStore.Seed(access.SeedTenants(), access.SeedApps(), access.SeedGrants())
	accessResolver := access.NewAccessResolver(accessStore, accessStore, &cache)

	store := NewMemoryStore()
	store.Seed(SeedDomains(), SeedVersions(), SeedNodeTypes(), SeedRelTypes(), SeedCrossDomainRules(), SeedQueryTemplates(), SeedStatusFieldConfigs())
	service := NewService(store, accessResolver)

	return service
}

func TestCreateQueryTemplateRejectsRawCypher(t *testing.T) {
	service := newTestService(t)

	_, err := service.CreateQueryTemplate(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", QueryTemplateCreateRequest{
		TemplateName: "unsafe",
		PatternSpec: map[string]any{
			"cypher": "MATCH (n) RETURN n",
		},
		ReturnFields: []string{"n.id"},
	})
	if err == nil {
		t.Fatal("CreateQueryTemplate() error = nil, want validation failure")
	}
}

func TestCreateQueryTemplateRejectsTooDeepTraversal(t *testing.T) {
	service := newTestService(t)
	hops := make([]any, 0, 6)
	for i := 0; i < 6; i++ {
		hops = append(hops, map[string]any{
			"rel_type":     "THAM_CHIEU",
			"to_node_type": "KhoanMau",
		})
	}

	_, err := service.CreateQueryTemplate(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", QueryTemplateCreateRequest{
		TemplateName: "too_deep",
		PatternSpec: map[string]any{
			"start": map[string]any{"node_type": "HopDongMau"},
			"hops":  hops,
		},
		ReturnFields: []string{"HopDongMau.ten"},
	})
	if err == nil {
		t.Fatal("CreateQueryTemplate() error = nil, want validation failure")
	}
}

func TestCreateAndActivateQueryTemplate(t *testing.T) {
	service := newTestService(t)

	template, err := service.CreateQueryTemplate(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", QueryTemplateCreateRequest{
		TemplateName: "contract_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "HopDongMau",
			},
			"hops": []any{
				map[string]any{
					"rel_type":     "THAM_CHIEU",
					"to_node_type": "KhoanMau",
				},
			},
		},
		ParamSchema: []ParameterSchema{
			{Name: "ma_hop_dong", Type: "string", Required: true},
		},
		ReturnFields: []string{"HopDongMau.ten"},
	})
	if err != nil {
		t.Fatalf("CreateQueryTemplate() error = %v", err)
	}
	if template.Status != "draft" {
		t.Fatalf("template status = %q", template.Status)
	}

	activated, err := service.ActivateQueryTemplate(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", "contract_lookup")
	if err != nil {
		t.Fatalf("ActivateQueryTemplate() error = %v", err)
	}
	if activated.Status != "active" {
		t.Fatalf("activated status = %q", activated.Status)
	}
}

func TestSampleDomainCanBeOnboardedThroughOntologyApis(t *testing.T) {
	service := newTestService(t)
	actor := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	domain, err := service.CreateDomain(actor, actor.TenantID, DomainCreateRequest{
		ID:         "sample_finance",
		Name:       "Sample Finance",
		Visibility: "public",
		Status:     "active",
	})
	if err != nil {
		t.Fatalf("CreateDomain() error = %v", err)
	}
	if domain.ID != "sample_finance" {
		t.Fatalf("domain id = %q", domain.ID)
	}
	if _, err := service.CreateNodeType(actor, actor.TenantID, "sample_finance", NodeTypeCreateRequest{
		NodeTypeName: "Invoice",
		RequiredProps: []PropertySchema{
			{Name: "invoice_no", Type: "string"},
		},
	}); err != nil {
		t.Fatalf("CreateNodeType() error = %v", err)
	}
	if _, err := service.CreateQueryTemplate(actor, actor.TenantID, "sample_finance", QueryTemplateCreateRequest{
		TemplateName: "invoice_lookup",
		PatternSpec: map[string]any{
			"start": map[string]any{
				"node_type": "Invoice",
				"match": map[string]any{
					"invoice_no": "$invoice_no",
				},
			},
		},
		ParamSchema:  []ParameterSchema{{Name: "invoice_no", Type: "string", Required: true}},
		ReturnFields: []string{"Invoice.invoice_no"},
	}); err != nil {
		t.Fatalf("CreateQueryTemplate() error = %v", err)
	}
	if _, err := service.UpsertStatusFieldConfig(actor, actor.TenantID, "sample_finance", StatusFieldConfigRequest{
		StatusFieldName:   "invoice_status",
		ValidStatusValues: []string{"open", "paid"},
	}); err != nil {
		t.Fatalf("UpsertStatusFieldConfig() error = %v", err)
	}

	details, err := service.GetDomainDetails(actor, "sample_finance")
	if err != nil {
		t.Fatalf("GetDomainDetails() error = %v", err)
	}
	if details.Domain.ID != "sample_finance" {
		t.Fatalf("domain details id = %q", details.Domain.ID)
	}
	if len(details.NodeTypes) != 1 || details.NodeTypes[0].NodeTypeName != "Invoice" {
		t.Fatalf("node types = %#v", details.NodeTypes)
	}
	if len(details.QueryTemplates) != 1 || details.QueryTemplates[0].TemplateName != "invoice_lookup" {
		t.Fatalf("query templates = %#v", details.QueryTemplates)
	}
	if details.StatusFieldConfig == nil || details.StatusFieldConfig.StatusFieldName != "invoice_status" {
		t.Fatalf("status field config = %#v", details.StatusFieldConfig)
	}
}

func TestSeedQueryTemplatesIncludesFiveSampleTemplates(t *testing.T) {
	templates := SeedQueryTemplates()
	if len(templates) != 5 {
		t.Fatalf("templates len = %d, want 5", len(templates))
	}
	want := map[string]bool{
		"action-guide":       false,
		"topic-routing":      false,
		"reference-check":    false,
		"obligation-summary": false,
		"schedule-trace":     false,
	}
	for _, template := range templates {
		if template.Status != "active" {
			t.Fatalf("template %s status = %q, want active", template.TemplateName, template.Status)
		}
		if _, ok := want[template.TemplateName]; ok {
			want[template.TemplateName] = true
		}
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("template %q missing from seed", name)
		}
	}
}

func TestUpsertStatusFieldConfig(t *testing.T) {
	service := newTestService(t)

	config, err := service.UpsertStatusFieldConfig(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "noi_bo_hop_dong", StatusFieldConfigRequest{
		StatusFieldName:   "tinh_trang",
		ValidStatusValues: []string{"hieu_luc"},
		CascadeRules: []CascadeRule{
			{FromNodeType: "HopDongMau", ViaRel: "THAM_CHIEU", ToNodeType: "KhoanMau"},
		},
	})
	if err != nil {
		t.Fatalf("UpsertStatusFieldConfig() error = %v", err)
	}
	if config.StatusFieldName != "tinh_trang" {
		t.Fatalf("status_field_name = %q", config.StatusFieldName)
	}
}

func TestResolveCrossDomainRulesFiltersBySourceNodeType(t *testing.T) {
	service := newTestService(t)

	rules := service.ResolveCrossDomainRules("noi_bo_hop_dong", "HopDongMau")
	if len(rules) != 1 {
		t.Fatalf("rules len = %d, want 1", len(rules))
	}

	other := service.ResolveCrossDomainRules("noi_bo_hop_dong", "KhoanMau")
	if len(other) != 0 {
		t.Fatalf("other rules len = %d, want 0", len(other))
	}
}

func TestCreateNodeTypeRejectsUnknownDomain(t *testing.T) {
	service := newTestService(t)

	_, err := service.CreateNodeType(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", "unknown-domain", NodeTypeCreateRequest{
		NodeTypeName:  "PhuLucHopDong",
		RequiredProps: []PropertySchema{{Name: "ten", Type: "string"}},
	})
	if err == nil {
		t.Fatal("CreateNodeType() error = nil, want not found")
	}
}

func TestValidateCrossDomainTargetRejectsMismatchedDomainAndType(t *testing.T) {
	service := newTestService(t)
	rule := service.ResolveCrossDomainRules("noi_bo_hop_dong", "HopDongMau")[0]

	if err := service.ValidateCrossDomainTarget(rule, "shared-domain", "PhuLucHopDong"); err == nil {
		t.Fatal("ValidateCrossDomainTarget() error = nil, want domain validation failure")
	}
	if err := service.ValidateCrossDomainTarget(rule, "noi_bo_hop_dong", "KhoanMau"); err == nil {
		t.Fatal("ValidateCrossDomainTarget() error = nil, want node-type validation failure")
	}
}

func TestValidateCrossDomainTargetRejectsMissingBridgeTarget(t *testing.T) {
	service := newTestService(t)
	rule := service.ResolveCrossDomainRules("noi_bo_hop_dong", "HopDongMau")[0]

	if err := service.ValidateCrossDomainTarget(rule, "", "PhuLucHopDong"); err == nil {
		t.Fatal("ValidateCrossDomainTarget() error = nil, want missing domain validation failure")
	}
	if err := service.ValidateCrossDomainTarget(rule, "noi_bo_hop_dong", ""); err == nil {
		t.Fatal("ValidateCrossDomainTarget() error = nil, want missing node-type validation failure")
	}
}

// A rejected search profile must say why. Every validation failure here used to be a plain
// errors.New, which writeError maps to 500 "Internal server error" — so an operator naming a field
// that is not selectable saw nothing actionable, and the one piece of information that would have
// resolved it (which fields ARE selectable) was thrown away. CreateNodeType has always returned
// VALIDATION_FAILED for the same class of mistake; this path had drifted.
func TestSearchProfileValidationIsAValidationError(t *testing.T) {
	service := newTestService(t)
	admin := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	_, err := service.UpsertSearchProfile(admin, admin.TenantID, "noi_bo_hop_dong", SearchProfile{
		SemanticFields: []IndexedField{{FieldName: "no_such_field", Weight: 1}},
		FTSLanguage:    "simple",
	})
	if err == nil {
		t.Fatal("UpsertSearchProfile() = nil, want a rejection for an undeclared field")
	}
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("error = %v; it must satisfy errors.Is(err, ErrValidation) or the handler answers 500", err)
	}
	if !strings.Contains(err.Error(), "no_such_field") {
		t.Fatalf("error = %v, want it to name the offending field", err)
	}
}

func TestSearchProfileWeightBoundsAreValidationErrors(t *testing.T) {
	service := newTestService(t)
	admin := access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}

	for name, weight := range map[string]float64{"too small": 0.01, "too large": 99} {
		_, err := service.UpsertSearchProfile(admin, admin.TenantID, "noi_bo_hop_dong", SearchProfile{
			SemanticFields: []IndexedField{{FieldName: "node_type", Weight: weight}},
			FTSLanguage:    "simple",
		})
		if !errors.Is(err, ErrValidation) {
			t.Errorf("%s: error = %v, want a validation error", name, err)
		}
	}
}
