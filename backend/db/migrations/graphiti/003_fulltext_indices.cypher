// Fulltext indices for BM25 keyword search

CREATE FULLTEXT INDEX entity_fulltext IF NOT EXISTS
    FOR (n:Entity) ON EACH [n.name, n.summary];

CREATE FULLTEXT INDEX episode_fulltext IF NOT EXISTS
    FOR (n:Episodic) ON EACH [n.content, n.source_description];

CREATE FULLTEXT INDEX community_fulltext IF NOT EXISTS
    FOR (n:Community) ON EACH [n.name, n.summary];

CREATE FULLTEXT INDEX entity_edge_fulltext IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON EACH [r.fact, r.name];
