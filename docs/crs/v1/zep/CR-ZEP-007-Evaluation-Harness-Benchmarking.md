# Change Request: CR-ZEP-007 — Evaluation Harness & Benchmarking

**CR ID:** CR-ZEP-007  
**Component:** `tools/eval-harness/` [NEW TOOL]  
**Priority:** Medium  
**Status:** In Progress
**Reference:** Zep PRD §6.3 F8-F9, SRS §5.4, URD §3.4  
**Benchmarks:** LoCoMo, LongMemEval

---

## 1. Mô tả

Xây dựng **Evaluation Harness** — công cụ đo lường chất lượng context retrieval của VNP Memory:

1. **End-to-End Pipeline**: Search → Context Evaluation → Generate → Grade.
2. **Two-Level Metrics**: Context Completeness (COMPLETE/PARTIAL/INSUFFICIENT) + Answer Accuracy (CORRECT/WRONG).
3. **Config Snapshotting**: Mỗi run lưu config snapshot để reproducible.
4. **Decoupled Ingestion**: Tách ingest users và ingest documents → combinatorial evaluation.
5. **Resilience**: Exponential backoff (max 8 retries) cho rate limits.
6. **LoCoMo + LongMemEval**: Support 2 benchmark datasets chuẩn ngành.

---

## 2. Vấn đề hiện tại

- VNP Memory chưa có hệ thống đo lường chất lượng retrieval tự động.
- Không biết context assembl đã đủ đầy (completeness) hay chưa.
- Thiếu khả năng so sánh các cấu hình ontology/search khác nhau.

---

## 3. Thay đổi đề xuất

### 3.1. [NEW] `tools/eval-harness/`

### 3.2. Pipeline Scripts

| Script | Mục đích | Output |
|--------|---------|-------|
| `ingest_users.py` | Ingest users + conversations + telemetry | `runs/users/{N}/manifest.json` |
| `chunk_documents.py` | Chunk + LLM-contextualize documents | `runs/chunk_sets/{N}/chunks.jsonl` |
| `ingest_documents.py` | Ingest document chunks vào graph | `runs/documents/{N}/manifest.json` |
| `evaluate.py` | Search → Context eval → Generate → Grade | `runs/evaluations/{N}/results.json` |
| `graph_inspect.py` | Print graph nodes và edges | stdout |

### 3.3. Evaluation Pipeline (4 bước)

```
1. SEARCH
   → Query VNP Memory với test question
   → Retrieve context block

2. CONTEXT EVALUATION (LLM Judge)
   → LLM đánh giá context:
   → COMPLETE | PARTIAL | INSUFFICIENT

3. GENERATE RESPONSE
   → LLM generate answer dùng retrieved context

4. GRADE ANSWER (LLM Judge)
   → So sánh với golden answer:
   → CORRECT | WRONG
```

### 3.4. Metrics Structure

```json
{
  "aggregate_scores": {
    "completeness": {
      "complete_rate": 0.87,
      "partial_rate": 0.10,
      "insufficient_rate": 0.03
    },
    "accuracy": {
      "accuracy_rate": 0.82,
      "error_rate": 0.18
    }
  },
  "category_scores": {
    "temporal_reasoning": { "complete_rate": 0.91 },
    "entity_relations": { "complete_rate": 0.85 },
    "preference_recall": { "complete_rate": 0.88 }
  },
  "user_scores": {
    "user_alice": { "accuracy_rate": 0.90 },
    "user_bob": { "accuracy_rate": 0.75 }
  }
}
```

### 3.5. Combinatorial Evaluation

```bash
# Tách ingest từ evaluate → mix-match configs
$ python ingest_users.py --custom-ontology    # Run 1: users with custom ontology
$ python ingest_users.py --default-ontology   # Run 2: users with default ontology

$ python chunk_documents.py --chunk-size 500  # ChunkSet 1
$ python chunk_documents.py --chunk-size 1000 # ChunkSet 2

# Evaluate mọi combination
$ python evaluate.py --user-run 1 --doc-run 1   # custom ontology + small chunks
$ python evaluate.py --user-run 1 --doc-run 2   # custom ontology + large chunks
$ python evaluate.py --user-run 2 --doc-run 1   # default ontology + small chunks
```

### 3.6. Resilience

```python
# Exponential backoff cho rate limits
@retry(
    stop=stop_after_attempt(8),
    wait=wait_exponential(multiplier=1, min=4, max=300),  # max 5 min delay
    retry=retry_if_exception_type(RateLimitError)
)
async def evaluate_with_retry(test_case: TestCase): ...

# Configurable concurrency
$ python evaluate.py --concurrency 30  # 30 parallel evaluations
```

### 3.7. Benchmark Support

| Benchmark | Dataset | Mô tả |
|-----------|---------|-------|
| **LoCoMo** | Long-Context Multi-Session Conversations | Test temporal reasoning qua nhiều sessions |
| **LongMemEval** | 500 multi-session conversations | Test memory retention sau long history |

```bash
# Run LoCoMo benchmark
$ python benchmark_locomo.py --retrieval-limit 10 --reranker rrf

# Run LongMemEval benchmark
$ python benchmark_longmemeval.py --edge-limit 20 --node-limit 10
```

---

## 4. Acceptance Criteria

- [ ] `evaluate.py` chạy với 100 test cases → output `results.json` đúng format.
- [ ] Completeness metric: `COMPLETE/PARTIAL/INSUFFICIENT` per test case.
- [ ] Config snapshot lưu trong mỗi run directory → reproduce bất kỳ run nào.
- [ ] Rate limit xảy ra → auto retry với exponential backoff, không crash.
- [ ] `--concurrency 30` hoạt động đúng (30 parallel evals, không race condition).
- [ ] LoCoMo benchmark runner hoạt động với dataset chuẩn.
- [ ] So sánh kết quả: custom ontology vs default → metric difference rõ ràng.
