ALTER TABLE cross_domain_rel_rules
    DROP COLUMN IF EXISTS bridge_property_key;

ALTER TABLE access_audit_log
    DROP COLUMN IF EXISTS metadata,
    DROP COLUMN IF EXISTS scope_value,
    DROP COLUMN IF EXISTS scope_type,
    DROP COLUMN IF EXISTS outcome,
    DROP COLUMN IF EXISTS resource_id,
    DROP COLUMN IF EXISTS resource_type,
    DROP COLUMN IF EXISTS resource_owner_app_id;

