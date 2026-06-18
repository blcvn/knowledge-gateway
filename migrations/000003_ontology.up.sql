CREATE TABLE domains (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    owner_tenant_id UUID NOT NULL REFERENCES tenants(id),
    parent_domain_id TEXT REFERENCES domains(id),
    search_profile JSONB,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'deprecated')),
    version INT NOT NULL DEFAULT 1,
    visibility TEXT NOT NULL DEFAULT 'private'
        CHECK (visibility IN ('public', 'tenant_shared', 'private')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE ontology_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id TEXT NOT NULL REFERENCES domains(id),
    version INT NOT NULL,
    changes JSONB NOT NULL,
    breaking_change BOOLEAN NOT NULL DEFAULT false,
    published_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_by UUID,
    UNIQUE (domain_id, version)
);

CREATE TABLE node_type_schemas (
    id TEXT PRIMARY KEY,
    domain_id TEXT NOT NULL REFERENCES domains(id),
    node_type_name TEXT NOT NULL,
    graph_label TEXT NOT NULL,
    required_props JSONB NOT NULL DEFAULT '[]'::jsonb,
    optional_props JSONB NOT NULL DEFAULT '[]'::jsonb,
    validation_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (domain_id, node_type_name)
);

CREATE TABLE rel_type_schemas (
    id TEXT PRIMARY KEY,
    domain_id TEXT NOT NULL REFERENCES domains(id),
    rel_type_name TEXT NOT NULL,
    from_node_type TEXT NOT NULL,
    to_node_type TEXT NOT NULL,
    same_domain BOOLEAN NOT NULL DEFAULT true,
    required_props JSONB NOT NULL DEFAULT '[]'::jsonb,
    optional_props JSONB NOT NULL DEFAULT '[]'::jsonb,
    UNIQUE (domain_id, rel_type_name, from_node_type, to_node_type)
);

CREATE TABLE cross_domain_rel_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rel_type_name TEXT NOT NULL,
    from_domain_id TEXT NOT NULL REFERENCES domains(id),
    to_domain_id TEXT NOT NULL REFERENCES domains(id),
    from_node_types TEXT[] NOT NULL DEFAULT ARRAY['*']::TEXT[],
    to_node_types TEXT[] NOT NULL,
    required BOOLEAN NOT NULL DEFAULT true,
    exception_types TEXT[] NOT NULL DEFAULT ARRAY[]::TEXT[],
    allowed_loai_values TEXT[] DEFAULT ARRAY['can_cu_chinh', 'tham_khao']::TEXT[]
);

CREATE TABLE domain_query_templates (
    id TEXT PRIMARY KEY,
    domain_id TEXT NOT NULL REFERENCES domains(id),
    template_name TEXT NOT NULL,
    pattern_spec JSONB NOT NULL,
    param_schema JSONB NOT NULL DEFAULT '[]'::jsonb,
    return_fields TEXT[] NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'deprecated')),
    created_by UUID,
    version INT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (domain_id, template_name)
);

CREATE TABLE domain_status_field_configs (
    domain_id TEXT PRIMARY KEY REFERENCES domains(id),
    status_field_name TEXT,
    valid_status_values TEXT[],
    warning_status_values TEXT[],
    cascade_rules JSONB NOT NULL DEFAULT '[]'::jsonb,
    authority_field_name TEXT,
    authority_values_map JSONB
);

CREATE TABLE query_strategies (
    key TEXT PRIMARY KEY,
    version INT NOT NULL DEFAULT 1,
    max_depth INT NOT NULL,
    params JSONB NOT NULL DEFAULT '{}'::jsonb
);
