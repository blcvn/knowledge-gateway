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

ALTER TABLE kg_projection_versions
    ALTER COLUMN entity_id TYPE TEXT USING entity_id::text,
    ALTER COLUMN source_event_id TYPE TEXT USING source_event_id::text;

ALTER TABLE kg_vector_documents
    ALTER COLUMN id TYPE TEXT USING id::text,
    ALTER COLUMN node_id TYPE TEXT USING node_id::text,
    ALTER COLUMN owner_tenant_id TYPE TEXT USING owner_tenant_id::text,
    ALTER COLUMN owner_app_id TYPE TEXT USING owner_app_id::text;

ALTER TABLE kg_outbox_events
    ALTER COLUMN id TYPE TEXT USING id::text,
    ALTER COLUMN aggregate_id TYPE TEXT USING aggregate_id::text;

ALTER TABLE kg_relationships
    ALTER COLUMN id TYPE TEXT USING id::text,
    ALTER COLUMN from_node_id TYPE TEXT USING from_node_id::text,
    ALTER COLUMN to_node_id TYPE TEXT USING to_node_id::text,
    ALTER COLUMN owner_tenant_id TYPE TEXT USING owner_tenant_id::text,
    ALTER COLUMN owner_app_id TYPE TEXT USING owner_app_id::text;

ALTER TABLE kg_nodes
    ALTER COLUMN id TYPE TEXT USING id::text,
    ALTER COLUMN owner_tenant_id TYPE TEXT USING owner_tenant_id::text,
    ALTER COLUMN owner_app_id TYPE TEXT USING owner_app_id::text;

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

ALTER TABLE kg_relationships
    ADD CONSTRAINT kg_relationships_from_node_id_fkey FOREIGN KEY (from_node_id) REFERENCES kg_nodes(id),
    ADD CONSTRAINT kg_relationships_to_node_id_fkey FOREIGN KEY (to_node_id) REFERENCES kg_nodes(id);

ALTER TABLE kg_vector_documents
    ADD CONSTRAINT kg_vector_documents_node_id_fkey FOREIGN KEY (node_id) REFERENCES kg_nodes(id) ON DELETE CASCADE;

CREATE POLICY kg_nodes_isolation ON kg_nodes
    USING (
        owner_tenant_id = current_setting('app.tenant_id')
        OR owner_tenant_id = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM access_grants g
            WHERE g.grantee_tenant_id = current_setting('app.tenant_id')
              AND (g.grantee_app_id = current_setting('app.app_id') OR g.grantee_app_id IS NULL)
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
        owner_tenant_id = current_setting('app.tenant_id')
        OR owner_tenant_id = '00000000-0000-0000-0000-000000000000'
        OR EXISTS (
            SELECT 1
            FROM access_grants g
            WHERE g.grantee_tenant_id = current_setting('app.tenant_id')
              AND (g.grantee_app_id = current_setting('app.app_id') OR g.grantee_app_id IS NULL)
              AND g.status = 'active'
              AND (g.expires_at IS NULL OR g.expires_at > now())
              AND (
                  g.scope_type = 'all'
                  OR (g.scope_type = 'domain' AND g.scope_value = kg_relationships.domain_id)
              )
        )
    );
