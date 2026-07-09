ALTER TABLE access_audit_log
    ADD COLUMN IF NOT EXISTS resource_owner_app_id UUID,
    ADD COLUMN IF NOT EXISTS resource_type TEXT,
    ADD COLUMN IF NOT EXISTS resource_id TEXT,
    ADD COLUMN IF NOT EXISTS outcome TEXT NOT NULL DEFAULT 'allow',
    ADD COLUMN IF NOT EXISTS scope_type TEXT,
    ADD COLUMN IF NOT EXISTS scope_value TEXT,
    ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE cross_domain_rel_rules
    ADD COLUMN IF NOT EXISTS bridge_property_key TEXT NOT NULL DEFAULT '';

