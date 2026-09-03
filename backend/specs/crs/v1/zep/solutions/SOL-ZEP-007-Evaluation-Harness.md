# Solution: SOL-ZEP-007 — Evaluation Harness & Benchmarking

**CR ID:** CR-ZEP-007  
**Solution ID:** SOL-ZEP-007  
**Status:** Draft  
**Date:** 2026-06-17  
**Author:** Antigravity AI  

---

## 1. Tóm tắt Giải pháp

Tạo mới `tools/eval-harness/` — Python toolset gồm 5 CLI scripts với 4-bước eval pipeline (Search → Context Eval → Generate → Grade). Hỗ trợ combinatorial evaluation và LoCoMo/LongMemEval benchmarks. Dùng tenacity cho exponential backoff retry.

---

## 2. Cấu trúc Tool

```
tools/eval-harness/
├── src/
│   ├── ingest_users.py          # Ingest users + conversations
│   ├── chunk_documents.py       # Chunk + LLM-contextualize documents
│   ├── ingest_documents.py      # Ingest document chunks into graph
│   ├── evaluate.py              # 4-step eval pipeline
│   ├── graph_inspect.py         # Print graph nodes/edges
│   ├── benchmark_locomo.py      # LoCoMo benchmark runner
│   ├── benchmark_longmemeval.py # LongMemEval benchmark runner
│   ├── pipeline/
│   │   ├── search.py            # Step 1: Search VNP Memory
│   │   ├── context_eval.py      # Step 2: LLM Context Evaluation (COMPLETE/PARTIAL/INSUFFICIENT)
│   │   ├── generate.py          # Step 3: Generate response with context
│   │   ├── grade.py             # Step 4: LLM Grade answer (CORRECT/WRONG)
│   │   └── retry.py             # Exponential backoff decorator
│   ├── metrics/
│   │   ├── completeness.py      # Completeness metric
│   │   ├── accuracy.py          # Accuracy metric
│   │   └── aggregator.py        # Aggregate + format results
│   └── config/
│       ├── snapshot.py          # Config snapshotting per run
│       └── schema.py            # Config schema (pydantic)
├── runs/                        # Auto-created, gitignored
│   ├── users/
│   ├── chunk_sets/
│   ├── documents/
│   └── evaluations/
├── datasets/
│   ├── locomo/                  # LoCoMo dataset files
│   └── longmemeval/             # LongMemEval dataset files
├── pyproject.toml
├── Makefile
└── README.md
```

---

## 3. Evaluation Pipeline (4 Bước)

### 3.1. Shared Types

```python
# tools/eval-harness/src/pipeline/types.py

from __future__ import annotations
from enum import Enum
from dataclasses import dataclass, field
from datetime import datetime


class ContextCompleteness(str, Enum):
    COMPLETE = "COMPLETE"
    PARTIAL = "PARTIAL"
    INSUFFICIENT = "INSUFFICIENT"

class AnswerAccuracy(str, Enum):
    CORRECT = "CORRECT"
    WRONG = "WRONG"
    UNKNOWN = "UNKNOWN"  # LLM couldn't determine

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

### 3.2. Step 1: Search

```python
# tools/eval-harness/src/pipeline/search.py

import time
from vnp_memory import AsyncVNPMemory
from .types import TestCase
from .retry import with_retry


@with_retry(max_attempts=8)
async def search_context(
    client: AsyncVNPMemory,
    test_case: TestCase,
    retrieval_limit: int = 10,
    reranker: str = "rrf",
) -> tuple[str, int]:
    """Step 1: Retrieve context from VNP Memory for a test question."""
    start = time.monotonic()

    context_resp = await client.thread.get_user_context(
        thread_id=f"eval_{test_case.user_id}",
    )

    # Also do graph search for additional facts
    graph_resp = await client.graph.search(
        user_id=test_case.user_id,
        query=test_case.question,
        scope="edges",
        reranker=reranker,
        limit=retrieval_limit,
    )

    # Combine context + search results
    facts_text = "\n".join(
        f"- {r.fact.fact}" for r in graph_resp.items if r.fact
    )
    full_context = f"{context_resp.context}\n\nADDITIONAL FACTS:\n{facts_text}"

    latency_ms = int((time.monotonic() - start) * 1000)
    return full_context, latency_ms
```

### 3.3. Step 2: Context Evaluation (LLM Judge)

```python
# tools/eval-harness/src/pipeline/context_eval.py

from openai import AsyncOpenAI
from .types import TestCase, ContextCompleteness
from .retry import with_retry


CONTEXT_EVAL_PROMPT = """You are evaluating whether a retrieved context contains sufficient information to answer a question.

Question: {question}
Expected answer: {golden_answer}
Retrieved context:
{context}

Rate the context completeness:
- COMPLETE: Context contains all information needed to answer the question correctly
- PARTIAL: Context has some relevant info but may lead to incomplete answer
- INSUFFICIENT: Context lacks key information needed to answer

Respond with ONLY one word: COMPLETE, PARTIAL, or INSUFFICIENT."""


@with_retry(max_attempts=8)
async def evaluate_context(
    llm: AsyncOpenAI,
    test_case: TestCase,
    context: str,
    model: str = "gpt-4o-mini",  # cheaper model for eval
) -> ContextCompleteness:
    """Step 2: LLM judges context completeness."""
    response = await llm.chat.completions.create(
        model=model,
        messages=[{
            "role": "user",
            "content": CONTEXT_EVAL_PROMPT.format(
                question=test_case.question,
                golden_answer=test_case.golden_answer,
                context=context,
            )
        }],
        temperature=0,
        max_tokens=10,
    )
    verdict = response.choices[0].message.content.strip().upper()
    try:
        return ContextCompleteness(verdict)
    except ValueError:
        return ContextCompleteness.INSUFFICIENT  # fallback
```

### 3.4. Step 3: Generate Response

```python
# tools/eval-harness/src/pipeline/generate.py

from openai import AsyncOpenAI
from .retry import with_retry


GENERATE_PROMPT = """Use the following context to answer the question as accurately as possible.

Context:
{context}

Question: {question}

Answer:"""


@with_retry(max_attempts=8)
async def generate_response(
    llm: AsyncOpenAI,
    question: str,
    context: str,
    model: str = "gpt-4o",
) -> str:
    """Step 3: Generate answer using retrieved context."""
    response = await llm.chat.completions.create(
        model=model,
        messages=[{
            "role": "user",
            "content": GENERATE_PROMPT.format(context=context, question=question)
        }],
        temperature=0,
        max_tokens=200,
    )
    return response.choices[0].message.content.strip()
```

### 3.5. Step 4: Grade Answer

```python
# tools/eval-harness/src/pipeline/grade.py

from openai import AsyncOpenAI
from .types import TestCase, AnswerAccuracy
from .retry import with_retry


GRADE_PROMPT = """Compare the generated answer to the golden answer.

Question: {question}
Golden answer: {golden_answer}
Generated answer: {generated_answer}

Is the generated answer correct? Consider it CORRECT if it captures the key facts,
even if worded differently. Respond with ONLY: CORRECT or WRONG."""


@with_retry(max_attempts=8)
async def grade_answer(
    llm: AsyncOpenAI,
    test_case: TestCase,
    generated_answer: str,
    model: str = "gpt-4o-mini",
) -> AnswerAccuracy:
    """Step 4: LLM judge grades the generated answer."""
    response = await llm.chat.completions.create(
        model=model,
        messages=[{
            "role": "user",
            "content": GRADE_PROMPT.format(
                question=test_case.question,
                golden_answer=test_case.golden_answer,
                generated_answer=generated_answer,
            )
        }],
        temperature=0,
        max_tokens=10,
    )
    verdict = response.choices[0].message.content.strip().upper()
    try:
        return AnswerAccuracy(verdict)
    except ValueError:
        return AnswerAccuracy.UNKNOWN
```

### 3.6. Retry with Exponential Backoff

```python
# tools/eval-harness/src/pipeline/retry.py

from tenacity import (
    retry,
    stop_after_attempt,
    wait_exponential,
    retry_if_exception_type,
    before_sleep_log,
)
import logging
import openai

logger = logging.getLogger(__name__)

# For LLM API calls: max 8 retries, 4s → 300s
def with_retry(max_attempts: int = 8):
    return retry(
        stop=stop_after_attempt(max_attempts),
        wait=wait_exponential(multiplier=1, min=4, max=300),
        retry=retry_if_exception_type((
            openai.RateLimitError,
            openai.APITimeoutError,
            openai.APIConnectionError,
        )),
        before_sleep=before_sleep_log(logger, logging.WARNING),
        reraise=True,
    )
```

---

## 4. Main Evaluate Script

```python
# tools/eval-harness/src/evaluate.py

import asyncio
import argparse
import json
import time
from pathlib import Path
from datetime import datetime

import asyncio
from openai import AsyncOpenAI

from vnp_memory import AsyncVNPMemory
from pipeline.search import search_context
from pipeline.context_eval import evaluate_context
from pipeline.generate import generate_response
from pipeline.grade import grade_answer
from pipeline.types import EvalResult
from metrics.aggregator import aggregate_results
from config.snapshot import save_config_snapshot


async def evaluate_single(
    client: AsyncVNPMemory,
    llm: AsyncOpenAI,
    test_case,
    config: dict,
) -> EvalResult:
    """Run 4-step evaluation pipeline for a single test case."""
    try:
        # Step 1: Search
        context, search_latency = await search_context(
            client, test_case,
            retrieval_limit=config["retrieval_limit"],
            reranker=config["reranker"],
        )

        # Step 2: Context Evaluation
        completeness = await evaluate_context(llm, test_case, context)

        # Step 3: Generate
        generated = await generate_response(llm, test_case.question, context)

        # Step 4: Grade
        accuracy = await grade_answer(llm, test_case, generated)

        return EvalResult(
            case_id=test_case.case_id,
            user_id=test_case.user_id,
            category=test_case.category,
            question=test_case.question,
            golden_answer=test_case.golden_answer,
            retrieved_context=context,
            generated_answer=generated,
            completeness=completeness,
            accuracy=accuracy,
            latency_ms=search_latency,
        )

    except Exception as e:
        return EvalResult(
            case_id=test_case.case_id,
            user_id=test_case.user_id,
            category=test_case.category,
            question=test_case.question,
            golden_answer=test_case.golden_answer,
            retrieved_context="",
            generated_answer="",
            completeness=ContextCompleteness.INSUFFICIENT,
            accuracy=AnswerAccuracy.WRONG,
            latency_ms=0,
            error=str(e),
        )


async def main(args: argparse.Namespace) -> None:
    client = AsyncVNPMemory(api_key=args.api_key)
    llm = AsyncOpenAI()

    # Load test cases from user run manifest
    test_cases = load_test_cases(args.user_run, args.doc_run)
    config = build_config(args)

    # Create run directory
    run_id = datetime.now().strftime("%Y%m%d_%H%M%S")
    run_dir = Path(f"runs/evaluations/{run_id}")
    run_dir.mkdir(parents=True)

    # Save config snapshot for reproducibility
    save_config_snapshot(run_dir / "config.json", config)

    # Run evaluations with configurable concurrency
    semaphore = asyncio.Semaphore(args.concurrency)
    
    async def bounded_eval(tc):
        async with semaphore:
            return await evaluate_single(client, llm, tc, config)

    results = await asyncio.gather(*[bounded_eval(tc) for tc in test_cases])

    # Aggregate metrics
    summary = aggregate_results(results)

    # Save results
    output = {
        "run_id": run_id,
        "config": config,
        "total_cases": len(results),
        "aggregate_scores": summary,
        "results": [r.__dict__ for r in results],
    }
    (run_dir / "results.json").write_text(json.dumps(output, indent=2, default=str))
    print(f"✅ Evaluation complete. Results: {run_dir}/results.json")
    print_summary(summary)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--api-key", required=True)
    parser.add_argument("--user-run", type=int, required=True)
    parser.add_argument("--doc-run", type=int, default=None)
    parser.add_argument("--concurrency", type=int, default=30)
    parser.add_argument("--retrieval-limit", type=int, default=10)
    parser.add_argument("--reranker", default="rrf",
                        choices=["rrf","mmr","cross_encoder","node_distance","episode_mentions"])
    asyncio.run(main(parser.parse_args()))
```

---

## 5. Metrics Aggregator

```python
# tools/eval-harness/src/metrics/aggregator.py

from collections import defaultdict
from pipeline.types import EvalResult, ContextCompleteness, AnswerAccuracy


def aggregate_results(results: list[EvalResult]) -> dict:
    total = len(results)
    if total == 0:
        return {}

    # Overall completeness
    completeness_counts = defaultdict(int)
    accuracy_counts = defaultdict(int)

    # Per-category breakdown
    category_completeness = defaultdict(lambda: defaultdict(int))
    category_accuracy = defaultdict(lambda: defaultdict(int))

    # Per-user breakdown
    user_accuracy = defaultdict(lambda: defaultdict(int))

    for r in results:
        completeness_counts[r.completeness] += 1
        accuracy_counts[r.accuracy] += 1
        category_completeness[r.category][r.completeness] += 1
        category_accuracy[r.category][r.accuracy] += 1
        user_accuracy[r.user_id][r.accuracy] += 1

    def rates(counts, total):
        return {
            "complete_rate":      counts[ContextCompleteness.COMPLETE] / total if ContextCompleteness.COMPLETE in counts else counts.get("CORRECT", 0) / total,
            "partial_rate":       counts.get(ContextCompleteness.PARTIAL, 0) / total,
            "insufficient_rate":  counts.get(ContextCompleteness.INSUFFICIENT, 0) / total,
        }

    return {
        "aggregate_scores": {
            "completeness": {
                "complete_rate":     completeness_counts[ContextCompleteness.COMPLETE] / total,
                "partial_rate":      completeness_counts[ContextCompleteness.PARTIAL] / total,
                "insufficient_rate": completeness_counts[ContextCompleteness.INSUFFICIENT] / total,
            },
            "accuracy": {
                "accuracy_rate": accuracy_counts[AnswerAccuracy.CORRECT] / total,
                "error_rate":    accuracy_counts[AnswerAccuracy.WRONG] / total,
            },
        },
        "category_scores": {
            category: {
                "complete_rate":     counts[ContextCompleteness.COMPLETE] / sum(counts.values()),
                "partial_rate":      counts[ContextCompleteness.PARTIAL] / sum(counts.values()),
            }
            for category, counts in category_completeness.items()
        },
        "user_scores": {
            user_id: {
                "accuracy_rate": counts[AnswerAccuracy.CORRECT] / sum(counts.values()),
            }
            for user_id, counts in user_accuracy.items()
        },
    }
```

---

## 6. Config Snapshot

```python
# tools/eval-harness/src/config/snapshot.py

import json
import platform
import subprocess
from pathlib import Path
from datetime import datetime


def save_config_snapshot(path: Path, config: dict) -> None:
    """Save config + environment snapshot for reproducibility."""
    snapshot = {
        "timestamp": datetime.now().isoformat(),
        "git_commit": _get_git_commit(),
        "python_version": platform.python_version(),
        "config": config,
    }
    path.write_text(json.dumps(snapshot, indent=2))

def _get_git_commit() -> str:
    try:
        return subprocess.check_output(
            ["git", "rev-parse", "--short", "HEAD"],
            text=True
        ).strip()
    except Exception:
        return "unknown"
```

---

## 7. Benchmark CLI

```bash
# LoCoMo benchmark (temporal reasoning)
python benchmark_locomo.py \
    --api-key ${VNP_MEMORY_API_KEY} \
    --dataset datasets/locomo/locomo_v1.json \
    --retrieval-limit 10 \
    --reranker rrf \
    --concurrency 20

# LongMemEval benchmark (memory retention)
python benchmark_longmemeval.py \
    --api-key ${VNP_MEMORY_API_KEY} \
    --dataset datasets/longmemeval/test.json \
    --edge-limit 20 \
    --node-limit 10 \
    --concurrency 15

# Combinatorial evaluation (compare configs)
python ingest_users.py --run-id 1 --custom-ontology
python ingest_users.py --run-id 2 --default-ontology
python evaluate.py --user-run 1 --doc-run 1 --reranker rrf
python evaluate.py --user-run 2 --doc-run 1 --reranker mmr
```

---

## 8. Lộ trình Triển khai

| Phase | Nội dung | Ước tính |
|-------|---------|---------|
| **P1** | Tool skeleton + pyproject.toml + config schema | 1 ngày |
| **P2** | Pipeline Steps 1-4 với retry | 2 ngày |
| **P3** | Metrics aggregator (completeness + accuracy + category breakdown) | 1 ngày |
| **P4** | Config snapshot + run directory management | 0.5 ngày |
| **P5** | ingest_users.py + ingest_documents.py | 1.5 ngày |
| **P6** | evaluate.py main script với concurrency | 1 ngày |
| **P7** | LoCoMo benchmark runner | 1 ngày |
| **P8** | LongMemEval benchmark runner | 1 ngày |
| **P9** | graph_inspect.py + tests | 1 ngày |

**Tổng:** ~10 ngày (Wave 6)

---

## 9. Acceptance Criteria Mapping

| AC | Giải pháp |
|----|-----------|
| evaluate.py 100 test cases → results.json đúng format | 4-step pipeline + aggregator |
| COMPLETE/PARTIAL/INSUFFICIENT per case | ContextCompleteness enum + LLM judge |
| Config snapshot per run | save_config_snapshot() vào run_dir |
| Rate limit → auto retry, không crash | tenacity với stop_after_attempt(8) |
| --concurrency 30 OK | asyncio.Semaphore(30) |
| LoCoMo benchmark runner | benchmark_locomo.py với dataset loading |
| Custom vs default ontology → metric difference | --user-run 1 vs --user-run 2 comparison |
