package neo4j

const (
	upsertEntityQuery = `
		MERGE (n:Entity:Tenant_%s {name: $name, entity_type: $type})
		SET n.description = $description, n.updated_at = datetime(), n.id = $id
		RETURN n
	`
	upsertRelationshipQuery = `
		MATCH (s:Entity:Tenant_%s {id: $source_id})
		MATCH (t:Entity:Tenant_%s {id: $target_id})
		MERGE (s)-[r:RELATES_TO {relation: $relation}]->(t)
		SET r.weight = $weight, r.source_chunk_id = $chunk_id, r.updated_at = datetime()
	`
	upsertCommunityQuery = `
		MERGE (c:Community:Tenant_%s {id: $id})
		SET c.summary = $summary, c.level = $level, c.updated_at = datetime()
		WITH c
		UNWIND $entity_ids AS entity_id
		MATCH (e:Entity:Tenant_%[1]s {id: entity_id})
		MERGE (e)-[:BELONGS_TO]->(c)
	`
	getEntityByNameQuery = `
		MATCH (n:Entity:Tenant_%s {name: $name})
		RETURN n.id AS id, n.name AS name, n.entity_type AS type, n.description AS description
	`
)
