package ontology

import (
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
		if domain.ID == "van_ban_phap_luat" {
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

	_, err = service.CreateDomain(access.Identity{
		TenantID: "11111111-1111-1111-1111-111111111111",
		AppID:    "11111111-admin-1111-admin-111111111111",
		AppType:  "admin_tool",
	}, "11111111-1111-1111-1111-111111111111", DomainCreateRequest{
		ID:   "noi_bo_hop_dong",
		Name: "Noi Bo Hop Dong",
	})
	if err != nil {
		t.Fatalf("bootstrap CreateDomain() error = %v", err)
	}

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
