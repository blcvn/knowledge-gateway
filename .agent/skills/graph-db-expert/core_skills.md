# Core Skills — Graph Database Engineering

## Ontology Design for Knowledge Graphs

### Node & Relationship Design for This Platform
```cypher
// Core Nodes
(:Role {id, name, description})              // Actors: Admin, Customer, etc.
(:Entity {id, name, normalized_name})         // Domain objects: Order, Product
(:Field {id, name, type, required, default})  // Entity properties
(:Action {id, verb, description})             // Verbs: Approve, Edit, Delete
(:Requirement {id, text, type, source_line})  // Functional/non-functional
(:State {id, name, entity_id})               // Status values: Pending, Approved
(:Screen {id, type, title})                  // UI element linked to requirement

// Core Relationships
(r:Role)-[:PERFORMS]->(a:Action)
(a:Action)-[:MANIPULATES]->(e:Entity)
(e:Entity)-[:HAS_FIELD]->(f:Field)
(s1:State)-[:TRANSITIONS_TO {condition}]->(s2:State)
(req:Requirement)-[:REQUIRES]->(a:Action)
(req:Requirement)-[:HAS_CONDITION]->(rule:Rule)
(screen:Screen)-[:IMPLEMENTS]->(req:Requirement)   // Traceability link
```

### Idempotent Write Patterns (Critical)
```cypher
// ALWAYS use MERGE, never CREATE alone, for nodes that may already exist
MERGE (e:Entity {normalized_name: $name})
ON CREATE SET e.id = $id, e.name = $name, e.created_at = timestamp()
ON MATCH SET e.updated_at = timestamp()

// MERGE relationships with unique keys
MERGE (r:Role {name: $roleName})-[:PERFORMS]->(a:Action {verb: $verb})
```

### Query Patterns

**Completeness Validation (every Requirement has a Screen):**
```cypher
MATCH (req:Requirement)
WHERE NOT EXISTS { MATCH (req)<-[:IMPLEMENTS]-(s:Screen) }
RETURN req.id, req.text AS unmapped_requirements
```

**Traceability — What requirements does a screen implement?**
```cypher
MATCH (s:Screen {id: $screenId})-[:IMPLEMENTS]->(req:Requirement)
RETURN s.title, collect(req.text) AS requirements
```

**Full Entity Context (for UI Schema generation):**
```cypher
MATCH (e:Entity {id: $entityId})
OPTIONAL MATCH (e)-[:HAS_FIELD]->(f:Field)
OPTIONAL MATCH (r:Role)-[:PERFORMS]->(a:Action)-[:MANIPULATES]->(e)
OPTIONAL MATCH (s:State)-[:TRANSITIONS_TO]->(s2:State)
RETURN e, collect(DISTINCT f) AS fields, collect(DISTINCT {role: r.name, action: a.verb}) AS actions
```

## Performance & Index Strategy
```cypher
// Unique constraints — enforce data integrity AND create implicit indexes
CREATE CONSTRAINT entity_name_unique IF NOT EXISTS
  FOR (e:Entity) REQUIRE e.normalized_name IS UNIQUE;

CREATE CONSTRAINT requirement_id_unique IF NOT EXISTS
  FOR (req:Requirement) REQUIRE req.id IS UNIQUE;

// Explicit index for frequent lookups
CREATE INDEX role_name_index IF NOT EXISTS FOR (r:Role) ON (r.name);
```

## Schema Evolution
- **Adding a new node type:** Create new `CONSTRAINT` and populate via migration script. Never delete old nodes; deprecate with a `deprecated: true` property.
- **Adding a new relationship type:** MERGE-based scripts are safe to run multiple times.
- **Renaming a property:** Add the new property first, migrate data, then remove the old one — never rename in a single operation.
