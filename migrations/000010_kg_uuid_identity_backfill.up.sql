DO $$
BEGIN
    CREATE TEMP TABLE kg_uuid_identity_map (
        entity_kind TEXT NOT NULL,
        old_id TEXT NOT NULL,
        new_id UUID NOT NULL,
        PRIMARY KEY (entity_kind, old_id)
    ) ON COMMIT DROP;

    INSERT INTO kg_uuid_identity_map (entity_kind, old_id, new_id)
    SELECT 'kg_node', id::text,
           CASE
               WHEN id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN id::text::uuid
               ELSE gen_random_uuid()
           END
    FROM kg_nodes;

    INSERT INTO kg_uuid_identity_map (entity_kind, old_id, new_id)
    SELECT 'kg_relationship', id::text,
           CASE
               WHEN id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN id::text::uuid
               ELSE gen_random_uuid()
           END
    FROM kg_relationships;

    INSERT INTO kg_uuid_identity_map (entity_kind, old_id, new_id)
    SELECT 'kg_outbox_event', id::text,
           CASE
               WHEN id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN id::text::uuid
               ELSE gen_random_uuid()
           END
    FROM kg_outbox_events;

    INSERT INTO kg_uuid_identity_map (entity_kind, old_id, new_id)
    SELECT 'kg_vector_document', id::text,
           CASE
               WHEN id::text ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN id::text::uuid
               ELSE gen_random_uuid()
           END
    FROM kg_vector_documents;
END $$;

CREATE OR REPLACE FUNCTION kg_uuid_identity_lookup(p_entity_kind TEXT, value TEXT)
RETURNS UUID
LANGUAGE plpgsql
VOLATILE
AS $$
DECLARE
    mapped UUID;
BEGIN
    SELECT new_id INTO mapped
    FROM kg_uuid_identity_map
    WHERE kg_uuid_identity_map.entity_kind = p_entity_kind
      AND old_id = value;

    IF FOUND THEN
        RETURN mapped;
    END IF;

    IF value ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$' THEN
        RETURN value::uuid;
    END IF;

    RETURN gen_random_uuid();
END;
$$;

ALTER TABLE kg_relationships
    DROP CONSTRAINT IF EXISTS kg_relationships_from_node_id_fkey,
    DROP CONSTRAINT IF EXISTS kg_relationships_to_node_id_fkey;

ALTER TABLE kg_vector_documents
    DROP CONSTRAINT IF EXISTS kg_vector_documents_node_id_fkey;

DROP POLICY IF EXISTS kg_nodes_isolation ON kg_nodes;
DROP POLICY IF EXISTS kg_relationships_isolation ON kg_relationships;

DROP INDEX IF EXISTS idx_kg_nodes_fts_vector;
ALTER TABLE kg_nodes
    DROP COLUMN IF EXISTS fts_vector;

ALTER TABLE kg_nodes
    ALTER COLUMN id DROP DEFAULT,
    ALTER COLUMN id TYPE UUID USING kg_uuid_identity_lookup('kg_node', id::text),
    ALTER COLUMN id SET DEFAULT gen_random_uuid(),
    ALTER COLUMN owner_tenant_id TYPE UUID USING owner_tenant_id::uuid,
    ALTER COLUMN owner_app_id TYPE UUID USING NULLIF(owner_app_id::text, '')::uuid;

ALTER TABLE kg_relationships
    ALTER COLUMN id TYPE UUID USING kg_uuid_identity_lookup('kg_relationship', id::text),
    ALTER COLUMN from_node_id TYPE UUID USING kg_uuid_identity_lookup('kg_node', from_node_id::text),
    ALTER COLUMN to_node_id TYPE UUID USING kg_uuid_identity_lookup('kg_node', to_node_id::text),
    ALTER COLUMN owner_tenant_id TYPE UUID USING owner_tenant_id::uuid,
    ALTER COLUMN owner_app_id TYPE UUID USING NULLIF(owner_app_id::text, '')::uuid;

ALTER TABLE kg_outbox_events
    ALTER COLUMN id TYPE UUID USING kg_uuid_identity_lookup('kg_outbox_event', id::text),
    ALTER COLUMN aggregate_id TYPE UUID USING CASE
        WHEN aggregate_type = 'kg_relationship' THEN kg_uuid_identity_lookup('kg_relationship', aggregate_id::text)
        ELSE kg_uuid_identity_lookup('kg_node', aggregate_id::text)
    END;

ALTER TABLE kg_vector_documents
    ALTER COLUMN id TYPE UUID USING kg_uuid_identity_lookup('kg_vector_document', id::text),
    ALTER COLUMN node_id TYPE UUID USING kg_uuid_identity_lookup('kg_node', node_id::text),
    ALTER COLUMN owner_tenant_id TYPE UUID USING owner_tenant_id::uuid,
    ALTER COLUMN owner_app_id TYPE UUID USING NULLIF(owner_app_id::text, '')::uuid;

ALTER TABLE kg_projection_versions
    ALTER COLUMN entity_id TYPE UUID USING CASE
        WHEN entity_kind = 'kg_relationship' THEN kg_uuid_identity_lookup('kg_relationship', entity_id::text)
        ELSE kg_uuid_identity_lookup('kg_node', entity_id::text)
    END,
    ALTER COLUMN source_event_id TYPE UUID USING kg_uuid_identity_lookup('kg_outbox_event', source_event_id::text);

ALTER TABLE kg_relationships
    ADD CONSTRAINT kg_relationships_from_node_id_fkey FOREIGN KEY (from_node_id) REFERENCES kg_nodes(id),
    ADD CONSTRAINT kg_relationships_to_node_id_fkey FOREIGN KEY (to_node_id) REFERENCES kg_nodes(id);

ALTER TABLE kg_vector_documents
    ADD CONSTRAINT kg_vector_documents_node_id_fkey FOREIGN KEY (node_id) REFERENCES kg_nodes(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_kg_nodes_external_ref_active
    ON kg_nodes(external_ref)
    WHERE external_ref IS NOT NULL AND NOT is_deleted;

CREATE POLICY kg_nodes_isolation ON kg_nodes
    USING (
        owner_tenant_id = current_setting('app.tenant_id')::uuid
        OR owner_tenant_id = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM access_grants g
            WHERE g.grantee_tenant_id = current_setting('app.tenant_id')::uuid
              AND (g.grantee_app_id = current_setting('app.app_id')::uuid OR g.grantee_app_id IS NULL)
              AND g.status = 'active'
              AND (g.expires_at IS NULL OR g.expires_at > now())
              AND (
                  g.scope_type = 'all'
                  OR (g.scope_type = 'domain' AND g.scope_value = kg_nodes.domain_id)
                  OR (g.scope_type = 'node_type' AND g.scope_value = kg_nodes.node_type)
              )
        )
    );

CREATE POLICY kg_relationships_isolation ON kg_relationships
    USING (
        owner_tenant_id = current_setting('app.tenant_id')::uuid
        OR owner_tenant_id = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM access_grants g
            WHERE g.grantee_tenant_id = current_setting('app.tenant_id')::uuid
              AND (g.grantee_app_id = current_setting('app.app_id')::uuid OR g.grantee_app_id IS NULL)
              AND g.status = 'active'
              AND (g.expires_at IS NULL OR g.expires_at > now())
              AND (
                  g.scope_type = 'all'
                  OR (g.scope_type = 'domain' AND g.scope_value = kg_relationships.domain_id)
              )
        )
    );

ALTER TABLE kg_nodes
    ADD COLUMN fts_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', COALESCE(id::text, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(node_type, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(domain_id, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(external_ref, '')), 'C') ||
        setweight(to_tsvector('simple', COALESCE(status_value, '')), 'D') ||
        COALESCE(jsonb_to_tsvector('simple', properties, '["string"]'), ''::tsvector)
    ) STORED;

CREATE INDEX idx_kg_nodes_fts_vector
    ON kg_nodes
    USING GIN (fts_vector);

DROP FUNCTION kg_uuid_identity_lookup(TEXT, TEXT);
