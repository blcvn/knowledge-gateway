// Property indices for filtering by group_id and temporal fields

CREATE INDEX entity_group_id IF NOT EXISTS
    FOR (n:Entity) ON (n.group_id);

CREATE INDEX episode_group_id IF NOT EXISTS
    FOR (n:Episodic) ON (n.group_id);

CREATE INDEX episode_valid_at IF NOT EXISTS
    FOR (n:Episodic) ON (n.valid_at);

CREATE INDEX community_group_id IF NOT EXISTS
    FOR (n:Community) ON (n.group_id);

// RELATES_TO edge property indices (for temporal filtering)
CREATE INDEX edge_group_id IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.group_id);

CREATE INDEX edge_valid_at IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.valid_at);

CREATE INDEX edge_invalid_at IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.invalid_at);

CREATE INDEX edge_created_at IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.created_at);
