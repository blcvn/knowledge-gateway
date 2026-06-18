package bootstrap

import (
	"log"
	"strings"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
)

func bootstrapSampleOntology(service *ontology.Service, actor access.Identity) {
	mustCreateDomain(service, actor, "sample-registry", "Sample Registry", access.PlatformTenantID)
	mustCreateDomain(service, actor, "sample-policy", "Sample Policy", access.PlatformTenantID)
	mustCreateDomain(service, actor, "shared-domain", "Shared Domain", "22222222-2222-2222-2222-222222222222")

	for _, schema := range ontology.SeedNodeTypes() {
		mustCreateNodeType(service, actor, schema.DomainID, schema)
	}
	for _, schema := range ontology.SeedRelTypes() {
		mustCreateRelType(service, actor, schema.DomainID, schema)
	}
	for _, template := range ontology.SeedQueryTemplates() {
		mustCreateQueryTemplate(service, actor, template.DomainID, template)
	}
	for _, cfg := range ontology.SeedStatusFieldConfigs() {
		mustUpsertStatusFieldConfig(service, actor, cfg.DomainID, cfg)
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
		log.Printf("bootstrap domain %s: %v", id, err)
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
		log.Printf("bootstrap node type %s.%s: %v", schema.DomainID, schema.NodeTypeName, err)
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
		log.Printf("bootstrap rel type %s.%s: %v", schema.DomainID, schema.RelTypeName, err)
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
		log.Printf("bootstrap template %s.%s: %v", template.DomainID, template.TemplateName, err)
		return
	}
	if _, err := service.ActivateQueryTemplate(actor, tenantID, template.DomainID, template.TemplateName); err != nil {
		log.Printf("bootstrap activate template %s.%s: %v", template.DomainID, template.TemplateName, err)
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
		log.Printf("bootstrap status config %s: %v", cfg.DomainID, err)
	}
}

func mustSeedQueryStrategy(service *ontology.Service, strategy ontology.QueryStrategy) {
	if _, err := service.UpsertQueryStrategy(access.Identity{TenantID: access.PlatformTenantID, AppType: "admin_tool"}, access.PlatformTenantID, strategy); err != nil {
		if !strings.Contains(err.Error(), "forbidden") {
			log.Printf("bootstrap query strategy %s: %v", strategy.Key, err)
		}
	}
}
