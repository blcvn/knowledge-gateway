package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"kg-service/internal/access"
	"kg-service/internal/ontology"
)

var (
	_ access.TenantAppStore = (*Repository)(nil)
	_ ontology.DomainStore  = (*Repository)(nil)
)

func (r *Repository) GetTenant(id string) (access.Tenant, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, slug, name, status, tier, default_sharing_policy, created_at, updated_at
		FROM tenants
		WHERE id = $1
	`, id)
	return scanTenant(row)
}

func (r *Repository) CreateTenant(tenant access.Tenant) access.Tenant {
	_, _ = r.exec(context.Background(), `
		INSERT INTO tenants (
			id, slug, name, status, tier, default_sharing_policy, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			tier = EXCLUDED.tier,
			default_sharing_policy = EXCLUDED.default_sharing_policy,
			created_at = EXCLUDED.created_at,
			updated_at = EXCLUDED.updated_at
	`, tenant.ID, tenant.Slug, tenant.Name, tenant.Status, tenant.Tier, tenant.DefaultSharingPolicy, tenant.CreatedAt, tenant.UpdatedAt)
	return tenant
}

func (r *Repository) UpdateTenant(tenant access.Tenant) (access.Tenant, bool) {
	tenant.UpdatedAt = tenant.UpdatedAt.UTC()
	res, err := r.exec(context.Background(), `
		UPDATE tenants
		SET slug = $2,
			name = $3,
			status = $4,
			tier = $5,
			default_sharing_policy = $6,
			updated_at = $7
		WHERE id = $1
	`, tenant.ID, tenant.Slug, tenant.Name, tenant.Status, tenant.Tier, tenant.DefaultSharingPolicy, tenant.UpdatedAt)
	if err != nil {
		return access.Tenant{}, false
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return access.Tenant{}, false
	}
	return tenant, true
}

func (r *Repository) GetAppByAPIKeyHash(hash string) (access.App, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, tenant_id, slug, name, type, api_key_hash, api_key_prefix, status, created_at, revoked_at
		FROM apps
		WHERE api_key_hash = $1
	`, hash)
	return scanApp(row)
}

func (r *Repository) GetAppByID(id string) (access.App, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, tenant_id, slug, name, type, api_key_hash, api_key_prefix, status, created_at, revoked_at
		FROM apps
		WHERE id = $1
	`, id)
	return scanApp(row)
}

func (r *Repository) ListAppsByTenant(tenantID string) []access.App {
	rows, err := r.query(context.Background(), `
		SELECT id, tenant_id, slug, name, type, api_key_hash, api_key_prefix, status, created_at, revoked_at
		FROM apps
		WHERE tenant_id = $1
		ORDER BY created_at, id
	`, tenantID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]access.App, 0)
	for rows.Next() {
		app, ok := scanApp(rows)
		if !ok {
			continue
		}
		result = append(result, app)
	}
	return result
}

func (r *Repository) CreateApp(app access.App) access.App {
	_, _ = r.exec(context.Background(), `
		INSERT INTO apps (
			id, tenant_id, slug, name, type, api_key_hash, api_key_prefix, status, created_at, revoked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			slug = EXCLUDED.slug,
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			api_key_hash = EXCLUDED.api_key_hash,
			api_key_prefix = EXCLUDED.api_key_prefix,
			status = EXCLUDED.status,
			created_at = EXCLUDED.created_at,
			revoked_at = EXCLUDED.revoked_at
	`, app.ID, app.TenantID, app.Slug, app.Name, app.Type, app.APIKeyHash, app.APIKeyPrefix, app.Status, app.CreatedAt, app.RevokedAt)
	return app
}

func (r *Repository) UpdateApp(app access.App) (access.App, bool) {
	res, err := r.exec(context.Background(), `
		UPDATE apps
		SET tenant_id = $2,
			slug = $3,
			name = $4,
			type = $5,
			api_key_hash = $6,
			api_key_prefix = $7,
			status = $8,
			created_at = $9,
			revoked_at = $10
		WHERE id = $1
	`, app.ID, app.TenantID, app.Slug, app.Name, app.Type, app.APIKeyHash, app.APIKeyPrefix, app.Status, app.CreatedAt, app.RevokedAt)
	if err != nil {
		return access.App{}, false
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return access.App{}, false
	}
	return app, true
}

func (r *Repository) ListGrantsForGrantee(tenantID, appID string) []access.AccessGrant {
	rows, err := r.query(context.Background(), `
		SELECT id, grantor_tenant_id, grantor_app_id, grantee_tenant_id, grantee_app_id,
		       scope_type, scope_value, permission, status, expires_at, created_at, revoked_at
		FROM access_grants
		WHERE grantee_tenant_id = $1
		  AND (grantee_app_id IS NULL OR grantee_app_id = $2)
		ORDER BY created_at, id
	`, tenantID, nullString(appID))
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]access.AccessGrant, 0)
	for rows.Next() {
		grant, ok := scanGrant(rows)
		if !ok {
			continue
		}
		result = append(result, grant)
	}
	return result
}

func (r *Repository) CreateGrant(grant access.AccessGrant) access.AccessGrant {
	_, _ = r.exec(context.Background(), `
		INSERT INTO access_grants (
			id, grantor_tenant_id, grantor_app_id, grantee_tenant_id, grantee_app_id,
			scope_type, scope_value, permission, status, expires_at, created_at, revoked_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			grantor_tenant_id = EXCLUDED.grantor_tenant_id,
			grantor_app_id = EXCLUDED.grantor_app_id,
			grantee_tenant_id = EXCLUDED.grantee_tenant_id,
			grantee_app_id = EXCLUDED.grantee_app_id,
			scope_type = EXCLUDED.scope_type,
			scope_value = EXCLUDED.scope_value,
			permission = EXCLUDED.permission,
			status = EXCLUDED.status,
			expires_at = EXCLUDED.expires_at,
			created_at = EXCLUDED.created_at,
			revoked_at = EXCLUDED.revoked_at
	`, grant.ID, grant.GrantorTenantID, nullString(grant.GrantorAppID), grant.GranteeTenantID, nullString(grant.GranteeAppID), grant.ScopeType, nullString(grant.ScopeValue), grant.Permission, grant.Status, grant.ExpiresAt, grant.CreatedAt, grant.RevokedAt)
	return grant
}

func (r *Repository) ListGrants(filter access.GrantListFilter) []access.AccessGrant {
	query := `
		SELECT id, grantor_tenant_id, grantor_app_id, grantee_tenant_id, grantee_app_id,
		       scope_type, scope_value, permission, status, expires_at, created_at, revoked_at
		FROM access_grants
		WHERE 1 = 1
	`
	args := make([]any, 0, 2)
	if filter.GrantorTenantID != "" {
		query += fmt.Sprintf(" AND grantor_tenant_id = $%d", len(args)+1)
		args = append(args, filter.GrantorTenantID)
	}
	if filter.GranteeTenantID != "" {
		query += fmt.Sprintf(" AND grantee_tenant_id = $%d", len(args)+1)
		args = append(args, filter.GranteeTenantID)
	}
	query += " ORDER BY created_at, id"
	rows, err := r.query(context.Background(), query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]access.AccessGrant, 0)
	for rows.Next() {
		grant, ok := scanGrant(rows)
		if !ok {
			continue
		}
		result = append(result, grant)
	}
	return result
}

func (r *Repository) GetGrantByID(id string) (access.AccessGrant, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, grantor_tenant_id, grantor_app_id, grantee_tenant_id, grantee_app_id,
		       scope_type, scope_value, permission, status, expires_at, created_at, revoked_at
		FROM access_grants
		WHERE id = $1
	`, id)
	return scanGrant(row)
}

func (r *Repository) UpdateGrant(grant access.AccessGrant) (access.AccessGrant, bool) {
	res, err := r.exec(context.Background(), `
		UPDATE access_grants
		SET grantor_tenant_id = $2,
			grantor_app_id = $3,
			grantee_tenant_id = $4,
			grantee_app_id = $5,
			scope_type = $6,
			scope_value = $7,
			permission = $8,
			status = $9,
			expires_at = $10,
			created_at = $11,
			revoked_at = $12
		WHERE id = $1
	`, grant.ID, grant.GrantorTenantID, nullString(grant.GrantorAppID), grant.GranteeTenantID, nullString(grant.GranteeAppID), grant.ScopeType, nullString(grant.ScopeValue), grant.Permission, grant.Status, grant.ExpiresAt, grant.CreatedAt, grant.RevokedAt)
	if err != nil {
		return access.AccessGrant{}, false
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return access.AccessGrant{}, false
	}
	return grant, true
}

func (r *Repository) CreateAuditLog(entry access.AuditLogEntry) access.AuditLogEntry {
	_, _ = r.exec(context.Background(), `
		INSERT INTO access_audit_log (
			id, requester_tenant_id, requester_app_id, action, resource_owner_tenant_id,
			resource_owner_app_id, resource_type, resource_id, outcome, reason,
			scope_type, scope_value, metadata, allowed, request_id, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id, created_at) DO UPDATE SET
			requester_tenant_id = EXCLUDED.requester_tenant_id,
			requester_app_id = EXCLUDED.requester_app_id,
			action = EXCLUDED.action,
			resource_owner_tenant_id = EXCLUDED.resource_owner_tenant_id,
			resource_owner_app_id = EXCLUDED.resource_owner_app_id,
			resource_type = EXCLUDED.resource_type,
			resource_id = EXCLUDED.resource_id,
			outcome = EXCLUDED.outcome,
			reason = EXCLUDED.reason,
			scope_type = EXCLUDED.scope_type,
			scope_value = EXCLUDED.scope_value,
			metadata = EXCLUDED.metadata,
			allowed = EXCLUDED.allowed,
			request_id = EXCLUDED.request_id
	`, entry.ID, entry.RequesterTenantID, nullString(entry.RequesterAppID), entry.Action, entry.ResourceOwnerTenantID, nullString(entry.ResourceOwnerAppID), entry.ResourceType, nullString(entry.ResourceID), entry.Outcome, nullString(entry.Reason), nullString(entry.ScopeType), nullString(entry.ScopeValue), jsonValue(entry.Metadata), entry.Outcome == "allow", nil, entry.CreatedAt)
	return entry
}

func (r *Repository) ListAuditLogs(filter access.AuditListFilter) []access.AuditLogEntry {
	query := `
		SELECT id, requester_tenant_id, requester_app_id, action, resource_owner_tenant_id,
		       resource_owner_app_id, resource_type, resource_id, outcome, reason,
		       scope_type, scope_value, metadata, created_at
		FROM access_audit_log
		WHERE 1 = 1
	`
	args := make([]any, 0, 1)
	if filter.ResourceOwnerTenantID != "" {
		query += fmt.Sprintf(" AND resource_owner_tenant_id = $%d", len(args)+1)
		args = append(args, filter.ResourceOwnerTenantID)
	}
	query += " ORDER BY created_at, id"
	rows, err := r.query(context.Background(), query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]access.AuditLogEntry, 0)
	for rows.Next() {
		entry, ok := scanAuditLog(rows)
		if !ok {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func (r *Repository) CreateDomain(domain ontology.Domain) ontology.Domain {
	_, _ = r.exec(context.Background(), `
		INSERT INTO domains (
			id, name, description, owner_tenant_id, parent_domain_id, search_profile, status, version, visibility, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			owner_tenant_id = EXCLUDED.owner_tenant_id,
			parent_domain_id = EXCLUDED.parent_domain_id,
			search_profile = EXCLUDED.search_profile,
			status = EXCLUDED.status,
			version = EXCLUDED.version,
			visibility = EXCLUDED.visibility,
			created_at = EXCLUDED.created_at,
			updated_at = EXCLUDED.updated_at
	`, domain.ID, domain.Name, nullString(domain.Description), domain.OwnerTenantID, nullString(domain.ParentDomainID), jsonValue(domain.SearchProfile), domain.Status, domain.Version, domain.Visibility, domain.CreatedAt, domain.UpdatedAt)
	return domain
}

func (r *Repository) GetDomain(id string) (ontology.Domain, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, name, description, owner_tenant_id, parent_domain_id, search_profile, status, version, visibility, created_at, updated_at
		FROM domains
		WHERE id = $1
	`, id)
	return scanDomain(row)
}

func (r *Repository) ListDomains() []ontology.Domain {
	rows, err := r.query(context.Background(), `
		SELECT id, name, description, owner_tenant_id, parent_domain_id, search_profile, status, version, visibility, created_at, updated_at
		FROM domains
		ORDER BY id
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]ontology.Domain, 0)
	for rows.Next() {
		domain, ok := scanDomain(rows)
		if !ok {
			continue
		}
		result = append(result, domain)
	}
	return result
}

func (r *Repository) CreateVersion(version ontology.OntologyVersion) {
	_, _ = r.exec(context.Background(), `
		INSERT INTO ontology_versions (
			domain_id, version, changes, breaking_change, published_at, published_by
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (domain_id, version) DO UPDATE SET
			changes = EXCLUDED.changes,
			breaking_change = EXCLUDED.breaking_change,
			published_at = EXCLUDED.published_at,
			published_by = EXCLUDED.published_by
	`, version.DomainID, version.Version, jsonValue(orEmptyMap(version.Changes)), version.BreakingChange, version.PublishedAt, nil)
}

func (r *Repository) GetCurrentVersion(domainID string) (ontology.OntologyVersion, bool) {
	row := r.queryRow(context.Background(), `
		SELECT domain_id, version, breaking_change, changes, published_at
		FROM ontology_versions
		WHERE domain_id = $1
		ORDER BY version DESC
		LIMIT 1
	`, domainID)
	return scanVersion(row)
}

func (r *Repository) UpsertSearchProfile(domainID string, profile ontology.SearchProfile) ontology.SearchProfile {
	_, _ = r.exec(context.Background(), `
		UPDATE domains
		SET search_profile = $2,
			updated_at = now()
		WHERE id = $1
	`, domainID, jsonValue(profile))
	return profile
}

func (r *Repository) GetSearchProfile(domainID string) (ontology.SearchProfile, bool) {
	row := r.queryRow(context.Background(), `
		SELECT search_profile
		FROM domains
		WHERE id = $1
		  AND search_profile IS NOT NULL
	`, domainID)
	var raw []byte
	if err := row.Scan(&raw); err != nil {
		return ontology.SearchProfile{}, false
	}
	var profile ontology.SearchProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return ontology.SearchProfile{}, false
	}
	return profile, true
}

func (r *Repository) UpsertQueryStrategy(strategy ontology.QueryStrategy) ontology.QueryStrategy {
	_, _ = r.exec(context.Background(), `
		INSERT INTO query_strategies (key, version, max_depth, params)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (key) DO UPDATE SET
			version = EXCLUDED.version,
			max_depth = EXCLUDED.max_depth,
			params = EXCLUDED.params
	`, strategy.Key, strategy.Version, strategy.MaxDepth, jsonValue(orEmptyMap(strategy.Params)))
	return strategy
}

func (r *Repository) GetQueryStrategy(key string) (ontology.QueryStrategy, bool) {
	row := r.queryRow(context.Background(), `
		SELECT key, version, max_depth, params
		FROM query_strategies
		WHERE key = $1
	`, key)
	return scanQueryStrategy(row)
}

func (r *Repository) ListQueryStrategies() []ontology.QueryStrategy {
	rows, err := r.query(context.Background(), `
		SELECT key, version, max_depth, params
		FROM query_strategies
		ORDER BY key
	`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]ontology.QueryStrategy, 0)
	for rows.Next() {
		strategy, ok := scanQueryStrategy(rows)
		if !ok {
			continue
		}
		result = append(result, strategy)
	}
	return result
}

func (r *Repository) DeleteQueryStrategy(key string) bool {
	res, err := r.exec(context.Background(), `
		DELETE FROM query_strategies
		WHERE key = $1
	`, key)
	if err != nil {
		return false
	}
	affected, _ := res.RowsAffected()
	return affected > 0
}

func (r *Repository) CreateNodeType(schema ontology.NodeTypeSchema) ontology.NodeTypeSchema {
	_, _ = r.exec(context.Background(), `
		INSERT INTO node_type_schemas (
			id, domain_id, node_type_name, graph_label, required_props, optional_props, validation_rules, version, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			domain_id = EXCLUDED.domain_id,
			node_type_name = EXCLUDED.node_type_name,
			graph_label = EXCLUDED.graph_label,
			required_props = EXCLUDED.required_props,
			optional_props = EXCLUDED.optional_props,
			validation_rules = EXCLUDED.validation_rules,
			version = EXCLUDED.version,
			created_at = EXCLUDED.created_at
	`, schema.ID, schema.DomainID, schema.NodeTypeName, schema.GraphLabel, jsonValue(schema.RequiredProps), jsonValue(schema.OptionalProps), jsonValue(orEmptyStrings(schema.ValidationRules)), schema.Version, schema.CreatedAt)
	return schema
}

func (r *Repository) GetNodeType(domainID, nodeTypeName string) (ontology.NodeTypeSchema, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, domain_id, node_type_name, graph_label, required_props, optional_props, validation_rules, version, created_at
		FROM node_type_schemas
		WHERE domain_id = $1 AND node_type_name = $2
	`, domainID, nodeTypeName)
	return scanNodeType(row)
}

func (r *Repository) ListNodeTypes(domainID string) []ontology.NodeTypeSchema {
	rows, err := r.query(context.Background(), `
		SELECT id, domain_id, node_type_name, graph_label, required_props, optional_props, validation_rules, version, created_at
		FROM node_type_schemas
		WHERE domain_id = $1
		ORDER BY node_type_name
	`, domainID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]ontology.NodeTypeSchema, 0)
	for rows.Next() {
		schema, ok := scanNodeType(rows)
		if !ok {
			continue
		}
		result = append(result, schema)
	}
	return result
}

func (r *Repository) CreateRelType(schema ontology.RelTypeSchema) ontology.RelTypeSchema {
	_, _ = r.exec(context.Background(), `
		INSERT INTO rel_type_schemas (
			id, domain_id, rel_type_name, from_node_type, to_node_type, same_domain, required_props, optional_props
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			domain_id = EXCLUDED.domain_id,
			rel_type_name = EXCLUDED.rel_type_name,
			from_node_type = EXCLUDED.from_node_type,
			to_node_type = EXCLUDED.to_node_type,
			same_domain = EXCLUDED.same_domain,
			required_props = EXCLUDED.required_props,
			optional_props = EXCLUDED.optional_props
	`, schema.ID, schema.DomainID, schema.RelTypeName, schema.FromNodeType, schema.ToNodeType, schema.SameDomain, jsonValue(schema.RequiredProps), jsonValue(schema.OptionalProps))
	return schema
}

func (r *Repository) GetRelType(domainID, relTypeName, fromNodeType, toNodeType string) (ontology.RelTypeSchema, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, domain_id, rel_type_name, from_node_type, to_node_type, same_domain, required_props, optional_props
		FROM rel_type_schemas
		WHERE domain_id = $1 AND rel_type_name = $2 AND from_node_type = $3 AND to_node_type = $4
	`, domainID, relTypeName, fromNodeType, toNodeType)
	return scanRelType(row)
}

func (r *Repository) ListRelTypes(domainID string) []ontology.RelTypeSchema {
	rows, err := r.query(context.Background(), `
		SELECT id, domain_id, rel_type_name, from_node_type, to_node_type, same_domain, required_props, optional_props
		FROM rel_type_schemas
		WHERE domain_id = $1
		ORDER BY rel_type_name, from_node_type, to_node_type
	`, domainID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]ontology.RelTypeSchema, 0)
	for rows.Next() {
		schema, ok := scanRelType(rows)
		if !ok {
			continue
		}
		result = append(result, schema)
	}
	return result
}

func (r *Repository) CreateCrossDomainRule(rule ontology.CrossDomainRelRule) ontology.CrossDomainRelRule {
	_, _ = r.exec(context.Background(), `
		INSERT INTO cross_domain_rel_rules (
			id, rel_type_name, from_domain_id, to_domain_id, from_node_types, to_node_types, required, exception_types, bridge_property_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			rel_type_name = EXCLUDED.rel_type_name,
			from_domain_id = EXCLUDED.from_domain_id,
			to_domain_id = EXCLUDED.to_domain_id,
			from_node_types = EXCLUDED.from_node_types,
			to_node_types = EXCLUDED.to_node_types,
			required = EXCLUDED.required,
			exception_types = EXCLUDED.exception_types,
			bridge_property_key = EXCLUDED.bridge_property_key
	`, rule.ID, rule.RelTypeName, rule.FromDomainID, rule.ToDomainID, stringsToArrayValue(rule.FromNodeTypes), stringsToArrayValue(rule.ToNodeTypes), rule.Required, stringsToArrayValue(rule.ExceptionTypes), rule.BridgePropertyKey)
	return rule
}

func (r *Repository) ListCrossDomainRules(fromDomainID string) []ontology.CrossDomainRelRule {
	rows, err := r.query(context.Background(), `
		SELECT id, rel_type_name, from_domain_id, to_domain_id, from_node_types, to_node_types, required, exception_types, bridge_property_key
		FROM cross_domain_rel_rules
		WHERE from_domain_id = $1
		ORDER BY rel_type_name, id
	`, fromDomainID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]ontology.CrossDomainRelRule, 0)
	for rows.Next() {
		rule, ok := scanCrossDomainRule(rows)
		if !ok {
			continue
		}
		result = append(result, rule)
	}
	return result
}

func (r *Repository) CreateQueryTemplate(template ontology.QueryTemplate) ontology.QueryTemplate {
	_, _ = r.exec(context.Background(), `
		INSERT INTO domain_query_templates (
			id, domain_id, template_name, pattern_spec, param_schema, return_fields, description, status, version, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			domain_id = EXCLUDED.domain_id,
			template_name = EXCLUDED.template_name,
			pattern_spec = EXCLUDED.pattern_spec,
			param_schema = EXCLUDED.param_schema,
			return_fields = EXCLUDED.return_fields,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			version = EXCLUDED.version,
			created_at = EXCLUDED.created_at
	`, template.ID, template.DomainID, template.TemplateName, jsonValue(template.PatternSpec), jsonValue(template.ParamSchema), stringsToArrayValue(template.ReturnFields), nullString(template.Description), template.Status, template.Version, template.CreatedAt)
	return template
}

func (r *Repository) GetQueryTemplate(domainID, templateName string) (ontology.QueryTemplate, bool) {
	row := r.queryRow(context.Background(), `
		SELECT id, domain_id, template_name, pattern_spec, param_schema, return_fields, description, status, version, created_at
		FROM domain_query_templates
		WHERE domain_id = $1 AND template_name = $2
	`, domainID, templateName)
	return scanQueryTemplate(row)
}

func (r *Repository) UpdateQueryTemplate(template ontology.QueryTemplate) (ontology.QueryTemplate, bool) {
	res, err := r.exec(context.Background(), `
		UPDATE domain_query_templates
		SET domain_id = $2,
			template_name = $3,
			pattern_spec = $4,
			param_schema = $5,
			return_fields = $6,
			description = $7,
			status = $8,
			version = $9,
			created_at = $10
		WHERE id = $1
	`, template.ID, template.DomainID, template.TemplateName, jsonValue(template.PatternSpec), jsonValue(template.ParamSchema), stringsToArrayValue(template.ReturnFields), nullString(template.Description), template.Status, template.Version, template.CreatedAt)
	if err != nil {
		return ontology.QueryTemplate{}, false
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ontology.QueryTemplate{}, false
	}
	return template, true
}

func (r *Repository) ListQueryTemplates(domainID string) []ontology.QueryTemplate {
	rows, err := r.query(context.Background(), `
		SELECT id, domain_id, template_name, pattern_spec, param_schema, return_fields, description, status, version, created_at
		FROM domain_query_templates
		WHERE domain_id = $1
		ORDER BY template_name
	`, domainID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	result := make([]ontology.QueryTemplate, 0)
	for rows.Next() {
		template, ok := scanQueryTemplate(rows)
		if !ok {
			continue
		}
		result = append(result, template)
	}
	return result
}

func (r *Repository) UpsertStatusFieldConfig(config ontology.StatusFieldConfig) ontology.StatusFieldConfig {
	_, _ = r.exec(context.Background(), `
		INSERT INTO domain_status_field_configs (
			domain_id, status_field_name, valid_status_values, warning_status_values, cascade_rules, authority_field_name, authority_values_map
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (domain_id) DO UPDATE SET
			status_field_name = EXCLUDED.status_field_name,
			valid_status_values = EXCLUDED.valid_status_values,
			warning_status_values = EXCLUDED.warning_status_values,
			cascade_rules = EXCLUDED.cascade_rules,
			authority_field_name = EXCLUDED.authority_field_name,
			authority_values_map = EXCLUDED.authority_values_map
	`, config.DomainID, nullString(config.StatusFieldName), stringsToArrayValue(config.ValidStatusValues), stringsToArrayValue(config.WarningStatusValues), jsonValue(config.CascadeRules), nullString(config.AuthorityFieldName), jsonValue(orEmptyMapInt(config.AuthorityValuesMap)))
	return config
}

func (r *Repository) GetStatusFieldConfig(domainID string) (ontology.StatusFieldConfig, bool) {
	row := r.queryRow(context.Background(), `
		SELECT domain_id, status_field_name, valid_status_values, warning_status_values, cascade_rules, authority_field_name, authority_values_map
		FROM domain_status_field_configs
		WHERE domain_id = $1
	`, domainID)
	return scanStatusFieldConfig(row)
}

func scanTenant(row rowScanner) (access.Tenant, bool) {
	var tenant access.Tenant
	if err := row.Scan(&tenant.ID, &tenant.Slug, &tenant.Name, &tenant.Status, &tenant.Tier, &tenant.DefaultSharingPolicy, &tenant.CreatedAt, &tenant.UpdatedAt); err != nil {
		return access.Tenant{}, false
	}
	return tenant, true
}

func scanApp(row rowScanner) (access.App, bool) {
	var app access.App
	var revokedAt sql.NullTime
	if err := row.Scan(&app.ID, &app.TenantID, &app.Slug, &app.Name, &app.Type, &app.APIKeyHash, &app.APIKeyPrefix, &app.Status, &app.CreatedAt, &revokedAt); err != nil {
		return access.App{}, false
	}
	if revokedAt.Valid {
		app.RevokedAt = &revokedAt.Time
	}
	return app, true
}

func scanGrant(row rowScanner) (access.AccessGrant, bool) {
	var grant access.AccessGrant
	var grantorApp, granteeApp sql.NullString
	var expiresAt, revokedAt sql.NullTime
	if err := row.Scan(&grant.ID, &grant.GrantorTenantID, &grantorApp, &grant.GranteeTenantID, &granteeApp, &grant.ScopeType, &grant.ScopeValue, &grant.Permission, &grant.Status, &expiresAt, &grant.CreatedAt, &revokedAt); err != nil {
		return access.AccessGrant{}, false
	}
	grant.GrantorAppID = grantorApp.String
	grant.GranteeAppID = granteeApp.String
	if expiresAt.Valid {
		grant.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		grant.RevokedAt = &revokedAt.Time
	}
	return grant, true
}

func scanAuditLog(row rowScanner) (access.AuditLogEntry, bool) {
	var entry access.AuditLogEntry
	var requesterApp, ownerApp, resourceID, scopeType, scopeValue, reason, resourceType, outcome sql.NullString
	var metadata []byte
	if err := row.Scan(&entry.ID, &entry.RequesterTenantID, &requesterApp, &entry.Action, &entry.ResourceOwnerTenantID, &ownerApp, &resourceType, &resourceID, &outcome, &reason, &scopeType, &scopeValue, &metadata, &entry.CreatedAt); err != nil {
		return access.AuditLogEntry{}, false
	}
	entry.RequesterAppID = requesterApp.String
	entry.ResourceOwnerAppID = ownerApp.String
	entry.ResourceType = resourceType.String
	entry.ResourceID = resourceID.String
	entry.Outcome = outcome.String
	entry.Reason = reason.String
	entry.ScopeType = scopeType.String
	entry.ScopeValue = scopeValue.String
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &entry.Metadata)
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}
	return entry, true
}

func scanDomain(row rowScanner) (ontology.Domain, bool) {
	var domain ontology.Domain
	var description, parentDomain sql.NullString
	var searchProfile []byte
	if err := row.Scan(&domain.ID, &domain.Name, &description, &domain.OwnerTenantID, &parentDomain, &searchProfile, &domain.Status, &domain.Version, &domain.Visibility, &domain.CreatedAt, &domain.UpdatedAt); err != nil {
		return ontology.Domain{}, false
	}
	domain.Description = description.String
	domain.ParentDomainID = parentDomain.String
	if len(searchProfile) > 0 {
		var profile ontology.SearchProfile
		if err := json.Unmarshal(searchProfile, &profile); err == nil {
			domain.SearchProfile = &profile
		}
	}
	return domain, true
}

func scanVersion(row rowScanner) (ontology.OntologyVersion, bool) {
	var version ontology.OntologyVersion
	var changes []byte
	if err := row.Scan(&version.DomainID, &version.Version, &version.BreakingChange, &changes, &version.PublishedAt); err != nil {
		return ontology.OntologyVersion{}, false
	}
	if len(changes) > 0 {
		_ = json.Unmarshal(changes, &version.Changes)
	}
	if version.Changes == nil {
		version.Changes = map[string]any{}
	}
	return version, true
}

func scanNodeType(row rowScanner) (ontology.NodeTypeSchema, bool) {
	var schema ontology.NodeTypeSchema
	var requiredProps, optionalProps, validationRules []byte
	if err := row.Scan(&schema.ID, &schema.DomainID, &schema.NodeTypeName, &schema.GraphLabel, &requiredProps, &optionalProps, &validationRules, &schema.Version, &schema.CreatedAt); err != nil {
		return ontology.NodeTypeSchema{}, false
	}
	if len(requiredProps) > 0 {
		_ = json.Unmarshal(requiredProps, &schema.RequiredProps)
	}
	if len(optionalProps) > 0 {
		_ = json.Unmarshal(optionalProps, &schema.OptionalProps)
	}
	if len(validationRules) > 0 {
		_ = json.Unmarshal(validationRules, &schema.ValidationRules)
	}
	if schema.RequiredProps == nil {
		schema.RequiredProps = []ontology.PropertySchema{}
	}
	if schema.OptionalProps == nil {
		schema.OptionalProps = []ontology.PropertySchema{}
	}
	if schema.ValidationRules == nil {
		schema.ValidationRules = []string{}
	}
	return schema, true
}

func scanRelType(row rowScanner) (ontology.RelTypeSchema, bool) {
	var schema ontology.RelTypeSchema
	var requiredProps, optionalProps []byte
	if err := row.Scan(&schema.ID, &schema.DomainID, &schema.RelTypeName, &schema.FromNodeType, &schema.ToNodeType, &schema.SameDomain, &requiredProps, &optionalProps); err != nil {
		return ontology.RelTypeSchema{}, false
	}
	if len(requiredProps) > 0 {
		_ = json.Unmarshal(requiredProps, &schema.RequiredProps)
	}
	if len(optionalProps) > 0 {
		_ = json.Unmarshal(optionalProps, &schema.OptionalProps)
	}
	if schema.RequiredProps == nil {
		schema.RequiredProps = []ontology.PropertySchema{}
	}
	if schema.OptionalProps == nil {
		schema.OptionalProps = []ontology.PropertySchema{}
	}
	return schema, true
}

func scanCrossDomainRule(row rowScanner) (ontology.CrossDomainRelRule, bool) {
	var rule ontology.CrossDomainRelRule
	var fromNodeTypes, toNodeTypes, exceptionTypes []string
	if err := row.Scan(&rule.ID, &rule.RelTypeName, &rule.FromDomainID, &rule.ToDomainID, &fromNodeTypes, &toNodeTypes, &rule.Required, &exceptionTypes, &rule.BridgePropertyKey); err != nil {
		return ontology.CrossDomainRelRule{}, false
	}
	rule.FromNodeTypes = append([]string(nil), fromNodeTypes...)
	rule.ToNodeTypes = append([]string(nil), toNodeTypes...)
	rule.ExceptionTypes = append([]string(nil), exceptionTypes...)
	return rule, true
}

func scanQueryTemplate(row rowScanner) (ontology.QueryTemplate, bool) {
	var template ontology.QueryTemplate
	var patternSpec, paramSchema []byte
	var returnFields []string
	var description sql.NullString
	if err := row.Scan(&template.ID, &template.DomainID, &template.TemplateName, &patternSpec, &paramSchema, &returnFields, &description, &template.Status, &template.Version, &template.CreatedAt); err != nil {
		return ontology.QueryTemplate{}, false
	}
	if len(patternSpec) > 0 {
		_ = json.Unmarshal(patternSpec, &template.PatternSpec)
	}
	if len(paramSchema) > 0 {
		_ = json.Unmarshal(paramSchema, &template.ParamSchema)
	}
	template.ReturnFields = append([]string(nil), returnFields...)
	template.Description = description.String
	if template.PatternSpec == nil {
		template.PatternSpec = map[string]any{}
	}
	if template.ParamSchema == nil {
		template.ParamSchema = []ontology.ParameterSchema{}
	}
	if template.ReturnFields == nil {
		template.ReturnFields = []string{}
	}
	return template, true
}

func scanQueryStrategy(row rowScanner) (ontology.QueryStrategy, bool) {
	var strategy ontology.QueryStrategy
	var params []byte
	if err := row.Scan(&strategy.Key, &strategy.Version, &strategy.MaxDepth, &params); err != nil {
		return ontology.QueryStrategy{}, false
	}
	if len(params) > 0 {
		_ = json.Unmarshal(params, &strategy.Params)
	}
	if strategy.Params == nil {
		strategy.Params = map[string]any{}
	}
	return strategy, true
}

func scanStatusFieldConfig(row rowScanner) (ontology.StatusFieldConfig, bool) {
	var config ontology.StatusFieldConfig
	var validStatusValues, warningStatusValues []string
	var cascadeRules, authorityValuesMap []byte
	if err := row.Scan(&config.DomainID, &config.StatusFieldName, &validStatusValues, &warningStatusValues, &cascadeRules, &config.AuthorityFieldName, &authorityValuesMap); err != nil {
		return ontology.StatusFieldConfig{}, false
	}
	config.ValidStatusValues = append([]string(nil), validStatusValues...)
	config.WarningStatusValues = append([]string(nil), warningStatusValues...)
	if len(cascadeRules) > 0 {
		_ = json.Unmarshal(cascadeRules, &config.CascadeRules)
	}
	if len(authorityValuesMap) > 0 {
		_ = json.Unmarshal(authorityValuesMap, &config.AuthorityValuesMap)
	}
	if config.CascadeRules == nil {
		config.CascadeRules = []ontology.CascadeRule{}
	}
	if config.AuthorityValuesMap == nil {
		config.AuthorityValuesMap = map[string]int{}
	}
	return config, true
}

func jsonValue(value any) any {
	if value == nil {
		return nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return b
}

func orEmptyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func orEmptyMapInt(value map[string]int) map[string]int {
	if value == nil {
		return map[string]int{}
	}
	return value
}

func orEmptyStrings(value []string) []string {
	if value == nil {
		return []string{}
	}
	return value
}

func stringsToArrayValue(values []string) any {
	return orEmptyStrings(values)
}
