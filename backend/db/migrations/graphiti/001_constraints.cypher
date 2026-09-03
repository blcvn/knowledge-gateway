// Uniqueness constraints — ensure UUID uniqueness per node label

CREATE CONSTRAINT entity_node_uuid IF NOT EXISTS
    FOR (n:Entity) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT episodic_node_uuid IF NOT EXISTS
    FOR (n:Episodic) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT community_node_uuid IF NOT EXISTS
    FOR (n:Community) REQUIRE n.uuid IS UNIQUE;

CREATE CONSTRAINT saga_node_uuid IF NOT EXISTS
    FOR (n:Saga) REQUIRE n.uuid IS UNIQUE;
