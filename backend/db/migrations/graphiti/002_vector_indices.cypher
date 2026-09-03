// Vector indices for semantic similarity search
// Requires Neo4j 5.11+ Enterprise or AuraDB

CREATE VECTOR INDEX entity_name_embedding IF NOT EXISTS
    FOR (n:Entity) ON (n.name_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};

CREATE VECTOR INDEX entity_edge_fact_embedding IF NOT EXISTS
    FOR ()-[r:RELATES_TO]-() ON (r.fact_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};

CREATE VECTOR INDEX community_name_embedding IF NOT EXISTS
    FOR (n:Community) ON (n.name_embedding)
    OPTIONS {indexConfig: {
        `vector.dimensions`: 1536,
        `vector.similarity_function`: 'cosine'
    }};
