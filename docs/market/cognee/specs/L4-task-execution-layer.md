# L4 — Task Execution Layer

> **Layer**: 4 (Processing Units)  
> **Responsibility**: Implement atomic, composable pipeline tasks  
> **Dependencies**: L5 (Domain Modules), L6 (Infrastructure Adapters)  
> **Path**: `cognee/tasks/`

---

## 1. Tổng Quan

Layer 4 chứa các **hàm async độc lập** (atomic tasks) — mỗi hàm thực hiện một bước xử lý cụ thể trong pipeline. Các task được L3 Pipeline Orchestration gọi tuần tự, output task N → input task N+1.

---

## 2. Task Directory Map

| Directory | Số files | Chức năng chính |
|-----------|----------|----------------|
| `tasks/ingestion/` | 8+ | Data ingestion: resolve paths, extract content, DLT sources |
| `tasks/documents/` | 5+ | Document classification, chunk extraction |
| `tasks/chunks/` | 3+ | Chunk processing, merging, filtering |
| `tasks/graph/` | 6+ | Knowledge graph extraction from text |
| `tasks/storage/` | 4+ | Persist data points to graph + vector DBs |
| `tasks/summarization/` | 3+ | LLM-powered text summarization |
| `tasks/completion/` | 3+ | LLM completion generation |
| `tasks/entity_completion/` | 2+ | Entity-level completion |
| `tasks/temporal_awareness/` | 2+ | Time-aware processing |
| `tasks/temporal_graph/` | 3+ | Temporal event extraction + graph building |
| `tasks/memify/` | 2+ | Graph enrichment tasks |
| `tasks/schema/` | 2+ | Schema extraction |
| `tasks/translation/` | 2+ | Text translation |
| `tasks/web_scraper/` | 3+ | Web scraping + content extraction |
| `tasks/cleanup/` | 2+ | Data cleanup |
| `tasks/codingagents/` | 2+ | Code-specific processing |

---

## 3. Core Pipeline Tasks (Chi tiết)

### 3.1 Ingestion Tasks (`tasks/ingestion/`)

#### `resolve_data_directories(data)`
- Resolve file paths, URLs, S3 paths thành actual content
- Handle directory traversal (`include_subdirectories`)
- Output: List of resolved data items

#### `ingest_data(data, dataset_name, user, ...)`
- Extract text content từ various file formats
- Save data items to storage (relational DB)
- Create Dataset + Data records
- Output: List of Data records

#### `resolve_dlt_sources(data, dataset_name, user)`
- Expand DLT resources, CSV files, connection strings thành standard DataItems
- Auto-detect data format

#### `DataItem` model
- Structured data wrapper: content, content_type, metadata

---

### 3.2 Document Tasks (`tasks/documents/`)

#### `classify_documents(data_documents)`
- LLM classifies document type: Text, Audio, Image, Video, Multimedia, 3D, Procedural
- Each type has 50+ sub-classifications (xem `DefaultContentPrediction` model)
- Output: Typed Document objects

#### `extract_chunks_from_documents(documents, max_chunk_size, chunker)`
- Split documents into semantic text chunks
- Chunker options: `TextChunker` (paragraph-based), `LangchainChunker` (recursive character splitting)
- `max_chunk_size` auto-calculated: `min(embedding_max, llm_max // 2)`
- Output: List of text chunks with metadata

---

### 3.3 Graph Extraction Tasks (`tasks/graph/`)

#### `extract_graph_and_summarize(chunks, graph_model, config, custom_prompt)`
- Combined task: entity extraction + summarization
- LLM extracts entities và relationships theo `graph_model` schema
- Uses **Instructor** hoặc **BAML** cho structured output
- Ontology grounding: match entities against OWL ontology
- Summarize each chunk cho hierarchical retrieval
- Output: KnowledgeGraph objects (nodes + edges)

#### `extract_graph_from_data(chunks, graph_model)`
- Standalone graph extraction (without summarization)
- LLM → entities (Node) + relationships (Edge) theo schema

---

### 3.4 Storage Tasks (`tasks/storage/`)

#### `add_data_points(data_points, embed_triplets)`
- Persist nodes và edges to Graph DB
- Embed text content → vectors → store in Vector DB
- Optional: embed triplets (subject-predicate-object) cho triplet search
- Supports batched writing (`task_config={"batch_size": N}`)

---

### 3.5 Temporal Tasks (`tasks/temporal_graph/`)

#### `extract_events_and_timestamps(chunks)`
- Extract temporal events, dates, timestamps from text chunks
- Output: Event objects with temporal metadata

#### `extract_knowledge_graph_from_events(events)`
- Build time-aware knowledge graph from extracted events
- Temporal relationships between events

---

### 3.6 Summarization Tasks (`tasks/summarization/`)

#### `summarize_text(chunks)`
- LLM-powered hierarchical summarization
- Multiple levels: chunk → section → document → corpus

---

### 3.7 Web Scraper Tasks (`tasks/web_scraper/`)

- Web content extraction via BeautifulSoup hoặc Tavily API
- CSS selector / XPath rules cho targeted extraction
- Concurrent crawling with configurable delay

---

### 3.8 DLT Foreign Key Tasks

#### `extract_dlt_fk_edges(data)`
- Extract foreign key-based relationships cho tabular data
- Auto-detect FK constraints → create graph edges

---

## 4. Task Composition Pattern

Tasks are **pure functions** — no side effects beyond their explicit I/O:

```python
# Input: list of data items from previous task
# Output: list of processed items for next task
async def my_task(input_data, param1=None, param2=None):
    results = []
    for item in input_data:
        processed = await process(item, param1, param2)
        results.append(processed)
    return results

# Generator variant (streaming)
async def my_streaming_task(input_data):
    for item in input_data:
        yield await process(item)
```

---

## 5. Default Pipeline Sequences

### 5.1 ADD Pipeline
```
resolve_data_directories → ingest_data
```

### 5.2 COGNIFY Pipeline (Default)
```
classify_documents
  → extract_chunks_from_documents
  → extract_graph_and_summarize
  → add_data_points
  → extract_dlt_fk_edges
```

### 5.3 COGNIFY Pipeline (Temporal)
```
classify_documents
  → extract_chunks_from_documents
  → extract_events_and_timestamps
  → extract_knowledge_graph_from_events
  → add_data_points
```

---

## 6. Key Design Decisions

1. **Composability** — Tasks can be reordered, replaced, or augmented without changing pipeline logic
2. **Batching** — `task_config={"batch_size": N}` controls how many items are processed per batch
3. **Generator support** — Tasks can yield results incrementally (streaming processing)
4. **Drop sentinel** — Tasks can filter items by returning `Drop`
5. **Enrichment mode** — `enriches=True` tasks pass-through input when output is None
