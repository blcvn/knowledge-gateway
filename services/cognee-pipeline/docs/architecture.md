# cognee-pipeline — Architecture

> **Pattern**: Pipeline Merge (Ingestion + Processing → Single Binary)

## Internal Layer Structure

```
services/cognee-pipeline/
├── internal/
│   ├── domain/
│   │   ├── ingestion/      # Dataset, DataItem, DataSource entities
│   │   └── cognify/        # CognifyJob, PipelineStage, Ontology entities
│   ├── usecase/
│   │   ├── ingest/         # IngestFile, IngestText, IngestUrl, ManageDataset
│   │   └── cognify/        # TriggerCognify (LOCAL call from ingest), 7 pipeline stages
│   ├── adapter/
│   │   ├── grpc/
│   │   │   ├── ingestion_handler.go   # CogneeIngestionService (proto unchanged)
│   │   │   └── cognify_handler.go     # CogneeCognifyService (proto unchanged)
│   │   ├── repository/
│   │   │   ├── postgres/   # Dataset, DataItem, Job tables
│   │   │   ├── neo4j/      # Knowledge graph CRUD
│   │   │   └── pgvector/   # Entity/chunk embeddings (post-Qdrant migration)
│   │   └── event/nats/     # Emit: cognee.pipeline.completed
│   └── infra/
```

## Key Design Decisions

1. **Pipeline merge**: `cognee.data.ingested` NATS event eliminated — ingestion triggers cognify via local function call
2. **Dual gRPC services**: Single binary exposes both CogneeIngestionService and CogneeCognifyService — proto backward compatible
3. **Shared DB connections**: PostgreSQL, Neo4j, and vector store connections shared across ingestion and cognify usecases

## External Dependencies

| Dependency | Purpose |
|-----------|---------|
| PostgreSQL + pgvector | Metadata storage + entity embeddings |
| Neo4j | Knowledge graph (entities, relationships, communities) |
| MinIO/S3 | Raw file object storage |
| NATS | Emit `cognee.pipeline.completed` → cognee-search |
| Bifrost (LLM) | Entity/relationship extraction, ontology classification |
