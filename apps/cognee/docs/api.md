# REST API Reference — Cognee App

> **Base URL**: `http://localhost:8080` | **Format**: JSON | **Auth**: JWT Bearer or X-Tenant-ID header (dev mode)

## Response Format

### Success
```json
{
  "data": { ... },
  "meta": { "request_id": "..." }
}
```

### Error
```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable description"
  }
}
```

---

## Ingestion Endpoints

### POST `/api/v1/cognee/add`

Ingest text or URL content into a dataset.

**Request Body:**
```json
{
  "dataset_id": "uuid",
  "text": "Content to ingest",
  "source_name": "my-doc",
  "metadata": { "key": "value" }
}
```

Or for URL:
```json
{
  "dataset_id": "uuid",
  "url": "https://example.com/article"
}
```

**Response (201):**
```json
{
  "data": {
    "item_id": "uuid",
    "dataset_id": "uuid",
    "source": "text",
    "size_bytes": 1234,
    "text_preview": "First 200 chars...",
    "is_duplicate": false
  }
}
```

### GET `/api/v1/cognee/datasets`

List datasets for the current tenant.

**Query Params:** `cursor` (string), `limit` (int, default 50)

**Response (200):**
```json
{
  "data": {
    "datasets": [...],
    "next_cursor": "..."
  }
}
```

### DELETE `/api/v1/cognee/datasets/{id}`

Delete a dataset and all its data items.

**Response (200):**
```json
{ "data": { "status": "deleted" } }
```

---

## Cognify Endpoints

### POST `/api/v1/cognee/cognify`

Start the 8-stage knowledge graph construction pipeline.

**Request Body:**
```json
{
  "dataset_id": "uuid",
  "chunk_size": 1024,
  "chunk_overlap": 128,
  "skip_dedup": false,
  "skip_summarize": false,
  "ontology_id": "optional-uuid"
}
```

**Response (202):**
```json
{
  "data": {
    "job_id": "uuid",
    "dataset_id": "uuid",
    "status": "running",
    "metrics": { ... },
    "duration_ms": 0
  }
}
```

### GET `/api/v1/cognee/cognify/{id}/status`

Get the status of a cognification job.

**Response (200):** Returns the full `CognifyJob` object with stage progress and metrics.

---

## Search Endpoints

### POST `/api/v1/cognee/search`

Execute multi-strategy search.

**Request Body:**
```json
{
  "query": "What is knowledge graph?",
  "strategies": ["similarity", "chunks", "cypher"],
  "top_k": 10,
  "rerank": true
}
```

**Response (200):**
```json
{ "data": { "results": [...] } }
```

### GET `/api/v1/cognee/search/explore`

Explore graph neighborhood of a node.

**Query Params:** `node_id` (string), `depth` (int, default 2)

### POST `/api/v1/cognee/search/rag`

RAG (Retrieval-Augmented Generation) completion.

**Request Body:**
```json
{
  "query": "Explain memory types",
  "strategies": ["similarity"],
  "top_k": 5
}
```

**Response (200):**
```json
{
  "data": {
    "answer": "Generated answer...",
    "sources": [...]
  }
}
```

---

## Health Endpoints

### GET `/healthz`
Liveness probe. Returns `200` if process is running.

### GET `/readyz`
Readiness probe. Returns `200` if all dependencies are healthy, `503` otherwise.
