---
skill_id: SKILL-005
version: 1.0.0
status: active
priority: P0
group: Backend Development
created_at: 2026-04-24
---

# SKILL-005 · Graph Database Engineering (Neo4j / ArangoDB)

## Mô tả

Thiết kế ontology, xây dựng và tối ưu hóa Knowledge Graph. Chuyên sâu về Cypher query language, graph schema design, và tích hợp graph DB vào pipeline.

## Agents sử dụng

- `knowledge-graph-agent`
- `traceability-validator-agent`

## Tài liệu liên kết

- `services/knowledge-graph-service/docs/`

---

## Năng lực cốt lõi

### 1. Ontology Design

```cypher
// Core Node Types
(:Actor)       // Người dùng, hệ thống, external party
(:Action)      // Hành động / use case
(:Entity)      // Business object (Order, Product)
(:Constraint)  // Business rule, limitation
(:Screen)      // UI screen
(:Component)   // UI component
(:Requirement) // Source requirement text
(:Document)    // Source document

// Core Relationships
(Actor)-[:PERFORMS]->(Action)
(Action)-[:OPERATES_ON]->(Entity)
(Action)-[:REQUIRES]->(Constraint)
(Requirement)-[:DEFINES]->(Action)
(Document)-[:CONTAINS]->(Requirement)
(Action)-[:RENDERED_AS]->(Screen)
(Screen)-[:CONTAINS]->(Component)
(Screen)-[:TRACED_TO]->(Requirement)
```

### 2. Idempotent MERGE Pattern (quan trọng nhất)

```cypher
// KHÔNG dùng CREATE — LUÔN dùng MERGE để tránh duplicate
MERGE (a:Actor {id: $actor_id})
ON CREATE SET a.name = $name, a.type = $type, a.created_at = datetime()
ON MATCH SET  a.name = $name, a.type = $type, a.updated_at = datetime()

// Batch upsert với UNWIND (performance cho nhiều nodes)
UNWIND $actors AS actor
MERGE (a:Actor {id: actor.id})
ON CREATE SET a.name = actor.name, a.type = actor.type, a.created_at = datetime()
ON MATCH SET  a.name = actor.name, a.type = actor.type, a.updated_at = datetime()
```

### 3. Index & Constraint Design

```cypher
// Unique constraints (tự động tạo index)
CREATE CONSTRAINT actor_id_unique IF NOT EXISTS FOR (a:Actor) REQUIRE a.id IS UNIQUE;
CREATE CONSTRAINT action_id_unique IF NOT EXISTS FOR (a:Action) REQUIRE a.id IS UNIQUE;
CREATE CONSTRAINT entity_id_unique IF NOT EXISTS FOR (e:Entity) REQUIRE e.id IS UNIQUE;

// Additional indexes
CREATE INDEX actor_name_idx IF NOT EXISTS FOR (a:Actor) ON (a.name);
CREATE FULLTEXT INDEX req_text_search IF NOT EXISTS FOR (r:Requirement) ON EACH [r.text];
```

### 4. Complex Traversal Queries

```cypher
// Subgraph cho một document
MATCH (doc:Document {id: $doc_id})-[:CONTAINS]->(req)-[:DEFINES]->(action)
OPTIONAL MATCH (actor)-[:PERFORMS]->(action)
OPTIONAL MATCH (action)-[:OPERATES_ON]->(entity)
RETURN doc, req, action, actor, entity

// Traceability: Screen → Requirements
MATCH (screen:Screen {id: $screen_id})-[:TRACED_TO]->(req)<-[:CONTAINS]-(doc)
RETURN screen.name, collect(req.text), doc.name
```

### 5. Golang Integration

```go
// Connection với pooling
driver, _ := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(user, pass, ""),
    func(c *neo4j.Config) {
        c.MaxConnectionPoolSize = 50
        c.ConnectionAcquisitionTimeout = 10 * time.Second
    },
)

// Idempotent upsert
func (r *GraphRepo) UpsertActor(ctx context.Context, actor *Actor) error {
    session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
    defer session.Close(ctx)
    _, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
        return tx.Run(ctx, `
            MERGE (a:Actor {id: $id})
            ON CREATE SET a.name = $name, a.type = $type, a.created_at = datetime()
            ON MATCH SET  a.name = $name, a.type = $type, a.updated_at = datetime()
        `, map[string]any{"id": actor.ID, "name": actor.Name, "type": actor.Type})
    })
    return err
}
```

### 6. Graph Traversal Algorithms

```cypher
// Connected components
CALL gds.wcc.stream('myGraph') YIELD nodeId, componentId
RETURN componentId, count(nodeId) AS size ORDER BY size DESC

// PageRank — tìm nodes quan trọng nhất
CALL gds.pageRank.stream('myGraph') YIELD nodeId, score
MATCH (n) WHERE id(n) = nodeId
RETURN n.name, score ORDER BY score DESC LIMIT 10
```

---

## Anti-patterns cần tránh

```cypher
-- ❌ Gây duplicate
CREATE (a:Actor {id: $id, name: $name})

-- ❌ String concatenation (injection risk)
MATCH (a:Actor {name: '"+userInput+"'})

-- ❌ Không có LIMIT trên production
MATCH (n) RETURN n

-- ✅ Đúng: MERGE + parameterized + LIMIT
MERGE (a:Actor {id: $id}) SET a.name = $name
MATCH (n:Actor) RETURN n LIMIT 100
```

## Checklist

- [ ] Mọi node type có `id` UUID + unique constraint
- [ ] Dùng MERGE thay vì CREATE cho tất cả upsert
- [ ] Index tạo cho các properties hay dùng trong WHERE
- [ ] Batch operations dùng UNWIND
- [ ] Cypher queries đều parameterized
- [ ] Graph schema document tại `services/knowledge-graph-service/docs/`
