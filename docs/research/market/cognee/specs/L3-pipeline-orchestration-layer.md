# L3 — Pipeline Orchestration Layer

> **Layer**: 3 (Workflow Engine)  
> **Responsibility**: Orchestrate multi-step data processing pipelines  
> **Dependencies**: L4 (Task Execution)  
> **Path**: `cognee/modules/pipelines/`

---

## 1. Tổng Quan

Layer 3 là **workflow engine** của Cognee. Nó nhận danh sách tasks từ L2, quản lý thứ tự thực thi, batching, caching, và execution mode (blocking vs background).

```
┌────────────────────────────────────────────────────────┐
│              Pipeline Orchestration Layer               │
├──────────┬──────────┬──────────┬───────────────────────┤
│  Task    │ Pipeline │ Pipeline │ Pipeline              │
│  System  │ Runner   │ Queue    │ Layers                │
├──────────┼──────────┼──────────┼───────────────────────┤
│ Task     │ run_     │ queue    │ resolve_authorized_   │
│ BoundTask│ pipeline │ manager  │   user_dataset        │
│ TaskSpec │          │          │ reset_pipeline_status │
│ task()   │          │          │ pipeline_execution_   │
│          │          │          │   mode                │
└──────────┴──────────┴──────────┴───────────────────────┘
```

---

## 2. Task System

### 2.1 Class Hierarchy

```
task() (decorator/factory)
  └─ TaskSpec (callable wrapper)
       └─ BoundTask (pre-bound kwargs, ready for pipeline)
            └─ Task (core executor)
```

### 2.2 `Task` Class

**File**: `cognee/modules/pipelines/tasks/task.py`

Core executor class hỗ trợ 4 loại callable:

| Task Type | Detection | Execution Method |
|-----------|-----------|-----------------|
| `Async Generator` | `inspect.isasyncgenfunction` | `execute_async_generator` |
| `Generator` | `inspect.isgeneratorfunction` | `execute_generator` |
| `Coroutine` | `inspect.iscoroutinefunction` | `execute_coroutine` |
| `Function` | `inspect.isfunction` | `execute_function` |

**Key properties**:
- `executable` — underlying callable
- `task_config` — `{"batch_size": N}` for batched processing
- `default_params` — pre-bound `args` and `kwargs`
- `enriches: bool` — nếu True, return input khi output là None
- `accepts_ctx: bool` — có nhận `PipelineContext` parameter hay không

**Execution**:
```python
async def execute(self, args, kwargs, next_batch_size=None):
    batch_size = next_batch_size if next_batch_size is not None else 1
    async for result in self._execute_method(args, kwargs, batch_size):
        yield result
```

Batching logic: kết quả được gom thành batch theo `batch_size` trước khi yield cho task tiếp theo.

### 2.3 `TaskSpec` Class

Callable wrapper trả về bởi `@task` decorator:

```python
@task(batch_size=20)
async def extract_graph(chunks, graph_model=None):
    ...

# Calling TaskSpec returns BoundTask (NOT execution)
bound = extract_graph(graph_model=KnowledgeGraph)  # → BoundTask

# Direct execution for testing
result = await extract_graph.direct(chunks, graph_model=KnowledgeGraph)
```

### 2.4 `BoundTask` Class

Pre-bound kwargs ready for pipeline chaining:

```python
bound = BoundTask(inner_task=Task(...), graph_model=KnowledgeGraph)
# bound.task → Task instance
# bound.kwargs → {"graph_model": KnowledgeGraph}
```

### 2.5 `task()` Factory/Decorator

```python
# As decorator
@task
async def classify(data): ...

@task(batch_size=20)
async def extract(chunks, graph_model=None): ...

# As functional wrapper
extract_task = task(extract_graph_existing, batch_size=20)
```

### 2.6 `task_summary()` Decorator

Attach human-readable summary template cho logging:

```python
@task_summary("Classified {n} document(s)")
async def classify_documents(data_documents):
    ...
```

---

## 3. Pipeline Runner

### 3.1 `run_pipeline()`

**File**: `cognee/modules/pipelines/` (operations directory)

Nhận list of Tasks, chạy tuần tự, truyền output của task N làm input cho task N+1.

**Parameters**:
- `tasks: list[Task | BoundTask]` — ordered list of pipeline steps
- `datasets: list[UUID]` — dataset IDs to process
- `data` — initial input data
- `user: User` — execution context
- `pipeline_name: str` — cho tracking/caching
- `vector_db_config`, `graph_db_config` — optional per-pipeline DB config
- `use_pipeline_cache: bool` — skip already-processed datasets
- `incremental_loading: bool` — skip already-processed data items
- `data_per_batch: int` — batch size cho initial data

### 3.2 Pipeline Execution Modes

**File**: `cognee/modules/pipelines/layers/pipeline_execution_mode.py`

```python
pipeline_executor_func = get_pipeline_executor(run_in_background=True)

result = await pipeline_executor_func(
    pipeline=run_pipeline,
    tasks=tasks,
    datasets=datasets,
    ...
)
```

| Mode | Behavior |
|------|----------|
| `run_in_background=False` | Blocking — wait for completion |
| `run_in_background=True` | Fire-and-forget — return immediately |

---

## 4. Pipeline Layers

**Path**: `cognee/modules/pipelines/layers/`

Pre-pipeline và post-pipeline hooks:

| Layer | Chức năng |
|-------|-----------|
| `resolve_authorized_user_dataset` | Resolve/create user + dataset, assign permissions |
| `reset_dataset_pipeline_run_status` | Clear previous pipeline run states |
| `pipeline_execution_mode` | Determine blocking vs background execution |

---

## 5. Pipeline Queue

**Path**: `cognee/modules/pipelines/queues/`

Queue management cho concurrent pipeline executions. Đảm bảo không có 2 pipeline chạy đồng thời trên cùng dataset.

---

## 6. Pipeline Models

**Path**: `cognee/modules/pipelines/models/`

Data models cho pipeline tracking:

- **PipelineRun** — execution record (status, timestamps, run_id)
- **PipelineRunStatus** — enum: pending, running, completed, failed

---

## 7. Pipeline Methods

**Path**: `cognee/modules/pipelines/methods/`

Utility functions:
- `get_pipeline_run()` — lookup by run ID
- `get_pipeline_run_by_dataset()` — lookup by dataset
- `get_pipeline_runs_by_dataset()` — list all runs
- `reset_pipeline_run_status()` — reset state

---

## 8. Drop Sentinel

**File**: `cognee/pipelines/types.py`

`_Drop` sentinel class — khi một task yields `Drop`, item bị loại bỏ khỏi pipeline:

```python
from cognee.pipelines import Drop

async def filter_task(item):
    if should_skip(item):
        return Drop  # item removed from pipeline
    return item
```

---

## 9. Usage Pattern (from L2)

```python
# In cognify():
tasks = [
    Task(classify_documents),
    Task(extract_chunks_from_documents, max_chunk_size=512, chunker=TextChunker),
    Task(extract_graph_and_summarize, graph_model=KnowledgeGraph, 
         task_config={"batch_size": 100}),
    Task(add_data_points, embed_triplets=True, task_config={"batch_size": 100}),
    Task(extract_dlt_fk_edges),
]

pipeline_executor_func = get_pipeline_executor(run_in_background=False)
result = await pipeline_executor_func(
    pipeline=run_pipeline,
    tasks=tasks,
    datasets=datasets,
    pipeline_name="cognify_pipeline",
)
```
