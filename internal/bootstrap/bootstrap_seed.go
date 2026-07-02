package bootstrap

import (
	"errors"
	"log"
	"strings"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
)

func bootstrapSampleOntology(service *ontology.Service, actor access.Identity) {
	mustCreateDomain(service, actor, "sample-registry", "Sample Registry", access.PlatformTenantID)
	mustCreateDomain(service, actor, "sample-policy", "Sample Policy", access.PlatformTenantID)
	mustCreateDomain(service, actor, "shared-domain", "Shared Domain", "22222222-2222-2222-2222-222222222222")
	mustCreateDomain(service, actor, "noi_bo_hop_dong", "Nội bộ Hợp đồng", "11111111-1111-1111-1111-111111111111")

	domainOwners := make(map[string]string, len(ontology.SeedDomains()))
	for _, d := range ontology.SeedDomains() {
		domainOwners[d.ID] = d.OwnerTenantID
	}

	for _, schema := range ontology.SeedNodeTypes() {
		ownerTenantID := domainOwners[schema.DomainID]
		if ownerTenantID == "" {
			log.Printf("bootstrap node type %s.%s: domain %q not in SeedDomains — skipping", schema.DomainID, schema.NodeTypeName, schema.DomainID)
			continue
		}
		mustCreateNodeType(service, actor, ownerTenantID, schema)
	}
	for _, schema := range ontology.SeedRelTypes() {
		ownerTenantID := domainOwners[schema.DomainID]
		if ownerTenantID == "" {
			log.Printf("bootstrap rel type %s.%s: domain %q not in SeedDomains — skipping", schema.DomainID, schema.RelTypeName, schema.DomainID)
			continue
		}
		mustCreateRelType(service, actor, ownerTenantID, schema)
	}
	for _, template := range ontology.SeedQueryTemplates() {
		ownerTenantID := domainOwners[template.DomainID]
		if ownerTenantID == "" {
			log.Printf("bootstrap template %s.%s: domain %q not in SeedDomains — skipping", template.DomainID, template.TemplateName, template.DomainID)
			continue
		}
		mustCreateQueryTemplate(service, actor, ownerTenantID, template)
	}
	for _, cfg := range ontology.SeedStatusFieldConfigs() {
		ownerTenantID := domainOwners[cfg.DomainID]
		if ownerTenantID == "" {
			log.Printf("bootstrap status config %s: domain %q not in SeedDomains — skipping", cfg.DomainID, cfg.DomainID)
			continue
		}
		mustUpsertStatusFieldConfig(service, actor, ownerTenantID, cfg)
	}
	mustSeedQueryStrategy(service, ontology.QueryStrategy{
		Key:      "default",
		Version:  1,
		MaxDepth: 5,
		Params: map[string]any{
			"direction":     "out",
			"depth_mode":    "fixed",
			"acl_predicate": "any_hop",
		},
	})
	mustSeedQueryStrategy(service, ontology.QueryStrategy{
		Key:      "deep_traversal",
		Version:  1,
		MaxDepth: 10,
		Params: map[string]any{
			"direction":     "out",
			"depth_mode":    "variable",
			"acl_predicate": "start_only",
		},
	})
}

func mustCreateDomain(service *ontology.Service, actor access.Identity, id, name, ownerTenantID string) {
	if _, err := service.CreateDomain(actor, ownerTenantID, ontology.DomainCreateRequest{ID: id, Name: name, Status: "active", Visibility: "public"}); err != nil {
		if isAlreadyExists(err) {
			return
		}
		log.Printf("bootstrap domain %s owner=%s: %v", id, ownerTenantID, err)
	}
}

func mustCreateNodeType(service *ontology.Service, actor access.Identity, tenantID string, schema ontology.NodeTypeSchema) {
	if _, err := service.CreateNodeType(actor, tenantID, schema.DomainID, ontology.NodeTypeCreateRequest{
		NodeTypeName:    schema.NodeTypeName,
		GraphLabel:      schema.GraphLabel,
		RequiredProps:   schema.RequiredProps,
		OptionalProps:   schema.OptionalProps,
		ValidationRules: schema.ValidationRules,
	}); err != nil {
		if isAlreadyExists(err) {
			return
		}
		log.Printf("bootstrap node type %s.%s tenant=%s: %v", schema.DomainID, schema.NodeTypeName, tenantID, err)
	}
}

func mustCreateRelType(service *ontology.Service, actor access.Identity, tenantID string, schema ontology.RelTypeSchema) {
	if _, err := service.CreateRelType(actor, tenantID, schema.DomainID, ontology.RelTypeCreateRequest{
		RelTypeName:   schema.RelTypeName,
		FromNodeType:  schema.FromNodeType,
		ToNodeType:    schema.ToNodeType,
		SameDomain:    schema.SameDomain,
		RequiredProps: schema.RequiredProps,
		OptionalProps: schema.OptionalProps,
	}); err != nil {
		if isAlreadyExists(err) {
			return
		}
		log.Printf("bootstrap rel type %s.%s tenant=%s: %v", schema.DomainID, schema.RelTypeName, tenantID, err)
	}
}

func mustCreateQueryTemplate(service *ontology.Service, actor access.Identity, tenantID string, template ontology.QueryTemplate) {
	if _, err := service.CreateQueryTemplate(actor, tenantID, template.DomainID, ontology.QueryTemplateCreateRequest{
		TemplateName: template.TemplateName,
		PatternSpec:  template.PatternSpec,
		ParamSchema:  template.ParamSchema,
		ReturnFields: template.ReturnFields,
		Description:  template.Description,
	}); err != nil {
		if isAlreadyExists(err) {
			return
		}
		log.Printf("bootstrap template %s.%s tenant=%s: %v", template.DomainID, template.TemplateName, tenantID, err)
		return
	}
	if _, err := service.ActivateQueryTemplate(actor, tenantID, template.DomainID, template.TemplateName); err != nil {
		log.Printf("bootstrap activate template %s.%s tenant=%s: %v", template.DomainID, template.TemplateName, tenantID, err)
	}
}

func mustUpsertStatusFieldConfig(service *ontology.Service, actor access.Identity, tenantID string, cfg ontology.StatusFieldConfig) {
	if _, err := service.UpsertStatusFieldConfig(actor, tenantID, cfg.DomainID, ontology.StatusFieldConfigRequest{
		StatusFieldName:     cfg.StatusFieldName,
		ValidStatusValues:   cfg.ValidStatusValues,
		WarningStatusValues: cfg.WarningStatusValues,
		CascadeRules:        cfg.CascadeRules,
		AuthorityFieldName:  cfg.AuthorityFieldName,
		AuthorityValuesMap:  cfg.AuthorityValuesMap,
	}); err != nil {
		log.Printf("bootstrap status config %s tenant=%s: %v", cfg.DomainID, tenantID, err)
	}
}

func mustSeedQueryStrategy(service *ontology.Service, strategy ontology.QueryStrategy) {
	if _, err := service.UpsertQueryStrategy(access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}, access.PlatformTenantID, strategy); err != nil {
		if errors.Is(err, ontology.ErrForbidden) {
			return
		}
		log.Printf("bootstrap query strategy %s: %v", strategy.Key, err)
	}
}

// isAlreadyExists reports whether err is a "validation: ... already exists" error.
// These are safe to ignore on restart since the seed data is idempotent.
func isAlreadyExists(err error) bool {
	return errors.Is(err, ontology.ErrValidation) && strings.Contains(err.Error(), "already exists")
}
