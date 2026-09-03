# TASK-ZEP-016 — tools/eval-harness: Evaluation Pipeline & Benchmarks

**Task ID:** TASK-ZEP-016  
**Wave:** 6 (Quality)  
**Solution:** [SOL-ZEP-007](../solutions/SOL-ZEP-007-Evaluation-Harness.md)  
**Depends on:** TẤT CẢ waves trước (full system required for evaluation)  
**Ước tính:** 5h  
**Priority:** Medium — quality measurement

**Trạng thái:** ⏳ Pending  
**Ghi chú:** Evaluation harness not implemented  
---

## Mục tiêu

Tạo `tools/eval-harness/` — Python CLI toolset cho evaluation:
1. **4-step pipeline**: Search → Context Eval → Generate → Grade
2. **Metrics**: COMPLETE/PARTIAL/INSUFFICIENT + CORRECT/WRONG
3. **Config snapshotting**: mỗi run có config.json để reproduce
4. **Exponential backoff retry**: tenacity, max 8 attempts, 4s→300s
5. **Concurrency**: configurable `--concurrency N` với asyncio.Semaphore
6. **Benchmarks**: LoCoMo + LongMemEval runners

---

## Công việc cụ thể

### 1. Setup Package

**`tools/eval-harness/pyproject.toml`**:
```toml
[project]
name = "vnp-eval-harness"
version = "1.0.0"
requires-python = ">=3.10"
dependencies = [
    "openai>=1.0.0",
    "vnp-memory>=1.0.0",
    "tenacity>=8.0.0",
    "aiohttp>=3.0.0",
    "pydantic>=2.0.0",
]
```

### 2. Tạo Core Pipeline Modules

#### `src/pipeline/types.py`
```python
from enum import Enum
from dataclasses import dataclass, field

class ContextCompleteness(str, Enum):
    COMPLETE = "COMPLETE"
    PARTIAL = "PARTIAL"
    INSUFFICIENT = "INSUFFICIENT"

class AnswerAccuracy(str, Enum):
    CORRECT = "CORRECT"
    WRONG = "WRONG"
    UNKNOWN = "UNKNOWN"

@dataclass
class TestCase:
    case_id: str
    user_id: str
    question: str
    golden_answer: str
    category: str  # "temporal_reasoning" | "entity_relations" | "preference_recall"
    session_ids: list[str] = field(default_factory=list)

@dataclass
class EvalResult:
    case_id: str
    user_id: str
    category: str
    question: str
    golden_answer: str
    retrieved_context: str
    generated_answer: str
    completeness: ContextCompleteness
    accuracy: AnswerAccuracy
    latency_ms: int
    error: str | None = None
```

#### `src/pipeline/retry.py`
```python
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type
import openai

# Max 8 attempts, wait 4s → 300s (5 min max delay)
def with_retry(max_attempts: int = 8):
    return retry(
        stop=stop_after_attempt(max_attempts),
        wait=wait_exponential(multiplier=1, min=4, max=300),
        retry=retry_if_exception_type((
            openai.RateLimitError,
            openai.APITimeoutError,
            openai.APIConnectionError,
        )),
        reraise=True,
    )
```

#### `src/pipeline/search.py` — Step 1
```python
@with_retry(max_attempts=8)
async def search_context(client, test_case, retrieval_limit=10, reranker="rrf") -> tuple[str, int]:
    """Retrieve context from VNP Memory (get_user_context + graph search)."""
```

#### `src/pipeline/context_eval.py` — Step 2
```python
CONTEXT_EVAL_PROMPT = """...""" # LLM judge prompt
@with_retry(max_attempts=8)
async def evaluate_context(llm, test_case, context, model="gpt-4o-mini") -> ContextCompleteness:
    """LLM judges: COMPLETE | PARTIAL | INSUFFICIENT"""
```

#### `src/pipeline/generate.py` — Step 3
```python
@with_retry(max_attempts=8)
async def generate_response(llm, question, context, model="gpt-4o") -> str:
    """Generate answer using retrieved context."""
```

#### `src/pipeline/grade.py` — Step 4
```python
@with_retry(max_attempts=8)
async def grade_answer(llm, test_case, generated_answer, model="gpt-4o-mini") -> AnswerAccuracy:
    """LLM judge: CORRECT | WRONG"""
```

### 3. Tạo Metrics Aggregator

**`src/metrics/aggregator.py`**:
```python
def aggregate_results(results: list[EvalResult]) -> dict:
    """Aggregate results into:
    {
      "aggregate_scores": {
        "completeness": {"complete_rate": 0.87, "partial_rate": 0.10, "insufficient_rate": 0.03},
        "accuracy": {"accuracy_rate": 0.82, "error_rate": 0.18}
      },
      "category_scores": {"temporal_reasoning": {...}, ...},
      "user_scores": {"user_alice": {...}, ...}
    }
    """
```

### 4. Tạo Config Snapshot

**`src/config/snapshot.py`**:
```python
def save_config_snapshot(path: Path, config: dict) -> None:
    """Save config + git commit + python version for reproducibility."""
    snapshot = {
        "timestamp": datetime.now().isoformat(),
        "git_commit": _get_git_commit(),  # git rev-parse --short HEAD
        "python_version": platform.python_version(),
        "config": config,
    }
```

### 5. Tạo Main `evaluate.py` Script

```python
# evaluate.py CLI flags:
#   --api-key        VNP_MEMORY_API_KEY
#   --user-run N     integer run ID from ingest_users
#   --doc-run N      integer run ID from ingest_documents (optional)
#   --concurrency 30 parallel evaluations (default 30)
#   --retrieval-limit 10
#   --reranker rrf|mmr|cross_encoder|...

# Flow:
# 1. Load test cases từ runs/users/{N}/test_cases.json
# 2. Create run directory: runs/evaluations/{timestamp}/
# 3. save_config_snapshot(run_dir / "config.json", config)
# 4. asyncio.gather(*[bounded_eval(tc) for tc in test_cases])
# 5. aggregate_results() + save results.json
# 6. Print summary table
```

### 6. Tạo Benchmark Runners

**`src/benchmark_locomo.py`**:
```python
# LoCoMo benchmark — temporal reasoning across multi-session conversations
# Dataset: datasets/locomo/
# CLI: python benchmark_locomo.py --api-key KEY --retrieval-limit 10 --reranker rrf

# Loads LoCoMo dataset, creates TestCase objects with temporal questions
# e.g. "Where did Alice work in 2022?"
```

**`src/benchmark_longmemeval.py`**:
```python
# LongMemEval benchmark — 500 multi-session conversations
# Dataset: datasets/longmemeval/
# CLI: python benchmark_longmemeval.py --edge-limit 20 --node-limit 10
```

### 7. Tests

**`tests/test_pipeline.py`**:
- `test_context_eval_complete`: COMPLETE when context has answer
- `test_context_eval_insufficient`: INSUFFICIENT when context empty
- `test_grade_correct`: generated=golden → CORRECT
- `test_grade_wrong`: wrong answer → WRONG
- `test_aggregate_completeness_rate`: 10 results → correct rates
- `test_retry_on_rate_limit`: RateLimitError → retries (mock)
- `test_semaphore_limits_concurrency`: semaphore(3) → max 3 concurrent

---

## Output Format Specification

**`runs/evaluations/{timestamp}/results.json`**:
```json
{
  "run_id": "20260617_150000",
  "config": {"retrieval_limit": 10, "reranker": "rrf", ...},
  "total_cases": 100,
  "aggregate_scores": {
    "completeness": {"complete_rate": 0.87, ...},
    "accuracy": {"accuracy_rate": 0.82, ...}
  },
  "category_scores": {...},
  "user_scores": {...},
  "results": [
    {
      "case_id": "tc_001",
      "completeness": "COMPLETE",
      "accuracy": "CORRECT",
      "latency_ms": 150,
      ...
    }
  ]
}
```

---

## Acceptance Criteria

- [ ] `python evaluate.py --help` hiển thị tất cả flags
- [ ] Run 10 test cases → `results.json` với đúng format
- [ ] `completeness` có 3 possible values: COMPLETE/PARTIAL/INSUFFICIENT
- [ ] `accuracy` có 2 values: CORRECT/WRONG (+ UNKNOWN)
- [ ] Rate limit error → auto retry tối đa 8 lần, không crash
- [ ] `--concurrency 30` → asyncio.Semaphore(30) hoạt động
- [ ] Config snapshot lưu git commit + python version
- [ ] LoCoMo runner loads dataset và runs evaluations
- [ ] `pytest tests/` pass

---

## Files tạo ra

```
tools/eval-harness/
├── src/
│   ├── pipeline/
│   │   ├── types.py
│   │   ├── retry.py
│   │   ├── search.py
│   │   ├── context_eval.py
│   │   ├── generate.py
│   │   └── grade.py
│   ├── metrics/
│   │   └── aggregator.py
│   ├── config/
│   │   └── snapshot.py
│   ├── evaluate.py
│   ├── ingest_users.py
│   ├── ingest_documents.py
│   ├── chunk_documents.py
│   ├── graph_inspect.py
│   ├── benchmark_locomo.py
│   └── benchmark_longmemeval.py
├── tests/
│   └── test_pipeline.py
├── datasets/
│   ├── locomo/.gitkeep
│   └── longmemeval/.gitkeep
├── runs/.gitignore
└── pyproject.toml
```

## Sau khi hoàn thành

```bash
cd tools/eval-harness
pytest tests/
python src/evaluate.py --help
```
