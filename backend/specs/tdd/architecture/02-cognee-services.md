# Cognee Ingestion Service

> **Service**: `cognee-ingestion` | **gRPC Port**: 9011 | **Origin**: Cognee L2-L4

---

## 1. Responsibility

Multi-modal data ingestion pipeline: file upload, text extraction, URL scraping, dataset management. Tổ chức data theo dataset/namespace per tenant.

## 2. gRPC API

```protobuf
service CogneeIngestionService {
  rpc CreateDataset(CreateDatasetRequest) returns (Dataset);
  rpc DeleteDataset(DeleteDatasetRequest) returns (Empty);
  rpc AddData(stream AddDataRequest) returns (AddDataResponse);
  rpc AddText(AddTextRequest) returns (AddTextResponse);
  rpc AddUrl(AddUrlRequest) returns (AddUrlResponse);
  rpc GetDatasetStatus(GetDatasetStatusRequest) returns (DatasetStatus);
  rpc ListDatasets(ListDatasetsRequest) returns (ListDatasetsResponse);
}
```

## 3. Use Cases

| Use Case | Description | Output |
|----------|-------------|--------|
| `IngestFile` | Upload + extract text (PDF/DOCX/PPTX/CSV) | Metadata + raw text stored |
| `IngestText` | Direct text input | Text stored in dataset |
| `IngestUrl` | Web scraping + extraction | Extracted content stored |
| `ManageDataset` | CRUD dataset per tenant | Dataset lifecycle |

## 4. Domain Model

```go
type Dataset struct {
    ID        uuid.UUID
    TenantID  string
    Name      string
    Status    DatasetStatus  // PENDING, READY, COGNIFYING, ERROR
    FileCount int
    CreatedAt time.Time
}

type DataItem struct {
    ID        uuid.UUID
    DatasetID uuid.UUID
    Source    DataSource  // FILE, TEXT, URL
    MimeType string
    RawText  string
    Metadata map[string]string
}
```

## 5. NATS Events

| Event | Payload | Subscriber |
|-------|---------|------------|
| `cognee.data.ingested` | `{dataset_id, tenant_id, item_ids[]}` | cognee-cognify |

## 6. Storage

- **PostgreSQL**: Dataset metadata, data items
- **MinIO/S3**: Raw file storage
- **Redis**: Upload progress cache

---

# Cognee Cognify Service

> **Service**: `cognee-cognify` | **gRPC Port**: 9012 | **Origin**: Cognee L3-L5

---

## 1. Responsibility

Knowledge graph construction pipeline: classify → chunk → extract entities → build graph → embed. Core LLM-intensive processing service.

## 2. gRPC API

```protobuf
service CogneeCognifyService {
  rpc TriggerCognify(TriggerCognifyRequest) returns (CognifyJob);
  rpc GetJobStatus(GetJobStatusRequest) returns (CognifyJob);
  rpc CancelJob(CancelJobRequest) returns (Empty);
}
```

## 3. Pipeline Stages

```
classify → chunk → extract_entities → extract_relationships
    → deduplicate → build_graph(Neo4j) → embed(Qdrant)
    → summarize_communities
```

## 4. LLM Integration

| Stage | LLM Call | Model |
|-------|---------|-------|
| Classify | Content type detection | Fast model |
| Extract entities | NER + relation extraction | GPT-4o / custom |
| Deduplicate | Entity resolution | GPT-4o-mini |
| Summarize | Community summaries | GPT-4o-mini |

## 5. Storage

- **Neo4j**: Knowledge graph nodes + edges
- **Qdrant**: Entity/chunk embeddings
- **PostgreSQL**: Job status, pipeline state

---

# Cognee Search Service

> **Service**: `cognee-search` | **gRPC Port**: 9013 | **Origin**: Cognee L5

---

## 1. Responsibility

15 retrieval strategies over knowledge graph + vector store. RAG completion with LLM.

## 2. gRPC API

```protobuf
service CogneeSearchService {
  rpc Search(SearchRequest) returns (SearchResponse);
  rpc RAGComplete(RAGRequest) returns (RAGResponse);
  rpc GetChunks(GetChunksRequest) returns (ChunksResponse);
}
```

## 3. Search Strategies

| # | Strategy | Source | Description |
|---|----------|--------|-------------|
| 1 | SIMILARITY | Qdrant | Vector cosine similarity |
| 2 | GRAPH_COMPLETION | Neo4j+LLM | Graph traverse + LLM answer |
| 3 | RAG_COMPLETION | Qdrant+LLM | Traditional RAG |
| 4 | NATURAL_LANGUAGE | Neo4j | NL → Cypher query |
| 5 | CHUNKS | Qdrant | Raw chunk retrieval |
| 6 | LEXICAL | BM25 | Keyword/exact match |
| 7-15 | (other strategies) | Various | Domain-specific retrievers |

## 4. Storage

- **Qdrant**: Vector similarity search
- **Neo4j**: Graph traversal
- **Redis**: Search result cache (TTL 5min)
